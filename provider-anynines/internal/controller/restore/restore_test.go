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

package restore

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	cmnv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/test"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	a9sbackupmanager "github.com/anynines/klutchio/clients/a9s-backup-manager"
	fakebkpmgr "github.com/anynines/klutchio/clients/a9s-backup-manager/fake"
	bkpv1 "github.com/anynines/klutchio/provider-anynines/apis/backup/v1"
	v1 "github.com/anynines/klutchio/provider-anynines/apis/restore/v1"
	dsv1 "github.com/anynines/klutchio/provider-anynines/apis/serviceinstance/v1"
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
	RestoreOption func(p *v1.Restore)
	BackupOption  func(p *bkpv1.Backup)
)

const (
	errBackupNotFound = utilerr.PlainUserErr("backup was not found")
)

func newRestore(opts ...RestoreOption) *v1.Restore {
	rst := &v1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-6sf265",
			Labels: map[string]string{
				"crossplane.io/claim-name":      "test",
				"crossplane.io/claim-namespace": "test-1",
			},
		},
		Spec: v1.RestoreSpec{
			ForProvider: v1.RestoreParameters{
				BackupName: "test-bkp",
			},
		},
		Status: v1.RestoreStatus{},
	}
	for _, modifier := range opts {
		modifier(rst)
	}
	return rst
}

func withStatus(state, triggeredAt, finishedAt string, conditions []cmnv1.Condition) RestoreOption {
	return func(rst *v1.Restore) {
		rst.Status.AtProvider.RestoreID = ptr.To[int](2)
		rst.Status.AtProvider.State = state
		rst.Status.AtProvider.TriggeredAt = triggeredAt
		rst.Status.AtProvider.FinishedAt = finishedAt
		rst.Status.ConditionedStatus.Conditions = conditions
		rst.Status.AtProvider.InstanceID = "40a5148f-dba2-41f2-b1b7-0ca90e1501c5"
		rst.Status.AtProvider.BackupID = ptr.To[int](29)
	}
}

func afterCreation() RestoreOption {
	return func(rst *v1.Restore) {
		meta.AddAnnotations(rst, map[string]string{
			AnnotationKeyRestoreID: "2",
		})
	}
}

func initializeRestoreStatus(instanceID string, backupID int) RestoreOption {
	return func(rst *v1.Restore) {
		rst.Status.AtProvider.InstanceID = instanceID
		rst.Status.AtProvider.BackupID = ptr.To[int](backupID)
	}
}

func addDeletionTimeStamp() RestoreOption {
	return func(rst *v1.Restore) {
		rst.DeletionTimestamp = &metav1.Time{}
	}
}

func newRestoreResponse(status, triggeredAt, finishedAt string) *a9sbackupmanager.GetRestoreResponse {
	rstResponse := &a9sbackupmanager.GetRestoreResponse{
		BackupID:    ptr.To[int](1),
		RestoreID:   ptr.To[int](2),
		Status:      status,
		TriggeredAt: triggeredAt,
		FinishedAt:  finishedAt,
	}

	return rstResponse
}

func newBackup(modifiers ...BackupOption) *bkpv1.Backup {
	backup := &bkpv1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-bkp-56sa4d",
			Labels: map[string]string{
				"crossplane.io/claim-name":      "test-bkp",
				"crossplane.io/claim-namespace": "test-1",
			},
		},
		Spec: bkpv1.BackupSpec{
			ForProvider: bkpv1.BackupParameters{
				InstanceName: "test-instance",
			},
		},
	}

	for _, modifier := range modifiers {
		modifier(backup)
	}
	return backup
}

func backupWithName(name string) BackupOption {
	return func(backup *bkpv1.Backup) {
		backup.Name = name
	}
}

func backupWithStatus(status bkpv1.BackupObservation) BackupOption {
	return func(backup *bkpv1.Backup) {
		backup.Status.AtProvider = status
	}
}

func backupWithLabels(labels map[string]string) BackupOption {
	return func(backup *bkpv1.Backup) {
		if backup.Labels == nil {
			backup.Labels = map[string]string{}
		}
		for key, value := range labels {
			backup.Labels[key] = value
		}
	}
}

func TestObserve(t *testing.T) {
	t.Parallel()

	type args struct {
		getRestoreReaction fakebkpmgr.GetRestoreReaction
		managedResource    resource.Managed
		backup             bkpv1.Backup
		otherResources     []client.Object
	}

	type want struct {
		observation     managed.ExternalObservation
		managedResource resource.Managed
		err             error
	}

	cases := map[string]struct {
		service         a9sbackupmanager.Client
		ctx             context.Context
		managedResource resource.Managed
		args            args
		want            want
	}{
		"errorServiceInstanceNotFound": {
			args: args{
				getRestoreReaction: fakebkpmgr.GetRestoreReaction{
					Error: a9sbackupmanager.HTTPStatusCodeError{
						StatusCode:    http.StatusNotFound,
						ErrorMessage:  ptr.To[string]("InstanceNotFound"),
						Description:   ptr.To[string]("Instance not found."),
						ResponseError: nil,
					},
				},
				managedResource: newRestore(
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
				),
				backup: *newBackup(backupWithStatus(
					bkpv1.BackupObservation{
						InstanceID: "40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						BackupID:   ptr.To[int](29),
					},
				)),
			},
			want: want{
				observation: managed.ExternalObservation{},
				managedResource: newRestore(
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
				),
			},
		},
		"errorManagedResourceIsNotRestore": {
			args: args{
				managedResource: &dsv1.ServiceInstance{},
			},
			want: want{
				err:             utilerr.ErrInternal,
				managedResource: &dsv1.ServiceInstance{},
			},
		},
		"successRestoreCompleted": {
			args: args{
				getRestoreReaction: fakebkpmgr.GetRestoreReaction{
					Response: newRestoreResponse(
						"done",
						"2023-04-26T15:55:22.658Z",
						"2023-04-26T15:57:59.752Z",
					),
				},
				managedResource: newRestore(
					afterCreation(),
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
				),
				backup: *newBackup(backupWithStatus(
					bkpv1.BackupObservation{
						InstanceID: "40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						BackupID:   ptr.To[int](29),
					},
				)),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				},
				managedResource: newRestore(
					afterCreation(),
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
					withStatus(
						v1.StatusDone,
						"2023-04-26T15:55:22.658Z",
						"2023-04-26T15:57:59.752Z",
						[]cmnv1.Condition{
							{
								Type:    cmnv1.TypeReady,
								Status:  corev1.ConditionTrue,
								Reason:  cmnv1.ReasonAvailable,
								Message: "Restore completed successfully",
							},
						},
					),
				),
			},
		},
		"successObservedQueuedRestore": {
			args: args{
				getRestoreReaction: fakebkpmgr.GetRestoreReaction{
					Response: newRestoreResponse(
						"queued",
						"2023-04-26T15:55:22.658Z",
						"2023-04-26T15:57:59.752Z",
					),
				},
				managedResource: newRestore(
					afterCreation(),
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
				),
				backup: *newBackup(backupWithStatus(
					bkpv1.BackupObservation{
						InstanceID: "40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						BackupID:   ptr.To[int](29),
					},
				)),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				},
				managedResource: newRestore(
					afterCreation(),
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
					withStatus(
						v1.StatusQueued,
						"2023-04-26T15:55:22.658Z",
						"2023-04-26T15:57:59.752Z",
						[]cmnv1.Condition{
							{
								Type:   cmnv1.TypeReady,
								Status: corev1.ConditionFalse,
								Reason: cmnv1.ReasonCreating,
							},
						},
					),
				),
			},
		},
		"successObserveRunningRestore": {
			args: args{
				getRestoreReaction: fakebkpmgr.GetRestoreReaction{
					Response: newRestoreResponse(
						"running",
						"2023-04-26T15:55:22.658Z",
						"2023-04-26T15:57:59.752Z",
					),
				},
				managedResource: newRestore(
					afterCreation(),
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
				),
				backup: *newBackup(backupWithStatus(
					bkpv1.BackupObservation{
						InstanceID: "40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						BackupID:   ptr.To[int](29),
					},
				)),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				},
				managedResource: newRestore(
					afterCreation(),
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
					withStatus(
						v1.StatusRunning,
						"2023-04-26T15:55:22.658Z",
						"2023-04-26T15:57:59.752Z",
						[]cmnv1.Condition{
							{
								Type:   cmnv1.TypeReady,
								Status: corev1.ConditionFalse,
								Reason: cmnv1.ReasonCreating,
							},
						},
					),
				),
			},
		},
		"errorRestoreToDeleteIsCurrentlyQueued": {
			args: args{
				getRestoreReaction: fakebkpmgr.GetRestoreReaction{
					Response: newRestoreResponse(
						"queued",
						"2023-04-26T15:55:22.658Z",
						"2023-04-26T15:57:59.752Z",
					),
				},
				managedResource: newRestore(
					afterCreation(),
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
					addDeletionTimeStamp(),
					withStatus(
						v1.StatusQueued,
						"2023-04-26T15:55:22.658Z",
						"2023-04-26T15:57:59.752Z",
						[]cmnv1.Condition{},
					)),
				backup: *newBackup(backupWithStatus(
					bkpv1.BackupObservation{
						InstanceID: "40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						BackupID:   ptr.To[int](29),
					},
				)),
			},

			want: want{
				err: errRestoreQueued,
			},
		},
		"errorRestoreToDeleteIsCurrentlyRunning": {
			args: args{
				getRestoreReaction: fakebkpmgr.GetRestoreReaction{
					Response: newRestoreResponse(
						"running",
						"2023-04-26T15:55:22.658Z",
						"2023-04-26T15:57:59.752Z",
					),
				},
				managedResource: newRestore(
					afterCreation(),
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
					addDeletionTimeStamp(),
					withStatus(
						v1.StatusRunning,
						"2023-04-26T15:55:22.658Z",
						"2023-04-26T15:57:59.752Z",
						[]cmnv1.Condition{},
					)),
				backup: *newBackup(backupWithStatus(
					bkpv1.BackupObservation{
						InstanceID: "40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						BackupID:   ptr.To[int](29),
					},
				)),
			},
			want: want{
				err: errRestoreRunning,
			},
		},
		"successObservedFailedRestore": {
			args: args{
				getRestoreReaction: fakebkpmgr.GetRestoreReaction{
					Response: newRestoreResponse(
						"failed",
						"2023-04-26T15:55:22.658Z",
						"2023-04-26T15:57:59.752Z",
					),
				},
				managedResource: newRestore(
					afterCreation(),
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
				),
				backup: *newBackup(backupWithStatus(
					bkpv1.BackupObservation{
						InstanceID: "40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						BackupID:   ptr.To[int](29),
					},
				)),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				},
				managedResource: newRestore(
					afterCreation(),
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
					withStatus(
						v1.StatusFailed,
						"2023-04-26T15:55:22.658Z",
						"2023-04-26T15:57:59.752Z",
						[]cmnv1.Condition{
							{
								Type:    cmnv1.TypeReady,
								Status:  corev1.ConditionFalse,
								Reason:  cmnv1.ReasonUnavailable,
								Message: "Restore has failed",
							},
						},
					),
				),
			},
		},
		"successObservedDeletedRestore": {
			args: args{
				getRestoreReaction: fakebkpmgr.GetRestoreReaction{
					Response: newRestoreResponse(
						"deleted",
						"2023-04-26T15:55:22.658Z",
						"2023-04-26T15:57:59.752Z",
					),
				},
				managedResource: newRestore(
					afterCreation(),
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
				),
				backup: *newBackup(backupWithStatus(
					bkpv1.BackupObservation{
						InstanceID: "40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						BackupID:   ptr.To[int](29),
					},
				)),
			},
			want: want{
				observation: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				},
				managedResource: newRestore(
					afterCreation(),
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
					withStatus(
						v1.StatusDeleted,
						"2023-04-26T15:55:22.658Z",
						"2023-04-26T15:57:59.752Z",
						[]cmnv1.Condition{
							{
								Type:    cmnv1.TypeReady,
								Status:  corev1.ConditionFalse,
								Reason:  cmnv1.ReasonUnavailable,
								Message: "Restore has been deleted",
							},
						},
					),
				),
			},
		},
		"errorRestoreAndBackupObjectsInDifferentNamespaces": {
			args: args{
				managedResource: newRestore(),
				backup: *newBackup(
					backupWithLabels(map[string]string{
						"crossplane.io/claim-name":      "test-bkp",
						"crossplane.io/claim-namespace": "test-2",
					}),
					backupWithStatus(
						bkpv1.BackupObservation{
							InstanceID:   "40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
							BackupID:     ptr.To[int](29),
							SizeInBytes:  10,
							TriggeredAt:  "2023-05-01T01:30:00.742Z",
							FinishedAt:   "2023-05-01T01:30:28.300Z",
							Downloadable: true,
						}),
				),
			},
			want: want{
				err:             errBackupNotFound,
				managedResource: newRestore(),
			},
		},
		"errorBackupToRestoreNotFound": {
			args: args{
				managedResource: newRestore(),
			},
			want: want{
				err:             utilerr.PlainUserErr("backup was not found"),
				managedResource: newRestore(),
			},
		},
		"errorMoreThanOneMatchingBackupsExist": {
			args: args{
				managedResource: newRestore(
					afterCreation(),
				),
				backup: *newBackup(backupWithStatus(
					bkpv1.BackupObservation{
						InstanceID:   "40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						BackupID:     ptr.To[int](29),
						SizeInBytes:  10,
						TriggeredAt:  "2023-05-01T01:30:00.742Z",
						FinishedAt:   "2023-05-01T01:30:28.300Z",
						Downloadable: true,
					},
				)),
				otherResources: []client.Object{
					newBackup(
						backupWithName("test-bkp-56sa5d"),
						backupWithLabels(
							map[string]string{
								"crossplane.io/claim-name":      "test-bkp",
								"crossplane.io/claim-namespace": "test-1",
							}),
						backupWithStatus(
							bkpv1.BackupObservation{
								InstanceID:   "40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
								BackupID:     ptr.To[int](29),
								SizeInBytes:  10,
								TriggeredAt:  "2023-05-01T01:30:00.742Z",
								FinishedAt:   "2023-05-01T01:30:28.300Z",
								Downloadable: true,
							},
						),
					),
				},
			},
			want: want{
				managedResource: newRestore(
					afterCreation(),
				),
				err: utilerr.ErrInternal,
			},
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fakeBackupManager := fakebkpmgr.NewFakeClient(&fakebkpmgr.FakeClientConfiguration{
				GetRestoreReaction: tc.args.getRestoreReaction,
			})

			sc := runtime.NewScheme()
			sc.AddKnownTypes(bkpv1.SchemeGroupVersion, &bkpv1.Backup{}, &bkpv1.BackupList{})

			objs := make([]runtime.Object, len(tc.args.otherResources)+1) // change to appease the linter
			objs[0] = &tc.args.backup
			for i, resource := range tc.args.otherResources {
				objs[i+1] = resource
			}

			e := utilerr.Decorator{
				ExternalClient: &external{
					service: fakeBackupManager,
					kube:    fake.NewClientBuilder().WithRuntimeObjects(objs...).WithScheme(sc).Build(),
				},
				Logger: a9stest.TestLogger(t),
			}

			got, err := e.Observe(context.Background(), tc.args.managedResource)
			if diff := cmp.Diff(tc.want.observation, got); diff != "" {
				t.Errorf("Observe(...): -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("Observe(...): -want error, +got error:\n%s", diff)
			}
			if tc.want.managedResource != nil {
				if diff := cmp.Diff(tc.want.managedResource, tc.args.managedResource); diff != "" {
					t.Errorf("Observe(...): -want managed resource, +got managed resource:\n%s", diff)
				}
			}
		})
	}
}

func TestCreate(t *testing.T) {
	t.Parallel()
	type args struct {
		createRestoreReaction fakebkpmgr.CreateRestoreReaction
		managedResource       resource.Managed
	}

	type want struct {
		err             error
		managedResource resource.Managed
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"errorBackupToBeRestoredIsNotDoneYet": {
			args: args{
				createRestoreReaction: fakebkpmgr.CreateRestoreReaction{
					Error: fmt.Errorf("backup is in a non-restorable state: %w", a9sbackupmanager.HTTPStatusCodeError{
						StatusCode:    http.StatusUnprocessableEntity,
						ErrorMessage:  ptr.To[string]("InvalidStatusError"),
						Description:   ptr.To[string]("Cannot restore from backup 29, since it is in state not done."),
						ResponseError: nil,
					}),
				},
				managedResource: newRestore(
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
				),
			},
			want: want{
				err: utilerr.PlainUserErr("Cannot restore from backup 29, since it is in state not done."),
				managedResource: newRestore(
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
				),
			},
		},
		"errorBackupToRestoreFromNotInitialized": {
			args: args{
				createRestoreReaction: fakebkpmgr.CreateRestoreReaction{
					Error: fmt.Errorf("backup is in a non-restorable state: %w", a9sbackupmanager.HTTPStatusCodeError{
						StatusCode:   http.StatusUnprocessableEntity,
						ErrorMessage: ptr.To[string]("InvalidStatusError"),
						Description:  ptr.To[string]("Cannot restore from backup 29, since it is in state not done."),
					}),
				},
				managedResource: newRestore(
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
				),
			},
			want: want{
				err: utilerr.PlainUserErr("Cannot restore from backup 29, since it is in state not done."),
				managedResource: newRestore(
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
				),
			},
		},
		"errorBackupToRestoreFromNotFound": {
			args: args{
				createRestoreReaction: fakebkpmgr.CreateRestoreReaction{
					Error: fmt.Errorf("backup not found: %w", a9sbackupmanager.HTTPStatusCodeError{
						StatusCode:   http.StatusNotFound,
						ErrorMessage: ptr.To[string]("NotFound"),
						Description:  ptr.To[string]("The backup 29 was not found for the instance 40a5148f-dba2-41f2-b1b7-0ca90e1501c5."),
					}),
				},
				managedResource: newRestore(
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
				),
			},
			want: want{
				err: utilerr.PlainUserErr("The backup 29 was not found for the instance 40a5148f-dba2-41f2-b1b7-0ca90e1501c5."),
				managedResource: newRestore(
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
				),
			},
		},
		"errorServiceInstanceToRestoreToIsGone": {
			args: args{
				createRestoreReaction: fakebkpmgr.CreateRestoreReaction{
					Error: a9sbackupmanager.HTTPStatusCodeError{
						StatusCode:    http.StatusGone,
						ErrorMessage:  ptr.To[string]("Gone"),
						Description:   ptr.To[string]("The dataservice instance 40a5148f-dba2-41f2-b1b7-0ca90e1501c5 has been permanently deleted."),
						ResponseError: nil,
					},
				},
				managedResource: newRestore(
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
				),
			}, want: want{
				err: utilerr.PlainUserErr("The dataservice instance 40a5148f-dba2-41f2-b1b7-0ca90e1501c5 has been permanently deleted."),
				managedResource: newRestore(
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
				),
			},
		},
		"errorManagedResourceIsNotRestore": {
			args: args{
				managedResource: &dsv1.ServiceInstance{},
			},
			want: want{
				err:             utilerr.ErrInternal,
				managedResource: &dsv1.ServiceInstance{},
			},
		},
		"errorServiceInstanceAlreadyHasRestoreInProgress": {
			args: args{
				createRestoreReaction: fakebkpmgr.CreateRestoreReaction{
					Error: fmt.Errorf("restore already in progress: %w", a9sbackupmanager.HTTPStatusCodeError{
						StatusCode:   http.StatusConflict,
						ErrorMessage: ptr.To[string]("ConcurrencyError"),
						Description:  ptr.To[string]("A restore is already in progress."),
					}),
				},
				managedResource: newRestore(
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
				),
			},
			want: want{
				err: utilerr.PlainUserErr("A restore is already in progress."),
				managedResource: newRestore(
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
				),
			},
		},
		"successBackupRetrievedAndRestoreIsQueued": {
			args: args{
				createRestoreReaction: fakebkpmgr.CreateRestoreReaction{
					Response: &a9sbackupmanager.CreateRestoreResponse{
						RestoreID: ptr.To[int](2),
					},
				},
				managedResource: newRestore(
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
				),
			},
			want: want{
				managedResource: newRestore(
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
					afterCreation(),
				),
			},
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fakeBackupManager := fakebkpmgr.NewFakeClient(&fakebkpmgr.FakeClientConfiguration{
				CreateRestoreReaction: tc.args.createRestoreReaction,
			})

			e := utilerr.Decorator{
				ExternalClient: &external{
					service: fakeBackupManager,
				},
				Logger: a9stest.TestLogger(t),
			}

			_, err := e.Create(context.Background(), tc.args.managedResource)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("Create(...): -want error, +got error:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.managedResource, tc.args.managedResource); diff != "" {
				t.Errorf("Create(...): -want managed resource, +got managed resource:\n%s", diff)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()
	type args struct {
		managedResource resource.Managed
	}

	type want struct {
		err error
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"notRestore": {
			// Should fail because the Managed Resource is not a restore
			args: args{
				managedResource: &dsv1.ServiceInstance{},
			},
			want: want{
				err: utilerr.ErrInternal,
			},
		},
		"undeletableStateQueued": {
			// Should fail because a queued restore cannot be deleted
			args: args{
				managedResource: newRestore(
					afterCreation(),
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
					withStatus(
						v1.StatusQueued,
						"2023-04-26T15:55:22.658Z",
						"2023-04-26T15:57:59.752Z",
						[]cmnv1.Condition{},
					)),
			},
			want: want{
				err: errRestoreQueued,
			},
		},
		"undeletableStateRunning": {
			// Should fail because a running restore cannot be deleted
			args: args{
				managedResource: newRestore(
					afterCreation(),
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
					withStatus(
						v1.StatusRunning,
						"2023-04-26T15:55:22.658Z",
						"2023-04-26T15:57:59.752Z",
						[]cmnv1.Condition{},
					)),
			},
			want: want{
				err: errRestoreRunning,
			},
		},
		"successfullyDeleted": {
			// Should succeed because the restore can be successfully deleted
			args: args{
				managedResource: newRestore(
					afterCreation(),
					initializeRestoreStatus(
						"40a5148f-dba2-41f2-b1b7-0ca90e1501c5",
						29,
					),
					withStatus(
						v1.StatusDone,
						"2023-04-26T15:55:22.658Z",
						"2023-04-26T15:57:59.752Z",
						[]cmnv1.Condition{},
					)),
			},
			want: want{},
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			config := a9sbackupmanager.DefaultClientConfiguration()
			client, err := a9sbackupmanager.NewClient(config)
			if err != nil {
				t.Fatalf("Error creating new client: %v", err.Error())
			}
			e := utilerr.Decorator{
				ExternalClient: &external{service: client},
				Logger:         a9stest.TestLogger(t),
			}
			var inputMR resource.Managed
			if tc.args.managedResource == nil {
				inputMR = newRestore()
			} else {
				inputMR = tc.args.managedResource
			}
			err = e.Delete(context.Background(), inputMR)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("Observe(...): -want error, +got error:\n%s", diff)
			}
		})
	}
}
