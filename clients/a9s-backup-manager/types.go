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

package backupmanager

// CreateBackupRequest represents a request to create a new backup for a
// data service instance.
type CreateBackupRequest struct {
	// InstanceID is the ID of the data service instance from which the
	// backup should be taken.
	InstanceID string `json:"instance_id"`
}

// CreateBackupResponse is sent in response to a create backup call.
type CreateBackupResponse struct {
	// BackupID represents the ID of the backup.
	BackupID *int `json:"id"`
	// Message represents the response message from the backup
	// manager api.
	Message *string `json:"message,omitempty"`
}

type GetBackupsRequest struct {
	// InstanceID is the ID of the data service instance from which the
	// backups should be fetched.
	InstanceID string `json:"instance_id"`
}

type GetBackupsResponse struct {
	Backups []GetBackupResponse
}

type GetBackupRequest struct {
	// InstanceID is the ID of the data service instance from which the
	// backups should be fetched.
	InstanceID string `json:"instance_id"`

	// BackupID is the ID of the backup which should be polled.
	BackupID string `json:"backup_id"`
}

type GetBackupResponse struct {
	BackupID     *int   `json:"id"`
	Size         int    `json:"size"`
	Status       string `json:"status"`
	TriggeredAt  string `json:"triggered_at"`
	FinishedAt   string `json:"finished_at"`
	Downloadable bool   `json:"downloadable"`
}

// GetInstanceConfigRequest represents a request to retrieve the config of
// a backup from the backup manager
type GetInstanceConfigRequest struct {
	// InstanceID is the ID of the data service instance from which the
	// config is requested.
	InstanceID string `json:"instance_id"`
}

// GetInstanceConfigResponse is sent in response to a get instance config
// call.
type GetInstanceConfigResponse struct {
	// The minimum amount of backups per instance. When the amount of
	// backups exceeds this number, the older backups will be deleted. The
	// minimum value is 0 and the default value is defined in
	// {Rails.root}/config/config.yml at the key
	// 'min_backups_per_instance'.
	MinBackupCount *int `json:"min_backup_count" validate:"gte=0"`
	// The retention time is measured in days and defines a time range
	// after which backups can be deleted. The minimum value is 0 and the
	// default value is defined in {Rails.root}/config/config.yml at the
	// key 'min_backup_age'.
	RetentionTime *int `json:"retention_time" validate:"gte=0"`
	// The minimum length of the backups' encryption key. Default is 8.
	MinEncryptionKeyLength *int `json:"min_encryption_key_length" validate:"gte=0"`
	// States whether this instance will be excluded from scheduled
	// backups. Default is false.
	ExcludeFromAutoBackup *bool `json:"exclude_from_auto_backup"`
	// The type of the backup. Currently only supports "postgresql_wal".
	BackupType *string `json:"backup_type"`
}

// UpdateBackupConfigRequest represents a request to update the backup config for a
// data service instance.
type UpdateBackupConfigRequest struct {
	// InstanceID is the ID of the data service instance.
	InstanceID string `json:"instance_id"`
	// EncryptionKey is the key used to encrypt backups.
	EncryptionKey *string `json:"encryption_key"`
	// ExcludeFromAutoBackup indicates whether the data service instance will be
	// excluded from the backup schedule.
	// https://docs.anynines.com/docs/35.0.0/platform-operator/a9s-backup-service/a9s-po-backup-service-backup-process#regular-backup-cycle
	ExcludeFromAutoBackup *bool `json:"exclude_from_auto_backup"`
	// CredentialsUpdatedByUser indicates whether credentials are updated by the user.
	CredentialsUpdatedByUser *bool `json:"credentials_updated_by_user"`
}

// UpdateBackupConfigResponse is sent in response to a update backup config call.
type UpdateBackupConfigResponse struct {
	// Message represents the response message from the backup-manager api.
	Message *string `json:"message"`
}

type updateBackupConfigRequestBody struct {
	EncryptionKey            *string `json:"encryption_key,omitempty"`
	ExcludeFromAutoBackup    *bool   `json:"exclude_from_auto_backup,omitempty"`
	CredentialsUpdatedByUser *bool   `json:"credentials_updated_by_user,omitempty"`
}

type CreateRestoreRequest struct {
	// InstanceID is the ID of the data service instance from which the
	// backup should be restored.
	InstanceID string `json:"instance_id"`

	// BackupID is the ID used by the Backup Manager to
	// refer to the specific backup.
	BackupID string `json:"backup_id"`
}

type CreateRestoreResponse struct {
	// RestoreID represents the ID of the restore which will be restored.
	RestoreID *int `json:"id"`
}

type GetRestoreRequest struct {
	// InstanceID is the ID of the data service instance from which the
	// restore should be fetched.
	InstanceID string `json:"instance_id"`

	// RestoreID is the ID of the restore which should be polled.
	RestoreID string `json:"restore_id"`
}

type GetRestoreResponse struct {
	RestoreID   *int   `json:"id"`
	BackupID    *int   `json:"backup_id"`
	Status      string `json:"status"`
	TriggeredAt string `json:"triggered_at"`
	FinishedAt  string `json:"finished_at"`
}

type GetRestoresRequest struct {
	// InstanceID is the ID of the data service instance from which the
	// restore should be fetched.
	InstanceID string `json:"instance_id"`
}

type GetRestoresResponse struct {
	Restores []GetRestoreResponse
}

// DeleteBackupRequest represents a request to delete a given backup of a
// data service instance.
type DeleteBackupRequest struct {
	// InstanceID is the ID of the data service instance from which the
	// backup was taken.
	InstanceID string `json:"instance_id"`
	// BackupID represents the ID of the backup to delete.
	BackupID *int `json:"backup_id"`
}

// DeleteBackupResponse is sent in response to a delete backup call.
type DeleteBackupResponse struct {
	// Message represents the response message from the backup
	// manager api.
	Message *string `json:"message,omitempty"`
}
