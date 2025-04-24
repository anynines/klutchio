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
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	osbclient "github.com/anynines/klutchio/clients/a9s-open-service-broker"
	fakeosb "github.com/anynines/klutchio/clients/a9s-open-service-broker/fake"
	v1 "github.com/anynines/klutchio/provider-anynines/apis/servicebinding/v1"
	dsv1 "github.com/anynines/klutchio/provider-anynines/apis/serviceinstance/v1"
	apisv1 "github.com/anynines/klutchio/provider-anynines/apis/v1"
	a9stest "github.com/anynines/klutchio/provider-anynines/internal/controller/test"
	utilerr "github.com/anynines/klutchio/provider-anynines/pkg/utilerr"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/test"
	corev1 "k8s.io/api/core/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

var defaultBindingParameters = &v1.ServiceBindingParameters{
	InstanceName:      "postgres-1",
	AcceptsIncomplete: false,
}

func TestMain(m *testing.M) {
	if err := apisv1.AddToScheme(scheme.Scheme); err != nil {
		panic("failed to add API github.com/anynines/klutchio/provider-anynines/apis/v1alpha1 to scheme")
	}

	os.Exit(m.Run())
}

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

func TestObserve(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		sb                          resource.Managed
		expectedExternalObservation managed.ExternalObservation
		expectedServiceBinding      resource.Managed
		getInstancesReaction        *GetInstancesReaction
		serviceInstance             dsv1.ServiceInstance
		otherResources              []client.Object
		kube                        client.Client
	}{
		"sb_not_initialized_yet": {
			sb: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				withAtProvider("Pending", 0),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"",
					"",
				),
			),
			serviceInstance: *serviceInstance(
				withStatusInstanceID("6e2c036c-254f-11ee-be56-0242ac120002"),
				afterInstanceCreation(),
				withAnnotations(
					map[string]string{
						"crossplane.io/claim-name":      "postgres-1",
						"crossplane.io/claim-namespace": "test",
					}),
				withStatusServiceID("76c0089e-254e-11ee-be56-0242ac120002"),
				withStatusPlanID("63d05ec8-254e-11ee-be56-0242ac120002"),
			),
			expectedExternalObservation: managed.ExternalObservation{},
			expectedServiceBinding: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				withAtProvider("Pending", 0),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"",
					"",
				),
			),
		},
		"sb_does_not_exist_on_broker": {
			sb: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				withAtProvider("Pending", 0),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"test.URL.com",
					"5432",
				),
			),
			expectedExternalObservation: managed.ExternalObservation{
				ResourceExists:          false,
				ResourceUpToDate:        true,
				ResourceLateInitialized: false,
				ConnectionDetails:       managed.ConnectionDetails{},
			},
			expectedServiceBinding: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				withAtProvider("Pending", 0),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"test.URL.com",
					"5432",
				),
			),
			serviceInstance: *serviceInstance(
				withStatusInstanceID("6e2c036c-254f-11ee-be56-0242ac120002"),
				afterInstanceCreation(),
				withAnnotations(
					map[string]string{
						"crossplane.io/claim-name":      "postgres-1",
						"crossplane.io/claim-namespace": "test",
					}),
				withStatusServiceID("76c0089e-254e-11ee-be56-0242ac120002"),
				withStatusPlanID("63d05ec8-254e-11ee-be56-0242ac120002"),
			),

			getInstancesReaction: &GetInstancesReaction{
				Response: &osbclient.GetInstancesResponse{},
				Error:    nil,
			},
		},
		"sb_does_not_exist_but_unrelated_sb_exists": {
			sb: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				withAtProvider("Pending", 0),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"test.URL.com",
					"5432",
				),
			),
			expectedExternalObservation: managed.ExternalObservation{
				ResourceExists:          false,
				ResourceUpToDate:        true,
				ResourceLateInitialized: false,
				ConnectionDetails:       managed.ConnectionDetails{},
			},
			serviceInstance: *serviceInstance(
				withStatusInstanceID("6e2c036c-254f-11ee-be56-0242ac120002"),
				afterInstanceCreation(),
				withAnnotations(
					map[string]string{
						"crossplane.io/claim-name":      "postgres-1",
						"crossplane.io/claim-namespace": "test",
					}),
				withStatusServiceID("76c0089e-254e-11ee-be56-0242ac120002"),
				withStatusPlanID("63d05ec8-254e-11ee-be56-0242ac120002"),
			),
			expectedServiceBinding: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"test.URL.com",
					"5432",
				),
				withAtProvider("Pending", 0),
			),
			getInstancesReaction: &GetInstancesReaction{
				Response: &osbclient.GetInstancesResponse{
					Resources: []osbclient.GetInstanceResponse{
						{
							GUIDAtTenant: "6e2c036c-254f-11ee-be56-0242ac120002",
							Credentials: []osbclient.Credential{
								{
									GUIDAtTenant: "unrelated-sb-id",
								},
							},
						},
					},
				},
				Error: nil,
			},
		},
		"sb_exists_and_available_condition_is_not_set": {
			sb: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"test.URL.com",
					"5432",
				),
				withAtProvider("Pending", 0),
			),
			serviceInstance: *serviceInstance(
				withStatusInstanceID("6e2c036c-254f-11ee-be56-0242ac120002"),
				afterInstanceCreation(),
				withAnnotations(
					map[string]string{
						"crossplane.io/claim-name":      "postgres-1",
						"crossplane.io/claim-namespace": "test",
					}),
				withStatusServiceID("76c0089e-254e-11ee-be56-0242ac120002"),
				withStatusPlanID("63d05ec8-254e-11ee-be56-0242ac120002"),
			),
			expectedExternalObservation: managed.ExternalObservation{
				ResourceExists:          true,
				ResourceUpToDate:        true,
				ResourceLateInitialized: false,
				ConnectionDetails:       managed.ConnectionDetails{},
			},
			expectedServiceBinding: serviceBinding(
				withServiceBindingParameters(&v1.ServiceBindingParameters{
					InstanceName:      "postgres-1",
					AcceptsIncomplete: false,
				}),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"test.URL.com",
					"5432",
				),
				withConditions(xpv1.Available()),
				withAtProvider("Created", 0),
			),
			getInstancesReaction: &GetInstancesReaction{
				Response: &osbclient.GetInstancesResponse{
					Resources: []osbclient.GetInstanceResponse{
						{
							PlanGUID:     "63d05ec8-254e-11ee-be56-0242ac120002",
							GUIDAtTenant: "6e2c036c-254f-11ee-be56-0242ac120002",
							Credentials: []osbclient.Credential{
								{
									GUIDAtTenant: "1a6a6b3e-254e-11ee-be56-0242ac120002",
								},
							},
						},
					},
				},
				Error: nil,
			},
		},
		"sb_exists_and_unrelated_sb_exists": {
			sb: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				withAtProvider("Pending", 0),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"test.URL.com",
					"5432",
				),
			),
			serviceInstance: *serviceInstance(
				withStatusInstanceID("6e2c036c-254f-11ee-be56-0242ac120002"),
				afterInstanceCreation(),
				withAnnotations(
					map[string]string{
						"crossplane.io/claim-name":      "postgres-1",
						"crossplane.io/claim-namespace": "test",
					}),
				withStatusServiceID("76c0089e-254e-11ee-be56-0242ac120002"),
				withStatusPlanID("63d05ec8-254e-11ee-be56-0242ac120002"),
			),
			expectedExternalObservation: managed.ExternalObservation{
				ResourceExists:          true,
				ResourceUpToDate:        true,
				ResourceLateInitialized: false,
				ConnectionDetails:       managed.ConnectionDetails{},
			},
			expectedServiceBinding: serviceBinding(
				withServiceBindingParameters(&v1.ServiceBindingParameters{
					InstanceName:      "postgres-1",
					AcceptsIncomplete: false,
				}),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"test.URL.com",
					"5432",
				),
				withConditions(xpv1.Available()),
				withAtProvider("Created", 0),
			),
			getInstancesReaction: &GetInstancesReaction{
				Response: &osbclient.GetInstancesResponse{
					Resources: []osbclient.GetInstanceResponse{
						{
							PlanGUID:     "63d05ec8-254e-11ee-be56-0242ac120002",
							GUIDAtTenant: "6e2c036c-254f-11ee-be56-0242ac120002",
							Credentials: []osbclient.Credential{
								{
									GUIDAtTenant: "1a6a6b3e-254e-11ee-be56-0242ac120002",
								},
								{
									GUIDAtTenant: "unrelated-sb-id",
								},
							},
						},
					},
				},
				Error: nil,
			},
		},
		"sb_is_being_deleted": {
			sb: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"test.URL.com",
					"5432",
				),
				withAtProvider("Created", 0),
				deletionTimestamp(),
			),
			serviceInstance: *serviceInstance(
				withStatusInstanceID("6e2c036c-254f-11ee-be56-0242ac120002"),
				afterInstanceCreation(),
				withAnnotations(
					map[string]string{
						"crossplane.io/claim-name":      "postgres-1",
						"crossplane.io/claim-namespace": "test",
					}),
				withStatusServiceID("76c0089e-254e-11ee-be56-0242ac120002"),
				withStatusPlanID("63d05ec8-254e-11ee-be56-0242ac120002"),
			),
			expectedExternalObservation: managed.ExternalObservation{
				ResourceExists:          true,
				ResourceUpToDate:        true,
				ResourceLateInitialized: false,
				ConnectionDetails:       managed.ConnectionDetails{},
			},
			expectedServiceBinding: serviceBinding(
				withServiceBindingParameters(&v1.ServiceBindingParameters{
					InstanceName:      "postgres-1",
					AcceptsIncomplete: false,
				}),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"test.URL.com",
					"5432",
				),
				deletionTimestamp(),
				withConditions(xpv1.Deleting()),
				withAtProvider("Deleting", 0),
			),
			getInstancesReaction: &GetInstancesReaction{
				Response: &osbclient.GetInstancesResponse{
					Resources: []osbclient.GetInstanceResponse{
						{
							PlanGUID:     "63d05ec8-254e-11ee-be56-0242ac120002",
							GUIDAtTenant: "6e2c036c-254f-11ee-be56-0242ac120002",
							Credentials: []osbclient.Credential{
								{
									GUIDAtTenant: "1a6a6b3e-254e-11ee-be56-0242ac120002",
								},
							},
						},
					},
				},
				Error: nil,
			},
		},
		"sb_exists_and_available_condition_is_set": {
			sb: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"test.URL.com",
					"5432",
				),
				withConditions(xpv1.Available()),
				withAtProvider("Created", 0),
			),
			serviceInstance: *serviceInstance(
				withStatusInstanceID("6e2c036c-254f-11ee-be56-0242ac120002"),
				afterInstanceCreation(),
				withAnnotations(
					map[string]string{
						"crossplane.io/claim-name":      "postgres-1",
						"crossplane.io/claim-namespace": "test",
					}),
				withStatusServiceID("76c0089e-254e-11ee-be56-0242ac120002"),
				withStatusPlanID("63d05ec8-254e-11ee-be56-0242ac120002"),
			),
			expectedExternalObservation: managed.ExternalObservation{
				ResourceExists:          true,
				ResourceUpToDate:        true,
				ResourceLateInitialized: false,
				ConnectionDetails:       managed.ConnectionDetails{},
			},
			expectedServiceBinding: serviceBinding(
				withServiceBindingParameters(&v1.ServiceBindingParameters{
					InstanceName:      "postgres-1",
					AcceptsIncomplete: false,
				}),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"test.URL.com",
					"5432",
				),
				withConditions(xpv1.Available()),
				withAtProvider("Created", 0),
			),
			getInstancesReaction: &GetInstancesReaction{
				Response: &osbclient.GetInstancesResponse{
					Resources: []osbclient.GetInstanceResponse{
						{
							PlanGUID:     "63d05ec8-254e-11ee-be56-0242ac120002",
							GUIDAtTenant: "6e2c036c-254f-11ee-be56-0242ac120002",
							Credentials: []osbclient.Credential{
								{
									GUIDAtTenant: "1a6a6b3e-254e-11ee-be56-0242ac120002",
								},
							},
						},
					},
				},
				Error: nil,
			},
		},
		"fails_not_a_service_binding": {
			sb:                     &dsv1.ServiceInstance{},
			expectedServiceBinding: &dsv1.ServiceInstance{},
			getInstancesReaction: &GetInstancesReaction{
				Error: utilerr.ErrInternal,
			},
		},
		"instanceNotFound": {
			sb: serviceBinding(
				withInstanceName("dummy"),
			),
			serviceInstance: *serviceInstance(
				withStatusInstanceID("6e2c036c-254f-11ee-be56-0242ac120002"),
				afterInstanceCreation(),
				withAnnotations(
					map[string]string{
						"crossplane.io/claim-name":      "postgres-1",
						"crossplane.io/claim-namespace": "test",
					}),
				withStatusServiceID("76c0089e-254e-11ee-be56-0242ac120002"),
				withStatusPlanID("63d05ec8-254e-11ee-be56-0242ac120002"),
			),
			getInstancesReaction: &GetInstancesReaction{
				Error: errServiceInstanceNotFound,
			},
			expectedServiceBinding: serviceBinding(
				withInstanceName("dummy"),
			),
		},

		"moreThanOneMatchingInstancesExist": {
			sb: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
			),
			serviceInstance: *serviceInstance(
				withStatusInstanceID("6e2c036c-254f-11ee-be56-0242ac120002"),
				afterInstanceCreation(),
				withAnnotations(
					map[string]string{
						"crossplane.io/claim-name":      "postgres-1",
						"crossplane.io/claim-namespace": "test",
					}),
				withStatusServiceID("76c0089e-254e-11ee-be56-0242ac120002"),
				withStatusPlanID("63d05ec8-254e-11ee-be56-0242ac120002"),
			),
			otherResources: []client.Object{
				serviceInstance(
					serviceInstanceWithName("postgres-1-sdjl"),
					withAnnotations(
						map[string]string{
							"crossplane.io/claim-name":      "postgres-1",
							"crossplane.io/claim-namespace": "test",
						}),
					serviceInstanceWithStatus(
						dsv1.ServiceInstanceObservation{
							InstanceID: "23df2cf9-2ecc-414c-9333-6401f0c54365",
							ServiceID:  "76c0089e-254e-11ee-be56-0242ac120002",
							PlanID:     "63d05ec8-254e-11ee-be56-0242ac120002",
						},
					),
				),
			},
			expectedServiceBinding: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
			),
			getInstancesReaction: &GetInstancesReaction{
				Error: utilerr.ErrInternal,
			},
		},
		"crossNamespace": {
			sb: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
			),
			serviceInstance: *serviceInstance(serviceInstanceWithLabels(
				map[string]string{
					"crossplane.io/claim-name":      "postgres-1",
					"crossplane.io/claim-namespace": "test-1",
				},
			),
				serviceInstanceWithStatus(
					dsv1.ServiceInstanceObservation{
						InstanceID: "23df2cf9-2ecc-414c-9333-6401f0c54365",
					},
				)),

			getInstancesReaction: &GetInstancesReaction{
				Error: errServiceInstanceNotFound,
			},
			expectedServiceBinding: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
			),
		},
		"fails_instance_not_found": {
			sb: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
			),
			getInstancesReaction: &GetInstancesReaction{
				Error: errServiceInstanceNotFound,
			},
			expectedServiceBinding: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
			),
		},
		"fails_service_id_not_initialized": {
			sb: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
			),
			serviceInstance: *serviceInstance(
				withStatusInstanceID("6e2c036c-254f-11ee-be56-0242ac120002"),
				afterInstanceCreation(),
				withAnnotations(
					map[string]string{
						"crossplane.io/claim-name":      "postgres-1",
						"crossplane.io/claim-namespace": "test",
					}),
				withStatusPlanID("63d05ec8-254e-11ee-be56-0242ac120002"),
			),
			getInstancesReaction: &GetInstancesReaction{
				Error: errInstanceNotReady,
			},
			expectedServiceBinding: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"",
					"",
					"",
				),
			),
		},
		"status_connection_details_not_initialized": {
			sb: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"",
					"",
				),
			),
			serviceInstance: *serviceInstance(
				afterInstanceCreation(),
				withAnnotations(
					map[string]string{
						"crossplane.io/claim-name":      "postgres-1",
						"crossplane.io/claim-namespace": "test",
					}),
				serviceInstanceWithStatus(
					dsv1.ServiceInstanceObservation{
						InstanceID: "6e2c036c-254f-11ee-be56-0242ac120002",
						PlanID:     "63d05ec8-254e-11ee-be56-0242ac120002",
						ServiceID:  "76c0089e-254e-11ee-be56-0242ac120002",
					},
				),
			),
			otherResources: []client.Object{
				a9stest.Secret(a9stest.Name[corev1.Secret]("test-sb-creds"),
					a9stest.Namespace[corev1.Secret]("test"),
					a9stest.WithKey("host", "test.URL.com"),
					a9stest.WithKey("port", "5432"),
				),
			},
			getInstancesReaction: &GetInstancesReaction{
				Error: errServiceBindingIsUnset,
			},
			expectedServiceBinding: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"test.URL.com",
					"5432",
				),
			),
		},
		"status_connection_details_not_initialized_secret_not_found": {
			sb: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"",
					"",
				),
			),
			serviceInstance: *serviceInstance(
				afterInstanceCreation(),
				withAnnotations(
					map[string]string{
						"crossplane.io/claim-name":      "postgres-1",
						"crossplane.io/claim-namespace": "test",
					}),
				serviceInstanceWithStatus(
					dsv1.ServiceInstanceObservation{
						InstanceID: "6e2c036c-254f-11ee-be56-0242ac120002",
						PlanID:     "63d05ec8-254e-11ee-be56-0242ac120002",
						ServiceID:  "76c0089e-254e-11ee-be56-0242ac120002",
					},
				),
			),
			getInstancesReaction: &GetInstancesReaction{
				Error: fmt.Errorf("Internal error in provider"),
			},
			expectedServiceBinding: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"",
					"",
				),
			),
		},
		"fails_plan_id_not_initialized": {
			sb: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
			),
			serviceInstance: *serviceInstance(
				withStatusInstanceID("6e2c036c-254f-11ee-be56-0242ac120002"),
				afterInstanceCreation(),
				withAnnotations(
					map[string]string{
						"crossplane.io/claim-name":      "postgres-1",
						"crossplane.io/claim-namespace": "test",
					}),
				withStatusServiceID("76c0089e-254e-11ee-be56-0242ac120002"),
			),
			getInstancesReaction: &GetInstancesReaction{
				Error: errInstanceNotReady,
			},
			expectedServiceBinding: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"",
					"",
				),
			),
		},
		"fails_list_not_gettable_from_kubernetes": {
			sb: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
			),
			kube: fake.NewClientBuilder().Build(),

			getInstancesReaction: &GetInstancesReaction{
				Error: utilerr.ErrInternal,
			},
			expectedServiceBinding: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
			),
		},
		"sb_exists_but_plainID_is_out_of_sync": {
			sb: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"test.URL.com",
					"5432",
				),
				withConditions(xpv1.Available()),
				withAtProvider("Created", 0),
			),
			serviceInstance: *serviceInstance(
				withStatusInstanceID("6e2c036c-254f-11ee-be56-0242ac120002"),
				afterInstanceCreation(),
				withAnnotations(
					map[string]string{
						"crossplane.io/claim-name":      "postgres-1",
						"crossplane.io/claim-namespace": "test",
					}),
				withStatusServiceID("76c0089e-254e-11ee-be56-0242ac120002"),
				withStatusPlanID("63d05ec8-254e-11ee-be56-0242ac120003"),
			),
			expectedExternalObservation: managed.ExternalObservation{
				ResourceExists:          true,
				ResourceUpToDate:        true,
				ResourceLateInitialized: false,
				ConnectionDetails:       managed.ConnectionDetails{},
			},
			expectedServiceBinding: serviceBinding(
				withServiceBindingParameters(&v1.ServiceBindingParameters{
					InstanceName:      "postgres-1",
					AcceptsIncomplete: false,
				}),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120003",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"test.URL.com",
					"5432",
				),
				withConditions(xpv1.Available()),
				withAtProvider("Created", 0),
			),
			getInstancesReaction: &GetInstancesReaction{
				Response: &osbclient.GetInstancesResponse{
					Resources: []osbclient.GetInstanceResponse{
						{
							PlanGUID:     "63d05ec8-254e-11ee-be56-0242ac120003",
							GUIDAtTenant: "6e2c036c-254f-11ee-be56-0242ac120002",
							Credentials: []osbclient.Credential{
								{
									GUIDAtTenant: "1a6a6b3e-254e-11ee-be56-0242ac120002",
								},
							},
						},
					},
				},
				Error: nil,
			},
		},
	}

	for name, testCase := range cases {
		// Rebind testCase into this lexical scope. Details on the why at
		// https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		testCase := testCase

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fakeOSB := fakeosb.NewFakeClient(fakeosb.FakeClientConfiguration{
				GetInstancesReaction: testCase.getInstancesReaction,
			})

			if testCase.kube == nil {
				testCase.kube = newKubeMock(testCase.serviceInstance.DeepCopyObject(),
					testCase.otherResources)
			}

			e := utilerr.Decorator{
				Logger: a9stest.TestLogger(t),
				ExternalClient: &external{
					kube:    testCase.kube,
					service: fakeOSB,
				},
			}

			got, err := e.Observe(context.TODO(), testCase.sb)

			if testCase.getInstancesReaction == nil {
				if err != nil {
					t.Errorf("Unexpected error occurred when trying to observe ServiceBinding %+v : %s", testCase.sb, err)
				}
			} else {
				if diff := cmp.Diff(testCase.getInstancesReaction.Error, err, test.EquateErrors()); diff != "" {
					t.Errorf("Observe(...): -want error, +got error:\n%s", diff)
				}
			}

			if dif := cmp.Diff(testCase.expectedExternalObservation, got); dif != "" {
				t.Errorf("Return from Observe differs from expected externalObservation: %s", dif)
			}

			if diff := cmp.Diff(testCase.expectedServiceBinding,
				testCase.sb,
				test.EquateConditions()); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestServiceBindingConnectionDetailsStatusPopulation(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		sb                          resource.Managed
		expectedExternalObservation managed.ExternalObservation
		expectedServiceBinding      resource.Managed
		getInstancesReaction        *GetInstancesReaction
		serviceInstance             dsv1.ServiceInstance
		otherResources              []client.Object
		kube                        client.Client
	}{
		"status_connection_details_logme2": {
			sb: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"",
					"",
				),
			),
			serviceInstance: *serviceInstance(
				afterInstanceCreation(),
				withAnnotations(
					map[string]string{
						"crossplane.io/claim-name":      "logme-1",
						"crossplane.io/claim-namespace": "test",
					}),
				serviceInstanceWithStatus(
					dsv1.ServiceInstanceObservation{
						InstanceID: "6e2c036c-254f-11ee-be56-0242ac120002",
						PlanID:     "63d05ec8-254e-11ee-be56-0242ac120002",
						ServiceID:  "76c0089e-254e-11ee-be56-0242ac120002",
					},
				),
			),
			otherResources: []client.Object{
				a9stest.Secret(a9stest.Name[corev1.Secret]("test-sb-creds"),
					a9stest.Namespace[corev1.Secret]("test"),
					a9stest.WithKey("cacrt", "-----BEGIN CERTIFICATE-----\nMIIDGzszfasde....8tn9ebYK0k2Qt\n-----END CERTIFICATE-----\n"),
					a9stest.WithKey("host", "https://d765411-os.service.dc1.dsf2.a9ssvc:9200"),
					a9stest.WithKey("password", "a9scbe8462ee571f12d95b3a950e1bf8b2445a59983"),
					a9stest.WithKey("syslog_drain_url", "syslog-tls://d765411-fluentd.service.dc1.dsf2.a9ssvc:6514"),
					a9stest.WithKey("username", "a9s94bd153ddf5978f1eae7c88b57a27721430600d2"),
				),
			},
			getInstancesReaction: &GetInstancesReaction{
				Error: errServiceBindingIsUnset,
			},
			expectedServiceBinding: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"https://d765411-os.service.dc1.dsf2.a9ssvc",
					"9200",
				),
			),
		},
		"status_connection_details_mariadb": {
			sb: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"",
					"",
				),
			),
			serviceInstance: *serviceInstance(
				afterInstanceCreation(),
				withAnnotations(
					map[string]string{
						"crossplane.io/claim-name":      "mariadb",
						"crossplane.io/claim-namespace": "test",
					}),
				serviceInstanceWithStatus(
					dsv1.ServiceInstanceObservation{
						InstanceID: "6e2c036c-254f-11ee-be56-0242ac120002",
						PlanID:     "63d05ec8-254e-11ee-be56-0242ac120002",
						ServiceID:  "76c0089e-254e-11ee-be56-0242ac120002",
					},
				),
			),
			otherResources: []client.Object{
				a9stest.Secret(a9stest.Name[corev1.Secret]("test-sb-creds"),
					a9stest.Namespace[corev1.Secret]("test"),
					a9stest.WithKey("host", "d15575b.service.dc1.a9s-mariadb-consul"),
					a9stest.WithKey("name", "d15575b"),
					a9stest.WithKey("password", "a9s-password"),
					a9stest.WithKey("port", "3306"),
					a9stest.WithKey("uri", "mysql://a9s-brk-usr:a9s-password@d15575b.service.dc1.a9s-mariadb-consul:3306/d15575b"),
					a9stest.WithKey("username", "a9s-brk-usr"),
				),
			},
			getInstancesReaction: &GetInstancesReaction{
				Error: errServiceBindingIsUnset,
			},
			expectedServiceBinding: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"d15575b.service.dc1.a9s-mariadb-consul",
					"3306",
				),
			),
		},
		"status_connection_details_messaging": {
			sb: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"",
					"",
				),
			),
			serviceInstance: *serviceInstance(
				afterInstanceCreation(),
				withAnnotations(
					map[string]string{
						"crossplane.io/claim-name":      "msg",
						"crossplane.io/claim-namespace": "test",
					}),
				serviceInstanceWithStatus(
					dsv1.ServiceInstanceObservation{
						InstanceID: "6e2c036c-254f-11ee-be56-0242ac120002",
						PlanID:     "63d05ec8-254e-11ee-be56-0242ac120002",
						ServiceID:  "76c0089e-254e-11ee-be56-0242ac120002",
					},
				),
			),
			otherResources: []client.Object{
				a9stest.Secret(a9stest.Name[corev1.Secret]("test-sb-creds"),
					a9stest.Namespace[corev1.Secret]("test"),
					a9stest.WithKey("host", "hostname.node.dcx.consul"),
					a9stest.WithKey("hosts", ` ["hostname.node.dcx.consul"]`),
					a9stest.WithKey("password", "password"),
					a9stest.WithKey("port", "5672"),
					a9stest.WithKey("http_api_uri", "http://username:password@hostname.node.dcx.consul/api/"),
					a9stest.WithKey("http_api_uris", "['http://username:password@hostname.node.dcx.consul/api/']"),
					a9stest.WithKey("protocols", `{
						"amqp": {
							"host": "hostname.node.dcx.consul",
							"hosts": [
								"hostname.node.dcx.consul"
							],
							"password": "password",
							"port": 5672,
							"ssl": false,
							"uri": "amqp://username:password@hostname.node.dcx.consul:5672",
							"username": "username"
						},
						"management": {
							"username": "username",
							"password": "password",
							"path": "/api",
							"ssl": false,
							"host": "hostname.node.dcx.consul",
							"hosts": [
								"hostname.node.dcx.consul"
							],
							"uri": "http://username:password@hostname.node.dcx.consul",
							"uris": ["http://username:password@hostname.node.dcx.consul"]
						}
					}`),
				),
			},
			getInstancesReaction: &GetInstancesReaction{
				Error: errServiceBindingIsUnset,
			},
			expectedServiceBinding: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"hostname.node.dcx.consul",
					"5672",
				),
			),
		},
		"status_connection_details_mongodb": {
			sb: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"",
					"",
				),
			),
			serviceInstance: *serviceInstance(
				afterInstanceCreation(),
				withAnnotations(
					map[string]string{
						"crossplane.io/claim-name":      "mongodb",
						"crossplane.io/claim-namespace": "test",
					}),
				serviceInstanceWithStatus(
					dsv1.ServiceInstanceObservation{
						InstanceID: "6e2c036c-254f-11ee-be56-0242ac120002",
						PlanID:     "63d05ec8-254e-11ee-be56-0242ac120002",
						ServiceID:  "76c0089e-254e-11ee-be56-0242ac120002",
					},
				),
			),
			otherResources: []client.Object{
				a9stest.Secret(a9stest.Name[corev1.Secret]("test-sb-creds"),
					a9stest.Namespace[corev1.Secret]("test"),
					a9stest.WithKey("default_database", "d22906"),
					a9stest.WithKey("password", "EXAMPLE-PASSWORD"),
					a9stest.WithKey("username", "EXAMPLE-USERNAME"),
					a9stest.WithKey("hosts", ` ["test-mongodb-0.node.dc1.dsf2.a9ssvc:27017"]`),
					a9stest.WithKey("uri", "mongodb://a9s-brk-usr-test:test@test-mongodb-0.node.dc1.dsf2.a9ssvc:27017/test?ssl=true"),
				),
			},
			getInstancesReaction: &GetInstancesReaction{
				Error: errServiceBindingIsUnset,
			},
			expectedServiceBinding: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"test-mongodb-0.node.dc1.dsf2.a9ssvc",
					"27017",
				),
			),
		},
		"status_connection_details_postgresql": {
			sb: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"",
					"",
				),
			),
			serviceInstance: *serviceInstance(
				afterInstanceCreation(),
				withAnnotations(
					map[string]string{
						"crossplane.io/claim-name":      "postgres-1",
						"crossplane.io/claim-namespace": "test",
					}),
				serviceInstanceWithStatus(
					dsv1.ServiceInstanceObservation{
						InstanceID: "6e2c036c-254f-11ee-be56-0242ac120002",
						PlanID:     "63d05ec8-254e-11ee-be56-0242ac120002",
						ServiceID:  "76c0089e-254e-11ee-be56-0242ac120002",
					},
				),
			),
			otherResources: []client.Object{
				a9stest.Secret(a9stest.Name[corev1.Secret]("test-sb-creds"),
					a9stest.Namespace[corev1.Secret]("test"),
					a9stest.WithKey("host", "EXAMPLE-HOST"),
					a9stest.WithKey("hosts", "[EXAMPLE-HOST]"),
					a9stest.WithKey("name", "d92e2bd"),
					a9stest.WithKey("password", "EXAMPLE-PASSWORD"),
					a9stest.WithKey("port", "5432"),
					a9stest.WithKey("uri", "EXAMPLE-URI"),
					a9stest.WithKey("username", "EXAMPLE-USERNAME"),
				),
			},
			getInstancesReaction: &GetInstancesReaction{
				Error: errServiceBindingIsUnset,
			},
			expectedServiceBinding: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"EXAMPLE-HOST",
					"5432",
				),
			),
		},
		"status_connection_details_prometheus": {
			sb: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"",
					"",
				),
			),
			serviceInstance: *serviceInstance(
				afterInstanceCreation(),
				withAnnotations(
					map[string]string{
						"crossplane.io/claim-name":      "prometheus",
						"crossplane.io/claim-namespace": "test",
					}),
				serviceInstanceWithStatus(
					dsv1.ServiceInstanceObservation{
						InstanceID: "6e2c036c-254f-11ee-be56-0242ac120002",
						PlanID:     "63d05ec8-254e-11ee-be56-0242ac120002",
						ServiceID:  "76c0089e-254e-11ee-be56-0242ac120002",
					},
				),
			),
			otherResources: []client.Object{
				a9stest.Secret(a9stest.Name[corev1.Secret]("test-sb-creds"),
					a9stest.Namespace[corev1.Secret]("test"),
					a9stest.WithKey("username", "EXAMPLE-USERNAME"),
					a9stest.WithKey("password", "EXAMPLE-PASSWORD"),
					a9stest.WithKey("alertmanager_urls", `[http://test-alertmanager-0.node.dc1.dsf2.a9ssvc:9093/service-instances/test/alertmanager/]`),
					a9stest.WithKey("grafana_urls", `["http://test-grafana-0.node.dc1.dsf2.a9ssvc:3000/service-instances/test/grafana/"]`),
					a9stest.WithKey("prometheus_urls", `["http://test-prometheus-0.node.dc1.dsf2.a9ssvc:9090/service-instances/test/prometheus/"]`),
					a9stest.WithKey("graphite_exporter_port", "9109"),
					a9stest.WithKey("graphite_exporters", `["test-prometheus-0.node.dc1.dsf2.a9ssvc"]`),
				),
			},
			getInstancesReaction: &GetInstancesReaction{
				Error: errServiceBindingIsUnset,
			},
			expectedServiceBinding: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"http://test-prometheus-0.node.dc1.dsf2.a9ssvc",
					"9090",
				),
			),
		},
		"status_connection_details_search": {
			sb: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"",
					"",
				),
			),
			serviceInstance: *serviceInstance(
				afterInstanceCreation(),
				withAnnotations(
					map[string]string{
						"crossplane.io/claim-name":      "search",
						"crossplane.io/claim-namespace": "test",
					}),
				serviceInstanceWithStatus(
					dsv1.ServiceInstanceObservation{
						InstanceID: "6e2c036c-254f-11ee-be56-0242ac120002",
						PlanID:     "63d05ec8-254e-11ee-be56-0242ac120002",
						ServiceID:  "76c0089e-254e-11ee-be56-0242ac120002",
					},
				),
			),
			otherResources: []client.Object{
				a9stest.Secret(a9stest.Name[corev1.Secret]("test-sb-creds"),
					a9stest.Namespace[corev1.Secret]("test"),
					a9stest.WithKey("host", `["EXAMPLE_HOST"]`),
					a9stest.WithKey("hosts", `["EXAMPLE_HOST"]`),
					a9stest.WithKey("password", "EXAMPLE_PASSWORD"),
					a9stest.WithKey("username", "EXAMPLE_USER"),
					a9stest.WithKey("scheme", "http"),
					a9stest.WithKey("port", "9200"),
				),
			},
			getInstancesReaction: &GetInstancesReaction{
				Error: errServiceBindingIsUnset,
			},
			expectedServiceBinding: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"EXAMPLE-HOST",
					"9200",
				),
			),
		},
	}

	for name, testCase := range cases {

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fakeOSB := fakeosb.NewFakeClient(fakeosb.FakeClientConfiguration{
				GetInstancesReaction: testCase.getInstancesReaction,
			})

			if testCase.kube == nil {
				testCase.kube = newKubeMock(testCase.serviceInstance.DeepCopyObject(),
					testCase.otherResources)
			}

			e := utilerr.Decorator{
				Logger: a9stest.TestLogger(t),
				ExternalClient: &external{
					kube:    testCase.kube,
					service: fakeOSB,
				},
			}

			got, err := e.Observe(context.TODO(), testCase.sb)

			if testCase.getInstancesReaction == nil {
				if err != nil {
					t.Errorf("Unexpected error occurred when trying to observe ServiceBinding %+v : %s", testCase.sb, err)
				}
			} else {
				if diff := cmp.Diff(testCase.getInstancesReaction.Error, err, test.EquateErrors()); diff != "" {
					t.Errorf("Observe(...): -want error, +got error:\n%s", diff)
				}
			}

			if dif := cmp.Diff(testCase.expectedExternalObservation, got); dif != "" {
				t.Errorf("Return from Observe differs from expected externalObservation: %s", dif)
			}

			if diff := cmp.Diff(testCase.expectedServiceBinding,
				testCase.sb,
				test.EquateConditions()); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestObserveReturnsError(t *testing.T) {
	t.Parallel()

	sb := serviceBinding(
		withServiceBindingParameters(defaultBindingParameters),
		afterBindingCreation(),
		initializeSBStatus(
			"6e2c036c-254f-11ee-be56-0242ac120002",
			"63d05ec8-254e-11ee-be56-0242ac120002",
			"76c0089e-254e-11ee-be56-0242ac120002",
			"test.URL.com",
			"5432",
		),
	)
	getInstancesReaction := &GetInstancesReaction{
		Response: &osbclient.GetInstancesResponse{},
		Error:    errors.New("error in observe"),
	}
	serviceInstance := *serviceInstance(
		withStatusInstanceID("6e2c036c-254f-11ee-be56-0242ac120002"),
		afterInstanceCreation(),
		withAnnotations(
			map[string]string{
				"crossplane.io/claim-name":      "postgres-1",
				"crossplane.io/claim-namespace": "test",
			}),
		withStatusServiceID("76c0089e-254e-11ee-be56-0242ac120002"),
		withStatusPlanID("63d05ec8-254e-11ee-be56-0242ac120002"),
	)
	otherResources := []client.Object{}
	expectedExternalObservation := managed.ExternalObservation{
		ResourceExists:          false,
		ResourceUpToDate:        false,
		ResourceLateInitialized: false,
	}

	expectedServiceBinding := serviceBinding(
		withServiceBindingParameters(defaultBindingParameters),
		afterBindingCreation(),
		initializeSBStatus(
			"6e2c036c-254f-11ee-be56-0242ac120002",
			"63d05ec8-254e-11ee-be56-0242ac120002",
			"76c0089e-254e-11ee-be56-0242ac120002",
			"test.URL.com",
			"5432",
		),
	)

	fakeOSB := fakeosb.NewFakeClient(fakeosb.FakeClientConfiguration{
		GetInstancesReaction: getInstancesReaction,
	})

	e := utilerr.Decorator{
		ExternalClient: &external{
			service: fakeOSB,
			kube:    newKubeMock(serviceInstance.DeepCopyObject(), otherResources),
		},
		Logger: a9stest.TestLogger(t),
	}
	got, err := e.Observe(context.TODO(), sb)

	if err == nil {
		t.Errorf("Observe method did not return expected error for ServiceBinding \n%+v",
			sb)
	}

	if !errors.Is(err, utilerr.ErrInternal) {
		t.Errorf("Expected Observe method to return %s but got error \n%s",
			utilerr.ErrInternal, err)
	}

	if dif := cmp.Diff(sb, expectedServiceBinding); dif != "" {
		t.Errorf("serviceBinding differs from expected one: %s", dif)
	}

	if dif := cmp.Diff(expectedExternalObservation, got); dif != "" {
		t.Errorf("Return from Observe differs from expected externalObservation: %s",
			dif)
	}
}

func TestConnect(t *testing.T) {
	t.Parallel()

	pc := a9stest.ProviderConfig(a9stest.Name[apisv1.ProviderConfig]("test-provider"),
		a9stest.WithProviderConfigSpec("test.com",
			a9stest.SecretRef("test-secret", "test", "username"),
			a9stest.SecretRef("test-secret", "test", "password"),
			xpv1.CredentialsSourceSecret),
	)

	secret := a9stest.Secret(a9stest.Name[corev1.Secret]("test-secret"),
		a9stest.Namespace[corev1.Secret]("test"),
		a9stest.WithKey("username", "test"),
		a9stest.WithKey("password", "secure-test-password"),
	)

	fakeK8s := fake.NewClientBuilder().WithObjects(pc, secret).WithScheme(scheme.Scheme).Build()

	var gotUsername, gotPassword []byte
	var gotURL string

	con := connector{
		kube: fakeK8s,
		usage: resource.NewProviderConfigUsageTracker(fakeK8s,
			&apisv1.ProviderConfigUsage{}),
		newServiceFn: func(username, password []byte, url string) (osbclient.Client, error) {
			gotPassword = password
			gotUsername = username
			gotURL = url

			return nil, nil
		},
	}

	sb := serviceBinding(a9stest.Uid[v1.ServiceBinding]("test-uid"), withProviderRef("test-provider"))

	_, err := con.Connect(context.TODO(), sb)
	if err != nil {
		t.Errorf("Unexpected error while executing Connect: %s", err)
	}

	if dif := cmp.Diff(gotUsername, []byte("test")); dif != "" {
		t.Errorf("Connect did not propagate username correctly to OSB client: %s", dif)
	}

	if dif := cmp.Diff(gotPassword, []byte("secure-test-password")); dif != "" {
		t.Errorf("Connect did not propagate password correctly to OSB client: %s", dif)
	}

	if dif := cmp.Diff(gotURL, "test.com"); dif != "" {
		t.Errorf("Connect did not propagate URL correctly to OSB client: %s", dif)
	}
}

func TestCreate(t *testing.T) {
	t.Parallel()

	type args struct {
		bindReaction *BindReaction
		sb           resource.Managed
	}

	type want struct {
		bindRequest      *osbclient.BindRequest
		err              error
		externalCreation managed.ExternalCreation
		sb               resource.Managed
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"sb_successfully_created_for_postgresql": {
			args: args{
				sb: serviceBinding(
					withServiceBindingParameters(defaultBindingParameters),
					initializeSBStatus(
						"6e2c036c-254f-11ee-be56-0242ac120002",
						"63d05ec8-254e-11ee-be56-0242ac120002",
						"76c0089e-254e-11ee-be56-0242ac120002",
						"",
						"",
					),
				),
				bindReaction: &BindReaction{
					Response: &osbclient.BindResponse{
						Credentials: map[string]interface{}{
							"host":     "hmm133825-psql-master-alias.node.dc1.dsf2.a9ssvc",
							"hosts":    []string{"hmm133825-pg-0.node.dc1.dsf2.a9ssvc"},
							"name":     "hmm133825",
							"password": "a9s28d0270ac3dasda8a66easdccc315e00",
							"port":     5432,
							"uri":      "postgres://a9se0e1dasd30f0510b5asd87ba44c45:a9s28d0270ac3dasda8a66easdccc315e00@hmm133825-psql-master-alias.node.dc1.dsf2.a9ssvc:5432/hmm133825",
							"username": "a9se0e1dasd30f0510b5asd87ba44c45",
						},
					},
					Error: nil,
				},
			},
			want: want{
				bindRequest: &osbclient.BindRequest{
					BindingID:         "1a6a6b3e-254e-11ee-be56-0242ac120002",
					InstanceID:        "6e2c036c-254f-11ee-be56-0242ac120002",
					AcceptsIncomplete: false,
					ServiceID:         "76c0089e-254e-11ee-be56-0242ac120002",
					PlanID:            "63d05ec8-254e-11ee-be56-0242ac120002",
				},
				externalCreation: managed.ExternalCreation{
					ConnectionDetails: managed.ConnectionDetails{
						"host":     []byte(fmt.Sprintf("%+v", "hmm133825-psql-master-alias.node.dc1.dsf2.a9ssvc")),
						"hosts":    []byte(fmt.Sprintf("%+v", []string{"hmm133825-pg-0.node.dc1.dsf2.a9ssvc"})),
						"name":     []byte(fmt.Sprintf("%+v", "hmm133825")),
						"password": []byte(fmt.Sprintf("%+v", "a9s28d0270ac3dasda8a66easdccc315e00")),
						"port":     []byte(fmt.Sprintf("%+v", 5432)),
						"uri":      []byte(fmt.Sprintf("%+v", "postgres://a9se0e1dasd30f0510b5asd87ba44c45:a9s28d0270ac3dasda8a66easdccc315e00@hmm133825-psql-master-alias.node.dc1.dsf2.a9ssvc:5432/hmm133825")),
						"username": []byte(fmt.Sprintf("%+v", "a9se0e1dasd30f0510b5asd87ba44c45")),
					},
				},
				sb: serviceBinding(
					withServiceBindingParameters(defaultBindingParameters),
					initializeSBStatus(
						"6e2c036c-254f-11ee-be56-0242ac120002",
						"63d05ec8-254e-11ee-be56-0242ac120002",
						"76c0089e-254e-11ee-be56-0242ac120002",
						"",
						"",
					),
				),
			},
		},
		"sb_successfully_created_for_search": {
			args: args{
				sb: serviceBinding(
					withServiceBindingParameters(defaultBindingParameters),
					initializeSBStatus(
						"6e2c036c-254f-11ee-be56-0242ac120002",
						"63d05ec8-254e-11ee-be56-0242ac120002",
						"76c0089e-254e-11ee-be56-0242ac120002",
						"",
						"",
					),
				),
				bindReaction: &BindReaction{
					Response: &osbclient.BindResponse{
						Credentials: map[string]interface{}{
							"cacrt": `-----BEGIN CERTIFICATE-----
			MIIDHTCCAgWgAwIBAgIUKkLgDK+arSRPon6zV53l0adF2+IwDQYJKoZIhvcNAQEL
			BQAwHjEcMBoGA1UEAxMTYTlzIFNlYXJjaCAtIFNQSSBDQTAeFw0yMzAzMDExNTEw
			MjZaFw0yNDAyMjkxNTEwMjZaMB4xHDAaBgNVBAMTE2E5cyBTZWFyY2ggLSBTUEkg
			Q0EwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCsdm9j5ga5N8InQuBO
			1VXw9rWq3S6XBsp0Io5+yCEJ0EdimNJ3EqjH/we/5n8EmrKPeRAqNOYgaFG+EaRY
			Gfb896FliKItuPEb3q93HPKcuXIl8MxNBdnFj5tpevLY3vVINKXDQ6+Qv1JMa5ME
			8rmbqasMWL5P0fkqLbiVhHPeIzN5n4fkFXAtDTLpf7l7LuuaMLI3HbU9hkGfZBLA
			5V1Pu56+ZRN58s2H7Jf/YT48l8yBNEkym/0eNBgKqXGi1esynDGTDl6LjbddhEcV
			ykgdpGpOjSEihQTQ9hApMR/nKww3VKTG9dJO9ZJAgdYZ0xsP9uzvyWzsQrypE+l9
			GOd9AgMBAAGjUzBRMB0GA1UdDgQWBBTIUmpNXuexGAyjMK3L89Vu3HDsoDAfBgNV
			HSMEGDAWgBTIUmpNXuexGAyjMK3L89Vu3HDsoDAPBgNVHRMBAf8EBTADAQH/MA0G
			CSqGSIb3DQEBCwUAA4IBAQADpJkIyhkifUVFDh3t0M2digafrfaDzcwIMj8PCRot
			r9WmqiM9uGs1l2dJL4hKaATLh5T3xi2c3tColDm8o717vAkFiak3WsrQC/JrH/4B
			z099Pm4H9ejlkYSJ4xYT3BQM+qBem1YxE7+SbBcUJABLu4QwonAVpDboLmzrou42
			aCTyJ4ZuI7Os2YSSnW3cRX7fNmyPLXfFvj8RlnBC4n1MxnjIsswkl/vPvzbxggMg
			fRNYVbID6FU2Gv29xULWTQBA3zSoOojVDGkFZDMKG1kaCiPWmQcNgADX7KN9mhcq
			I3TyN3NIdf/7whFnJeX23L6lHLlrypmbENs/FxD4YZKL
			-----END CERTIFICATE-----`,
							"host":     []string{"https://mdod72216c.service.dc1.dsf2.a9ssvc:9200"},
							"hosts":    []string{"mdod72216c-os-0.node.dc1.dsf2.a9ssvc"},
							"password": "a9sd21f426c693f8d751cef80c76c2c23980c438cca",
							"port":     9200,
							"scheme":   "https",
							"username": "a9sc56b7fde44da65146f3b14f862bc11be6311801f",
						},
					},
					Error: nil,
				},
			},
			want: want{
				bindRequest: &osbclient.BindRequest{
					BindingID:         "1a6a6b3e-254e-11ee-be56-0242ac120002",
					InstanceID:        "6e2c036c-254f-11ee-be56-0242ac120002",
					AcceptsIncomplete: false,
					ServiceID:         "76c0089e-254e-11ee-be56-0242ac120002",
					PlanID:            "63d05ec8-254e-11ee-be56-0242ac120002",
				},
				externalCreation: managed.ExternalCreation{
					ConnectionDetails: managed.ConnectionDetails{
						"cacrt": []byte(fmt.Sprintf("%+v", `-----BEGIN CERTIFICATE-----
			MIIDHTCCAgWgAwIBAgIUKkLgDK+arSRPon6zV53l0adF2+IwDQYJKoZIhvcNAQEL
			BQAwHjEcMBoGA1UEAxMTYTlzIFNlYXJjaCAtIFNQSSBDQTAeFw0yMzAzMDExNTEw
			MjZaFw0yNDAyMjkxNTEwMjZaMB4xHDAaBgNVBAMTE2E5cyBTZWFyY2ggLSBTUEkg
			Q0EwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCsdm9j5ga5N8InQuBO
			1VXw9rWq3S6XBsp0Io5+yCEJ0EdimNJ3EqjH/we/5n8EmrKPeRAqNOYgaFG+EaRY
			Gfb896FliKItuPEb3q93HPKcuXIl8MxNBdnFj5tpevLY3vVINKXDQ6+Qv1JMa5ME
			8rmbqasMWL5P0fkqLbiVhHPeIzN5n4fkFXAtDTLpf7l7LuuaMLI3HbU9hkGfZBLA
			5V1Pu56+ZRN58s2H7Jf/YT48l8yBNEkym/0eNBgKqXGi1esynDGTDl6LjbddhEcV
			ykgdpGpOjSEihQTQ9hApMR/nKww3VKTG9dJO9ZJAgdYZ0xsP9uzvyWzsQrypE+l9
			GOd9AgMBAAGjUzBRMB0GA1UdDgQWBBTIUmpNXuexGAyjMK3L89Vu3HDsoDAfBgNV
			HSMEGDAWgBTIUmpNXuexGAyjMK3L89Vu3HDsoDAPBgNVHRMBAf8EBTADAQH/MA0G
			CSqGSIb3DQEBCwUAA4IBAQADpJkIyhkifUVFDh3t0M2digafrfaDzcwIMj8PCRot
			r9WmqiM9uGs1l2dJL4hKaATLh5T3xi2c3tColDm8o717vAkFiak3WsrQC/JrH/4B
			z099Pm4H9ejlkYSJ4xYT3BQM+qBem1YxE7+SbBcUJABLu4QwonAVpDboLmzrou42
			aCTyJ4ZuI7Os2YSSnW3cRX7fNmyPLXfFvj8RlnBC4n1MxnjIsswkl/vPvzbxggMg
			fRNYVbID6FU2Gv29xULWTQBA3zSoOojVDGkFZDMKG1kaCiPWmQcNgADX7KN9mhcq
			I3TyN3NIdf/7whFnJeX23L6lHLlrypmbENs/FxD4YZKL
			-----END CERTIFICATE-----`)),
						"host":     []byte(fmt.Sprintf("%+v", []string{"https://mdod72216c.service.dc1.dsf2.a9ssvc:9200"})),
						"hosts":    []byte(fmt.Sprintf("%+v", []string{"mdod72216c-os-0.node.dc1.dsf2.a9ssvc"})),
						"password": []byte(fmt.Sprintf("%+v", "a9sd21f426c693f8d751cef80c76c2c23980c438cca")),
						"port":     []byte(fmt.Sprintf("%+v", 9200)),
						"scheme":   []byte(fmt.Sprintf("%+v", "https")),
						"username": []byte(fmt.Sprintf("%+v", "a9sc56b7fde44da65146f3b14f862bc11be6311801f")),
					},
				},
				sb: serviceBinding(
					withServiceBindingParameters(defaultBindingParameters),
					initializeSBStatus(
						"6e2c036c-254f-11ee-be56-0242ac120002",
						"63d05ec8-254e-11ee-be56-0242ac120002",
						"76c0089e-254e-11ee-be56-0242ac120002",
						"",
						"",
					),
				),
			},
		},
		"sb_successfully_created_for_mongodb": {
			args: args{
				sb: serviceBinding(
					withServiceBindingParameters(defaultBindingParameters),
					initializeSBStatus(
						"6e2c036c-254f-11ee-be56-0242ac120002",
						"63d05ec8-254e-11ee-be56-0242ac120002",
						"76c0089e-254e-11ee-be56-0242ac120002",
						"",
						"",
					),
				),
				bindReaction: &BindReaction{
					Response: &osbclient.BindResponse{
						Credentials: map[string]interface{}{
							"default_database": "zzd661341",
							"hosts":            []string{"mdod661341-mongodb-0.node.dc1.dsf2.a9ssvc:27017"},
							"password":         "a9s253a1dcacaa93027bccc939260638e12e48eb629",
							"uri":              "mongodb://a9s-brk-usr-ce2536939a9aa77140c3e196186f4aad559ad112:a9s253a1dcacaa93027bccc939260638e12e48eb629@zzd661341-mongodb-0.node.dc1.dsf2.a9ssvc:27017/mdod661341",
							"username":         "a9s-brk-usr-ce2536939a9aa77140c3e196186f4aad559ad112",
						},
					},
					Error: nil,
				},
			},
			want: want{
				bindRequest: &osbclient.BindRequest{
					BindingID:         "1a6a6b3e-254e-11ee-be56-0242ac120002",
					InstanceID:        "6e2c036c-254f-11ee-be56-0242ac120002",
					AcceptsIncomplete: false,
					ServiceID:         "76c0089e-254e-11ee-be56-0242ac120002",
					PlanID:            "63d05ec8-254e-11ee-be56-0242ac120002",
				},
				externalCreation: managed.ExternalCreation{
					ConnectionDetails: managed.ConnectionDetails{
						"default_database": []byte(fmt.Sprintf("%+v", "zzd661341")),
						"hosts":            []byte(fmt.Sprintf("%+v", []string{"mdod661341-mongodb-0.node.dc1.dsf2.a9ssvc:27017"})),
						"password":         []byte(fmt.Sprintf("%+v", "a9s253a1dcacaa93027bccc939260638e12e48eb629")),
						"uri":              []byte(fmt.Sprintf("%+v", "mongodb://a9s-brk-usr-ce2536939a9aa77140c3e196186f4aad559ad112:a9s253a1dcacaa93027bccc939260638e12e48eb629@zzd661341-mongodb-0.node.dc1.dsf2.a9ssvc:27017/mdod661341")),
						"username":         []byte(fmt.Sprintf("%+v", "a9s-brk-usr-ce2536939a9aa77140c3e196186f4aad559ad112")),
					},
				},
				sb: serviceBinding(
					withServiceBindingParameters(defaultBindingParameters),
					initializeSBStatus(
						"6e2c036c-254f-11ee-be56-0242ac120002",
						"63d05ec8-254e-11ee-be56-0242ac120002",
						"76c0089e-254e-11ee-be56-0242ac120002",
						"",
						"",
					),
				),
			},
		},
		"sb_successfully_created_for_mariadb": {
			args: args{
				sb: serviceBinding(
					withServiceBindingParameters(defaultBindingParameters),
					initializeSBStatus(
						"6e2c036c-254f-11ee-be56-0242ac120002",
						"63d05ec8-254e-11ee-be56-0242ac120002",
						"76c0089e-254e-11ee-be56-0242ac120002",
						"",
						"",
					),
				),
				bindReaction: &BindReaction{
					Response: &osbclient.BindResponse{
						Credentials: map[string]interface{}{
							"host":     "d17775b.service.dc1.a9s-mariadb-consul",
							"name":     "d17775b",
							"password": "a9sd21f426c693f8d771cef80c76c2c23980c438cca",
							"port":     3306,
							"uri":      "mysql://a9s-brk-usr-a9s011b86e584ce3664fc86dc94c5f0:a9sd21f426c693f8d771cef80c76c2c23980c438cca@d17775b.service.dc1.a9s-mariadb-consul:3306/d17775b",
							"username": "a9s-brk-usr-a9s011b86e584ce3664fc86dc94c5f0",
						},
					},
					Error: nil,
				},
			},
			want: want{
				bindRequest: &osbclient.BindRequest{
					BindingID:         "1a6a6b3e-254e-11ee-be56-0242ac120002",
					InstanceID:        "6e2c036c-254f-11ee-be56-0242ac120002",
					AcceptsIncomplete: false,
					ServiceID:         "76c0089e-254e-11ee-be56-0242ac120002",
					PlanID:            "63d05ec8-254e-11ee-be56-0242ac120002",
				},
				externalCreation: managed.ExternalCreation{
					ConnectionDetails: managed.ConnectionDetails{
						"host":     []byte(fmt.Sprintf("%+v", "d17775b.service.dc1.a9s-mariadb-consul")),
						"name":     []byte(fmt.Sprintf("%+v", "d17775b")),
						"password": []byte(fmt.Sprintf("%+v", "a9sd21f426c693f8d771cef80c76c2c23980c438cca")),
						"port":     []byte(fmt.Sprintf("%+v", 3306)),
						"uri":      []byte(fmt.Sprintf("%+v", "mysql://a9s-brk-usr-a9s011b86e584ce3664fc86dc94c5f0:a9sd21f426c693f8d771cef80c76c2c23980c438cca@d17775b.service.dc1.a9s-mariadb-consul:3306/d17775b")),
						"username": []byte(fmt.Sprintf("%+v", "a9s-brk-usr-a9s011b86e584ce3664fc86dc94c5f0")),
					},
				},
				sb: serviceBinding(
					withServiceBindingParameters(defaultBindingParameters),
					initializeSBStatus(
						"6e2c036c-254f-11ee-be56-0242ac120002",
						"63d05ec8-254e-11ee-be56-0242ac120002",
						"76c0089e-254e-11ee-be56-0242ac120002",
						"",
						"",
					),
				),
			},
		},
		"sb_successfully_created_for_messaging": {
			args: args{
				sb: serviceBinding(
					withServiceBindingParameters(defaultBindingParameters),
					initializeSBStatus(
						"6e2c036c-254f-11ee-be56-0242ac120002",
						"63d05ec8-254e-11ee-be56-0242ac120002",
						"76c0089e-254e-11ee-be56-0242ac120002",
						"",
						"",
					),
				),
				bindReaction: &BindReaction{
					Response: &osbclient.BindResponse{
						Credentials: map[string]interface{}{
							"host":         "d67701c.service.dc1.a9svs",
							"hosts":        []string{"d67701c.service.dc1.a9svs"},
							"password":     "b4c2d8e1f5a0a9d6c9a7e5b8c5a7b0a6b2d4c0",
							"port":         5672,
							"http_api_uri": "http://a9s-brk-usr-1a5b9f4e6d7c3a8e0f2b3d4e5a6d5e:b4c2d8e1f5a0a9d6c9a7e5b8c5a7b0a6b2d4c0@d67701c.service.dc1.a9svs/api/",
							"http_api_uris": []string{
								"http://a9s-brk-usr-1a5b9f4e6d7c3a8e0f2b3d4e5a6d5e:b4c2d8e1f5a0a9d6c9a7e5b8c5a7b0a6b2d4c0@d67701c.service.dc1.a9svs/api/",
							},
							"protocols": map[string]interface{}{
								"amqp": map[string]interface{}{
									"host":     "d67701c.service.dc1.a9svs",
									"hosts":    []string{"d67701c.service.dc1.a9svs"},
									"password": "b4c2d8e1f5a0a9d6c9a7e5b8c5a7b0a6b2d4c0",
									"port":     5672,
									"ssl":      false,
									"uri":      "amqp://a9s-brk-usr-1a5b9f4e6d7c3a8e0f2b3d4e5a6d5e:b4c2d8e1f5a0a9d6c9a7e5b8c5a7b0a6b2d4c0@d67701c.service.dc1.a9svs:5672",
									"username": "a9s-brk-usr-1a5b9f4e6d7c3a8e0f2b3d4e5a6d5e",
								},
								"management": map[string]interface{}{
									"username": "a9s-brk-usr-1a5b9f4e6d7c3a8e0f2b3d4e5a6d5e",
									"password": "b4c2d8e1f5a0a9d6c9a7e5b8c5a7b0a6b2d4c0",
									"path":     "/api",
									"ssl":      false,
									"host":     "d67701c.service.dc1.a9svs",
									"hosts":    []string{"d67701c.service.dc1.a9svs"},
									"uri":      "http://a9s-brk-usr-1a5b9f4e6d7c3a8e0f2b3d4e5a6d5e:b4c2d8e1f5a0a9d6c9a7e5b8c5a7b0a6b2d4c0@d67701c.service.dc1.a9svs",
									"uris":     []string{"http://a9s-brk-usr-1a5b9f4e6d7c3a8e0f2b3d4e5a6d5e:b4c2d8e1f5a0a9d6c9a7e5b8c5a7b0a6b2d4c0@d67701c.service.dc1.a9svs"},
								},
							},
							"ssl":      false,
							"uri":      "a9s-brk-usr-1a5b9f4e6d7c3a8e0f2b3d4e5a6d5e:b4c2d8e1f5a0a9d6c9a7e5b8c5a7b0a6b2d4c0@d67701c.service.dc1.a9svs:5672",
							"username": "a9s-brk-usr-1a5b9f4e6d7c3a8e0f2b3d4e5a6d5e",
						},
					},
					Error: nil,
				},
			},
			want: want{
				bindRequest: &osbclient.BindRequest{
					BindingID:         "1a6a6b3e-254e-11ee-be56-0242ac120002",
					InstanceID:        "6e2c036c-254f-11ee-be56-0242ac120002",
					AcceptsIncomplete: false,
					ServiceID:         "76c0089e-254e-11ee-be56-0242ac120002",
					PlanID:            "63d05ec8-254e-11ee-be56-0242ac120002",
				},
				externalCreation: managed.ExternalCreation{
					ConnectionDetails: managed.ConnectionDetails{
						"http_api_uri":                  []byte(fmt.Sprintf("%+v", "http://a9s-brk-usr-1a5b9f4e6d7c3a8e0f2b3d4e5a6d5e:b4c2d8e1f5a0a9d6c9a7e5b8c5a7b0a6b2d4c0@d67701c.service.dc1.a9svs/api/")),
						"http_api_uris":                 []byte(fmt.Sprintf("%+v", []string{"http://a9s-brk-usr-1a5b9f4e6d7c3a8e0f2b3d4e5a6d5e:b4c2d8e1f5a0a9d6c9a7e5b8c5a7b0a6b2d4c0@d67701c.service.dc1.a9svs/api/"})),
						"protocols.amqp.host":           []byte(fmt.Sprintf("%+v", "d67701c.service.dc1.a9svs")),
						"protocols.amqp.hosts":          []byte(fmt.Sprintf("%+v", []string{"d67701c.service.dc1.a9svs"})),
						"protocols.amqp.password":       []byte(fmt.Sprintf("%+v", "b4c2d8e1f5a0a9d6c9a7e5b8c5a7b0a6b2d4c0")),
						"protocols.amqp.port":           []byte(fmt.Sprintf("%+v", 5672)),
						"protocols.amqp.ssl":            []byte(fmt.Sprintf("%+v", false)),
						"protocols.amqp.uri":            []byte(fmt.Sprintf("%+v", "amqp://a9s-brk-usr-1a5b9f4e6d7c3a8e0f2b3d4e5a6d5e:b4c2d8e1f5a0a9d6c9a7e5b8c5a7b0a6b2d4c0@d67701c.service.dc1.a9svs:5672")),
						"protocols.amqp.username":       []byte(fmt.Sprintf("%+v", "a9s-brk-usr-1a5b9f4e6d7c3a8e0f2b3d4e5a6d5e")),
						"protocols.management.host":     []byte(fmt.Sprintf("%+v", "d67701c.service.dc1.a9svs")),
						"protocols.management.hosts":    []byte(fmt.Sprintf("%+v", []string{"d67701c.service.dc1.a9svs"})),
						"protocols.management.password": []byte(fmt.Sprintf("%+v", "b4c2d8e1f5a0a9d6c9a7e5b8c5a7b0a6b2d4c0")),
						"protocols.management.path":     []byte(fmt.Sprintf("%+v", "/api")),
						"protocols.management.ssl":      []byte(fmt.Sprintf("%+v", "false")),
						"protocols.management.uri":      []byte(fmt.Sprintf("%+v", "http://a9s-brk-usr-1a5b9f4e6d7c3a8e0f2b3d4e5a6d5e:b4c2d8e1f5a0a9d6c9a7e5b8c5a7b0a6b2d4c0@d67701c.service.dc1.a9svs")),
						"protocols.management.uris":     []byte(fmt.Sprintf("%+v", []string{"http://a9s-brk-usr-1a5b9f4e6d7c3a8e0f2b3d4e5a6d5e:b4c2d8e1f5a0a9d6c9a7e5b8c5a7b0a6b2d4c0@d67701c.service.dc1.a9svs"})),
						"protocols.management.username": []byte(fmt.Sprintf("%+v", "a9s-brk-usr-1a5b9f4e6d7c3a8e0f2b3d4e5a6d5e")),
						"uri":                           []byte(fmt.Sprintf("%+v", "a9s-brk-usr-1a5b9f4e6d7c3a8e0f2b3d4e5a6d5e:b4c2d8e1f5a0a9d6c9a7e5b8c5a7b0a6b2d4c0@d67701c.service.dc1.a9svs:5672")),
					},
				},
				sb: serviceBinding(
					withServiceBindingParameters(defaultBindingParameters),
					initializeSBStatus(
						"6e2c036c-254f-11ee-be56-0242ac120002",
						"63d05ec8-254e-11ee-be56-0242ac120002",
						"76c0089e-254e-11ee-be56-0242ac120002",
						"",
						"",
					),
				),
			},
		},
		"sb_successfully_created_for_logme2": {
			args: args{
				sb: serviceBinding(
					withServiceBindingParameters(defaultBindingParameters),
					initializeSBStatus(
						"6e2c036c-254f-11ee-be56-0242ac120002",
						"63d05ec8-254e-11ee-be56-0242ac120002",
						"76c0089e-254e-11ee-be56-0242ac120002",
						"",
						"",
					),
				),
				bindReaction: &BindReaction{
					Response: &osbclient.BindResponse{
						Credentials: map[string]interface{}{
							"cacrt": `-----BEGIN CERTIFICATE-----
			MIIDHTCCAgWgAwIBAgIUawyLe2ioKWcZVHq6+IEQQeTmfp8wDQYJKoZIhvcNAQEL
			BQAwHjEcMBoGA1UEAxMTYTlzIExvZ01lMiAtIFNQSSBDQTAeFw0yMzAyMjcxMTU3
			NDRaFw0yNDAyMjcxMTU3NDRaMB4xHDAaBgNVBAMTE2E5cyBMb2dNZTIgLSBTUEkg
			Q0EwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC+POlrVAm65aVCWfk3
			S+CJfEUbjMpPh9tWGL8NtmVpmf/E1zGjXIVBH7nksx7oQDxbwoqMcr/fMVGUhwrv
			v16TytKz/v4us5yEI1Sh8niIeexZlukuF7VundxjF4W3U9s6WqzK/0JErEM6duhy
			rEY1Z9T1eob8LViXuc4GuLGduIRp28FLNdMBNAm3kJ4WqxL/ja8JIMCqswYXVE9I
			0AqR6CSoxPHsOxtGFAf78RwjrjCjI/DoYLkAVqhd+fqkH6/BJIvi2HWSBqpxP5Td
			W4S2wipowuXKM4RJ+Ex6Va67BlHRdqjeu3OXAMuxZFASWHkFacMcEQykRDhuYsGa
			fpp5AgMBAAGjUzBRMB0GA1UdDgQWBBQXS8UPTqUbI4vkJMFhk1quC9JUxDAfBgNV
			HSMEGDAWgBQXS8UPTqUbI4vkJMFhk1quC9JUxDAPBgNVHRMBAf8EBTADAQH/MA0G
			CSqGSIb3DQEBCwUAA4IBAQBq2Pi4zIl5rRpXinhaH2F/RwfbbXddrASd/8Hr512j
			FFvOn44X28PJ7onG9m2RBzHm7WF6pHdrSL3bAbrXg7b9pbggOktPf+cEnJtQ6SOj
			komSQDfLvVONnlnWEwxXGePOkZ3XXUp0Yx7tK7nuqZAgyD66+0E1TqVBhqu/BPFu
			EhrYd9MP6uXynzyHg7RDToac/RW8uB6eYHe/6WPhWNXRPJIi+QsizHpPByF0xagz
			aFxOp5nDVg+MBKfqsHuO7uPACefNh5VbGKBahtVcHtCkkaxd0uJ14/wZogUbVZh7
			K5VZYh0ZdLd68Ftx93Ub6k6IpzJ6ruZXedfTBJSCYDgs
			-----END CERTIFICATE-----`,
							"host":             "https://mdod4340b3-os.service.dc1.dsf2.a9ssvc:9200",
							"password":         "a9sb71101e4dcb38d4d1b9d127ca061419a3486d59b",
							"syslog_drain_url": "syslog-tls://mdod4340b3-fluentd.service.dc1.dsf2.a9ssvc:6514",
							"username":         "a9s8fb94579c777baded1f6b5877c162a54f9461a57",
						},
					},
					Error: nil,
				},
			},
			want: want{
				bindRequest: &osbclient.BindRequest{
					BindingID:         "1a6a6b3e-254e-11ee-be56-0242ac120002",
					InstanceID:        "6e2c036c-254f-11ee-be56-0242ac120002",
					AcceptsIncomplete: false,
					ServiceID:         "76c0089e-254e-11ee-be56-0242ac120002",
					PlanID:            "63d05ec8-254e-11ee-be56-0242ac120002",
				},
				externalCreation: managed.ExternalCreation{
					ConnectionDetails: managed.ConnectionDetails{
						"cacrt": []byte(fmt.Sprintf("%+v", `-----BEGIN CERTIFICATE-----
			MIIDHTCCAgWgAwIBAgIUawyLe2ioKWcZVHq6+IEQQeTmfp8wDQYJKoZIhvcNAQEL
			BQAwHjEcMBoGA1UEAxMTYTlzIExvZ01lMiAtIFNQSSBDQTAeFw0yMzAyMjcxMTU3
			NDRaFw0yNDAyMjcxMTU3NDRaMB4xHDAaBgNVBAMTE2E5cyBMb2dNZTIgLSBTUEkg
			Q0EwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC+POlrVAm65aVCWfk3
			S+CJfEUbjMpPh9tWGL8NtmVpmf/E1zGjXIVBH7nksx7oQDxbwoqMcr/fMVGUhwrv
			v16TytKz/v4us5yEI1Sh8niIeexZlukuF7VundxjF4W3U9s6WqzK/0JErEM6duhy
			rEY1Z9T1eob8LViXuc4GuLGduIRp28FLNdMBNAm3kJ4WqxL/ja8JIMCqswYXVE9I
			0AqR6CSoxPHsOxtGFAf78RwjrjCjI/DoYLkAVqhd+fqkH6/BJIvi2HWSBqpxP5Td
			W4S2wipowuXKM4RJ+Ex6Va67BlHRdqjeu3OXAMuxZFASWHkFacMcEQykRDhuYsGa
			fpp5AgMBAAGjUzBRMB0GA1UdDgQWBBQXS8UPTqUbI4vkJMFhk1quC9JUxDAfBgNV
			HSMEGDAWgBQXS8UPTqUbI4vkJMFhk1quC9JUxDAPBgNVHRMBAf8EBTADAQH/MA0G
			CSqGSIb3DQEBCwUAA4IBAQBq2Pi4zIl5rRpXinhaH2F/RwfbbXddrASd/8Hr512j
			FFvOn44X28PJ7onG9m2RBzHm7WF6pHdrSL3bAbrXg7b9pbggOktPf+cEnJtQ6SOj
			komSQDfLvVONnlnWEwxXGePOkZ3XXUp0Yx7tK7nuqZAgyD66+0E1TqVBhqu/BPFu
			EhrYd9MP6uXynzyHg7RDToac/RW8uB6eYHe/6WPhWNXRPJIi+QsizHpPByF0xagz
			aFxOp5nDVg+MBKfqsHuO7uPACefNh5VbGKBahtVcHtCkkaxd0uJ14/wZogUbVZh7
			K5VZYh0ZdLd68Ftx93Ub6k6IpzJ6ruZXedfTBJSCYDgs
			-----END CERTIFICATE-----`)),
						"host":             []byte(fmt.Sprintf("%+v", "https://mdod4340b3-os.service.dc1.dsf2.a9ssvc:9200")),
						"password":         []byte(fmt.Sprintf("%+v", "a9sb71101e4dcb38d4d1b9d127ca061419a3486d59b")),
						"syslog_drain_url": []byte(fmt.Sprintf("%+v", "syslog-tls://mdod4340b3-fluentd.service.dc1.dsf2.a9ssvc:6514")),
						"username":         []byte(fmt.Sprintf("%+v", "a9s8fb94579c777baded1f6b5877c162a54f9461a57")),
					},
				},
				sb: serviceBinding(
					withServiceBindingParameters(defaultBindingParameters),
					initializeSBStatus(
						"6e2c036c-254f-11ee-be56-0242ac120002",
						"63d05ec8-254e-11ee-be56-0242ac120002",
						"76c0089e-254e-11ee-be56-0242ac120002",
						"",
						"",
					),
				),
			},
		},
		"sb_successfully_created_for_prometheus": {
			args: args{
				sb: serviceBinding(
					withServiceBindingParameters(defaultBindingParameters),
					initializeSBStatus(
						"6e2c036c-254f-11ee-be56-0242ac120002",
						"63d05ec8-254e-11ee-be56-0242ac120002",
						"76c0089e-254e-11ee-be56-0242ac120002",
						"",
						"",
					),
				),
				bindReaction: &BindReaction{
					Response: &osbclient.BindResponse{
						Credentials: map[string]interface{}{
							"alertmanager_urls":      "http://imd45dd56-alertmanager-0.node.dc1.dsf2.a9ssvc:9093",
							"grafana_urls":           "http://imd45dd56-grafana-0.node.dc1.dsf2.a9ssvc:3000",
							"graphite_exporter_port": "9109",
							"graphite_exporters":     "imd45dd56-prometheus-0.node.dc1.dsf2.a9ssvc",
							"password":               "a9se8abeaf0609999fd8dd40b9d4ac8335466afd22d",
							"prometheus_urls":        "http://imd45dd56-prometheus-0.node.dc1.dsf2.a9ssvc:9090/",
							"username":               "a9s-brk-usr-76b0c4a7ebb051402a8437783b76d9449d638054",
						},
					},
					Error: nil,
				},
			},
			want: want{
				bindRequest: &osbclient.BindRequest{
					BindingID:         "1a6a6b3e-254e-11ee-be56-0242ac120002",
					InstanceID:        "6e2c036c-254f-11ee-be56-0242ac120002",
					AcceptsIncomplete: false,
					ServiceID:         "76c0089e-254e-11ee-be56-0242ac120002",
					PlanID:            "63d05ec8-254e-11ee-be56-0242ac120002",
				},
				externalCreation: managed.ExternalCreation{
					ConnectionDetails: managed.ConnectionDetails{
						"alertmanager_urls":      []byte(fmt.Sprintf("%+v", "http://imd45dd56-alertmanager-0.node.dc1.dsf2.a9ssvc:9093")),
						"grafana_urls":           []byte(fmt.Sprintf("%+v", "http://imd45dd56-grafana-0.node.dc1.dsf2.a9ssvc:3000")),
						"graphite_exporter_port": []byte(fmt.Sprintf("%+v", "9109")),
						"graphite_exporters":     []byte(fmt.Sprintf("%+v", "imd45dd56-prometheus-0.node.dc1.dsf2.a9ssvc")),
						"password":               []byte(fmt.Sprintf("%+v", "a9se8abeaf0609999fd8dd40b9d4ac8335466afd22d")),
						"prometheus_urls":        []byte(fmt.Sprintf("%+v", "http://imd45dd56-prometheus-0.node.dc1.dsf2.a9ssvc:9090/")),
						"username":               []byte(fmt.Sprintf("%+v", "a9s-brk-usr-76b0c4a7ebb051402a8437783b76d9449d638054")),
					},
				},
				sb: serviceBinding(
					withServiceBindingParameters(defaultBindingParameters),
					initializeSBStatus(
						"6e2c036c-254f-11ee-be56-0242ac120002",
						"63d05ec8-254e-11ee-be56-0242ac120002",
						"76c0089e-254e-11ee-be56-0242ac120002",
						"",
						"",
					),
				),
			},
		},
		"fails_on_broker": {
			args: args{
				sb: serviceBinding(
					withServiceBindingParameters(defaultBindingParameters),
					initializeSBStatus(
						"postgres-1-id",
						"63d05ec8-254e-11ee-be56-0242ac120002",
						"76c0089e-254e-11ee-be56-0242ac120002",
						"",
						"",
					),
				),
				bindReaction: &BindReaction{
					Response: &osbclient.BindResponse{},
					Error:    errors.New("Error in client"),
				},
			},
			want: want{
				bindRequest: &osbclient.BindRequest{
					BindingID:         "1a6a6b3e-254e-11ee-be56-0242ac120002",
					InstanceID:        "postgres-1-id",
					AcceptsIncomplete: false,
					ServiceID:         "76c0089e-254e-11ee-be56-0242ac120002",
					PlanID:            "63d05ec8-254e-11ee-be56-0242ac120002",
				},
				err: utilerr.ErrInternal,
			},
		},
		"fails_not_a_service_binding": {
			args: args{
				sb: &dsv1.ServiceInstance{},
			},
			want: want{
				err: utilerr.ErrInternal,
				sb:  &dsv1.ServiceInstance{},
			},
		},
	}

	for name, testCase := range cases {
		// Rebind testCase into this lexical scope. Details on the why at
		// https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fakeOSB := fakeosb.NewFakeClient(fakeosb.FakeClientConfiguration{
				BindReaction: testCase.args.bindReaction,
			})

			external := utilerr.Decorator{
				Logger: a9stest.TestLogger(t),
				ExternalClient: &external{
					service: fakeOSB,
				},
			}
			got, err := external.Create(context.TODO(), testCase.args.sb)

			if diff := cmp.Diff(testCase.want.externalCreation, got); diff != "" {
				t.Errorf("Create(...): -want, +got:\n%s", diff)
			}

			if diff := cmp.Diff(testCase.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("Create(...): -want error, +got error:\n%s", diff)
			}

			if testCase.args.bindReaction != nil {
				bindReq := testCase.args.bindReaction.getLastBindRequest()
				if diff := cmp.Diff(testCase.want.bindRequest, bindReq); diff != "" {
					t.Errorf("Create(...): -want Bind Request, +got Bind Request:\n%s", diff)
				}
			}
		})
	}
}

func TestDeleteHappyPath(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		serviceBinding    *v1.ServiceBinding
		unbindResponse    *osbclient.UnbindResponse
		expectedUnbindReq *osbclient.UnbindRequest
	}{
		"empty_response": {
			serviceBinding: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"",
					"",
				),
			),
			unbindResponse: &osbclient.UnbindResponse{},
			expectedUnbindReq: &osbclient.UnbindRequest{
				BindingID:         "1a6a6b3e-254e-11ee-be56-0242ac120002",
				InstanceID:        "6e2c036c-254f-11ee-be56-0242ac120002",
				AcceptsIncomplete: false,
				ServiceID:         "76c0089e-254e-11ee-be56-0242ac120002",
				PlanID:            "63d05ec8-254e-11ee-be56-0242ac120002",
			},
		},
		"async_true_response": {
			serviceBinding: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"",
					"",
				),
			),
			unbindResponse: &osbclient.UnbindResponse{
				Async: true,
			},
			expectedUnbindReq: &osbclient.UnbindRequest{
				BindingID:         "1a6a6b3e-254e-11ee-be56-0242ac120002",
				InstanceID:        "6e2c036c-254f-11ee-be56-0242ac120002",
				AcceptsIncomplete: false,
				ServiceID:         "76c0089e-254e-11ee-be56-0242ac120002",
				PlanID:            "63d05ec8-254e-11ee-be56-0242ac120002",
			},
		},
		"async_false_response": {
			serviceBinding: serviceBinding(
				withServiceBindingParameters(defaultBindingParameters),
				afterBindingCreation(),
				initializeSBStatus(
					"6e2c036c-254f-11ee-be56-0242ac120002",
					"63d05ec8-254e-11ee-be56-0242ac120002",
					"76c0089e-254e-11ee-be56-0242ac120002",
					"",
					"",
				),
			),
			unbindResponse: &osbclient.UnbindResponse{
				Async: false,
			},
			expectedUnbindReq: &osbclient.UnbindRequest{
				BindingID:         "1a6a6b3e-254e-11ee-be56-0242ac120002",
				InstanceID:        "6e2c036c-254f-11ee-be56-0242ac120002",
				AcceptsIncomplete: false,
				ServiceID:         "76c0089e-254e-11ee-be56-0242ac120002",
				PlanID:            "63d05ec8-254e-11ee-be56-0242ac120002",
			},
		},
	}

	for name, testCase := range testCases {

		// Rebind testCase into this lexical scope. Details on the why at
		// https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		testCase := testCase

		t.Run(name, func(t *testing.T) {
			t.Parallel()
			unbindReaction := &UnbindReaction{
				Response: testCase.unbindResponse,
				Error:    nil,
			}

			fakeOSB := fakeosb.NewFakeClient(fakeosb.FakeClientConfiguration{
				UnbindReaction: unbindReaction,
			})
			external := utilerr.Decorator{
				ExternalClient: &external{
					service: fakeOSB,
				},
				Logger: a9stest.TestLogger(t),
			}

			_, err := external.Delete(context.TODO(), testCase.serviceBinding)
			if err != nil {
				t.Errorf("Unexpected error occurred when trying to "+
					"delete ServiceBinding %+v : %s",
					testCase.serviceBinding, err)
			}

			unbindReq := unbindReaction.getLastUnbindRequest()
			if unbindReq == nil {
				t.Errorf("OSB client was never called to delete ServiceBinding %+v.",
					testCase.serviceBinding)
			}

			if dif := cmp.Diff(testCase.expectedUnbindReq, unbindReq); dif != "" {
				t.Errorf("UnbindRequest given to OSB client differs from expected one: -want,+got %s", dif)
			}

			containsDeleteCond := false
			for _, c := range testCase.serviceBinding.Status.Conditions {
				if c.Reason == xpv1.ReasonDeleting {
					containsDeleteCond = true
				}
			}

			if !containsDeleteCond {
				t.Errorf("Condition \"Deleting\" was not set")
			}
		})
	}
}

func TestDeleteClientErr(t *testing.T) {
	t.Parallel()

	serviceBinding := serviceBinding(
		withServiceBindingParameters(defaultBindingParameters),
		afterBindingCreation(),
		initializeSBStatus(
			"6e2c036c-254f-11ee-be56-0242ac120002",
			"63d05ec8-254e-11ee-be56-0242ac120002",
			"76c0089e-254e-11ee-be56-0242ac120002",
			"test.URL.com",
			"5432",
		),
	)

	unbindResponse := &osbclient.UnbindResponse{}

	expectedUnbindReq := &osbclient.UnbindRequest{
		BindingID:         "1a6a6b3e-254e-11ee-be56-0242ac120002",
		InstanceID:        "6e2c036c-254f-11ee-be56-0242ac120002",
		AcceptsIncomplete: false,
		ServiceID:         "76c0089e-254e-11ee-be56-0242ac120002",
		PlanID:            "63d05ec8-254e-11ee-be56-0242ac120002",
	}
	expectedErr := utilerr.PlainUserErr("failed to delete ServiceBinding")

	unbindReaction := &UnbindReaction{
		Response: unbindResponse,
		Error:    errors.New("Error in client"),
	}

	fakeOSB := fakeosb.NewFakeClient(fakeosb.FakeClientConfiguration{
		UnbindReaction: unbindReaction,
	})
	external := utilerr.Decorator{
		ExternalClient: &external{
			kube: newKubeMock(
				serviceInstance(
					afterInstanceCreation(),
					withAnnotations(map[string]string{
						"crossplane.io/claim-name":      "postgres-1",
						"crossplane.io/claim-namespace": "test",
					}),
					withStatusServiceID("76c0089e-254e-11ee-be56-0242ac120002"),
					withStatusPlanID("63d05ec8-254e-11ee-be56-0242ac120002"),
				), nil,
			),
			service: fakeOSB,
		},
		Logger: a9stest.TestLogger(t),
	}

	_, err := external.Delete(context.TODO(), serviceBinding)
	if err == nil {
		t.Errorf("Delete method did not return expected error for ServiceBinding %+v",
			serviceBinding)
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected Delete method to return \"%s\" but got error \"%s\"", expectedErr, err)
	}

	unbindReq := unbindReaction.getLastUnbindRequest()
	if unbindReq == nil {
		t.Errorf("OSB client was never called to delete ServiceBinding %+v.",
			serviceBinding)
	}

	if dif := cmp.Diff(expectedUnbindReq, unbindReq); dif != "" {
		t.Errorf("UnbindRequest given to OSB client differs from expected one: -want, +got %s", dif)
	}
}

func withConditions(c ...xpv1.Condition) func(*v1.ServiceBinding) {
	return func(sb *v1.ServiceBinding) { sb.Status.SetConditions(c...) }
}

func withAtProvider(state string, id int) func(*v1.ServiceBinding) {
	return func(sb *v1.ServiceBinding) {
		sb.Status.AtProvider.State = state
		sb.Status.AtProvider.ServiceBindingID = id
	}
}

func serviceBinding(opts ...func(*v1.ServiceBinding)) *v1.ServiceBinding {
	sb := &v1.ServiceBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-sb-asty",
			Labels: map[string]string{
				"crossplane.io/claim-name":      "test-sb",
				"crossplane.io/claim-namespace": "test",
			},
			UID: "1a6a6b3e-254e-11ee-be56-0242ac120002",
		},
	}

	for _, opt := range opts {
		opt(sb)
	}

	return sb
}

func withServiceBindingParameters(params *v1.ServiceBindingParameters) func(*v1.ServiceBinding) {
	return func(sb *v1.ServiceBinding) {
		sb.Spec.ForProvider = *params
	}
}

func withInstanceName(instanceName string) func(*v1.ServiceBinding) {
	return func(sb *v1.ServiceBinding) {
		sb.Spec.ForProvider.InstanceName = instanceName
	}
}

func withAnnotations(annotations map[string]string) func(*dsv1.ServiceInstance) {
	return func(pg *dsv1.ServiceInstance) {
		meta.AddAnnotations(pg, annotations)
	}
}

func afterBindingCreation() func(*v1.ServiceBinding) {
	return func(sb *v1.ServiceBinding) {
		// The key strings here use the constants declared in this test file. This is done because we want the tests to
		// fail loudly if the annotation keys used by the controller change.
		meta.AddAnnotations(sb, map[string]string{
			"anynines.crossplane.io/servicebinding-created": "true",
		})
	}
}

func initializeSBStatus(instanceID, planID, serviceID, hostURL, port string) func(*v1.ServiceBinding) {
	return func(sb *v1.ServiceBinding) {
		sb.Status.AtProvider.InstanceID = instanceID
		sb.Status.AtProvider.PlanID = planID
		sb.Status.AtProvider.ServiceID = serviceID
		sb.AddConnectionDetails(hostURL, port)
	}
}

func deletionTimestamp() func(*v1.ServiceBinding) {
	return func(sb *v1.ServiceBinding) {
		sb.DeletionTimestamp = &metav1.Time{}
	}
}

func withProviderRef(name string) func(*v1.ServiceBinding) {
	return func(sb *v1.ServiceBinding) {
		sb.Spec.ProviderConfigReference = &xpv1.Reference{Name: name}
	}
}

type BindReaction struct {
	Request  *osbclient.BindRequest
	Response *osbclient.BindResponse
	Error    error
}

type GetInstancesReaction struct {
	Response *osbclient.GetInstancesResponse
	Error    error
}

func (s *GetInstancesReaction) React() (*osbclient.GetInstancesResponse, error) {
	return s.Response, s.Error
}

func (s *BindReaction) React(r *osbclient.BindRequest) (*osbclient.BindResponse, error) {
	s.Request = r

	return s.Response, s.Error
}

func (s *BindReaction) getLastBindRequest() *osbclient.BindRequest {
	return s.Request
}

type UnbindReaction struct {
	Request  *osbclient.UnbindRequest
	Response *osbclient.UnbindResponse
	Error    error
}

func (s *UnbindReaction) React(r *osbclient.UnbindRequest) (*osbclient.UnbindResponse, error) {
	s.Request = r

	return s.Response, s.Error
}

func (s *UnbindReaction) getLastUnbindRequest() *osbclient.UnbindRequest {
	return s.Request
}

type ServiceInstanceOption func(p *dsv1.ServiceInstance)

func afterInstanceCreation() func(*dsv1.ServiceInstance) {
	return func(pg *dsv1.ServiceInstance) {
		pg.Spec.ForProvider.ServiceName = ptr.To[string]("a9s-postgresql11")
		pg.Spec.ForProvider.PlanName = ptr.To[string]("postgresql-replica-small")
	}
}

func withStatusPlanID(planId string) func(*dsv1.ServiceInstance) {
	return func(pg *dsv1.ServiceInstance) {
		pg.Status.AtProvider.PlanID = planId
	}
}

func withStatusServiceID(serviceId string) func(*dsv1.ServiceInstance) {
	return func(pg *dsv1.ServiceInstance) {
		pg.Status.AtProvider.ServiceID = serviceId
	}
}

// withStatusInstanceID sets the InstanceID in withStatusInstanceID function, which is
// typically determined by the Observe method based on the annotation from the Create method.
// Note that the Create method will overwrite this field if no InstanceID annotation is present.
func withStatusInstanceID(id string) ServiceInstanceOption {
	return func(pg *dsv1.ServiceInstance) {
		pg.Status.AtProvider.InstanceID = id
	}
}

func newKubeMock(serviceInstanceObject runtime.Object, otherResources []client.Object) client.Client {
	sc := runtime.NewScheme()
	sc.AddKnownTypes(dsv1.SchemeGroupVersion, &dsv1.ServiceInstance{}, &dsv1.ServiceInstanceList{}, &corev1.Secret{})

	objs := make([]runtime.Object, len(otherResources)+1) // change to appease the linter
	objs[0] = serviceInstanceObject
	for i, resource := range otherResources {
		objs[i+1] = resource
	}

	return fake.NewClientBuilder().WithRuntimeObjects(objs...).WithScheme(sc).Build()
}

func serviceInstance(modifiers ...ServiceInstanceOption) *dsv1.ServiceInstance {
	serviceInstance := &dsv1.ServiceInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "postgres-1-sdjk",
			Labels: map[string]string{
				"crossplane.io/claim-name":      "postgres-1",
				"crossplane.io/claim-namespace": "test",
			},
		},
		Spec: dsv1.ServiceInstanceSpec{
			ForProvider: dsv1.ServiceInstanceParameters{
				AcceptsIncomplete: ptr.To[bool](true),
				ServiceName:       ptr.To[string]("a9s-postgresql13"),
				PlanName:          ptr.To[string]("postgresql-replica-small"),
				OrganizationGUID:  ptr.To[string]("a1612e60-3042-4bf2-bd7c-fa600a4f66b9"),
				SpaceGUID:         ptr.To[string]("009dbe05-925d-4f2a-ac0d-8d44dd723a11"),
			},
		},
	}

	for _, modifier := range modifiers {
		modifier(serviceInstance)
	}
	return serviceInstance
}

func serviceInstanceWithName(name string) ServiceInstanceOption {
	return func(serviceInstance *dsv1.ServiceInstance) {
		serviceInstance.Name = name
	}
}

func serviceInstanceWithStatus(status dsv1.ServiceInstanceObservation) ServiceInstanceOption {
	return func(serviceInstance *dsv1.ServiceInstance) {
		serviceInstance.Status.AtProvider = status
	}
}

func serviceInstanceWithLabels(labels map[string]string) ServiceInstanceOption {
	return func(serviceInstance *dsv1.ServiceInstance) {
		if serviceInstance.Labels == nil {
			serviceInstance.Labels = map[string]string{}
		}
		for key, value := range labels {
			serviceInstance.Labels[key] = value
		}
	}
}
