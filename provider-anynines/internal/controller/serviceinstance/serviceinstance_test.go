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
	"net/http"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/test"
	"k8s.io/apimachinery/pkg/util/intstr"

	osbclient "github.com/anynines/klutchio/clients/a9s-open-service-broker"
	fakeosb "github.com/anynines/klutchio/clients/a9s-open-service-broker/fake"
	sbv1 "github.com/anynines/klutchio/provider-anynines/apis/servicebinding/v1"
	v1 "github.com/anynines/klutchio/provider-anynines/apis/serviceinstance/v1"
	apisv1 "github.com/anynines/klutchio/provider-anynines/apis/v1"
	a9stest "github.com/anynines/klutchio/provider-anynines/internal/controller/test"
	utilerr "github.com/anynines/klutchio/provider-anynines/pkg/utilerr"
)

var defaultCatalogResponse = osbclient.CatalogResponse{
	Services: []osbclient.Service{
		{
			ID:                   "0f3f9e21-f960-41f4-b787-b2b47b567996",
			Name:                 "a9s-postgresql11-ms-1687789906",
			Description:          "This is a service creating and managing dedicated PostgreSQL service instances and clusters, powered by the anynines Service Framework",
			Bindable:             true,
			InstancesRetrievable: false,
			BindingsRetrievable:  false,
			PlanUpdatable:        ptr.To[bool](true),
			Plans: []osbclient.Plan{
				{
					ID:             "40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
					Name:           "postgresql-single-small",
					Description:    "a small single instance",
					Free:           ptr.To[bool](true),
					PlanUpdateable: ptr.To[bool](true),
				},
				{
					ID:             "2754eb09-a4cb-4fe3-bbd8-3ad208608840",
					Name:           "postgresql-single-big",
					Description:    "a big single instance",
					Free:           ptr.To[bool](true),
					PlanUpdateable: ptr.To[bool](true),
				},
			},
		},
	},
}

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

type serviceInstanceOption func(*v1.ServiceInstance)

func newServiceInstance(opts ...serviceInstanceOption) *v1.ServiceInstance {
	// Set Defaults
	p := &v1.ServiceInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test-ns",
		},
		Spec: v1.ServiceInstanceSpec{
			ForProvider: v1.ServiceInstanceParameters{
				ServiceName:       ptr.To[string]("a9s-postgresql11"),
				PlanName:          ptr.To[string]("postgresql-single-small"),
				AcceptsIncomplete: ptr.To[bool](true),
				OrganizationGUID:  ptr.To[string]("a1612e60-3042-4bf2-bd7c-fa600a4f66b9"),
				SpaceGUID:         ptr.To[string]("009dbe05-925d-4f2a-ac0d-8d44dd723a11"),
			},
		},
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

func invalidServiceInstanceConfigurationErrorResponse() sbError {
	return sbError{
		Error:       "InvalidServiceInstanceConfigurationError",
		Description: "Configuration parameters are missing",
	}
}

type instanceResponseOption func(gir *osbclient.GetInstanceResponse)

func withInstanceResponseState(state string) instanceResponseOption {
	return func(gir *osbclient.GetInstanceResponse) {
		gir.State = state
	}
}

func withInstanceResponseIntParameter(key string, value int) instanceResponseOption {
	return func(gir *osbclient.GetInstanceResponse) {
		if gir.Metadata.Parameters == nil {
			gir.Metadata.Parameters = map[string]interface{}{}
		}
		// to simulate the way the param would be parsed from JSON, convert to float64 here
		gir.Metadata.Parameters[key] = float64(value)
	}
}

func newInstanceResponse(opts ...instanceResponseOption) *osbclient.GetInstanceResponse {
	// Set defaults
	ir := &osbclient.GetInstanceResponse{
		ID:           1,
		PlanGUID:     "40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
		ServiceGUID:  "0f3f9e21-f960-41f4-b787-b2b47b567996",
		GUIDAtTenant: "40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
		TenantID:     "40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
		Metadata: osbclient.Metadata{
			InstanceGUIDAtTenant: "40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
			PlanGUID:             "40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
			OrganizationGUID:     "a1612e60-3042-4bf2-bd7c-fa600a4f66b9",
			SpaceGUID:            "009dbe05-925d-4f2a-ac0d-8d44dd723a11",
		},
	}

	for _, opt := range opts {
		opt(ir)
	}

	return ir
}

type getOperationResponseOption func(r *osbclient.GetOperationResponse)

func withOperationState(state string) getOperationResponseOption {
	return func(r *osbclient.GetOperationResponse) {
		r.State = state
	}
}

func newGetOperationResponse(opts ...getOperationResponseOption) *osbclient.GetOperationResponse {
	response := &osbclient.GetOperationResponse{}

	for _, opt := range opts {
		opt(response)
	}

	return response
}

type sbError struct {
	Error       string `json:"error"`
	Description string `json:"description"`
}

func catalogErrResponse() sbError {
	return sbError{
		Error:       "CatalogError",
		Description: "Service Offering does not exist in the catalog",
	}
}

func instanceLimitErrorResponse() sbError {
	return sbError{
		Error:       "InstanceLimitError",
		Description: "Service instance limit hit for the service plan",
	}
}

func InstanceNotProvisionedResponse() sbError {
	return sbError{
		Error:       "InstanceNotProvisioned",
		Description: "Instance is not provisioned currently.",
	}
}

func TestMain(m *testing.M) {
	if err := apisv1.AddToScheme(scheme.Scheme); err != nil {
		panic("failed to add API github.com/anynines/klutchio/provider-anynines/apis/v1 to scheme")
	}

	os.Exit(m.Run())
}

func TestObserve(t *testing.T) {
	type args struct {
		getInstanceReaction  *fakeosb.GetInstanceReaction
		catalogReaction      *fakeosb.CatalogReaction
		getOperationReaction *fakeosb.GetOperationReaction
		mr                   resource.Managed
	}

	type want struct {
		observation managed.ExternalObservation
		err         error
		mr          resource.Managed
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"errManagedResourceIsNotServiceInstance": {
			args: args{
				mr: &sbv1.ServiceBinding{},
			},
			want: want{
				err: utilerr.PlainUserErr("managed resource is not a ServiceInstance custom resource"),
				mr:  &sbv1.ServiceBinding{},
			},
		},
		"errServiceInstanceNotFoundOnBroker": {
			args: args{
				getInstanceReaction: &fakeosb.GetInstanceReaction{
					Error: osbclient.HTTPStatusCodeError{
						StatusCode:   http.StatusNotFound,
						ErrorMessage: ptr.To[string]("InstanceNotFound"),
						Description:  ptr.To[string]("Instance not found"),
					},
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
			},
			want: want{
				observation: managed.ExternalObservation{},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
			},
		},
		"errSpecifiedServiceNameIsNotInTheCatalog": {
			args: args{
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceName("non-existing-service"),
				),
			},
			want: want{
				observation: managed.ExternalObservation{},
				err:         errServiceNameNotFound{"non-existing-service"},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceName("non-existing-service"),
				),
			},
		},
		"errServiceDoesNotOfferSpecifiedPlan": {
			args: args{
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withPlanName("non-existing-plan"),
				),
			},
			want: want{
				observation: managed.ExternalObservation{},
				err:         &errPlanNameNotFound{"non-existing-plan"},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withPlanName("non-existing-plan"),
				),
			},
		},
		"successCorrectlyObservedInstanceWithStateCreating": {
			args: args{
				getInstanceReaction: &fakeosb.GetInstanceReaction{
					Response: newInstanceResponse(withInstanceResponseState("created")),
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:          true,
					ResourceLateInitialized: false,
					ResourceUpToDate:        true,
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withState("created"),
					withCondition(xpv1.Creating()),
				),
			},
		},
		"successCorrectlyObservedInstanceWithStateProvisioned": {
			args: args{
				getInstanceReaction: &fakeosb.GetInstanceReaction{
					Response: newInstanceResponse(withInstanceResponseState("provisioned")),
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:          true,
					ResourceLateInitialized: false,
					ResourceUpToDate:        true,
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withState("provisioned"),
					withCondition(xpv1.Available()),
				),
			},
		},
		"successCorrectlyObservedInstanceWithStateAvailable": {
			args: args{
				getInstanceReaction: &fakeosb.GetInstanceReaction{Response: newInstanceResponse(withInstanceResponseState("available"))},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:          true,
					ResourceLateInitialized: false,
					ResourceUpToDate:        true,
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withState("available"),
					withCondition(xpv1.Available()),
				),
			},
		},
		"successCorrectlyObservedInstanceWithStateDeleted": {
			args: args{
				getInstanceReaction: &fakeosb.GetInstanceReaction{Response: newInstanceResponse(withInstanceResponseState("deleted"))},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:          false,
					ResourceLateInitialized: false,
					ResourceUpToDate:        false,
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withState("deleted"),
				),
			},
		},
		"successCorrectlyObservedInstanceWithStateDeleting": {
			args: args{
				getInstanceReaction: &fakeosb.GetInstanceReaction{Response: newInstanceResponse(withInstanceResponseState("deleting"))},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:          true,
					ResourceLateInitialized: false,
					ResourceUpToDate:        true,
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withState("deleting"),
					withCondition(xpv1.Deleting()),
				),
			},
		},
		"successCorrectlyObservedInstanceWithStateDeploying": {
			args: args{
				getInstanceReaction: &fakeosb.GetInstanceReaction{Response: newInstanceResponse(withInstanceResponseState("deploying"))},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:          true,
					ResourceLateInitialized: false,
					ResourceUpToDate:        true,
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withState("deploying"),
					withCondition(xpv1.Creating()),
				),
			},
		},
		"successCorrectlyObservedInstanceWithStateFailed": {
			args: args{
				getInstanceReaction: &fakeosb.GetInstanceReaction{Response: newInstanceResponse(withInstanceResponseState("failed"))},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:          true,
					ResourceLateInitialized: false,
					ResourceUpToDate:        true,
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withState("failed"),
					withCondition(xpv1.Unavailable()),
				),
			},
		},
		"errObservedInstanceWithUnknownState": {
			args: args{
				getInstanceReaction: &fakeosb.GetInstanceReaction{Response: newInstanceResponse(withInstanceResponseState("unknown"))},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:          true,
					ResourceLateInitialized: false,
					ResourceUpToDate:        true,
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withState("unknown"),
					withCondition(xpv1.ReconcileError(errors.New("unable to determine state of instance"))),
				),
			},
		},
		"successPlanWasCorrectlyUpdated": {
			args: args{
				getInstanceReaction: &fakeosb.GetInstanceReaction{Response: newInstanceResponse(withInstanceResponseState("provisioned"))},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withPlanName("postgresql-single-big"),
				),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:          true,
					ResourceLateInitialized: false,
					ResourceUpToDate:        false,
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withPlanName("postgresql-single-big"),
					withState("provisioned"),
					withCondition(xpv1.Available()),
				),
			},
		},
		"pendingOperationIsPending": {
			args: args{
				getInstanceReaction:  &fakeosb.GetInstanceReaction{Response: newInstanceResponse(withInstanceResponseState("unknown"))},
				getOperationReaction: &fakeosb.GetOperationReaction{Response: newGetOperationResponse(withOperationState("pending"))},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withPlanName("postgresql-single-small"),
					withPendingOperation("27"),
				),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:          true,
					ResourceLateInitialized: false,
					ResourceUpToDate:        true,
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withPlanName("postgresql-single-small"),
					// pending operation remains while it's neither done nor failed
					withPendingOperation("27"),
					withState("unknown"),
					withCondition(xpv1.Unavailable()),
				),
			},
		},
		"pendingOperationIsDone": {
			args: args{
				getInstanceReaction:  &fakeosb.GetInstanceReaction{Response: newInstanceResponse(withInstanceResponseState("provisioned"))},
				getOperationReaction: &fakeosb.GetOperationReaction{Response: newGetOperationResponse(withOperationState("done"))},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withPlanName("postgresql-single-small"),
					withPendingOperation("27"),
				),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:          true,
					ResourceLateInitialized: false,
					ResourceUpToDate:        true,
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withPlanName("postgresql-single-small"),
					withState("provisioned"),
					withCondition(xpv1.Available()),
					// no `pendingOperation` after seeing "done"
				),
			},
		},
		"pendingOperationHasErrorState": {
			args: args{
				getInstanceReaction:  &fakeosb.GetInstanceReaction{Response: newInstanceResponse(withInstanceResponseState("provisioned"))},
				getOperationReaction: &fakeosb.GetOperationReaction{Response: newGetOperationResponse(withOperationState("error"))},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withPlanName("postgresql-single-small"),
					withPendingOperation("27"),
				),
			},
			want: want{
				err: utilerr.ErrInternal,
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withPlanName("postgresql-single-small"),
					withState("provisioned"),
					// no `pendingOperation` after seeing an error
				),
			},
		},
		"successParametersAreUnchanged": {
			args: args{
				getInstanceReaction: &fakeosb.GetInstanceReaction{
					Response: newInstanceResponse(
						withInstanceResponseState("provisioned"),
						withInstanceResponseIntParameter("max_connections", 200),
					),
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withPlanName("postgresql-single-small"),
					withIntParameter("maxConnections", 200),
				),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withPlanName("postgresql-single-small"),
					withState("provisioned"),
					withCondition(xpv1.Available()),
					withIntParameter("maxConnections", 200),
					withStatusIntParameter("maxConnections", 200),
				),
			},
		},
		"successParametersNeedUpdate": {
			args: args{
				getInstanceReaction: &fakeosb.GetInstanceReaction{
					Response: newInstanceResponse(
						withInstanceResponseState("provisioned"),
						withInstanceResponseIntParameter("max_connections", 100),
					),
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withPlanName("postgresql-single-small"),
					withIntParameter("maxConnections", 200),
				),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: false,
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withPlanName("postgresql-single-small"),
					withState("provisioned"),
					withCondition(xpv1.Available()),
					withIntParameter("maxConnections", 200),
					withStatusIntParameter("maxConnections", 100),
				),
			},
		},
		"successParametersAreRemoved": {
			args: args{
				getInstanceReaction: &fakeosb.GetInstanceReaction{
					Response: newInstanceResponse(
						withInstanceResponseState("provisioned"),
						withInstanceResponseIntParameter("max_connections", 100),
					),
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withPlanName("postgresql-single-small"),
				),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: false,
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withPlanName("postgresql-single-small"),
					withState("provisioned"),
					withCondition(xpv1.Available()),
					withStatusIntParameter("maxConnections", 100),
				),
			},
		},
		"successEmptyParametersDontCauseUpdate": {
			args: args{
				getInstanceReaction: &fakeosb.GetInstanceReaction{
					Response: newInstanceResponse(
						withInstanceResponseState("provisioned"),
					),
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withPlanName("postgresql-single-small"),
					withEmptyParameters(),
				),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withPlanName("postgresql-single-small"),
					withState("provisioned"),
					withCondition(xpv1.Available()),
					withEmptyParameters(),
				),
			},
		},
	}

	for name, tc := range cases {
		// Rebind tc into this lexical scope. Details on the why at
		// https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.args.catalogReaction == nil {
				tc.args.catalogReaction = &fakeosb.CatalogReaction{
					Response: &defaultCatalogResponse,
				}
			}

			// Set up the object under test.
			fakeOSB := fakeosb.NewFakeClient(fakeosb.FakeClientConfiguration{
				GetInstanceReaction:  tc.args.getInstanceReaction,
				CatalogReaction:      tc.args.catalogReaction,
				GetOperationReaction: tc.args.getOperationReaction,
			})
			e := utilerr.Decorator{
				ExternalClient: &external{
					logger: a9stest.TestLogger(t),
					osb:    fakeOSB,
				},
				Logger: a9stest.TestLogger(t),
			}

			// Invoke the method under test.
			got, err := e.Observe(context.Background(), tc.args.mr)
			if diff := cmp.Diff(tc.want.observation, got); diff != "" {
				t.Errorf("Observe(...): -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("Observe(...): -want error, +got error:\n%s", diff)
			}

			if diff := cmp.Diff(tc.want.mr, tc.args.mr); diff != "" {
				t.Errorf("Observe(...): -want mr, +got mr:\n%s", diff)
			}
		})
	}
}

func TestCreate(t *testing.T) {
	type args struct {
		provisionReaction *fakeosb.ProvisionReaction
		catalogReaction   *fakeosb.CatalogReaction
		mr                resource.Managed
	}

	type want struct {
		eo      managed.ExternalCreation
		err     error
		mr      resource.Managed
		actions []fakeosb.Action
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"errTriedToProvisionResourceWithInvalidServiceID": {
			args: args{
				provisionReaction: &fakeosb.ProvisionReaction{Error: osbclient.HTTPStatusCodeError{
					StatusCode:   http.StatusUnprocessableEntity,
					ErrorMessage: ptr.To[string]("CatalogError"),
					Description:  ptr.To[string]("Service Offering does not exist in the catalog"),
				}},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("invalid-service-id"),
				),
			},
			want: want{
				err: utilerr.PlainUserErr(catalogErrResponse().Description),
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("invalid-service-id")),
			},
		},
		"errManagedResourceIsNotServiceInstanceInstance": {
			args: args{
				mr: &sbv1.ServiceBinding{},
			},
			want: want{
				err: utilerr.PlainUserErr("managed resource is not a ServiceInstance custom resource"),
				mr:  &sbv1.ServiceBinding{},
			},
		},
		"errCatalogCouldNotBeRetrievedFromBroker": {
			args: args{
				catalogReaction: &fakeosb.CatalogReaction{
					Error: osbclient.HTTPStatusCodeError{
						StatusCode:   500,
						ErrorMessage: ptr.To[string]("BOOOM"),
						Description:  ptr.To[string]("test-error"),
					},
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
			},
			want: want{
				eo:  managed.ExternalCreation{},
				err: utilerr.ErrInternal,
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
			},
		},
		"errSpecifiedServiceNameIsNotInTheCatalog": {
			args: args{
				catalogReaction: &fakeosb.CatalogReaction{Response: &osbclient.CatalogResponse{
					Services: []osbclient.Service{}},
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceName("non-existing-service"),
				),
			},
			want: want{
				eo: managed.ExternalCreation{},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceName("non-existing-service"),
				),
				err: errServiceNameNotFound{"non-existing-service"},
			},
		},
		"errServiceDoesNotOfferSpecifiedPlan": {
			args: args{
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withPlanName("non-existing-plan"),
				),
			},
			want: want{
				eo: managed.ExternalCreation{},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withPlanName("non-existing-plan"),
				),
				err: &errPlanNameNotFound{"non-existing-plan"},
			},
		},
		"errInstanceLimitOnServiceBrokerIsReached": {
			args: args{
				provisionReaction: &fakeosb.ProvisionReaction{
					Error: osbclient.HTTPStatusCodeError{
						StatusCode:   http.StatusUnprocessableEntity,
						ErrorMessage: ptr.To[string]("InstanceLimitError"),
						Description:  ptr.To[string]("Service instance limit hit for the service plan"),
					},
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
			},
			want: want{
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
				err: utilerr.PlainUserErr(instanceLimitErrorResponse().Description),
			},
		},
		"errInstanceAlreadyExistsWithDifferentPlan": {
			args: args{
				provisionReaction: &fakeosb.ProvisionReaction{
					Error: osbclient.HTTPStatusCodeError{
						StatusCode:   http.StatusConflict,
						ErrorMessage: ptr.To[string]("InstanceAlreadyExistsError"),
						Description:  ptr.To[string]("The instance is already provisioned but with a different plan"),
					},
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
			},
			want: want{
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
				err: utilerr.PlainUserErr("The instance is already provisioned but with a different plan"),
			},
		},
		"errInstanceAlreadyExistsWithDifferentParameters": {
			args: args{
				provisionReaction: &fakeosb.ProvisionReaction{
					Error: osbclient.HTTPStatusCodeError{
						StatusCode:   http.StatusConflict,
						ErrorMessage: ptr.To[string]("InstanceAlreadyExistsError"),
						Description:  ptr.To[string]("The instance is already provisioned but with different parameters"),
					},
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
			},
			want: want{
				err: utilerr.PlainUserErr("The instance is already provisioned but with different parameters"),
				mr:  newServiceInstance(withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5")),
			},
		},
		"errInvalidServiceInstanceConfiguration": {
			args: args{
				provisionReaction: &fakeosb.ProvisionReaction{
					Error: osbclient.HTTPStatusCodeError{
						StatusCode:   http.StatusBadRequest,
						ErrorMessage: ptr.To[string]("InvalidServiceInstanceConfigurationError"),
						Description:  ptr.To[string]("Configuration parameters are missing"),
					},
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
			},
			want: want{
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
				err: utilerr.PlainUserErr(invalidServiceInstanceConfigurationErrorResponse().Description),
			},
		},
		"errServiceBrokerUnableToCreateInstance": {
			args: args{
				provisionReaction: &fakeosb.ProvisionReaction{
					Error: osbclient.HTTPStatusCodeError{
						StatusCode:    http.StatusBadRequest,
						ResponseError: errors.New("unexpected end of JSON input"),
					},
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
			},
			want: want{
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
				err: utilerr.ErrInternal,
			},
		},
		"successInstanceIsProvisionedCorrectlyOnServiceBroker": {
			args: args{
				provisionReaction: &fakeosb.ProvisionReaction{
					Response: &osbclient.ProvisionResponse{},
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withAnnotation("anynines.crossplane.io/instance-name",
						"test"),
				),
			},
			want: want{
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withAnnotation("anynines.crossplane.io/instance-name",
						"test"),
				),
			},
		},
		"successInstanceIsProvisionedWithParameters": {
			args: args{
				provisionReaction: &fakeosb.ProvisionReaction{
					Response: &osbclient.ProvisionResponse{},
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withIntParameter("max_connections", 100),
				),
			},
			want: want{
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withIntParameter("max_connections", 100),
				),
				actions: []fakeosb.Action{
					{Type: "GetCatalog"},
					{
						Type: "ProvisionInstance",
						Request: &osbclient.ProvisionRequest{
							InstanceID:        "40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
							AcceptsIncomplete: true,
							ServiceID:         "0f3f9e21-f960-41f4-b787-b2b47b567996",
							PlanID:            "40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
							OrganizationGUID:  "a1612e60-3042-4bf2-bd7c-fa600a4f66b9",
							SpaceGUID:         "009dbe05-925d-4f2a-ac0d-8d44dd723a11",
							Parameters: map[string]interface{}{
								"max_connections": 100,
							},
						},
					}},
			},
		},
	}

	for name, tc := range cases {
		// Rebind tc into this lexical scope. Details on the why at
		// https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.args.catalogReaction == nil {
				tc.args.catalogReaction = &fakeosb.CatalogReaction{
					Response: &defaultCatalogResponse,
				}
			}

			fakeOSB := fakeosb.NewFakeClient(fakeosb.FakeClientConfiguration{
				ProvisionReaction: tc.args.provisionReaction,
				CatalogReaction:   tc.args.catalogReaction,
			})

			e := utilerr.Decorator{
				ExternalClient: &external{
					logger: a9stest.TestLogger(t),
					osb:    fakeOSB,
				},
				Logger: a9stest.TestLogger(t),
			}

			// Invoke the method under test.
			got, err := e.Create(context.Background(), tc.args.mr)

			if diff := cmp.Diff(tc.want.eo, got); diff != "" {
				t.Errorf("Create(...): -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("Create(...): -want error, +got error:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.mr, tc.args.mr); diff != "" {
				t.Errorf("Create(...): -want mr, +got mr:\n%s", diff)
			}
			if tc.want.actions != nil {
				got := fakeOSB.Actions()
				if diff := cmp.Diff(tc.want.actions, got); diff != "" {
					t.Errorf("Create(...) actions: -want, +got:\n%s", diff)
				}
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	type args struct {
		updateInstanceReaction fakeosb.UpdateInstanceReaction
		catalogReaction        *fakeosb.CatalogReaction
		mr                     resource.Managed
	}

	type want struct {
		eo               managed.ExternalUpdate
		err              error
		pendingOperation string
		actions          []fakeosb.Action
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"errManagedResourceIsNotServiceInstanceInstance": {
			args: args{
				mr: &sbv1.ServiceBinding{},
			},
			want: want{
				err: errNotServiceInstance,
			},
		},
		"successUpdateSucceeded": {
			args: args{
				updateInstanceReaction: fakeosb.UpdateInstanceReaction{
					Response: &osbclient.UpdateInstanceResponse{},
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
			},
			want: want{
				actions: []fakeosb.Action{
					{Type: "GetCatalog"},
					{
						Type: "UpdateInstance",
						Request: &osbclient.UpdateInstanceRequest{
							InstanceID:        "40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
							AcceptsIncomplete: true,
							ServiceID:         "0f3f9e21-f960-41f4-b787-b2b47b567996",
							PlanID:            ptr.To[string]("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
						},
					},
				},
			},
		},
		"successUpdateWithParameters": {
			args: args{
				updateInstanceReaction: fakeosb.UpdateInstanceReaction{
					Response: &osbclient.UpdateInstanceResponse{},
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withIntParameter("maxConnections", 250),
				),
			},
			want: want{
				actions: []fakeosb.Action{
					{Type: "GetCatalog"},
					{
						Type: "UpdateInstance",
						Request: &osbclient.UpdateInstanceRequest{
							InstanceID:        "40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
							AcceptsIncomplete: true,
							ServiceID:         "0f3f9e21-f960-41f4-b787-b2b47b567996",
							PlanID:            ptr.To[string]("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
							Parameters: map[string]interface{}{
								"max_connections": 250,
							},
						},
					},
				},
			},
		},
		"successRemoveParameter": {
			args: args{
				updateInstanceReaction: fakeosb.UpdateInstanceReaction{
					Response: &osbclient.UpdateInstanceResponse{},
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withStatusIntParameter("maxConnections", 250),
				),
			},
			want: want{
				actions: []fakeosb.Action{
					{Type: "GetCatalog"},
					{
						Type: "UpdateInstance",
						Request: &osbclient.UpdateInstanceRequest{
							InstanceID:        "40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
							AcceptsIncomplete: true,
							ServiceID:         "0f3f9e21-f960-41f4-b787-b2b47b567996",
							PlanID:            ptr.To[string]("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
							Parameters: map[string]interface{}{
								"max_connections": nil,
							},
						},
					},
				},
			},
		},
		"SuccessAddParameter": {
			args: args{
				updateInstanceReaction: fakeosb.UpdateInstanceReaction{
					Response: &osbclient.UpdateInstanceResponse{},
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withIntParameter("maxConnections", 250),
					withStringParameter("synchronousCommit", "local"),
					withStatusIntParameter("maxConnections", 250),
				),
			},
			want: want{
				actions: []fakeosb.Action{
					{Type: "GetCatalog"},
					{
						Type: "UpdateInstance",
						Request: &osbclient.UpdateInstanceRequest{
							InstanceID:        "40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
							AcceptsIncomplete: true,
							ServiceID:         "0f3f9e21-f960-41f4-b787-b2b47b567996",
							PlanID:            ptr.To[string]("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
							Parameters: map[string]interface{}{
								"synchronous_commit": "local",
							},
						},
					},
				},
			},
		},
		"SuccessChangeExistingParameter": {
			args: args{
				updateInstanceReaction: fakeosb.UpdateInstanceReaction{
					Response: &osbclient.UpdateInstanceResponse{},
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withIntParameter("maxConnections", 200),
					withStatusIntParameter("maxConnections", 250),
				),
			},
			want: want{
				actions: []fakeosb.Action{
					{Type: "GetCatalog"},
					{
						Type: "UpdateInstance",
						Request: &osbclient.UpdateInstanceRequest{
							InstanceID:        "40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
							AcceptsIncomplete: true,
							ServiceID:         "0f3f9e21-f960-41f4-b787-b2b47b567996",
							PlanID:            ptr.To[string]("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
							Parameters: map[string]interface{}{
								"max_connections": 200,
							},
						},
					},
				},
			},
		},
		"successUpdateAsync": {
			args: args{
				updateInstanceReaction: fakeosb.UpdateInstanceReaction{
					Response: &osbclient.UpdateInstanceResponse{
						Async:        true,
						OperationKey: operationKey("job-xyz"),
					},
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
			},
			want: want{
				pendingOperation: "job-xyz",
			},
		},
		"errNotFound": {
			reason: "Should return error if instance to update is not found",
			args: args{
				updateInstanceReaction: fakeosb.UpdateInstanceReaction{
					Error: osbclient.HTTPStatusCodeError{
						StatusCode:   http.StatusNotFound,
						ErrorMessage: ptr.To[string]("InstanceNotFound"),
						Description:  ptr.To[string]("Instance not found"),
					},
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
			},
			want: want{
				err: utilerr.PlainUserErr("Instance not found"),
			},
		},
		"errInvalidRequestFail": {
			reason: "Should fail when invalid catalog GUIDs are used",
			args: args{
				updateInstanceReaction: fakeosb.UpdateInstanceReaction{
					Error: osbclient.HTTPStatusCodeError{
						StatusCode:   http.StatusInternalServerError,
						ErrorMessage: ptr.To[string]("InternalServerError"),
						Description:  ptr.To[string]("Okay, Houston, we've had a problem here. See the logs for more information."),
					},
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
			},
			want: want{
				err: utilerr.ErrInternal,
			},
		},
		"errInstanceIsNotProvisionedFail": {
			reason: "Should fail if instance is not yet provisioned",
			args: args{
				updateInstanceReaction: fakeosb.UpdateInstanceReaction{
					Error: osbclient.HTTPStatusCodeError{
						StatusCode:   http.StatusUnprocessableEntity,
						ErrorMessage: ptr.To[string]("InstanceNotProvisioned"),
						Description:  ptr.To[string]("Instance is not provisioned currently."),
					},
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
			},
			want: want{
				err: utilerr.PlainUserErr(InstanceNotProvisionedResponse().Description),
			},
		},

		"errServiceIDNotInCatalog": {
			args: args{
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceName("non-existent-service"),
					withPlanName("postgresql-single-small"),
				),
			},
			want: want{
				err: errServiceNameNotFound{"non-existent-service"},
			},
		},

		"errPlanNotAvailableForService": {
			args: args{
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceName("a9s-postgresql11"),
					withPlanName("non-existent-plan"),
				),
			},
			want: want{
				err: &errPlanNameNotFound{"non-existent-plan"},
			},
		},

		// TODO: Add test cases for PlanID and ServiceID not in catalog

	}

	for name, tc := range cases {
		// Rebind tc into this lexical scope. Details on the why at
		// https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.args.catalogReaction == nil {
				tc.args.catalogReaction = &fakeosb.CatalogReaction{
					Response: &defaultCatalogResponse,
				}
			}

			fakeOSB := fakeosb.NewFakeClient(fakeosb.FakeClientConfiguration{
				UpdateInstanceReaction: &tc.args.updateInstanceReaction,
				CatalogReaction:        tc.args.catalogReaction,
			})

			e := utilerr.Decorator{
				ExternalClient: &external{
					logger: a9stest.TestLogger(t),
					osb:    fakeOSB,
				},
				Logger: a9stest.TestLogger(t),
			}

			// Invoke the method under test.
			got, err := e.Update(context.Background(), tc.args.mr)

			if diff := cmp.Diff(tc.want.eo, got); diff != "" {
				t.Errorf("Update(...): -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("Update(...): -want error, +got error:\n%s", diff)
			}

			if tc.want.actions != nil {
				got := fakeOSB.Actions()
				if diff := cmp.Diff(tc.want.actions, got); diff != "" {
					t.Errorf("Update(...) actions: -want, +got:\n%s", diff)
				}
			}

			expectPendingOperation(t, tc.args.mr, tc.want.pendingOperation)
		})
	}
}

func TestDelete(t *testing.T) {
	type args struct {
		deprovisionReaction fakeosb.DeprovisionReaction
		pendingOperation    string
		mr                  resource.Managed
	}

	type want struct {
		err error
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"errManagedResourceIsNotServiceInstanceInstance": {
			args: args{
				mr: &sbv1.ServiceBinding{},
			},
			want: want{
				err: utilerr.PlainUserErr("managed resource is not a ServiceInstance custom resource"),
			},
		},
		"errDeleteFailedMismatchedInstanceID": {
			args: args{
				mr: newServiceInstance(withStatusInstanceID("mismatched-id"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
				deprovisionReaction: fakeosb.DeprovisionReaction{
					Error: osbclient.HTTPStatusCodeError{
						StatusCode:   http.StatusBadRequest,
						ErrorMessage: ptr.To[string]("InstanceError"),
						Description:  ptr.To[string]("Mismatch between the provided service ID and the service ID of the instance"),
					},
				},
			},
			want: want{
				err: utilerr.PlainUserErr("Mismatch between the provided service ID and the service ID of the instance"),
			},
		},
		"successInstanceDeleted": {
			args: args{
				deprovisionReaction: fakeosb.DeprovisionReaction{
					Response: &osbclient.DeprovisionResponse{},
				},
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
			},
		},
		"successInstanceDeletedAsync": {
			args: args{
				deprovisionReaction: fakeosb.DeprovisionReaction{
					Response: &osbclient.DeprovisionResponse{
						Async:        true,
						OperationKey: operationKey("123"),
					},
				},
				pendingOperation: "123",
				mr: newServiceInstance(
					withStatusInstanceID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
					withServiceID("0f3f9e21-f960-41f4-b787-b2b47b567996"),
					withPlanID("40a5148f-dba2-41f2-b1b7-0ca90e1501c5"),
				),
			},
		},
	}

	for name, tc := range cases {
		// Rebind tc into this lexical scope. Details on the why at
		// https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fakeOSB := fakeosb.NewFakeClient(fakeosb.FakeClientConfiguration{
				DeprovisionReaction: &tc.args.deprovisionReaction,
			})

			e := utilerr.Decorator{
				ExternalClient: &external{
					logger: a9stest.TestLogger(t),
					osb:    fakeOSB,
				},
				Logger: a9stest.TestLogger(t),
			}
			// Invoke the method under test.
			err := e.Delete(context.Background(), tc.args.mr)

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("Delete(...): -want error, +got error:\n%s", diff)
			}

			expectPendingOperation(t, tc.args.mr, tc.args.pendingOperation)
		})
	}
}

func TestConnect(t *testing.T) {
	pc := a9stest.ProviderConfig(a9stest.Name[apisv1.ProviderConfig]("test-provider"),
		a9stest.WithProviderConfigSpec("test.com",
			a9stest.SecretRef("test-secret",
				"test",
				"username"),
			a9stest.SecretRef("test-secret",
				"test",
				"password"),
			xpv1.CredentialsSourceSecret),
	)

	secret := a9stest.Secret(a9stest.Name[corev1.Secret]("test-secret"),
		a9stest.Namespace[corev1.Secret]("test"),
		a9stest.WithKey("username",
			"test"),
		a9stest.WithKey("password",
			"secure-test-password"),
	)

	fakeK8s := fake.NewClientBuilder().WithObjects(pc, secret).WithScheme(scheme.Scheme).Build()

	var gotUsername, gotPassword []byte
	var gotURL string

	con := utilerr.ConnectDecorator{
		Connector: &connector{
			kube: fakeK8s,
			usage: resource.NewProviderConfigUsageTracker(fakeK8s,
				&apisv1.ProviderConfigUsage{}),
			newServiceFn: func(username, password []byte, url string) (osbclient.Client, error) {
				gotPassword = password
				gotUsername = username
				gotURL = url

				return nil, nil
			},
		},
		Logger: a9stest.TestLogger(t),
	}

	pg := newServiceInstance(a9stest.Uid[v1.ServiceInstance]("test-uid"),
		withProviderRef("test-provider"))

	_, err := con.Connect(context.TODO(), pg)
	if err != nil {
		t.Errorf("Unexpected error while executing Connect: %s", err)
	}

	if diff := cmp.Diff(gotUsername, []byte("test")); diff != "" {
		t.Errorf("Connect did not propagate username correctly to OSB client: %s", diff)
	}

	if diff := cmp.Diff(gotPassword, []byte("secure-test-password")); diff != "" {
		t.Errorf("Connect did not propagate password correctly to OSB client: %s", diff)
	}

	if diff := cmp.Diff(gotURL, "test.com"); diff != "" {
		t.Errorf("Connect did not propagate URL correctly to OSB client: %s", diff)
	}
}

// withStatusInstanceID sets the InstanceID in withStatusInstanceID function, which is
// typically determined by the Observe method based on the annotation from the Create method.
// Note that the Create method will overwrite this field if no InstanceID annotation is present.
func withStatusInstanceID(id string) serviceInstanceOption {
	return func(pg *v1.ServiceInstance) {
		pg.Status.AtProvider.InstanceID = id
	}
}

func withProviderRef(name string) serviceInstanceOption {
	return func(pg *v1.ServiceInstance) {
		pg.Spec.ProviderConfigReference = &xpv1.Reference{Name: name}
	}
}

func withServiceID(id string) serviceInstanceOption {
	return func(pg *v1.ServiceInstance) {
		newID := id
		pg.Status.AtProvider.ServiceID = newID
	}
}

func withPlanID(id string) serviceInstanceOption {
	return func(pg *v1.ServiceInstance) {
		newID := id
		pg.Status.AtProvider.PlanID = newID
	}
}

func withServiceName(name string) serviceInstanceOption {
	return func(pg *v1.ServiceInstance) {
		pg.Spec.ForProvider.ServiceName = &name
	}
}

func withPlanName(name string) serviceInstanceOption {
	return func(pg *v1.ServiceInstance) {
		pg.Spec.ForProvider.PlanName = &name
	}
}

func withEmptyParameters() serviceInstanceOption {
	return func(pg *v1.ServiceInstance) {
		pg.Spec.ForProvider.Parameters = map[string]intstr.IntOrString{}
	}
}

func withIntParameter(key string, value int) serviceInstanceOption {
	return func(pg *v1.ServiceInstance) {
		if pg.Spec.ForProvider.Parameters == nil {
			pg.Spec.ForProvider.Parameters = map[string]intstr.IntOrString{}
		}
		pg.Spec.ForProvider.Parameters[key] = intstr.FromInt(value)
	}
}

func withStringParameter(key string, value string) serviceInstanceOption {
	return func(pg *v1.ServiceInstance) {
		if pg.Spec.ForProvider.Parameters == nil {
			pg.Spec.ForProvider.Parameters = map[string]intstr.IntOrString{}
		}
		pg.Spec.ForProvider.Parameters[key] = intstr.FromString(value)
	}
}

func withStatusIntParameter(key string, value int) serviceInstanceOption {
	return func(pg *v1.ServiceInstance) {
		if pg.Status.AtProvider.Parameters == nil {
			pg.Status.AtProvider.Parameters = map[string]intstr.IntOrString{}
		}
		pg.Status.AtProvider.Parameters[key] = intstr.FromInt(value)
	}
}

func withAnnotation(key, value string) serviceInstanceOption {
	return func(pg *v1.ServiceInstance) {
		meta.AddAnnotations(pg, map[string]string{key: value})
	}
}

func withState(state string) serviceInstanceOption {
	return func(pg *v1.ServiceInstance) {
		pg.Status.AtProvider.State = state
	}
}

func withCondition(condition xpv1.Condition) serviceInstanceOption {
	return func(pg *v1.ServiceInstance) {
		pg.Status.SetConditions(condition)
	}
}

func withPendingOperation(operationKey string) serviceInstanceOption {
	return func(pg *v1.ServiceInstance) {
		pg.Status.PendingOperation = &operationKey
	}
}

func operationKey(key string) *osbclient.OperationKey {
	opKey := osbclient.OperationKey(key)
	return &opKey
}

func expectPendingOperation(t *testing.T, mr resource.Managed, expectedOperation string) {
	dsi, ok := mr.(*v1.ServiceInstance)
	if !ok {
		return
	}

	pendingOperation := dsi.Status.PendingOperation
	switch {
	case expectedOperation == "" && pendingOperation != nil:
		t.Errorf("Delete(...): expected no pending operation, got %v", *pendingOperation)
	case expectedOperation != "" && pendingOperation == nil:
		t.Errorf("Delete(...): expected pending operation to be %v, but got none",
			expectedOperation)
	case expectedOperation != "" && *pendingOperation != expectedOperation:
		t.Errorf("Delete(...): expected pending operation to be %v, got %v",
			expectedOperation, *pendingOperation)
	}
}
