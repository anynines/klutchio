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

// AuthConfig is a union-type representing the possible auth configurations a
// client may use to authenticate to the backup manager. Currently, only basic auth is
// supported.
type AuthConfig struct {
	BasicAuthConfig *BasicAuthConfig
}

// BasicAuthConfig represents a set of basic auth credentials.
type BasicAuthConfig struct {
	// Username is the basic auth username.
	Username string
	// Password is the basic auth password.
	Password string
}

// ClientConfiguration represents the configuration of a Client.
type ClientConfiguration struct {
	// Name is the name to use for this client in log messages.
	Name string
	// URL is the URL to use to contact the backup manager.
	URL string
	// AuthInfo is the auth configuration the client should use to authenticate
	// to the backup manager.
	AuthConfig *AuthConfig
	// TimeoutSeconds is the length of the timeout of any request to the
	// backup manager, in seconds.
	TimeoutSeconds int
	// Verbose is whether the client will log to klog.
	Verbose bool
}

// DefaultClientConfiguration returns a default ClientConfiguration:
//   - 60 second timeout
func DefaultClientConfiguration() *ClientConfiguration {
	return &ClientConfiguration{
		TimeoutSeconds: 60,
	}
}

// Client defines the interface to the a9s backup-manager client.
//
// 1. Create a new backup of a data service instance with the CreateBackup method
// 2. Update the backup config of a data service instance with UpdateBackupConfig method
type Client interface {
	// CreateBackup requests that a new backup of a data service be
	// created and returns information about the backup or an error.
	// CreateBackup does a POST on the backup managers endpoint for the
	// requested instance ID (/instances/{instance-id}/backups).
	CreateBackup(r *CreateBackupRequest) (*CreateBackupResponse, error)

	// CreateRestore requests that a new restore of a backup of a data service
	// to be created and returns information about the restore or an error.
	// CreateRestore does a POST on the restore managers endpoint for the
	// requested instance ID and backup ID (/instance/{instance-id}/backups/{backup-id}/restore)
	CreateRestore(r *CreateRestoreRequest) (*CreateRestoreResponse, error)

	// GetBackup retrieves information about a specific backup for a specific
	// instance from the backup manager or returns an error. GetBackup does a
	// GET on the backup managers endpoint for the requested instance ID and
	// backup ID (/instances/{instance-id}/backups/{backup-id}).
	GetBackup(r *GetBackupRequest) (*GetBackupResponse, error)

	// GetBackups retrieves information about all existing backups for a
	// specific instance from the backup manager or returns an error.
	// GetBackups does a GET on the backup managers endpoint for the requested
	// instance ID (/instances/{instance-id}/backups).
	GetBackups(r *GetBackupsRequest) (*GetBackupsResponse, error)

	// GetInstanceConfig retrieves the configuration of a specific
	// data service instance from the backup manager or returns an error.
	// GetInstanceConfig does a GET on the backup manager's endpoint for
	// the requested instance ID (/instances/{instance-id}/config).
	GetInstanceConfig(r *GetInstanceConfigRequest) (*GetInstanceConfigResponse, error)

	// UpdateBackupConfig requests that the backup config of a data service instance
	// is updated and returns information about the update or an error.
	// UpdateBackupConfig does a PUT on the backup managers endpoint for the
	// requested instance ID (/instances/{instance-id}).
	UpdateBackupConfig(r *UpdateBackupConfigRequest) (*UpdateBackupConfigResponse, error)

	// GetRestore retrieves information about a specific restore that has been
	// performed on a specific instance from the backup manager or returns an
	// error. GetRestore does a GET on the backup managers endpoint for the
	// requested instance ID and restore ID
	// (/instances/{instance-id}/restores/{restore-id}).
	GetRestore(r *GetRestoreRequest) (*GetRestoreResponse, error)

	// GetRestores retrieves information about all previously performed restores
	// for a specific instance from the backup manager or returns an error.
	// GetRestores does a GET on the backup managers endpoint for the requested
	// instance ID (/instances/{instance-id}/restores).
	GetRestores(r *GetRestoresRequest) (*GetRestoresResponse, error)

	// DeleteBackup requests that a backup of a data service be deleted
	// and returns a confirmation of the deletion or an error.
	// This includes the metadata of the backup in the backup manager as
	// well as the actual file containing the backup.
	// DeleteBackup does a POST on the backup managers endpoint for the
	// requested instance ID and the requested backup id
	// (/instances/{instance-id}/backups/{backup-id})
	DeleteBackup(r *DeleteBackupRequest) (*DeleteBackupResponse, error)
}

// CreateFunc allows control over which implementation of a Client is
// returned. Users of the Client interface may need to create clients for
// multiple backup managers in a way that makes normal dependency injection
// prohibitive. In order to make such code testable, users of the API can
// inject a CreateFunc, and use the CreateFunc from the fake package in tests.
type CreateFunc func(*ClientConfiguration) (Client, error)
