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

package fake_test

import (
	"testing"

	backupmanager "github.com/anynines/klutchio/clients/a9s-backup-manager"
	"github.com/anynines/klutchio/clients/a9s-backup-manager/fake"
	"github.com/google/go-cmp/cmp"
	"k8s.io/utils/pointer"
)

func TestCreateBackup(t *testing.T) {
	type args struct {
		r   *backupmanager.CreateBackupRequest
		cfg *fake.FakeClientConfiguration
	}
	tests := []struct {
		name  string
		args  args
		want  *backupmanager.CreateBackupResponse
		error error
	}{
		{
			name: "successBackupCreated",
			args: args{
				cfg: &fake.FakeClientConfiguration{
					CreateBackupReaction: fake.CreateBackupReaction{
						Response: &backupmanager.CreateBackupResponse{
							BackupID: pointer.Int(1),
							Message:  pointer.String("job to backup is queued"),
						},
					},
				},
				r: &backupmanager.CreateBackupRequest{
					InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
				},
			},
			want: &backupmanager.CreateBackupResponse{
				BackupID: pointer.Int(1),
				Message:  pointer.String("job to backup is queued"),
			},
		},
		{
			name: "errorInstanceNotFound",
			args: args{
				r: &backupmanager.CreateBackupRequest{
					InstanceID: "non-existant-instance",
				},
				cfg: &fake.FakeClientConfiguration{
					CreateBackupReaction: fake.CreateBackupReaction{
						Error: backupmanager.HTTPStatusCodeError{
							StatusCode:   404,
							ErrorMessage: pointer.String("NotFound"),
							Description:  pointer.String("The instance \"non-existant-instance\" was not found"),
						},
					},
				},
			},
			error: backupmanager.HTTPStatusCodeError{
				StatusCode:   404,
				ErrorMessage: pointer.String("NotFound"),
				Description:  pointer.String("The instance \"non-existant-instance\" was not found"),
			},
		},
		{
			name: "errorUnexpectedAction",
			args: args{
				r: &backupmanager.CreateBackupRequest{
					InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
				},
			},
			error: fake.UnexpectedActionError(),
		},
	}
	for _, tt := range tests {

		tt := tt

		t.Run(tt.name, func(t *testing.T) {

			t.Parallel()

			c := fake.NewFakeClient(tt.args.cfg)
			got, err := c.CreateBackup(tt.args.r)
			if diff := cmp.Diff(tt.error, err, EquateHTTPSErrors()); diff != "" {
				t.Errorf("FakeClient.CreateBackup() -want error, +got error:\n%s", diff)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("FakeClient.CreateBackup() -want, +got:\n%s", diff)
			}
		})
	}
}

func TestCreateRestore(t *testing.T) {
	type args struct {
		r   *backupmanager.CreateRestoreRequest
		cfg *fake.FakeClientConfiguration
	}
	tests := []struct {
		name  string
		args  args
		want  *backupmanager.CreateRestoreResponse
		error error
	}{
		{
			name: "successRestoreCreated",
			args: args{
				cfg: &fake.FakeClientConfiguration{
					CreateRestoreReaction: fake.CreateRestoreReaction{
						Response: &backupmanager.CreateRestoreResponse{
							RestoreID: pointer.Int(1),
						},
					},
				},
				r: &backupmanager.CreateRestoreRequest{
					InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
				},
			},
			want: &backupmanager.CreateRestoreResponse{
				RestoreID: pointer.Int(1),
			},
		},
		{
			name: "errorInstanceNotFound",
			args: args{
				r: &backupmanager.CreateRestoreRequest{
					InstanceID: "non-existant-instance",
				},
				cfg: &fake.FakeClientConfiguration{
					CreateRestoreReaction: fake.CreateRestoreReaction{
						Error: backupmanager.HTTPStatusCodeError{
							StatusCode:   404,
							ErrorMessage: pointer.String("NotFound"),
							Description:  pointer.String("The instance \"non-existant-instance\" was not found"),
						},
					},
				},
			},
			error: backupmanager.HTTPStatusCodeError{
				StatusCode:   404,
				ErrorMessage: pointer.String("NotFound"),
				Description:  pointer.String("The instance \"non-existant-instance\" was not found"),
			},
		},
		{
			name: "errorUnexpectedAction",
			args: args{
				r: &backupmanager.CreateRestoreRequest{
					InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
				},
			},
			error: fake.UnexpectedActionError(),
		},
	}
	for _, tt := range tests {

		tt := tt

		t.Run(tt.name, func(t *testing.T) {

			t.Parallel()

			c := fake.NewFakeClient(tt.args.cfg)
			got, err := c.CreateRestore(tt.args.r)
			if diff := cmp.Diff(tt.error, err, EquateHTTPSErrors()); diff != "" {
				t.Errorf("FakeClient.CreateRestore() -want error, +got error:\n%s", diff)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("FakeClient.CreateRestore() -want, +got:\n%s", diff)
			}
		})
	}
}

func TestGetBackup(t *testing.T) {
	type args struct {
		r   *backupmanager.GetBackupRequest
		cfg *fake.FakeClientConfiguration
	}
	tests := []struct {
		name  string
		args  args
		want  *backupmanager.GetBackupResponse
		error error
	}{
		{
			name: "successBackupRetrieved",
			args: args{
				cfg: &fake.FakeClientConfiguration{
					GetBackupReaction: fake.GetBackupReaction{
						Response: &backupmanager.GetBackupResponse{
							BackupID:     pointer.Int(1),
							Size:         10,
							Status:       "done",
							TriggeredAt:  "2023-04-11T08:52:48.209Z",
							FinishedAt:   "2023-04-11T08:53:16.411Z",
							Downloadable: false,
						},
					},
				},
				r: &backupmanager.GetBackupRequest{
					InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
					BackupID:   "1",
				},
			},
			want: &backupmanager.GetBackupResponse{
				BackupID:     pointer.Int(1),
				Size:         10,
				Status:       "done",
				TriggeredAt:  "2023-04-11T08:52:48.209Z",
				FinishedAt:   "2023-04-11T08:53:16.411Z",
				Downloadable: false,
			},
		},
		{
			name: "errorInstanceNotFound",
			args: args{
				r: &backupmanager.GetBackupRequest{
					InstanceID: "non-existant-instance",
					BackupID:   "non-existant-backup",
				},
				cfg: &fake.FakeClientConfiguration{
					GetBackupReaction: fake.GetBackupReaction{
						Error: backupmanager.HTTPStatusCodeError{
							StatusCode:   404,
							ErrorMessage: pointer.String("NotFound"),
							Description:  pointer.String("The instance \"non-existant-instance\" was not found"),
						},
					},
				},
			},
			error: backupmanager.HTTPStatusCodeError{
				StatusCode:   404,
				ErrorMessage: pointer.String("NotFound"),
				Description:  pointer.String("The instance \"non-existant-instance\" was not found"),
			},
		},
		{
			name: "errorUnexpectedAction",
			args: args{
				r: &backupmanager.GetBackupRequest{
					InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
					BackupID:   "1",
				},
			},
			error: fake.UnexpectedActionError(),
		},
	}
	for _, tt := range tests {

		tt := tt

		t.Run(tt.name, func(t *testing.T) {

			t.Parallel()

			c := fake.NewFakeClient(tt.args.cfg)
			got, err := c.GetBackup(tt.args.r)
			if diff := cmp.Diff(tt.error, err, EquateHTTPSErrors()); diff != "" {
				t.Errorf("FakeClient.GetBackup() -want error, +got error:\n%s", diff)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("FakeClient.GetBackup() -want, +got:\n%s", diff)
			}
		})
	}
}

func TestGetBackups(t *testing.T) {
	type args struct {
		r   *backupmanager.GetBackupsRequest
		cfg *fake.FakeClientConfiguration
	}
	tests := []struct {
		name  string
		args  args
		want  *backupmanager.GetBackupsResponse
		error error
	}{
		{
			name: "successBackupsRetrieved",
			args: args{
				cfg: &fake.FakeClientConfiguration{
					GetBackupsReaction: fake.GetBackupsReaction{
						Response: &backupmanager.GetBackupsResponse{
							Backups: []backupmanager.GetBackupResponse{
								{
									BackupID:     pointer.Int(1),
									Size:         10,
									Status:       "done",
									TriggeredAt:  "2023-04-11T08:52:48.209Z",
									FinishedAt:   "2023-04-11T08:53:16.411Z",
									Downloadable: false,
								},
							},
						},
					},
				},
				r: &backupmanager.GetBackupsRequest{
					InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
				},
			},
			want: &backupmanager.GetBackupsResponse{
				Backups: []backupmanager.GetBackupResponse{
					{
						BackupID:     pointer.Int(1),
						Size:         10,
						Status:       "done",
						TriggeredAt:  "2023-04-11T08:52:48.209Z",
						FinishedAt:   "2023-04-11T08:53:16.411Z",
						Downloadable: false,
					},
				},
			},
		},
		{
			name: "errorInstanceNotFound",
			args: args{
				r: &backupmanager.GetBackupsRequest{
					InstanceID: "non-existant-instance",
				},
				cfg: &fake.FakeClientConfiguration{
					GetBackupsReaction: fake.GetBackupsReaction{
						Error: backupmanager.HTTPStatusCodeError{
							StatusCode:   404,
							ErrorMessage: pointer.String("NotFound"),
							Description:  pointer.String("The instance \"non-existant-instance\" was not found"),
						},
					},
				},
			},
			error: backupmanager.HTTPStatusCodeError{
				StatusCode:   404,
				ErrorMessage: pointer.String("NotFound"),
				Description:  pointer.String("The instance \"non-existant-instance\" was not found"),
			},
		},
		{
			name: "errorUnexpectedAction",
			args: args{
				r: &backupmanager.GetBackupsRequest{
					InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
				},
			},
			error: fake.UnexpectedActionError(),
		},
	}
	for _, tt := range tests {

		tt := tt

		t.Run(tt.name, func(t *testing.T) {

			t.Parallel()

			c := fake.NewFakeClient(tt.args.cfg)
			got, err := c.GetBackups(tt.args.r)
			if diff := cmp.Diff(tt.error, err, EquateHTTPSErrors()); diff != "" {
				t.Errorf("FakeClient.GetBackups() -want error, +got error:\n%s", diff)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("FakeClient.GetBackups() -want, +got:\n%s", diff)
			}
		})
	}
}

func TestGetInstanceConfig(t *testing.T) {
	type args struct {
		r   *backupmanager.GetInstanceConfigRequest
		cfg *fake.FakeClientConfiguration
	}
	tests := []struct {
		name  string
		args  args
		want  *backupmanager.GetInstanceConfigResponse
		error error
	}{
		{
			name: "successInstanceConfigRetrieved",
			args: args{
				cfg: &fake.FakeClientConfiguration{
					GetInstanceConfigReaction: fake.GetInstanceConfigReaction{
						Response: &backupmanager.GetInstanceConfigResponse{
							MinBackupCount:         pointer.Int(3),
							RetentionTime:          pointer.Int(2),
							MinEncryptionKeyLength: pointer.Int(5),
							ExcludeFromAutoBackup:  pointer.Bool(true),
							BackupType:             pointer.String("regular"),
						},
					},
				},
				r: &backupmanager.GetInstanceConfigRequest{
					InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
				},
			},
			want: &backupmanager.GetInstanceConfigResponse{
				MinBackupCount:         pointer.Int(3),
				RetentionTime:          pointer.Int(2),
				MinEncryptionKeyLength: pointer.Int(5),
				ExcludeFromAutoBackup:  pointer.Bool(true),
				BackupType:             pointer.String("regular"),
			},
		},
		{
			name: "errorInstanceNotFound",
			args: args{
				r: &backupmanager.GetInstanceConfigRequest{
					InstanceID: "non-existant-instance",
				},
				cfg: &fake.FakeClientConfiguration{
					GetInstanceConfigReaction: fake.GetInstanceConfigReaction{
						Error: backupmanager.HTTPStatusCodeError{
							StatusCode:   404,
							ErrorMessage: pointer.String("NotFound"),
							Description:  pointer.String("The instance \"non-existant-instance\" was not found"),
						},
					},
				},
			},
			error: backupmanager.HTTPStatusCodeError{
				StatusCode:   404,
				ErrorMessage: pointer.String("NotFound"),
				Description:  pointer.String("The instance \"non-existant-instance\" was not found"),
			},
		},
		{
			name: "errorUnexpectedAction",
			args: args{
				r: &backupmanager.GetInstanceConfigRequest{
					InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
				},
			},
			error: fake.UnexpectedActionError(),
		},
	}
	for _, tt := range tests {

		tt := tt

		t.Run(tt.name, func(t *testing.T) {

			t.Parallel()

			c := fake.NewFakeClient(tt.args.cfg)
			got, err := c.GetInstanceConfig(tt.args.r)
			if diff := cmp.Diff(tt.error, err, EquateHTTPSErrors()); diff != "" {
				t.Errorf("FakeClient.GetInstanceConfig() -want error, +got error:\n%s", diff)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("FakeClient.GetInstanceConfig() -want, +got:\n%s", diff)
			}
		})
	}
}

func TestUpdateBackupConfig(t *testing.T) {
	type args struct {
		r   *backupmanager.UpdateBackupConfigRequest
		cfg *fake.FakeClientConfiguration
	}
	tests := []struct {
		name  string
		args  args
		want  *backupmanager.UpdateBackupConfigResponse
		error error
	}{
		{
			name: "successBackupConfigUpdated",
			args: args{
				cfg: &fake.FakeClientConfiguration{
					UpdateBackupConfigReaction: fake.UpdateBackupConfigReaction{
						Response: &backupmanager.UpdateBackupConfigResponse{
							Message: pointer.String("Backup config successfully updated"),
						},
					},
				},
				r: &backupmanager.UpdateBackupConfigRequest{
					InstanceID:               "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
					EncryptionKey:            pointer.String("new-encryption-key"),
					ExcludeFromAutoBackup:    pointer.Bool(true),
					CredentialsUpdatedByUser: pointer.Bool(true),
				},
			},
			want: &backupmanager.UpdateBackupConfigResponse{
				Message: pointer.String("Backup config successfully updated"),
			},
		},
		{
			name: "errorInstanceNotFound",
			args: args{
				r: &backupmanager.UpdateBackupConfigRequest{
					InstanceID: "non-existant-instance",
				},
				cfg: &fake.FakeClientConfiguration{
					UpdateBackupConfigReaction: fake.UpdateBackupConfigReaction{
						Error: backupmanager.HTTPStatusCodeError{
							StatusCode:   404,
							ErrorMessage: pointer.String("NotFound"),
							Description:  pointer.String("The instance \"non-existant-instance\" was not found"),
						},
					},
				},
			},
			error: backupmanager.HTTPStatusCodeError{
				StatusCode:   404,
				ErrorMessage: pointer.String("NotFound"),
				Description:  pointer.String("The instance \"non-existant-instance\" was not found"),
			},
		},
		{
			name: "errorUnexpectedAction",
			args: args{
				r: &backupmanager.UpdateBackupConfigRequest{
					InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
				},
			},
			error: fake.UnexpectedActionError(),
		},
	}
	for _, tt := range tests {

		tt := tt

		t.Run(tt.name, func(t *testing.T) {

			t.Parallel()

			c := fake.NewFakeClient(tt.args.cfg)
			got, err := c.UpdateBackupConfig(tt.args.r)
			if diff := cmp.Diff(tt.error, err, EquateHTTPSErrors()); diff != "" {
				t.Errorf("FakeClient.UpdateBackupConfig() -want error, +got error:\n%s", diff)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("FakeClient.UpdateBackupConfig() -want, +got:\n%s", diff)
			}
		})
	}
}

func TestGetRestore(t *testing.T) {
	type args struct {
		r   *backupmanager.GetRestoreRequest
		cfg *fake.FakeClientConfiguration
	}
	tests := []struct {
		name  string
		args  args
		want  *backupmanager.GetRestoreResponse
		error error
	}{
		{
			name: "successRestoreRetrieved",
			args: args{
				cfg: &fake.FakeClientConfiguration{
					GetRestoreReaction: fake.GetRestoreReaction{
						Response: &backupmanager.GetRestoreResponse{
							RestoreID:   pointer.Int(1),
							BackupID:    pointer.Int(1),
							Status:      "done",
							TriggeredAt: "2023-04-11T08:52:48.209Z",
							FinishedAt:  "2023-04-11T08:53:16.411Z",
						},
					},
				},
				r: &backupmanager.GetRestoreRequest{
					InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
					RestoreID:  "1",
				},
			},
			want: &backupmanager.GetRestoreResponse{
				RestoreID:   pointer.Int(1),
				BackupID:    pointer.Int(1),
				Status:      "done",
				TriggeredAt: "2023-04-11T08:52:48.209Z",
				FinishedAt:  "2023-04-11T08:53:16.411Z",
			},
		},
		{
			name: "errorInstanceNotFound",
			args: args{
				r: &backupmanager.GetRestoreRequest{
					InstanceID: "non-existant-instance",
					RestoreID:  "non-existant-restore",
				},
				cfg: &fake.FakeClientConfiguration{
					GetRestoreReaction: fake.GetRestoreReaction{
						Error: backupmanager.HTTPStatusCodeError{
							StatusCode:   404,
							ErrorMessage: pointer.String("NotFound"),
							Description:  pointer.String("The instance \"non-existant-instance\" was not found"),
						},
					},
				},
			},
			error: backupmanager.HTTPStatusCodeError{
				StatusCode:   404,
				ErrorMessage: pointer.String("NotFound"),
				Description:  pointer.String("The instance \"non-existant-instance\" was not found"),
			},
		},
		{
			name: "errorUnexpectedAction",
			args: args{
				r: &backupmanager.GetRestoreRequest{
					InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
					RestoreID:  "1",
				},
			},
			error: fake.UnexpectedActionError(),
		},
	}
	for _, tt := range tests {

		tt := tt

		t.Run(tt.name, func(t *testing.T) {

			t.Parallel()

			c := fake.NewFakeClient(tt.args.cfg)
			got, err := c.GetRestore(tt.args.r)
			if diff := cmp.Diff(tt.error, err, EquateHTTPSErrors()); diff != "" {
				t.Errorf("FakeClient.GetRestore() -want error, +got error:\n%s", diff)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("FakeClient.GetRestore() -want, +got:\n%s", diff)
			}
		})
	}
}

func TestGetRestores(t *testing.T) {
	type args struct {
		r   *backupmanager.GetRestoresRequest
		cfg *fake.FakeClientConfiguration
	}
	tests := []struct {
		name  string
		args  args
		want  *backupmanager.GetRestoresResponse
		error error
	}{
		{
			name: "successRestoreRetrieved",
			args: args{
				cfg: &fake.FakeClientConfiguration{
					GetRestoresReaction: fake.GetRestoresReaction{
						Response: &backupmanager.GetRestoresResponse{
							Restores: []backupmanager.GetRestoreResponse{
								{
									RestoreID:   pointer.Int(1),
									BackupID:    pointer.Int(1),
									Status:      "done",
									TriggeredAt: "2023-04-11T08:52:48.209Z",
									FinishedAt:  "2023-04-11T08:53:16.411Z",
								},
							},
						},
					},
				},
				r: &backupmanager.GetRestoresRequest{
					InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
				},
			},
			want: &backupmanager.GetRestoresResponse{
				Restores: []backupmanager.GetRestoreResponse{
					{
						RestoreID:   pointer.Int(1),
						BackupID:    pointer.Int(1),
						Status:      "done",
						TriggeredAt: "2023-04-11T08:52:48.209Z",
						FinishedAt:  "2023-04-11T08:53:16.411Z",
					},
				},
			},
		},
		{
			name: "errorInstanceNotFound",
			args: args{
				r: &backupmanager.GetRestoresRequest{
					InstanceID: "non-existant-instance",
				},
				cfg: &fake.FakeClientConfiguration{
					GetRestoresReaction: fake.GetRestoresReaction{
						Error: backupmanager.HTTPStatusCodeError{
							StatusCode:   404,
							ErrorMessage: pointer.String("NotFound"),
							Description:  pointer.String("The instance \"non-existant-instance\" was not found"),
						},
					},
				},
			},
			error: backupmanager.HTTPStatusCodeError{
				StatusCode:   404,
				ErrorMessage: pointer.String("NotFound"),
				Description:  pointer.String("The instance \"non-existant-instance\" was not found"),
			},
		},
		{
			name: "errorUnexpectedAction",
			args: args{
				r: &backupmanager.GetRestoresRequest{
					InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
				},
			},
			error: fake.UnexpectedActionError(),
		}}
	for _, tt := range tests {

		tt := tt

		t.Run(tt.name, func(t *testing.T) {

			t.Parallel()

			c := fake.NewFakeClient(tt.args.cfg)
			got, err := c.GetRestores(tt.args.r)
			if diff := cmp.Diff(tt.error, err, EquateHTTPSErrors()); diff != "" {
				t.Errorf("FakeClient.GetRestores() -want error, +got error:\n%s", diff)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("FakeClient.GetRestores() -want, +got:\n%s", diff)
			}
		})
	}
}

func TestDeleteBackup(t *testing.T) {
	type args struct {
		r   *backupmanager.DeleteBackupRequest
		cfg *fake.FakeClientConfiguration
	}
	tests := []struct {
		name  string
		args  args
		want  *backupmanager.DeleteBackupResponse
		error error
	}{
		{
			name: "successBackupDeleted",
			args: args{
				cfg: &fake.FakeClientConfiguration{
					DeleteBackupReaction: fake.DeleteBackupReaction{
						Response: &backupmanager.DeleteBackupResponse{
							Message: pointer.String("Backup successfully deleted"),
						},
					},
				},
				r: &backupmanager.DeleteBackupRequest{
					InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
					BackupID:   pointer.Int(1),
				},
			},
			want: &backupmanager.DeleteBackupResponse{
				Message: pointer.String("Backup successfully deleted"),
			},
		},
		{
			name: "errorInstanceNotFound",
			args: args{
				r: &backupmanager.DeleteBackupRequest{
					InstanceID: "non-existant-instance",
					BackupID:   pointer.Int(1),
				},
				cfg: &fake.FakeClientConfiguration{
					DeleteBackupReaction: fake.DeleteBackupReaction{
						Error: backupmanager.HTTPStatusCodeError{
							StatusCode:   404,
							ErrorMessage: pointer.String("NotFound"),
							Description:  pointer.String("The instance \"non-existant-instance\" was not found"),
						},
					},
				},
			},
			error: backupmanager.HTTPStatusCodeError{
				StatusCode:   404,
				ErrorMessage: pointer.String("NotFound"),
				Description:  pointer.String("The instance \"non-existant-instance\" was not found"),
			},
		},
		{
			name: "errorUnexpectedAction",
			args: args{
				r: &backupmanager.DeleteBackupRequest{
					InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
					BackupID:   pointer.Int(1),
				},
			},
			error: fake.UnexpectedActionError(),
		},
	}
	for _, tt := range tests {

		tt := tt

		t.Run(tt.name, func(t *testing.T) {

			t.Parallel()

			c := fake.NewFakeClient(tt.args.cfg)
			got, err := c.DeleteBackup(tt.args.r)
			if diff := cmp.Diff(tt.error, err, EquateHTTPSErrors()); diff != "" {
				t.Errorf("FakeClient.DeleteBackup() -want error, +got error:\n%s", diff)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("FakeClient.DeleteBackup() -want, +got:\n%s", diff)
			}
		})
	}
}

func TestActions(t *testing.T) {

	cfg := &fake.FakeClientConfiguration{}

	want := []fake.Action{
		{
			Type: "CreateBackup",
			Request: &backupmanager.CreateBackupRequest{
				InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
			},
		},
		{
			Type: "CreateRestore",
			Request: &backupmanager.CreateRestoreRequest{
				InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
				BackupID:   "761e6d41-49c2-4f80-805f-7a03de9fe798",
			}},
		{
			Type: "DeleteBackup",
			Request: &backupmanager.DeleteBackupRequest{
				InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
				BackupID:   pointer.Int(1),
			},
		},
		{
			Type: "GetBackup",
			Request: &backupmanager.GetBackupRequest{
				InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
				BackupID:   "761e6d41-49c2-4f80-805f-7a03de9fe798",
			},
		},
		{
			Type: "GetBackups",
			Request: &backupmanager.GetBackupsRequest{
				InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
			},
		},
		{
			Type: "GetInstanceConfig",
			Request: &backupmanager.GetInstanceConfigRequest{
				InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
			},
		},
		{
			Type: "GetRestore",
			Request: &backupmanager.GetRestoreRequest{
				InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
				RestoreID:  "21284603-daf0-4ca4-8f2c-cad385ac1740",
			},
		},
		{
			Type: "GetRestores",
			Request: &backupmanager.GetRestoresRequest{
				InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
			},
		},
		{
			Type: "UpdateBackupConfig",
			Request: &backupmanager.UpdateBackupConfigRequest{
				InstanceID:               "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
				EncryptionKey:            pointer.String("test"),
				ExcludeFromAutoBackup:    pointer.Bool(true),
				CredentialsUpdatedByUser: pointer.Bool(true),
			},
		},
	}

	c := fake.NewFakeClient(cfg)

	c.CreateBackup(&backupmanager.CreateBackupRequest{
		InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
	})
	c.CreateRestore(&backupmanager.CreateRestoreRequest{
		InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
		BackupID:   "761e6d41-49c2-4f80-805f-7a03de9fe798",
	})
	c.DeleteBackup(&backupmanager.DeleteBackupRequest{
		InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
		BackupID:   pointer.Int(1),
	})
	c.GetBackup(&backupmanager.GetBackupRequest{
		InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
		BackupID:   "761e6d41-49c2-4f80-805f-7a03de9fe798",
	})
	c.GetBackups(&backupmanager.GetBackupsRequest{
		InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
	})
	c.GetInstanceConfig(&backupmanager.GetInstanceConfigRequest{
		InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
	})
	c.GetRestore(&backupmanager.GetRestoreRequest{
		InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
		RestoreID:  "21284603-daf0-4ca4-8f2c-cad385ac1740",
	})
	c.GetRestores(&backupmanager.GetRestoresRequest{
		InstanceID: "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
	})
	c.UpdateBackupConfig(&backupmanager.UpdateBackupConfigRequest{
		InstanceID:               "0b0001f9-38a2-4248-9ef8-5cbf27d78e8b",
		EncryptionKey:            pointer.String("test"),
		ExcludeFromAutoBackup:    pointer.Bool(true),
		CredentialsUpdatedByUser: pointer.Bool(true),
	})

	got := c.Actions()

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("FakeClient.Actions() +want, -got:\n%s", diff)
	}
}

func EquateHTTPSErrors() cmp.Option {
	return cmp.FilterValues(areConcreteErrors, cmp.Comparer(compareHTTPSErrors))
}

func areConcreteErrors(x, y interface{}) bool {
	_, ok1 := x.(backupmanager.HTTPStatusCodeError)
	_, ok2 := y.(backupmanager.HTTPStatusCodeError)
	return ok1 && ok2
}

func compareHTTPSErrors(x, y interface{}) bool {
	xe := x.(backupmanager.HTTPStatusCodeError)
	ye := y.(backupmanager.HTTPStatusCodeError)

	if xe.Description != ye.Description &&
		(xe.Description == nil || ye.Description == nil) &&
		*xe.Description != *ye.Description {
		return false
	}

	if xe.ErrorMessage != ye.ErrorMessage &&
		(xe.ErrorMessage == nil || ye.ErrorMessage == nil) &&
		*xe.ErrorMessage != *ye.ErrorMessage {
		return false

	}

	if xe.StatusCode != ye.StatusCode {
		return false
	}

	if xe.ResponseError != ye.ResponseError {
		return false
	}

	return true
}
