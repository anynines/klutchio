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

package backup_test

import (
	"context"
	"net/http"
	"testing"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/test"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	bkpmgrclient "github.com/anynines/klutchio/clients/a9s-backup-manager"
	fakebkpmgr "github.com/anynines/klutchio/clients/a9s-backup-manager/fake"
	v1 "github.com/anynines/klutchio/provider-anynines/apis/backup/v1"
	dsv1 "github.com/anynines/klutchio/provider-anynines/apis/serviceinstance/v1"
	"github.com/anynines/klutchio/provider-anynines/internal/controller/backup"
	bkpcontroller "github.com/anynines/klutchio/provider-anynines/internal/controller/backup"
	a9stest "github.com/anynines/klutchio/provider-anynines/internal/controller/test"
	utilerr "github.com/anynines/klutchio/provider-anynines/pkg/utilerr"
)

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

type (
	BackupOption           func(*v1.Backup)
	serviceInstanceOptions func(*dsv1.ServiceInstance)
)

func newBackup(modifiers ...BackupOption) *v1.Backup {
	bkp := &v1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-as5654",
		},
	}
	for _, modifier := range modifiers {
		modifier(bkp)
	}
	return bkp
}

func withSpec(forProvider v1.BackupParameters) BackupOption {
	return func(bkp *v1.Backup) {
		bkp.Spec.ForProvider = forProvider
	}
}

func withLabels(labels map[string]string) BackupOption {
	return func(bkp *v1.Backup) {
		if bkp.Labels == nil {
			bkp.Labels = map[string]string{}
		}
		for key, value := range labels {
			bkp.Labels[key] = value
		}
	}
}

func withAnnotations(annotations map[string]string) BackupOption {
	return func(bkp *v1.Backup) {
		if bkp.Annotations == nil {
			bkp.Annotations = map[string]string{}
		}
		for key, value := range annotations {
			bkp.Annotations[key] = value
		}
	}
}

func withStatusAtProvider() BackupOption {
	return func(bkp *v1.Backup) {
		bkp.Status = v1.BackupStatus{
			AtProvider: v1.BackupObservation{
				InstanceID:   "23df2cf9-2ecc-414c-9333-6401f0c54365",
				BackupID:     ptr.To[int](1),
				SizeInBytes:  10,
				TriggeredAt:  "2023-05-01T01:30:00.742Z",
				FinishedAt:   "2023-05-01T01:30:28.300Z",
				Downloadable: true,
			},
		}
	}
}

func withStatusAtProviderInstanceID() BackupOption {
	return func(bkp *v1.Backup) {
		bkp.Status = v1.BackupStatus{
			AtProvider: v1.BackupObservation{
				InstanceID: "23df2cf9-2ecc-414c-9333-6401f0c54365",
			},
		}
	}
}

func afterCreation() BackupOption {
	return withAnnotations(map[string]string{
		"anynines.crossplane.io/backup-id": "1",
	})
}

func expectedCondition(status string, condition xpv1.Condition) BackupOption {
	return func(bkp *v1.Backup) {
		bkp.Status.AtProvider.Status = status

		if condition != (xpv1.Condition{}) {
			bkp.Status.Conditions = append(bkp.Status.Conditions, condition)
		}
	}
}

func successfulRetrievalResponse(status string) *bkpmgrclient.GetBackupResponse {
	return &bkpmgrclient.GetBackupResponse{
		BackupID:     ptr.To[int](1),
		Size:         10,
		Status:       status,
		TriggeredAt:  "2023-05-01T01:30:00.742Z",
		FinishedAt:   "2023-05-01T01:30:28.300Z",
		Downloadable: true,
	}
}

func serviceInstance(modifiers ...serviceInstanceOptions) *dsv1.ServiceInstance {
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

func serviceInstanceWithName(name string) serviceInstanceOptions {
	return func(serviceInstance *dsv1.ServiceInstance) {
		serviceInstance.Name = name
	}
}

func serviceInstanceWithStatus(status dsv1.ServiceInstanceObservation) serviceInstanceOptions {
	return func(serviceInstance *dsv1.ServiceInstance) {
		serviceInstance.Status.AtProvider = status
	}
}

func serviceInstanceWithLabels(labels map[string]string) serviceInstanceOptions {
	return func(serviceInstance *dsv1.ServiceInstance) {
		if serviceInstance.Labels == nil {
			serviceInstance.Labels = map[string]string{}
		}
		for key, value := range labels {
			serviceInstance.Labels[key] = value
		}
	}
}

var successfulObservation = &managed.ExternalObservation{
	ResourceExists:    true,
	ResourceUpToDate:  true,
	ConnectionDetails: managed.ConnectionDetails{},
}

func initializeBackupStatus(instanceID string, backupID int) BackupOption {
	return func(bkp *v1.Backup) {
		bkp.Status.AtProvider.InstanceID = instanceID
		bkp.Status.AtProvider.BackupID = ptr.To[int](backupID)
	}
}

func TestObserve(t *testing.T) {
	t.Parallel()
	type args struct {
		serviceInstance   dsv1.ServiceInstance
		managedResource   resource.Managed
		otherResources    []client.Object
		getBackupReaction fakebkpmgr.GetBackupReaction
	}

	type want struct {
		observation     managed.ExternalObservation
		err             error
		managedResource resource.Managed
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"errorManagedResourceIsNotBackup": {
			args: args{
				managedResource: &dsv1.ServiceInstance{},
			},
			want: want{
				err: utilerr.ErrInternal,
			},
		},
		"errorFailedToGetBackup": {
			args: args{
				getBackupReaction: fakebkpmgr.GetBackupReaction{
					Error: bkpmgrclient.HTTPStatusCodeError{
						StatusCode:   http.StatusNotFound,
						ErrorMessage: ptr.To[string]("NotFound"),
						Description:  ptr.To[string]("The backup 1 was not found for the instance 23df2cf9-2ecc-414c-9333-6401f0c54365."),
					},
				},
				managedResource: newBackup(
					afterCreation(),
					initializeBackupStatus(
						"23df2cf9-2ecc-414c-9333-6401f0c54365",
						1,
					),
					withSpec(v1.BackupParameters{
						InstanceName: "postgres-1",
					},
					)),
				serviceInstance: *serviceInstance(serviceInstanceWithStatus(
					dsv1.ServiceInstanceObservation{
						InstanceID: "23df2cf9-2ecc-414c-9333-6401f0c54365",
					})),
			},
			want: want{
				err: utilerr.PlainUserErr("The backup 1 was not found for the instance 23df2cf9-2ecc-414c-9333-6401f0c54365."),
				managedResource: newBackup(
					afterCreation(),
					initializeBackupStatus(
						"23df2cf9-2ecc-414c-9333-6401f0c54365",
						1,
					),
					withSpec(v1.BackupParameters{
						InstanceName: "postgres-1",
					},
					)),
			},
		},
		"errorBackupHasUnknownStatusOnBackupManager": {
			args: args{
				getBackupReaction: fakebkpmgr.GetBackupReaction{
					Response: successfulRetrievalResponse("supercalifragilisticexpialidocious"),
				},
				managedResource: newBackup(
					afterCreation(),
					initializeBackupStatus(
						"23df2cf9-2ecc-414c-9333-6401f0c54365",
						1,
					),
					withSpec(v1.BackupParameters{
						InstanceName: "postgres-1",
					},
					)),
				serviceInstance: *serviceInstance(serviceInstanceWithStatus(
					dsv1.ServiceInstanceObservation{
						InstanceID: "23df2cf9-2ecc-414c-9333-6401f0c54365",
					})),
			},
			want: want{
				observation: managed.ExternalObservation{},
				managedResource: newBackup(
					withSpec(v1.BackupParameters{
						InstanceName: "postgres-1",
					},
					),
					afterCreation(),
					initializeBackupStatus(
						"23df2cf9-2ecc-414c-9333-6401f0c54365",
						1,
					),
					withStatusAtProvider(),
					expectedCondition("supercalifragilisticexpialidocious",
						xpv1.Condition{},
					)),
				err: utilerr.ErrInternal,
			},
		},
		"successBackupDoesNotExist": {
			args: args{
				managedResource: newBackup(
					withSpec(v1.BackupParameters{
						InstanceName: "postgres-1",
					},
					),
					initializeBackupStatus(
						"23df2cf9-2ecc-414c-9333-6401f0c54365",
						1,
					)),
				serviceInstance: *serviceInstance(serviceInstanceWithStatus(
					dsv1.ServiceInstanceObservation{
						InstanceID: "23df2cf9-2ecc-414c-9333-6401f0c54365",
					})),
			},
			want: want{
				observation: managed.ExternalObservation{},
			},
		},
		"successBackupObservedToBeQueued": {
			args: args{
				getBackupReaction: fakebkpmgr.GetBackupReaction{
					Response: successfulRetrievalResponse("queued"),
				},
				managedResource: newBackup(
					afterCreation(),
					initializeBackupStatus(
						"23df2cf9-2ecc-414c-9333-6401f0c54365",
						1,
					),
					withSpec(v1.BackupParameters{
						InstanceName: "postgres-1",
					},
					),
				),
				serviceInstance: *serviceInstance(serviceInstanceWithStatus(
					dsv1.ServiceInstanceObservation{
						InstanceID: "23df2cf9-2ecc-414c-9333-6401f0c54365",
					})),
			},
			want: want{
				observation: *successfulObservation,
				managedResource: newBackup(
					withSpec(v1.BackupParameters{
						InstanceName: "postgres-1",
					}),
					afterCreation(),
					withStatusAtProvider(),
					expectedCondition("queued", xpv1.Creating()),
				),
			},
		},
		"successBackupObservedToBeRunning": {
			args: args{
				getBackupReaction: fakebkpmgr.GetBackupReaction{
					Response: successfulRetrievalResponse("running"),
				},
				managedResource: newBackup(
					withSpec(v1.BackupParameters{
						InstanceName: "postgres-1",
					}),
					afterCreation(),
					initializeBackupStatus(
						"23df2cf9-2ecc-414c-9333-6401f0c54365",
						1,
					),
				),
				serviceInstance: *serviceInstance(serviceInstanceWithStatus(
					dsv1.ServiceInstanceObservation{
						InstanceID: "23df2cf9-2ecc-414c-9333-6401f0c54365",
					})),
			},
			want: want{
				observation: *successfulObservation,
				managedResource: newBackup(
					afterCreation(),
					withStatusAtProvider(),
					expectedCondition("running", xpv1.Creating()),
					withSpec(v1.BackupParameters{
						InstanceName: "postgres-1",
					},
					)),
			},
		},
		"successBackupObservedToBeDone": {
			args: args{
				getBackupReaction: fakebkpmgr.GetBackupReaction{
					Response: successfulRetrievalResponse("done"),
				},
				managedResource: newBackup(
					afterCreation(),
					initializeBackupStatus(
						"23df2cf9-2ecc-414c-9333-6401f0c54365",
						1,
					),
					withSpec(v1.BackupParameters{
						InstanceName: "postgres-1",
					},
					)),
				serviceInstance: *serviceInstance(serviceInstanceWithStatus(
					dsv1.ServiceInstanceObservation{
						InstanceID: "23df2cf9-2ecc-414c-9333-6401f0c54365",
					})),
			},
			want: want{
				observation: *successfulObservation,
				managedResource: newBackup(
					afterCreation(),
					withStatusAtProvider(),
					expectedCondition("done", xpv1.Available()),
					withSpec(v1.BackupParameters{
						InstanceName: "postgres-1",
					},
					)),
			},
		},
		"successBackupObservedToBeFailed": {
			args: args{
				getBackupReaction: fakebkpmgr.GetBackupReaction{
					Response: successfulRetrievalResponse("failed"),
				},
				managedResource: newBackup(
					afterCreation(),
					initializeBackupStatus(
						"23df2cf9-2ecc-414c-9333-6401f0c54365",
						1,
					),
					withSpec(v1.BackupParameters{
						InstanceName: "postgres-1",
					},
					)),
				serviceInstance: *serviceInstance(serviceInstanceWithStatus(
					dsv1.ServiceInstanceObservation{
						InstanceID: "23df2cf9-2ecc-414c-9333-6401f0c54365",
					})),
			},
			want: want{
				observation: *successfulObservation,
				managedResource: newBackup(
					afterCreation(),
					withStatusAtProvider(),
					expectedCondition("failed",
						xpv1.Unavailable().WithMessage("Backup has failed")),
					withSpec(v1.BackupParameters{
						InstanceName: "postgres-1",
					},
					)),
			},
		},
		"successBackupObservedToBeDeleted": {
			args: args{
				getBackupReaction: fakebkpmgr.GetBackupReaction{
					Response: successfulRetrievalResponse("deleted"),
				},
				managedResource: newBackup(
					afterCreation(),
					initializeBackupStatus(
						"23df2cf9-2ecc-414c-9333-6401f0c54365",
						1,
					),
					withSpec(v1.BackupParameters{
						InstanceName: "postgres-1",
					},
					)),
				serviceInstance: *serviceInstance(serviceInstanceWithStatus(
					dsv1.ServiceInstanceObservation{
						InstanceID: "23df2cf9-2ecc-414c-9333-6401f0c54365",
					})),
			},
			want: want{
				observation: managed.ExternalObservation{},
				managedResource: newBackup(
					afterCreation(),
					withStatusAtProvider(),
					expectedCondition("deleted",
						xpv1.Unavailable().WithMessage("Backup has been deleted")),
					withSpec(v1.BackupParameters{
						InstanceName: "postgres-1",
					},
					)),
			},
		},

		"errorServiceInstanceNotFound": {
			// Observe should fail because there is no matching ServiceInstance MR
			args: args{
				managedResource: newBackup(
					withLabels(
						map[string]string{
							"crossplane.io/claim-name":      "test",
							"crossplane.io/claim-namespace": "test",
						}),
					withSpec(v1.BackupParameters{
						InstanceName: "postgres-1",
					},
					)),
			},
			want: want{
				err: backup.ErrServiceInstanceNotFound,
			},
		},
		"errorMoreThanOneMatchingServiceInstanceExists": {
			// Observe should fail because there are multiple matching ServiceInstance MR
			args: args{
				managedResource: newBackup(
					withLabels(
						map[string]string{
							"crossplane.io/claim-name":      "test",
							"crossplane.io/claim-namespace": "test",
						}),
					withSpec(v1.BackupParameters{
						InstanceName: "postgres-1",
					},
					)),
				serviceInstance: *serviceInstance(serviceInstanceWithStatus(
					dsv1.ServiceInstanceObservation{
						InstanceID: "23df2cf9-2ecc-414c-9333-6401f0c54365",
					},
				)),
				otherResources: []client.Object{
					serviceInstance(
						serviceInstanceWithName("postgres-1-sdjl"),
						serviceInstanceWithStatus(
							dsv1.ServiceInstanceObservation{
								InstanceID: "23df2cf9-2ecc-414c-9333-6401f0c54365",
							},
						),
					),
				},
			},
			want: want{
				err: utilerr.ErrInternal,
				managedResource: newBackup(
					withLabels(
						map[string]string{
							"crossplane.io/claim-name":      "test",
							"crossplane.io/claim-namespace": "test",
						}),
					withSpec(v1.BackupParameters{
						InstanceName: "postgres-1",
					},
					)),
			},
		},
		"errorServiceInstanceAndBackupInDifferentNamespaces": {
			// Observe should fail because backup and ServiceInstance are in a different namespace
			args: args{
				managedResource: newBackup(
					withLabels(
						map[string]string{
							"crossplane.io/claim-name":      "test",
							"crossplane.io/claim-namespace": "test",
						}),
					withSpec(v1.BackupParameters{
						InstanceName: "postgres-1",
					},
					)),
				serviceInstance: *serviceInstance(serviceInstanceWithLabels(
					map[string]string{
						"crossplane.io/claim-name":      "postgres",
						"crossplane.io/claim-namespace": "test-1",
					},
				),
					serviceInstanceWithStatus(
						dsv1.ServiceInstanceObservation{
							InstanceID: "23df2cf9-2ecc-414c-9333-6401f0c54365",
						},
					)),
			},
			want: want{
				err: backup.ErrServiceInstanceNotFound,
				managedResource: newBackup(
					withLabels(
						map[string]string{
							"crossplane.io/claim-name":      "test",
							"crossplane.io/claim-namespace": "test",
						}),
					withSpec(v1.BackupParameters{
						InstanceName: "postgres-1",
					},
					)),
			},
		},
	}

	for name, tc := range cases {
		// Rebind tc into this lexical scope. Details on the why at
		// https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fakeBackupManager := fakebkpmgr.NewFakeClient(&fakebkpmgr.FakeClientConfiguration{
				GetBackupReaction: tc.args.getBackupReaction,
			})

			sc := runtime.NewScheme()
			sc.AddKnownTypes(dsv1.SchemeGroupVersion, &dsv1.ServiceInstance{}, &dsv1.ServiceInstanceList{})

			var objs []runtime.Object
			objs = append(objs, &tc.args.serviceInstance)

			for _, resources := range tc.args.otherResources {
				objs = append(objs, resources)
			}

			e := utilerr.Decorator{
				ExternalClient: &bkpcontroller.External{
					Client: fakeBackupManager,
					Kube:   fake.NewClientBuilder().WithRuntimeObjects(objs...).WithScheme(sc).Build(),
				},
				Logger: a9stest.TestLogger(t),
			}

			// invoke the method under test
			got, err := e.Observe(context.Background(), tc.args.managedResource)
			if diff := cmp.Diff(tc.want.observation, got); diff != "" {
				t.Errorf("\n%s\nObserve(...): -want error, +got error:\n%s\n", t.Name(), diff)
			}
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nObserve(...): -want error, +got error:\n%s\n", t.Name(), diff)
			}
			if tc.want.managedResource != nil {
				if diff := cmp.Diff(tc.want.managedResource, tc.args.managedResource); diff != "" {
					t.Errorf("\n%s\nObserve(...): -want managed resource, +got managed resource:\n%s", t.Name(), diff)
				}
			}
		})
	}
}

func TestCreate(t *testing.T) {
	t.Parallel()

	type args struct {
		createBackupReaction fakebkpmgr.CreateBackupReaction
		managedResource      resource.Managed
		serviceInstance      dsv1.ServiceInstance
		otherResources       []client.Object
	}

	type want struct {
		err              error
		externalCreation managed.ExternalCreation
		managedResource  resource.Managed
	}

	cases := map[string]struct {
		args args
		want want
	}{ // TODO: Add further error test cases , e.g. with required parameters for the request missing or malformed response JSONs
		"errorManagedResourceIsNotBackup": {
			// Create should throw an error because the Managed Resource is not a backup
			args: args{
				managedResource: &dsv1.ServiceInstance{},
			},
			want: want{
				err:              utilerr.ErrInternal,
				managedResource:  &dsv1.ServiceInstance{},
				externalCreation: managed.ExternalCreation{},
			},
		},
		"errorServiceInstanceToBackUpNotFound": {
			// Create should throw an error because creating the backup at the Backup Manager failed
			args: args{
				createBackupReaction: fakebkpmgr.CreateBackupReaction{
					Error: bkpmgrclient.InstanceNotFoundError{
						Reason: bkpmgrclient.HTTPStatusCodeError{
							StatusCode:   404,
							Description:  ptr.To[string]("instance 23df2cf9-2ecc-414c-9333-6401f0c54365 does not exist"),
							ErrorMessage: ptr.To[string]("NotFound"),
						},
					},
				},
				managedResource: newBackup(withLabels(
					map[string]string{
						"crossplane.io/claim-name":      "test",
						"crossplane.io/claim-namespace": "test",
					}),
					initializeBackupStatus(
						"23df2cf9-2ecc-414c-9333-6401f0c54365",
						1,
					),
					withSpec(v1.BackupParameters{
						InstanceName: "postgres-1",
					},
					)),
			},
			want: want{
				err: utilerr.PlainUserErr("instance 23df2cf9-2ecc-414c-9333-6401f0c54365 does not exist"),
				managedResource: newBackup(
					withLabels(
						map[string]string{
							"crossplane.io/claim-name":      "test",
							"crossplane.io/claim-namespace": "test",
						}),
					initializeBackupStatus(
						"23df2cf9-2ecc-414c-9333-6401f0c54365",
						1,
					),
					withSpec(v1.BackupParameters{
						InstanceName: "postgres-1",
					},
					)),
				externalCreation: managed.ExternalCreation{},
			},
		},
		"successBackupHasBeenCreated": {
			// Create should succeed because the Backup has been successfully created at the Backup Manager, its status is queued
			args: args{
				createBackupReaction: fakebkpmgr.CreateBackupReaction{
					Response: &bkpmgrclient.CreateBackupResponse{
						BackupID: ptr.To[int](1),
						Message:  ptr.To[string]("job to backup is queued"),
					},
				},
				managedResource: newBackup(
					withLabels(
						map[string]string{
							"crossplane.io/claim-name":      "test",
							"crossplane.io/claim-namespace": "test",
						}),
					initializeBackupStatus(
						"23df2cf9-2ecc-414c-9333-6401f0c54365",
						1,
					),
					withSpec(v1.BackupParameters{
						InstanceName: "postgres-1",
					},
					)),
			},
			want: want{
				managedResource: newBackup(
					withSpec(v1.BackupParameters{
						InstanceName: "postgres-1",
					}),
					initializeBackupStatus(
						"23df2cf9-2ecc-414c-9333-6401f0c54365",
						1,
					),
					withAnnotations(map[string]string{
						"anynines.crossplane.io/backup-id": "1",
					}),
					withLabels(
						map[string]string{
							"crossplane.io/claim-name":      "test",
							"crossplane.io/claim-namespace": "test",
						},
					)),

				externalCreation: managed.ExternalCreation{},
			},
		},
	}

	for name, tc := range cases {

		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fakeBackupManager := fakebkpmgr.NewFakeClient(&fakebkpmgr.FakeClientConfiguration{
				CreateBackupReaction: tc.args.createBackupReaction,
			})

			sc := runtime.NewScheme()
			sc.AddKnownTypes(dsv1.SchemeGroupVersion, &dsv1.ServiceInstance{}, &dsv1.ServiceInstanceList{})

			var objs []runtime.Object
			objs = append(objs, &tc.args.serviceInstance)

			for _, resources := range tc.args.otherResources {
				objs = append(objs, resources)
			}

			e := utilerr.Decorator{
				ExternalClient: &bkpcontroller.External{
					Client: fakeBackupManager,
					Kube:   fake.NewClientBuilder().WithRuntimeObjects(objs...).WithScheme(sc).Build(),
				},
				Logger: a9stest.TestLogger(t),
			}

			// invoke the method under test
			result, err := e.Create(context.Background(), tc.args.managedResource)

			if diff := cmp.Diff(tc.want.externalCreation, result); diff != "" {
				t.Errorf("\n%s\nCreate(...): -want external creation, +got external creation:\n%s", t.Name(), diff)
			}
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nCreate(...): -want error, +got error:\n%s", t.Name(), diff)
			}
			if diff := cmp.Diff(tc.want.managedResource, tc.args.managedResource); diff != "" {
				t.Errorf("\n%s\nCreate(...): -want managed resource, +got managed resource:\n%s", t.Name(), diff)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()
	type args struct {
		deleteBackupReaction fakebkpmgr.DeleteBackupReaction
		managedResource      resource.Managed
	}

	type want struct {
		err error
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"errorManagedResourceIsNotBackup": {
			args: args{
				managedResource: &dsv1.ServiceInstance{},
			},
			want: want{
				err: utilerr.ErrInternal,
			},
		},
		"errorServiceInstanceToBackUpNotFound": {
			args: args{
				deleteBackupReaction: fakebkpmgr.DeleteBackupReaction{
					Error: bkpmgrclient.InstanceNotFoundError{
						Reason: bkpmgrclient.HTTPStatusCodeError{
							StatusCode:    404,
							ErrorMessage:  ptr.To[string]("NotFound"),
							Description:   ptr.To[string]("instance 23df2cf9-2ecc-414c-9333-6401f0c54365 does not exist"),
							ResponseError: nil,
						},
					},
				},
			},
			want: want{
				err: utilerr.PlainUserErr("instance 23df2cf9-2ecc-414c-9333-6401f0c54365 does not exist"),
			},
		},
		"successNoBackupIDSet": {
			args: args{
				managedResource: newBackup(),
			},
			want: want{
				err: nil,
			},
		},
		"errorBackupIDNotANumber": {
			args: args{
				managedResource: newBackup(
					withAnnotations(map[string]string{
						"anynines.crossplane.io/backup-id":   "hello-world",
						"anynines.crossplane.io/instance-id": "23df2cf9-2ecc-414c-9333-6401f0c54365",
					}),
					withSpec(v1.BackupParameters{
						InstanceName: "postgres-1",
					},
					)),
			},
			want: want{
				err: utilerr.ErrInternal,
			},
		},
		"successBackupHasBeenDeleted": {
			args: args{
				deleteBackupReaction: fakebkpmgr.DeleteBackupReaction{
					Response: &bkpmgrclient.DeleteBackupResponse{
						Message: ptr.To[string]("backup file deleted"),
					},
				},
			},
		},
	}

	for name, tc := range cases {

		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fakeBackupManager := fakebkpmgr.NewFakeClient(&fakebkpmgr.FakeClientConfiguration{
				DeleteBackupReaction: tc.args.deleteBackupReaction,
			})

			e := utilerr.Decorator{
				ExternalClient: &bkpcontroller.External{Client: fakeBackupManager},
				Logger:         a9stest.TestLogger(t),
			}

			var inputMR resource.Managed
			if tc.args.managedResource == nil {
				inputMR = newBackup(afterCreation(), withStatusAtProviderInstanceID())
			} else {
				inputMR = tc.args.managedResource
			}

			// invoke the method under test
			err := e.Delete(context.Background(), inputMR)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nDelete(...): -want error, +got error:\n%s", t.Name(), diff)
			}
		})
	}
}
