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

package fake

import (
	"sync"

	backupmanager "github.com/anynines/klutch/clients/a9s-backup-manager"
	"k8s.io/utils/pointer"
)

/*
	The architecture of this fake client is as follows:
	For every function that the real Client interface contains we provide our own implementation and a struct that has the same name with the suffix "Reaction", i.e. CreateBackupReaction. Every FakeClient has one field of every reaction type that has the same name its type, e.g. the FakeClient's field of the type CreateBackupReaction is also called CreateBackupReaction.

	These structs always consist of a Response field, a Request field, an Error field and a React method.
	The React method fills the Request field with the request that was handed to the method on call and then returns the Response and Error fields belonging to its struct.

	Whenever one of the methods that are defined in the backupmanager.Client interface is called by a user of our fake client four things happen:
	1. The method requests the clients Mutex lock
	2. It logs the method call in the "actions" field of the client
	3. It executes the React method of its Reaction type
	4. It releases its lock

	If a backupmanager.Client method is called whose reaction field in the FakeClient is not filled then an UnexpectedAction-Error is returned.

	So in order to use this fake client one needs to create a variable of the type FakeClientConfiguration, then fill the reaction fields of the configuration with the responses and errors you want the FakeClient to return and finally give the FakeClientConfiguration to the NewFakeClient method.
*/

// NewFakeClient returns a new fake Client with the given
// FakeClientConfiguration.
func NewFakeClient(config *FakeClientConfiguration) *Client {
	if config == nil {
		return &Client{}
	}
	return &Client{
		CreateBackupReaction:       config.CreateBackupReaction,
		CreateRestoreReaction:      config.CreateRestoreReaction,
		DeleteBackupReaction:       config.DeleteBackupReaction,
		GetBackupReaction:          config.GetBackupReaction,
		GetBackupsReaction:         config.GetBackupsReaction,
		GetInstanceConfigReaction:  config.GetInstanceConfigReaction,
		GetRestoreReaction:         config.GetRestoreReaction,
		GetRestoresReaction:        config.GetRestoresReaction,
		UpdateBackupConfigReaction: config.UpdateBackupConfigReaction,
	}
}

// FakeClientConfiguration models the configuration of a FakeClient.
type FakeClientConfiguration struct {
	CreateBackupReaction       CreateBackupReaction
	CreateRestoreReaction      CreateRestoreReaction
	DeleteBackupReaction       DeleteBackupReaction
	GetBackupReaction          GetBackupReaction
	GetBackupsReaction         GetBackupsReaction
	GetInstanceConfigReaction  GetInstanceConfigReaction
	GetRestoreReaction         GetRestoreReaction
	GetRestoresReaction        GetRestoresReaction
	UpdateBackupConfigReaction UpdateBackupConfigReaction
}

// Action is a record of a method call on the FakeClient.
type Action struct {
	Type    ActionType
	Request interface{}
}

// ActionType is a typedef over the set of actions that can be taken on a
// FakeClient.
type ActionType string

// These are the set of actions that can be taken on a FakeClient.
const (
	CreateBackup       ActionType = "CreateBackup"
	CreateRestore      ActionType = "CreateRestore"
	DeleteBackup       ActionType = "DeleteBackup"
	GetBackup          ActionType = "GetBackup"
	GetBackups         ActionType = "GetBackups"
	GetInstanceConfig  ActionType = "GetInstanceConfig"
	GetRestore         ActionType = "GetRestore"
	GetRestores        ActionType = "GetRestores"
	UpdateBackupConfig ActionType = "UpdateBackupConfig"
)

// Client is a fake implementation of the backupmanager.Client interface. It records
// the actions that are taken on it and runs the appropriate reaction to those
// actions. If an action for which there is no reaction specified occurs, it
// returns an error.  Client is threadsafe.
type Client struct {
	CreateBackupReaction       CreateBackupReaction
	CreateRestoreReaction      CreateRestoreReaction
	DeleteBackupReaction       DeleteBackupReaction
	GetBackupReaction          GetBackupReaction
	GetBackupsReaction         GetBackupsReaction
	GetInstanceConfigReaction  GetInstanceConfigReaction
	GetRestoreReaction         GetRestoreReaction
	GetRestoresReaction        GetRestoresReaction
	UpdateBackupConfigReaction UpdateBackupConfigReaction

	sync.Mutex
	actions []Action
}

var _ backupmanager.Client = &Client{}

// Actions is a method defined on FakeClient that returns the actions taken on
// it. This method returns a copy of the client's actions array.
func (c *Client) Actions() []Action {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	actionCopy := make([]Action, len(c.actions))
	copy(actionCopy, c.actions)
	return actionCopy
}

// CreateBackupReaction defines the reaction to CreateBackup requests.
type CreateBackupReaction struct {
	Request  *backupmanager.CreateBackupRequest
	Response *backupmanager.CreateBackupResponse
	Error    error
}

func (r *CreateBackupReaction) React(req *backupmanager.CreateBackupRequest) (*backupmanager.CreateBackupResponse, error) {
	r.Request = req
	return r.Response, r.Error
}

// CreateBackup implements the Client.CreateBackup method for the FakeClient.
func (c *Client) CreateBackup(r *backupmanager.CreateBackupRequest) (*backupmanager.CreateBackupResponse, error) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	c.actions = append(c.actions, Action{Type: CreateBackup, Request: r})

	if c.CreateBackupReaction != (CreateBackupReaction{}) {
		return c.CreateBackupReaction.React(r)
	}

	return nil, UnexpectedActionError()
}

// CreateRestoreReaction defines the reaction to CreateRestore requests.
type CreateRestoreReaction struct {
	Request  *backupmanager.CreateRestoreRequest
	Response *backupmanager.CreateRestoreResponse
	Error    error
}

func (r *CreateRestoreReaction) React(req *backupmanager.CreateRestoreRequest) (*backupmanager.CreateRestoreResponse, error) {
	r.Request = req
	return r.Response, r.Error
}

// CreateRestore requests that a new restore of a backup of a data service
// to be created and returns information about the restore or an error.
// CreateRestore does a POST on the restore managers endpoint for the
// requested instance ID and backup ID (/instance/{instance-id}/backups/{backup-id}/restore)
func (c *Client) CreateRestore(r *backupmanager.CreateRestoreRequest) (*backupmanager.CreateRestoreResponse, error) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	c.actions = append(c.actions, Action{Type: CreateRestore, Request: r})

	if c.CreateRestoreReaction != (CreateRestoreReaction{}) {
		return c.CreateRestoreReaction.React(r)
	}

	return nil, UnexpectedActionError()
}

// GetBackupReaction defines the reaction to GetBackup requests.
type GetBackupReaction struct {
	Request  *backupmanager.GetBackupRequest
	Response *backupmanager.GetBackupResponse
	Error    error
}

func (r *GetBackupReaction) React(req *backupmanager.GetBackupRequest) (*backupmanager.GetBackupResponse, error) {
	r.Request = req
	return r.Response, r.Error
}

// GetBackup retrieves information about a specific backup for a specific
// instance from the backup manager or returns an error. GetBackup does a
// GET on the backup managers endpoint for the requested instance ID and
// backup ID (/instances/{instance-id}/backups/{backup-id}).
func (c *Client) GetBackup(r *backupmanager.GetBackupRequest) (*backupmanager.GetBackupResponse, error) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	c.actions = append(c.actions, Action{Type: GetBackup, Request: r})

	if c.GetBackupReaction != (GetBackupReaction{}) {
		return c.GetBackupReaction.React(r)
	}

	return nil, UnexpectedActionError()
}

// GetBackupsReaction defines the reaction to GetBackups requests.
type GetBackupsReaction struct {
	Request  *backupmanager.GetBackupsRequest
	Response *backupmanager.GetBackupsResponse
	Error    error
}

func (r *GetBackupsReaction) React() (*backupmanager.GetBackupsResponse, error) {
	return r.Response, r.Error
}

// GetBackups retrieves information about all existing backups for a
// specific instance from the backup manager or returns an error.
// GetBackups does a GET on the backup managers endpoint for the requested
// instance ID (/instances/{instance-id}/backups).
func (c *Client) GetBackups(r *backupmanager.GetBackupsRequest) (*backupmanager.GetBackupsResponse, error) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	c.actions = append(c.actions, Action{Type: GetBackups, Request: r})

	if c.GetBackupsReaction != (GetBackupsReaction{}) {
		return c.GetBackupsReaction.React()
	}

	return nil, UnexpectedActionError()
}

// GetInstanceConfigReactionv defines the reaction to GetInstanceConfig requests.
type GetInstanceConfigReaction struct {
	Request  *backupmanager.GetInstanceConfigRequest
	Response *backupmanager.GetInstanceConfigResponse
	Error    error
}

func (r *GetInstanceConfigReaction) React() (*backupmanager.GetInstanceConfigResponse, error) {
	return r.Response, r.Error
}

// GetInstanceConfig retrieves the configuration of a specific
// data service instance from the backup manager or returns an error.
// GetInstanceConfig does a GET on the backup manager's endpoint for
// the requested instance ID (/instances/{instance-id}/config).
func (c *Client) GetInstanceConfig(r *backupmanager.GetInstanceConfigRequest) (*backupmanager.GetInstanceConfigResponse, error) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	c.actions = append(c.actions, Action{Type: GetInstanceConfig, Request: r})

	if c.GetInstanceConfigReaction != (GetInstanceConfigReaction{}) {
		return c.GetInstanceConfigReaction.React()
	}

	return nil, UnexpectedActionError()
}

// UpdateBackupConfigReaction defines the reaction to UpdateBackupConfig requests.
type UpdateBackupConfigReaction struct {
	Request  *backupmanager.UpdateBackupConfigRequest
	Response *backupmanager.UpdateBackupConfigResponse
	Error    error
}

func (r *UpdateBackupConfigReaction) React(req *backupmanager.UpdateBackupConfigRequest) (*backupmanager.UpdateBackupConfigResponse, error) {
	r.Request = req
	return r.Response, r.Error
}

// UpdateBackupConfig requests that the backup config of a data service instance
// is updated and returns information about the update or an error.
// UpdateBackupConfig does a PUT on the backup managers endpoint for the
// requested instance ID (/instances/{instance-id}).
func (c *Client) UpdateBackupConfig(r *backupmanager.UpdateBackupConfigRequest) (*backupmanager.UpdateBackupConfigResponse, error) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	c.actions = append(c.actions, Action{Type: UpdateBackupConfig, Request: r})

	if c.UpdateBackupConfigReaction != (UpdateBackupConfigReaction{}) {
		return c.UpdateBackupConfigReaction.React(r)
	}

	return nil, UnexpectedActionError()
}

// GetRestoreReaction defines the reaction to GetRestore requests.
type GetRestoreReaction struct {
	Request  *backupmanager.GetRestoreRequest
	Response *backupmanager.GetRestoreResponse
	Error    error
}

func (r *GetRestoreReaction) React() (*backupmanager.GetRestoreResponse, error) {
	return r.Response, r.Error
}

// GetRestore retrieves information about a specific restore that has been
// performed on a specific instance from the backup manager or returns an
// error. GetRestore does a GET on the backup managers endpoint for the
// requested instance ID and restore ID
// (/instances/{instance-id}/restores/{restore-id}).
func (c *Client) GetRestore(r *backupmanager.GetRestoreRequest) (*backupmanager.GetRestoreResponse, error) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	c.actions = append(c.actions, Action{Type: GetRestore, Request: r})

	if c.GetRestoreReaction != (GetRestoreReaction{}) {
		return c.GetRestoreReaction.React()
	}

	return nil, UnexpectedActionError()
}

// GetRestoresReaction defines the reaction to GetRestores requests.
type GetRestoresReaction struct {
	Request  *backupmanager.GetRestoresRequest
	Response *backupmanager.GetRestoresResponse
	Error    error
}

func (r *GetRestoresReaction) React() (*backupmanager.GetRestoresResponse, error) {
	return r.Response, r.Error
}

// GetRestores retrieves information about all previously performed restores
// for a specific instance from the backup manager or returns an error.
// GetRestores does a GET on the backup managers endpoint for the requested
// instance ID (/instances/{instance-id}/restores).
func (c *Client) GetRestores(r *backupmanager.GetRestoresRequest) (*backupmanager.GetRestoresResponse, error) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	c.actions = append(c.actions, Action{Type: GetRestores, Request: r})

	if c.GetRestoresReaction != (GetRestoresReaction{}) {
		return c.GetRestoresReaction.React()
	}

	return nil, UnexpectedActionError()
}

// DeleteBackupReaction defines the reaction to DeleteBackup requests.
type DeleteBackupReaction struct {
	Request  *backupmanager.DeleteBackupRequest
	Response *backupmanager.DeleteBackupResponse
	Error    error
}

func (r *DeleteBackupReaction) React(req *backupmanager.DeleteBackupRequest) (*backupmanager.DeleteBackupResponse, error) {
	r.Request = req
	return r.Response, r.Error
}

// DeleteBackup requests that a backup of a data service be deleted
// and returns a confirmation of the deletion or an error.
// This includes the metadata of the backup in the backup manager as
// well as the actual file containing the backup.
// DeleteBackup does a POST on the backup managers endpoint for the
// requested instance ID and the requested backup id
// (/instances/{instance-id}/backups/{backup-id})
func (c *Client) DeleteBackup(r *backupmanager.DeleteBackupRequest) (*backupmanager.DeleteBackupResponse, error) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	c.actions = append(c.actions, Action{Type: DeleteBackup, Request: r})

	if c.DeleteBackupReaction != (DeleteBackupReaction{}) {
		return c.DeleteBackupReaction.React(r)
	}

	return nil, UnexpectedActionError()
}

// UnexpectedActionError returns an error message when an action is not found
// in the FakeClient's action array.
func UnexpectedActionError() error {
	return backupmanager.HTTPStatusCodeError{ErrorMessage: pointer.String("unexpected action")}
}
