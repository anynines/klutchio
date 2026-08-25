/*
Copyright 2024 Klutch Authors. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package servicebinding

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	osbclient "github.com/anynines/klutchio/clients/a9s-open-service-broker"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/anynines/klutchio/provider-anynines/apis/servicebinding/v1"
	dsv1 "github.com/anynines/klutchio/provider-anynines/apis/serviceinstance/v1"
	apisv1 "github.com/anynines/klutchio/provider-anynines/apis/v1"
	util "github.com/anynines/klutchio/provider-anynines/internal/controller/utils"
	client "github.com/anynines/klutchio/provider-anynines/pkg/client/osb"
	"github.com/anynines/klutchio/provider-anynines/pkg/constants"
	utilerr "github.com/anynines/klutchio/provider-anynines/pkg/utilerr"
	utils "github.com/anynines/klutchio/provider-anynines/pkg/utils"
)

const (
	// AnnotationKeyServiceBindingCreated is used to check that servicebinding has been
	// created or not.
	AnnotationKeyServiceBindingCreated = "anynines.crossplane.io/servicebinding-created"

	serviceBindingStatusCreated  = "Created"
	serviceBindingStatusDeleting = "Deleting"
	// serviceBindingStatusUnbound is set in AtProvider.State after a successful Unbind
	// call. Because AtProvider is part of the status subresource it is persisted by
	// Crossplane's Status().Update() call that follows Delete(). This prevents
	// re-calling Unbind on subsequent reconciliations while the broker asynchronously
	// finishes deleting the binding.
	serviceBindingStatusUnbound = "Unbound"
)

const (
	errNotServiceBinding = utilerr.PlainErr("something went wrong with crossplane as managed resource reconciled is not a ServiceBinding custom resource, THIS SHOULD NOT HAPPEN")
	// errServiceBindingIsUnset is the message of the error that is triggered when the status field
	// of servicebinding is unset
	errServiceBindingIsUnset   = utilerr.PlainUserErr("servicebinding status field is unset, setting required values")
	errInstanceNotReady        = utilerr.PlainUserErr("service instance is not ready")
	errNoSuchDataservice       = utilerr.PlainUserErr("referenced data service not found.")
	errServiceInstanceNotFound = utilerr.PlainUserErr("data service instance was not found")
	errNewClient               = "cannot create new Service"
)

var (
	errTrackPCUsage         = utilerr.FromStr("cannot track ProviderConfig usage")
	errGetPC                = utilerr.FromStr("cannot get ProviderConfig")
	errDeleteServiceBinding = utilerr.FromStr("failed to delete ServiceBinding")
)

// Setup adds a controller that reconciles ServiceBinding managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1.ServiceBindingGroupKind)
	cps := util.GetConnectionPublisher(mgr, o)
	log := o.Logger.WithValues("controller", name)
	logConnec := getExternalConnector(mgr, log)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1.ServiceBindingGroupVersionKind),
		managed.WithExternalConnecter(logConnec),
		managed.WithLogger(log),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1.ServiceBinding{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         k8sclient.Client
	usage        resource.Tracker
	newServiceFn func(username, password []byte, url string, insecureSkipVerify bool, caBundle []byte, overrideServerName string) (osbclient.Client, error)
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	sb, err := getServiceBindingFromResource(mg)
	if err != nil {
		return nil, err
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errTrackPCUsage.WithCause(err)
	}

	pc := &apisv1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: sb.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errGetPC.WithCause(err)
	}

	credentials, err := util.GetCredentialsFromProvider(ctx, pc, c.kube)
	if err != nil {
		return nil, err
	}

	svc, err := c.newServiceFn(credentials.Username, credentials.Password, pc.Spec.Url, credentials.InsecureSkipVerify, credentials.CABundle, credentials.OverrideServerName)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errNewClient, err)
	}

	return &external{
		service: svc,
		kube:    c.kube,
	}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	// A 'client' used to connect to the external resource API. In practice this
	// would be something like an AWS SDK client.
	service osbclient.Client

	// we need a k8s client in the external struct in order to retrieve the MRs
	// for Service Instances in order to resolve the Instance names into instance IDs
	kube k8sclient.Client
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	sb, err := getServiceBindingFromResource(mg)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	if obs, done, err := c.observeInitialization(ctx, sb); done {
		return obs, err
	}

	isDeleting := sb.DeletionTimestamp != nil

	if obs, done, err := c.observeUnboundState(ctx, sb, isDeleting); done {
		return obs, err
	}

	return c.observeBindingState(ctx, sb, isDeleting)
}

// resourceGoneDuringDeletion cleans up the servicebinding secret and reports
// the external resource as gone. Used by Observe when a deletion-in-progress
// resource can no longer be reconciled against the broker.
func (c *external) resourceGoneDuringDeletion(ctx context.Context, sb *v1.ServiceBinding) (managed.ExternalObservation, error) {
	if err := c.deleteServiceBindingSecret(ctx, sb); err != nil {
		return managed.ExternalObservation{}, errDeleteServiceBinding.WithCause(err)
	}
	return managed.ExternalObservation{ResourceExists: false}, nil
}

// observeInitialization populates the servicebinding status if needed and
// determines whether Observe should return immediately (done=true).
func (c *external) observeInitialization(ctx context.Context, sb *v1.ServiceBinding) (obs managed.ExternalObservation, done bool, err error) {
	if err := c.initializeServiceBindingStatus(ctx, sb); err != nil {
		// While deleting, only treat the binding as gone once we're sure there's
		// nothing left at the broker. Anything else (e.g. a transient kube-apiserver
		// error) gets propagated so Crossplane retries instead of leaking the binding.
		if sb.DeletionTimestamp != nil && confirmedNothingToUnbind(sb, err) {
			obs, err := c.resourceGoneDuringDeletion(ctx, sb)
			return obs, true, err
		}
		return managed.ExternalObservation{}, true, err
	}

	if sb.Annotations == nil || sb.Annotations[AnnotationKeyServiceBindingCreated] == "" {
		// Initiate creation of SB
		return managed.ExternalObservation{}, true, nil
	}

	return managed.ExternalObservation{}, false, nil
}

// confirmedNothingToUnbind reports whether it's safe to give up on the broker
// binding: either it was never created (no AnnotationKeyServiceBindingCreated,
// so Bind() was never called), or the ServiceInstance/data service it depends
// on is confirmed permanently gone. Any other error is unproven and must be
// retried instead, or a still-live binding could be leaked at the broker.
func confirmedNothingToUnbind(sb *v1.ServiceBinding, err error) bool {
	if sb.Annotations[AnnotationKeyServiceBindingCreated] != "true" {
		return true
	}
	return errors.Is(err, errServiceInstanceNotFound) || errors.Is(err, errNoSuchDataservice)
}

// observeUnboundState reports the resource as gone if Unbind was already
// called (State==Unbound persisted via Status().Update()), so Crossplane
// skips Delete() and proceeds to UnpublishConnection + RemoveFinalizer.
func (c *external) observeUnboundState(ctx context.Context, sb *v1.ServiceBinding, isDeleting bool) (obs managed.ExternalObservation, done bool, err error) {
	if isDeleting && sb.Status.AtProvider.State == serviceBindingStatusUnbound {
		obs, err := c.resourceGoneDuringDeletion(ctx, sb)
		return obs, true, err
	}
	return managed.ExternalObservation{}, false, nil
}

// observeBindingState retrieves the binding from the broker and reports
// whether it exists and is up to date.
func (c *external) observeBindingState(ctx context.Context, sb *v1.ServiceBinding, isDeleting bool) (managed.ExternalObservation, error) {
	// set deletion condition if MR is marked for deletion
	sb.SetDeletionStatusIfNotDeleted(serviceBindingStatusDeleting)

	// Get binding
	bindResponse, err := c.service.GetBinding(&osbclient.GetBindingRequest{
		InstanceID: sb.Status.AtProvider.InstanceID,
		BindingID:  string(sb.UID),
	})
	switch {
	case err != nil && isDeleting:
		// Resource does not exist and is marked for deletion
		return c.resourceGoneDuringDeletion(ctx, sb)
	case err != nil:
		return managed.ExternalObservation{}, fmt.Errorf("failed to get service binding: %w", err)
	}

	exists := bindResponse != nil && bindResponse.Credentials != nil
	if exists && !isDeleting {
		// do not set Available if servicebinding is being deleted
		sb.Status.SetConditions(xpv1.Available())
		sb.Status.AtProvider.State = serviceBindingStatusCreated
	}

	return managed.ExternalObservation{
		// Return false when the external resource does not exist. This lets
		// the managed resource reconciler know that it needs to call Create to
		// (re)create the resource, or that it has successfully been deleted.
		ResourceExists: exists,

		// Return false when the external resource exists, but it not up to date
		// with the desired managed resource state. This lets the managed
		// resource reconciler know that it needs to call Update.
		ResourceUpToDate: true,

		// Return any details that may be required to connect to the external
		// resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	sb, err := getServiceBindingFromResource(mg)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	bindReq := &osbclient.BindRequest{
		// Using the serviceBinding UID provided by Kubernetes as the BindingID
		// may result in collisions.
		BindingID:         string(sb.UID),
		InstanceID:        sb.Status.AtProvider.InstanceID,
		AcceptsIncomplete: sb.Spec.ForProvider.AcceptsIncomplete,
		ServiceID:         sb.Status.AtProvider.ServiceID,
		PlanID:            sb.Status.AtProvider.PlanID,
	}

	resp, err := c.service.Bind(bindReq)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	meta.AddAnnotations(sb, map[string]string{
		AnnotationKeyServiceBindingCreated: "true",
	})

	cd, err := generateConnectionDetails(resp)
	return managed.ExternalCreation{ConnectionDetails: cd}, err
}

func (c *external) GetServiceInstanceManagedResource(ctx context.Context, sb v1.ServiceBinding) (*dsv1.ServiceInstance, error) {
	// Get ServiceInstance Managed Resource
	instances := &dsv1.ServiceInstanceList{}

	// In Crossplane v2 (Namespaced XRs), composed MRs carry the label
	// crossplane.io/composite: <xr-name> instead of the v1 claim labels.
	labelSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{
			constants.LabelKeyComposite: sb.Spec.ForProvider.InstanceName,
		},
	})
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create label selector for "+
				"ServiceInstance managed resources: %w",
			err)
	}

	err = c.kube.List(ctx, instances, &k8sclient.ListOptions{
		LabelSelector: labelSelector,
		Namespace:     sb.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf(
			"failed to list ServiceInstance managed resource: %w",
			err)
	}

	return instances.ToServiceInstance(sb.Name)
}

// GetServiceBindingSecret retrieves the servicebinding secret whose location is
// set by the composition via spec.writeConnectionSecretToRef.
func (c *external) GetServiceBindingSecret(ctx context.Context, sb v1.ServiceBinding) (*corev1.Secret, error) {
	secretRef := sb.Spec.WriteConnectionSecretToReference
	if secretRef == nil {
		return nil, fmt.Errorf("ServiceBinding has no writeConnectionSecretToRef set")
	}

	secret := &corev1.Secret{}
	err := c.kube.Get(ctx, types.NamespacedName{
		Name:      secretRef.Name,
		Namespace: secretRef.Namespace,
	}, secret)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get ServiceBinding secret: %w",
			err)
	}

	return secret, nil
}

// initializeServiceBindingStatus initializes the status of ServiceBinding.
func (c *external) initializeServiceBindingStatus(ctx context.Context, sb *v1.ServiceBinding) error {
	if !sb.Status.AtProvider.HasMissingFields() &&
		sb.ConnectionDetailsIsNotEmpty() {
		return nil
	}

	// Populate ConnectionDetails
	if sb.Annotations[AnnotationKeyServiceBindingCreated] == "true" {
		err := c.initializeConnectionDetails(ctx, sb)
		if err != nil {
			return err
		}
	} else if !sb.Status.AtProvider.HasMissingFields() {
		return nil
	}

	err := c.initializeInstanceFields(ctx, sb)
	if err != nil {
		return err
	}

	return errServiceBindingIsUnset
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	fmt.Println("Updating Service Bindings is not supported!")

	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	sb, err := getServiceBindingFromResource(mg)
	if err != nil {
		return managed.ExternalDelete{}, err
	}

	sb.Status.SetConditions(xpv1.Deleting())

	// Always try to refresh instance fields before Unbind to pick up any plan
	// upgrades that happened after the binding was created. A stale PlanID causes
	// the broker to return 400 "Mismatch between the provided plan ID".
	// If the refresh fails (e.g. instance is mid-upgrade/deploying), fall back to
	// the cached values already in atProvider rather than blocking deletion entirely.
	if err := c.initializeInstanceFields(ctx, sb); err != nil {
		if sb.Status.AtProvider.HasMissingFields() {
			return managed.ExternalDelete{}, err
		}
		// Cached values are present — proceed with them.
	}

	deleteReq := &osbclient.UnbindRequest{
		BindingID:         string(sb.UID),
		InstanceID:        sb.Status.AtProvider.InstanceID,
		AcceptsIncomplete: sb.Spec.ForProvider.AcceptsIncomplete,
		ServiceID:         sb.Status.AtProvider.ServiceID,
		PlanID:            sb.Status.AtProvider.PlanID,
	}

	// TODO: handle response from client
	_, err = c.service.Unbind(deleteReq)
	if err != nil {
		return managed.ExternalDelete{}, errDeleteServiceBinding.WithCause(err)
	}

	if err := c.deleteServiceBindingSecret(ctx, sb); err != nil {
		return managed.ExternalDelete{}, errDeleteServiceBinding.WithCause(err)
	}

	// Mark that Unbind was called. Setting AtProvider.State persists via the
	// Status().Update() call that the managed reconciler makes after Delete() returns,
	// so the next Observe() will see State==Unbound and return ResourceExists=false.
	sb.Status.AtProvider.State = serviceBindingStatusUnbound

	return managed.ExternalDelete{}, nil
}

func (c *external) deleteServiceBindingSecret(ctx context.Context, sb *v1.ServiceBinding) error {
	if sb.Spec.WriteConnectionSecretToReference == nil {
		return nil
	}

	secret := &corev1.Secret{}
	err := c.kube.Get(ctx, types.NamespacedName{
		Name:      sb.Spec.WriteConnectionSecretToReference.Name,
		Namespace: sb.Spec.WriteConnectionSecretToReference.Namespace,
	}, secret)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	err = c.kube.Delete(ctx, secret)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}

func (c *external) Disconnect(ctx context.Context) error {
	// Unimplemented, required by newer versions of crossplane-runtime
	return nil
}

func generateConnectionDetails(res *osbclient.BindResponse) (managed.ConnectionDetails, error) {
	if len(res.Credentials) == 0 {
		return nil, fmt.Errorf("the service broker returned no credentials for service binding")
	}

	connDetails := utils.FlattenMap(res.Credentials, "")
	utils.ReplaceRootKeyWithNestedKey(connDetails)
	return connDetails, nil
}

// parseHostAndPort parses an input string in the format "host:port".
// It separates the host and port by finding the last occurrence of ':'.
// Returns the extracted host and port as separate strings.
func (c *external) parseHostAndPort(input string) (host, port string, err error) {
	if strings.Contains(input, ":") {
		index := strings.LastIndex(input, ":")
		host = input[:index]
		port = input[index+1:]
		return host, port, nil
	}
	return "", "", fmt.Errorf("invalid host:port format: %q", input)
}

func (c *external) extractBracketHost(sb *v1.ServiceBinding, secret map[string][]byte, key, label string) error {
	host, found := secret[key]
	if !found {
		return fmt.Errorf("%q field not found in secret", key)
	}
	if len(host) <= 2 || (host[0] != '[' && host[len(host)-1] != ']') {
		return fmt.Errorf("invalid host format: %q", host)
	}
	host = host[1 : len(host)-1]
	hostURL, port, err := c.parseHostAndPort(string(host))
	if err != nil {
		return err
	}
	sb.AddConnectionDetailsWithLabel(hostURL, port, label)
	return nil
}

func (c *external) extractPlainHost(sb *v1.ServiceBinding, secret map[string][]byte, key, label string) error {
	host, found := secret[key]
	if !found {
		return fmt.Errorf("%q field not found in secret", key)
	}
	if len(host) == 0 {
		return fmt.Errorf("invalid host format: %q", host)
	}
	parsedURL, err := url.Parse(string(host))
	if err != nil {
		return err
	}
	sb.AddConnectionDetailsWithLabel(parsedURL.Scheme+"://"+parsedURL.Hostname(), parsedURL.Port(), label)
	return nil
}

func (c *external) extractPrometheusURL(sb *v1.ServiceBinding, secret map[string][]byte, key string, label string) error {
	host, found := secret[key]
	if !found {
		return fmt.Errorf("%q field not found in secret", key)
	}
	if len(host) <= 2 || (host[0] != '[' && host[len(host)-1] != ']') {
		return fmt.Errorf("invalid host format: %q", host)
	}
	host = host[1 : len(host)-1]
	parsedURL, err := url.Parse(string(host))
	if err != nil {
		return err
	}
	hostURL := parsedURL.Scheme + "://" + parsedURL.Hostname()
	port := parsedURL.Port()
	sb.AddConnectionDetailsWithLabel(hostURL, port, label)
	return nil
}

func (c *external) extractGraphiteExporter(sb *v1.ServiceBinding, secret map[string][]byte) error {
	host, found := secret["graphite_exporters"]
	if !found {
		return fmt.Errorf("graphite_exporters field not found in secret")
	}
	if len(host) <= 2 || (host[0] != '[' && host[len(host)-1] != ']') {
		return fmt.Errorf("invalid host format: %q", host)
	}
	host = host[1 : len(host)-1]
	hostURL := string(host)
	port, found := secret["graphite_exporter_port"]
	if !found {
		return fmt.Errorf("graphite_exporter_port field not found in secret")
	}
	sb.AddConnectionDetailsWithLabel(hostURL, string(port), "Graphite Exporter")
	return nil
}

func (c *external) extractMessagingEndpoints(sb *v1.ServiceBinding, secret map[string][]byte) error {
	// HTTP API endpoint
	if httpAPI, found := secret["http_api_uri"]; found && len(httpAPI) > 0 {
		if err := c.parseAndAddURI(sb, string(httpAPI), "HTTP API"); err != nil {
			return err
		}
	}

	// AMQP SSL endpoint
	if amqpSSL, found := secret["protocols.amqp_ssl.uri"]; found && len(amqpSSL) > 0 {
		if err := c.parseAndAddURI(sb, string(amqpSSL), "AMQP SSL"); err != nil {
			return err
		}
	}

	// Management endpoint
	if mgmt, found := secret["protocols.management.uri"]; found && len(mgmt) > 0 {
		if err := c.parseAndAddURI(sb, string(mgmt), "Management"); err != nil {
			return err
		}
	}

	return nil
}

func (c *external) parseAndAddURI(sb *v1.ServiceBinding, uriStr string, label string) error {
	parsedURL, err := url.Parse(uriStr)
	if err != nil {
		return err
	}
	hostURL := parsedURL.Scheme + "://" + parsedURL.Hostname()
	port := parsedURL.Port()
	sb.AddConnectionDetailsWithLabel(hostURL, port, label)
	return nil
}

// serviceKind classifies an instance-type label into the connection-details
// dispatch key used by initializeConnectionDetails.
func serviceKind(instanceName string) string {
	switch {
	case strings.Contains(instanceName, "search"):
		return "search"
	case strings.Contains(instanceName, "logme2"):
		return "logme2"
	case strings.Contains(instanceName, "mongodb"):
		return "mongodb"
	case strings.Contains(instanceName, "prometheus"):
		return "prometheus"
	case strings.Contains(instanceName, "messaging"):
		return "messaging"
	case strings.Contains(instanceName, "keyvalue"):
		return "keyvalue"
	case strings.Contains(instanceName, "postgresql"), strings.Contains(instanceName, "mariadb"):
		return "sql"
	default:
		return ""
	}
}

// extractPrometheusDetails populates all Prometheus-related connection details
// (Prometheus, Alertmanager, Grafana and the Graphite Exporter).
func (c *external) extractPrometheusDetails(sb *v1.ServiceBinding, secret map[string][]byte) error {
	if err := c.extractPrometheusURL(sb, secret, "prometheus_urls", "Prometheus"); err != nil {
		return err
	}
	if err := c.extractPrometheusURL(sb, secret, "alertmanager_urls", "Alertmanager"); err != nil {
		return err
	}
	if err := c.extractPrometheusURL(sb, secret, "grafana_urls", "Grafana"); err != nil {
		return err
	}
	return c.extractGraphiteExporter(sb, secret)
}

// extractHostAndPort populates a connection detail pair from two flat secret
// fields, used by the KeyValue and SQL data services.
func (c *external) extractHostAndPort(sb *v1.ServiceBinding, secret map[string][]byte, hostKey, portKey, label string) error {
	hostURL, hostFound := secret[hostKey]
	if !hostFound {
		return fmt.Errorf("%s field not found in secret", hostKey)
	}
	port, portFound := secret[portKey]
	if !portFound {
		return fmt.Errorf("%s field not found in secret", portKey)
	}
	sb.AddConnectionDetailsWithLabel(string(hostURL), string(port), label)
	return nil
}

// connectionDetailsHandlers dispatches on the service kind (see serviceKind)
// to populate connection details from the servicebinding secret.
var connectionDetailsHandlers = map[string]func(c *external, sb *v1.ServiceBinding, secret map[string][]byte) error{
	"search": func(c *external, sb *v1.ServiceBinding, secret map[string][]byte) error {
		return c.extractBracketHost(sb, secret, "host", "Search")
	},
	"logme2": func(c *external, sb *v1.ServiceBinding, secret map[string][]byte) error {
		return c.extractPlainHost(sb, secret, "host", "Logme2")
	},
	"mongodb": func(c *external, sb *v1.ServiceBinding, secret map[string][]byte) error {
		return c.extractBracketHost(sb, secret, "hosts", "MongoDB")
	},
	"prometheus": func(c *external, sb *v1.ServiceBinding, secret map[string][]byte) error {
		return c.extractPrometheusDetails(sb, secret)
	},
	"messaging": func(c *external, sb *v1.ServiceBinding, secret map[string][]byte) error {
		return c.extractMessagingEndpoints(sb, secret)
	},
	"keyvalue": func(c *external, sb *v1.ServiceBinding, secret map[string][]byte) error {
		return c.extractHostAndPort(sb, secret, "host", "valkey.port", "KeyValue")
	},
	"sql": func(c *external, sb *v1.ServiceBinding, secret map[string][]byte) error {
		return c.extractHostAndPort(sb, secret, "host", "port", "SQL")
	},
}

// initializeConnectionDetails populates the servicebinding status with connection details
// mainly HostURl and Port.
func (c *external) initializeConnectionDetails(ctx context.Context, sb *v1.ServiceBinding) error {
	secret, err := c.GetServiceBindingSecret(ctx, *sb)
	if err != nil {
		return err
	}

	instanceName := sb.Labels["klutch.io/instance-type"]

	handler, ok := connectionDetailsHandlers[serviceKind(instanceName)]
	if !ok {
		return errNoSuchDataservice
	}

	return handler(c, sb, secret.Data)
}

// initializeInstanceFields populates the servicebinding status with service instance
// details like InstanceID, ServiceID and PlanID.
func (c *external) initializeInstanceFields(ctx context.Context, sb *v1.ServiceBinding) error {
	serviceInstance, err := c.GetServiceInstanceManagedResource(ctx, *sb)
	if err != nil {
		return err
	}

	sb.Status.AtProvider.InstanceID = serviceInstance.Status.AtProvider.InstanceID
	sb.Status.AtProvider.ServiceID = serviceInstance.Status.AtProvider.ServiceID
	sb.Status.AtProvider.PlanID = serviceInstance.Status.AtProvider.PlanID

	// Validate status
	if sb.Status.AtProvider.HasMissingFields() {
		return errInstanceNotReady
	}

	return nil
}

func getExternalConnector(mgr ctrl.Manager, log logging.Logger) utilerr.ConnectDecorator {
	connec := &connector{
		kube:         mgr.GetClient(),
		usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1.ProviderConfigUsage{}),
		newServiceFn: client.NewOsbServiceWithTLS,
	}
	logConnec := &utilerr.ConnectDecorator{
		Connector: connec,
		Logger:    log,
	}
	return *logConnec
}

func getServiceBindingFromResource(mg resource.Managed) (*v1.ServiceBinding, error) {
	sb, ok := mg.(*v1.ServiceBinding)
	if !ok {
		return nil, errNotServiceBinding
	}
	return sb, nil
}
