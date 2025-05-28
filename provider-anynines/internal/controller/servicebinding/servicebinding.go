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
	"fmt"
	"net/url"
	"strings"

	osbclient "github.com/anynines/klutchio/clients/a9s-open-service-broker"
	corev1 "k8s.io/api/core/v1"
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
	newServiceFn func(username, password []byte, url string) (osbclient.Client, error)
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

	svc, err := c.newServiceFn(credentials.Username, credentials.Password, pc.Spec.Url)
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

	err = c.initializeServiceBindingStatus(ctx, sb)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	if sb.Annotations == nil || sb.Annotations[AnnotationKeyServiceBindingCreated] == "" {
		return managed.ExternalObservation{}, nil
	}

	resp, err := c.service.GetInstances()
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("failed to get service instances: %w", err)
	}

	exists := false
	for _, instance := range resp.Resources {
		if creds := getInstanceCredentials(instance, sb); creds != nil {
			exists = true

			sb.Status.SetConditions(xpv1.Available())
			sb.Status.AtProvider.ServiceBindingID = creds.ID
			sb.Status.AtProvider.PlanID = instance.PlanGUID
			sb.Status.AtProvider.State = serviceBindingStatusCreated

			break
		}
	}

	sb.SetDeletionStatusIfNotDeleted(serviceBindingStatusDeleting)

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

func (c external) GetServiceInstanceManagedResource(ctx context.Context, sb v1.ServiceBinding) (*dsv1.ServiceInstance, error) {
	// Get ServiceInstance Managed Resource
	instances := &dsv1.ServiceInstanceList{}

	// Current assumption is that serviceBinding-claim exists in the same
	// namespace as serviceInstance-claim. This allows ServiceBindings to
	// work in the context of Consumer.
	labelSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{
			constants.LabelKeyClaimName:      sb.Spec.ForProvider.InstanceName,
			constants.LabelKeyClaimNamespace: sb.Labels[constants.LabelKeyClaimNamespace],
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
	})
	if err != nil {
		return nil, fmt.Errorf(
			"failed to list ServiceInstance managed resource: %w",
			err)
	}

	return instances.ToServiceInstance(sb.Name)
}

// GetServiceBindingSecret retrieves the servicebinding secret with postfix '-creds'.
func (c external) GetServiceBindingSecret(ctx context.Context, sb v1.ServiceBinding) (*corev1.Secret, error) {
	secret := &corev1.Secret{}

	err := c.kube.Get(ctx, types.NamespacedName{
		Name:      sb.Labels[constants.LabelKeyClaimName] + "-creds",
		Namespace: sb.Labels[constants.LabelKeyClaimNamespace],
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

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	// Unimplemented, required by newer versions of crossplane-runtime
	return nil
}

func generateConnectionDetails(res *osbclient.BindResponse) (managed.ConnectionDetails, error) {
	if len(res.Credentials) == 0 {
		return nil, fmt.Errorf("The service broker returned no credentials for service binding")
	}

	connDetails := utils.FlattenMap(res.Credentials, "")
	utils.ReplaceRootKeyWithNestedKey(connDetails)
	return connDetails, nil
}

func getInstanceCredentials(instance osbclient.GetInstanceResponse, sb *v1.ServiceBinding) *osbclient.Credential {
	if instance.GUIDAtTenant != sb.Status.AtProvider.InstanceID {
		return nil
	}

	for _, credential := range instance.Credentials {
		if credential.GUIDAtTenant == string(sb.UID) {
			return &credential
		}
	}
	return nil
}

// parseHostAndPort parses an input string in the format "host:port".
// It separates the host and port by finding the last occurrence of ':'.
// Returns the extracted host and port as separate strings.
func (c external) parseHostAndPort(input string) (host, port string, err error) {
	if strings.Contains(input, ":") {
		index := strings.LastIndex(input, ":")
		host = input[:index]
		port = input[index+1:]
		return host, port, nil
	}
	return "", "", fmt.Errorf("invalid host:port format: %q", input)
}

func (c external) extractBracketHost(sb *v1.ServiceBinding, secret map[string][]byte, key string) error {
	host, found := secret[key]
	if found && len(host) > 2 && host[0] == '[' && host[len(host)-1] == ']' {
		hostURL, port, err := c.parseHostAndPort(string(host[1 : len(host)-1]))
		if err != nil {
			return err
		}
		sb.AddConnectionDetails(hostURL, port)
	} else {
		return fmt.Errorf("invalid host format: %q", host)
	}
	return nil
}

func (c external) extractPlainHost(sb *v1.ServiceBinding, secret map[string][]byte, key string) error {
	host, found := secret[key]
	if found && len(host) > 0 {
		hostURL, port, err := c.parseHostAndPort(string(host))
		if err != nil {
			return err
		}
		sb.AddConnectionDetails(hostURL, port)
	} else {
		return fmt.Errorf("invalid host format: %q", host)
	}
	return nil
}

func (c external) extractPrometheusHost(sb *v1.ServiceBinding, secret map[string][]byte, key string, port string) error {
	host, found := secret[key]
	if found && len(host) > 2 && host[0] == '[' && host[len(host)-1] == ']' {
		if strings.Contains(key, "graphite_exporters") {
			hostURL := string(string(host[1 : len(host)-1]))
			port, found := secret[port]
			if found {
				sb.AddConnectionDetails(hostURL, string(port))
			}
		} else {
			parsedURL, err := url.Parse(string(host[1 : len(host)-1]))
			if err != nil {
				return err
			}
			hostURL, port, err := c.parseHostAndPort(parsedURL.Scheme + "://" + parsedURL.Host)
			if err != nil {
				return err
			}
			sb.AddConnectionDetails(hostURL, port)
		}
	} else {
		return fmt.Errorf("invalid host format: %q", host)
	}
	return nil
}

// initializeConnectionDetails populates the servicebinding status with connection details
// mainly HostURl and Port.
func (c external) initializeConnectionDetails(ctx context.Context, sb *v1.ServiceBinding) error {
	secret, err := c.GetServiceBindingSecret(ctx, *sb)
	if err != nil {
		return err
	}

	instanceName := sb.ObjectMeta.Labels["klutch.io/instance-type"]

	if strings.Contains(instanceName, "search") {
		err = c.extractBracketHost(sb, secret.Data, "host")
		if err != nil {
			return err
		}
	} else if strings.Contains(instanceName, "logme2") {
		err = c.extractPlainHost(sb, secret.Data, "host")
		if err != nil {
			return err
		}
	} else if strings.Contains(instanceName, "mongodb") {
		err = c.extractBracketHost(sb, secret.Data, "hosts")
		if err != nil {
			return err
		}
	} else if strings.Contains(instanceName, "prometheus") {
		err = c.extractPrometheusHost(sb, secret.Data, "prometheus_urls", "")
		if err != nil {
			return err
		}

		err = c.extractPrometheusHost(sb, secret.Data, "alertmanager_urls", "")
		if err != nil {
			return err
		}

		err = c.extractPrometheusHost(sb, secret.Data, "grafana_urls", "")
		if err != nil {
			return err
		}

		err = c.extractPrometheusHost(sb, secret.Data, "graphite_exporters", "graphite_exporter_port")
		if err != nil {
			return err
		}

	} else if strings.Contains(instanceName, "postgresql") ||
		strings.Contains(instanceName, "messaging") ||
		strings.Contains(instanceName, "mariadb") {
		hostURL, hostFound := secret.Data["host"]
		port, portFound := secret.Data["port"]
		if hostFound && portFound {
			sb.AddConnectionDetails(string(hostURL), string(port))
		}
	} else {
		return errNoSuchDataservice
	}

	return nil
}

// initializeInstanceFields populates the servicebinding status with service instance
// details like InstanceID, ServiceID and PlanID.
func (c external) initializeInstanceFields(ctx context.Context, sb *v1.ServiceBinding) error {
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
		newServiceFn: client.NewOsbService,
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
