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

package serviceinstance

import (
	"context"
	"errors"
	"fmt"
	"strings"

	osbclient "github.com/anynines/klutchio/clients/a9s-open-service-broker"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	v1 "github.com/anynines/klutchio/provider-anynines/apis/serviceinstance/v1"
	apisv1 "github.com/anynines/klutchio/provider-anynines/apis/v1"
	util "github.com/anynines/klutchio/provider-anynines/internal/controller/utils"
	anynines "github.com/anynines/klutchio/provider-anynines/pkg/client"
	client "github.com/anynines/klutchio/provider-anynines/pkg/client/osb"
	"github.com/anynines/klutchio/provider-anynines/pkg/client/serviceinstance"
	utilerr "github.com/anynines/klutchio/provider-anynines/pkg/utilerr"
)

const (
	maxRetryAttempts = 10
)

// Setup adds a controller that reconciles ServiceInstance managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1.ServiceInstanceGroupKind)
	cps := util.GetConnectionPublisher(mgr, o)

	log := o.Logger.WithValues("controller", name)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1.ServiceInstanceGroupVersionKind),
		managed.WithExternalConnecter(utilerr.ConnectDecorator{
			Connector: &connector{
				logger:       log,
				kube:         mgr.GetClient(),
				usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1.ProviderConfigUsage{}),
				newServiceFn: client.NewOsbService,
			},
			Logger: log,
		}),
		managed.WithLogger(log),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1.ServiceInstance{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	logger       logging.Logger
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
	dsi, ok := mg.(*v1.ServiceInstance)
	if !ok {
		return nil, errNotServiceInstance
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, fmt.Errorf("%s: %w", errTrackPCUsage, err)
	}

	pc := &apisv1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: dsi.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, fmt.Errorf("%s: %w", errGetPC, err)
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
		logger: c.logger,
		osb:    svc,
	}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	logger logging.Logger
	osb    osbclient.Client
}

// Observe makes observation about the external resource.
func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	dsi, err := c.getAndVerifyServiceInstance(mg)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	// We need to detect when the desired plan doesn't match the actual one, to trigger an update
	// to make them match. But we get the actual service and plan names by querying the a9s service
	// broker, and its response contains only service and plan IDs, not names. We only have names in
	// dsi's spec. So here we resolve the desired service and plan names into their IDs (by querying
	// the catalog of the service broker), so that we can perform the comparison.
	_, desiredPlanID, err := c.getServiceAndPlanIDs(*dsi.Spec.ForProvider.ServiceName, *dsi.Spec.ForProvider.PlanName)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	instance, err := c.getInstanceAndUpdateObservation(dsi)
	if err != nil {
		return managed.ExternalObservation{}, err
	} else if instance == nil {
		return managed.ExternalObservation{}, nil
	}

	err = c.processPendingOperation(dsi)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	// Set the conditions that indicate whether the instance is ready and synched
	setCrossplaneConditions(dsi)
	var resourceUpToDate bool = c.isResourceUpToDate(dsi, instance, desiredPlanID)

	return managed.ExternalObservation{
		// Return false when the external resource does not exist. This lets
		// the managed resource reconciler know that it needs to call Create to
		// (re)create the resource, or that it has successfully been deleted.
		ResourceExists:   true,
		ResourceUpToDate: resourceUpToDate,
	}, nil
}

// we put this in its own helper function in order to reduce the cyclomatic complexity of Observe
func assertServiceAndPlanNamesAreSet(dsi *v1.ServiceInstance) error {
	if dsi.Spec.ForProvider.ServiceName == nil {
		return errors.New("the field Spec.ForProvider.ServiceName is unset and must be set")
	}

	if dsi.Spec.ForProvider.PlanName == nil {
		return errors.New("the field Spec.ForProvider.PlanName is unset and must be set")
	}
	return nil
}

func genAndCheckUID(osb osbclient.Client, maxAttempts int) (string, error) {
	var err error
	for i := 0; i < maxAttempts; i++ {
		uid := anynines.GenUID()
		_, err = osb.GetInstance(&osbclient.GetInstanceRequest{InstanceID: uid})
		if client.IsNotFound(err) {
			return uid, nil
		}
	}
	return "", errInstanceIDNotUnique.WithCause(err)
}

// This function returns the IDs in the same order that it takes the names with Service first and then Plan
func (c *external) getServiceAndPlanIDs(servicePrefix, planName string) (string, string, error) {
	service, err := c.getServiceFromCatalog(servicePrefix)
	if err != nil {
		return "", "", err
	}
	plan, err := getPlanFromService(planName, service)
	if err != nil {
		return "", "", err
	}
	return service.ID, plan.ID, nil
}

func setCrossplaneConditions(dsi *v1.ServiceInstance) {
	if dsi.Status.PendingOperation != nil {
		dsi.Status.SetConditions(xpv1.Unavailable())
		return
	}
	switch dsi.Status.AtProvider.State {
	case v1.StateCreated:
		dsi.Status.SetConditions(xpv1.Creating())
	case v1.StateProvisioned:
		dsi.Status.SetConditions(xpv1.Available())
	case v1.StateAvailable:
		dsi.Status.SetConditions(xpv1.Available())
	case v1.StateDeleting:
		dsi.Status.SetConditions(xpv1.Deleting())
	case v1.StateDeploying:
		dsi.Status.SetConditions(xpv1.Creating())
	case v1.StateFailed:
		dsi.Status.SetConditions(xpv1.Unavailable())
	default:
		dsi.Status.SetConditions(xpv1.ReconcileError(
			errors.New("unable to determine state of instance")))
	}
}

// Create initiates creation of external resource.
func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	dsi, ok := mg.(*v1.ServiceInstance)
	if !ok {
		return managed.ExternalCreation{}, errNotServiceInstance
	}

	serviceID, planID, err := c.getServiceAndPlanIDs(*dsi.Spec.ForProvider.ServiceName, *dsi.Spec.ForProvider.PlanName)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	response, err := c.osb.ProvisionInstance(&osbclient.ProvisionRequest{
		// We use the Kubernetes resource UID to ensure that each managed resource is associated
		// with only one service instance throughout its lifecycle. The Instance UID need not be
		// provided in the managed resource on creation.
		InstanceID:        dsi.Status.AtProvider.InstanceID,
		AcceptsIncomplete: *dsi.Spec.ForProvider.AcceptsIncomplete,
		ServiceID:         serviceID,
		PlanID:            planID,
		OrganizationGUID:  *dsi.Spec.ForProvider.OrganizationGUID,
		SpaceGUID:         *dsi.Spec.ForProvider.SpaceGUID,
		Parameters:        serviceinstance.KubernetesParamsToServiceBroker(dsi.Spec.ForProvider.Parameters),
	})
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("cannot create ServiceInstance: %w", utilerr.HandleHttpError(err))
	}

	if response.Async {
		c.logAsyncAction(*response.OperationKey, dsi)
	}

	return managed.ExternalCreation{}, nil
}

// getServiceFromCatalog gets a service from catalog using a specified prefix
func (c *external) getServiceFromCatalog(servicePrefix string) (osbclient.Service, error) {
	catalog, err := c.osb.GetCatalog()
	if err != nil {
		return osbclient.Service{}, fmt.Errorf("cannot get service broker catalog: %w", err)
	}
	for _, service := range catalog.Services {
		/*
			A user will supply a ServiceName such as a9s-postgresql11. However, the catalog represents
			ServiceNames differently, appending a unique suffix associated with the parent Service
			Broker. Therefore, the equivalent ServiceName in the catalog might appear as
			a9s-postgresql11-ms-1687789907, where -ms-1687789907 is this unique identifier.

			This approach considers the ServiceName provided by the user to be a prefix which can be
			used to find the ServiceName. One issue one may raise with this approach is that
			strings.Prefix is less constrained than doing a string comparison with ==. We could
			potentially be more restrictive here but we do already provide validation at the level of
			the XRC via the XRD. This means that users can only request explicitly supported services
			via the XRC.
		*/
		if strings.HasPrefix(service.Name, servicePrefix) {
			return service, nil
		}
	}
	return osbclient.Service{}, errServiceNameNotFound{name: servicePrefix}
}

func getPlanFromService(specifiedName string, service osbclient.Service) (osbclient.Plan, error) {
	for _, plan := range service.Plans {
		if plan.Name == specifiedName {
			return plan, nil
		}
	}
	return osbclient.Plan{}, errPlanNameNotFound{specifiedName}
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	dsi, ok := mg.(*v1.ServiceInstance)
	if !ok {
		return managed.ExternalUpdate{}, errNotServiceInstance
	}

	desiredServiceID, desiredPlanID, err := c.getServiceAndPlanIDs(*dsi.Spec.ForProvider.ServiceName, *dsi.Spec.ForProvider.PlanName)
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("%s: %w", errUpdateServiceInstance, err)
	}

	parameterUpdate := c.updateParameters(dsi)

	response, err := c.osb.UpdateInstance(&osbclient.UpdateInstanceRequest{
		InstanceID:        dsi.Status.AtProvider.InstanceID,
		AcceptsIncomplete: *dsi.Spec.ForProvider.AcceptsIncomplete,
		ServiceID:         desiredServiceID,
		PlanID:            &desiredPlanID,
		Parameters:        parameterUpdate,
	})
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf(
			"%s: %w", errUpdateServiceInstance,
			utilerr.HandleHttpError(err))
	}

	if response.Async {
		c.logAsyncAction(*response.OperationKey, dsi)
	}

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) error {
	dsi, ok := mg.(*v1.ServiceInstance)
	if !ok {
		return errNotServiceInstance
	}

	dsi.SetConditions(xpv1.Deleting())

	response, err := c.osb.DeprovisionInstance(&osbclient.DeprovisionRequest{
		InstanceID:        dsi.Status.AtProvider.InstanceID,
		AcceptsIncomplete: *dsi.Spec.ForProvider.AcceptsIncomplete,
		ServiceID:         dsi.Status.AtProvider.ServiceID,
		PlanID:            dsi.Status.AtProvider.PlanID,
	})
	if err != nil && !client.IsNotFound(err) {
		return fmt.Errorf("%s: %w", errDeleteServiceInstance, utilerr.HandleHttpError(err))
	}

	if response.Async {
		c.logAsyncAction(*response.OperationKey, dsi)
	}

	return nil
}

func (c *external) setUidWithError(dsi *v1.ServiceInstance) error {
	uid, err := genAndCheckUID(c.osb, maxRetryAttempts)
	if err != nil {
		return err
	}
	dsi.Status.AtProvider.InstanceID = uid
	return errors.New(errInstanceIDStatusUnset)
}

func (c *external) updateParameters(dsi *v1.ServiceInstance) map[string]interface{} {
	parameterUpdate := serviceinstance.ParameterUpdateForBroker(
		serviceinstance.KubernetesParamsToServiceBroker(dsi.Status.AtProvider.Parameters),
		serviceinstance.KubernetesParamsToServiceBroker(dsi.Spec.ForProvider.Parameters),
	)

	if parameterUpdate != nil {
		c.logger.Debug("Updating instance parameters", "parameterUpdate", parameterUpdate)
	}
	return parameterUpdate
}

func (c *external) logAsyncAction(opKey osbclient.OperationKey, dsi *v1.ServiceInstance) {
	operationKey := string(opKey)
	dsi.Status.PendingOperation = &operationKey
	c.logger.Debug("Asynchronous operation now pending", "operationKey", operationKey)
}

func (c *external) getAndVerifyServiceInstance(mg resource.Managed) (*v1.ServiceInstance, error) {
	dsi, ok := mg.(*v1.ServiceInstance)
	if !ok {
		return nil, errNotServiceInstance
	}

	// If the InstanceID is not set, then we generate and check for a unique ID, assign it to
	// InstanceID in the status, and return an 'InstanceIDStatusUnset' error. This approach triggers
	// an object status update, ensuring the InstanceID is available for future reconciliations.
	// Without this error-induced early return, the updated status wouldn't be persisted, causing a
	// failure in the Reconciler's Create method and an endless reconciliation loop.
	if dsi.Status.AtProvider.InstanceID == "" {
		return nil, c.setUidWithError(dsi)
	}

	err := assertServiceAndPlanNamesAreSet(dsi)
	if err != nil {
		return nil, err
	}
	return dsi, nil
}

func (c *external) getInstanceAndUpdateObservation(dsi *v1.ServiceInstance) (*osbclient.GetInstanceResponse, error) {
	instance, err := c.osb.GetInstance(&osbclient.GetInstanceRequest{InstanceID: dsi.Status.AtProvider.InstanceID})
	if err != nil && client.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("%s: %w", errGetServiceInstance, err)
	}

	// Update the status
	dsi.Status.AtProvider = serviceinstance.GenerateObservation(*instance)

	if dsi.Status.AtProvider.State == v1.StateDeleted {
		// When a service instance is deleted the object is still retrievable from the API but it
		// is in a deleted state. Returning here with this value tells the provider that the deletion
		// was successful and that it can remove its finalizers on the Kubernetes object representing
		// the service instance.
		return nil, nil
	}
	return instance, nil
}

func (c *external) processPendingOperation(dsi *v1.ServiceInstance) error {
	if dsi.Status.PendingOperation != nil {
		response, err := c.osb.GetOperation(&osbclient.GetOperationRequest{
			OperationKey: osbclient.OperationKey(*dsi.Status.PendingOperation),
		})
		if err != nil {
			c.logger.Debug("Failed to check pending operation",
				"operationKey", *dsi.Status.PendingOperation,
				"error", err)
			return fmt.Errorf("%s: %w", errGetOperation, err)
		}

		c.logger.Debug("Checked pending operation",
			"operationKey", *dsi.Status.PendingOperation,
			"state", response.State)

		if response.IsDone() {
			dsi.Status.PendingOperation = nil
		} else if failed, err := response.IsFailure(); failed {
			// clear pending operation. After reaching a failure state an
			// operation will never complete. The next reconciliation will
			// compare spec and status again and try to re-apply any changes
			// necessary.
			dsi.Status.PendingOperation = nil

			return fmt.Errorf("%s: %w", errOperationFailed, err)
		}
	}
	return nil
}

func (c *external) isResourceUpToDate(dsi *v1.ServiceInstance, instance *osbclient.GetInstanceResponse, desiredPlanID string) bool {
	if dsi.Status.PendingOperation == nil {
		specMatchesObserved, diff := serviceinstance.SpecMatchesObservedState(dsi.Spec.ForProvider, *instance)
		if !specMatchesObserved {
			c.logger.Debug("Observed state differs from expected: " + diff)
			// Return false when the external resource exists, but it not up to date
			// with the desired managed resource state. This lets the managed
			// resource reconciler know that it needs to call Update.
			return false
		} else {
			// Since ServiceID and PlanID are part of Status instead of Spec we need
			// to separately check whether they are up to date as well.
			return desiredPlanID == dsi.Status.AtProvider.PlanID
		}
	}

	// While an operation is pending on the service-broker side, the resource
	// is considered "up to date", since there is nothing we can do to
	// to reconcile it's state until the operation finishes.
	return true
}
