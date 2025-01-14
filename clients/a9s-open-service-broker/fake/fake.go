package fake

import (
	"errors"
	"net/http"
	"sync"

	v2 "github.com/anynines/klutchio/clients/a9s-open-service-broker"
)

// NewFakeClientFunc returns a v2.CreateFunc that returns a FakeClient with
// the given FakeClientConfiguration.  It is useful for injecting the
// FakeClient in code that uses the v2.CreateFunc interface.
func NewFakeClientFunc(config FakeClientConfiguration) v2.CreateFunc {
	return func(_ *v2.ClientConfiguration) (v2.Client, error) {
		return NewFakeClient(config), nil
	}
}

// ReturnFakeClientFunc returns a v2.CreateFunc that returns the given
// FakeClient.
func ReturnFakeClientFunc(c *FakeClient) v2.CreateFunc {
	return func(_ *v2.ClientConfiguration) (v2.Client, error) {
		return c, nil
	}
}

// NewFakeClient returns a new fake Client with the given
// FakeClientConfiguration.
func NewFakeClient(config FakeClientConfiguration) *FakeClient {
	return &FakeClient{
		CatalogReaction:                  config.CatalogReaction,
		ProvisionReaction:                config.ProvisionReaction,
		UpdateInstanceReaction:           config.UpdateInstanceReaction,
		DeprovisionReaction:              config.DeprovisionReaction,
		GetInstanceReaction:              config.GetInstanceReaction,
		GetInstancesReaction:             config.GetInstancesReaction,
		PollLastOperationReaction:        config.PollLastOperationReaction,
		PollLastOperationReactions:       config.PollLastOperationReactions,
		PollBindingLastOperationReaction: config.PollBindingLastOperationReaction,
		BindReaction:                     config.BindReaction,
		UnbindReaction:                   config.UnbindReaction,
		GetBindingReaction:               config.GetBindingReaction,
		CheckAvailabilityReaction:        config.CheckAvailabilityReaction,
		GetOperationReaction:             config.GetOperationReaction,
	}
}

// FakeClientConfiguration models the configuration of a FakeClient.
type FakeClientConfiguration struct {
	CatalogReaction                  CatalogReactionInterface
	ProvisionReaction                ProvisionReactionInterface
	UpdateInstanceReaction           UpdateInstanceReactionInterface
	DeprovisionReaction              DeprovisionReactionInterface
	GetInstanceReaction              GetInstanceReactionInterface
	GetInstancesReaction             GetInstancesReactionInterface
	PollLastOperationReaction        PollLastOperationReactionInterface
	PollLastOperationReactions       map[v2.OperationKey]*PollLastOperationReaction
	PollBindingLastOperationReaction PollBindingLastOperationReactionInterface
	BindReaction                     BindReactionInterface
	UnbindReaction                   UnbindReactionInterface
	GetBindingReaction               GetBindingReactionInterface
	CheckAvailabilityReaction        CheckAvailabilityReactionInterface
	GetOperationReaction             GetOperationReactionInterface
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
	GetCatalog               ActionType = "GetCatalog"
	ProvisionInstance        ActionType = "ProvisionInstance"
	UpdateInstance           ActionType = "UpdateInstance"
	DeprovisionInstance      ActionType = "DeprovisionInstance"
	GetInstance              ActionType = "GetInstance"
	GetInstances             ActionType = "GetInstances"
	PollLastOperation        ActionType = "PollLastOperation"
	PollBindingLastOperation ActionType = "PollBindingLastOperation"
	Bind                     ActionType = "Bind"
	Unbind                   ActionType = "Unbind"
	GetBinding               ActionType = "GetBinding"
	CheckAvailability        ActionType = "CheckAvailability"
	GetOperation             ActionType = "GetOperation"
)

// FakeClient is a fake implementation of the v2.Client interface. It records
// the actions that are taken on it and runs the appropriate reaction to those
// actions. If an action for which there is no reaction specified occurs, it
// returns an error.  FakeClient is threadsafe.
type FakeClient struct {
	CatalogReaction                  CatalogReactionInterface
	ProvisionReaction                ProvisionReactionInterface
	UpdateInstanceReaction           UpdateInstanceReactionInterface
	DeprovisionReaction              DeprovisionReactionInterface
	GetInstanceReaction              GetInstanceReactionInterface
	GetInstancesReaction             GetInstancesReactionInterface
	PollLastOperationReaction        PollLastOperationReactionInterface
	PollLastOperationReactions       map[v2.OperationKey]*PollLastOperationReaction
	PollBindingLastOperationReaction PollBindingLastOperationReactionInterface
	BindReaction                     BindReactionInterface
	UnbindReaction                   UnbindReactionInterface
	GetBindingReaction               GetBindingReactionInterface
	CheckAvailabilityReaction        CheckAvailabilityReactionInterface
	GetOperationReaction             GetOperationReactionInterface

	sync.Mutex
	actions []Action
}

var _ v2.Client = &FakeClient{}

// Actions is a method defined on FakeClient that returns the actions taken on
// it.
func (c *FakeClient) Actions() []Action {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	return c.actions
}

// GetCatalog implements the Client.GetCatalog method for the FakeClient.
func (c *FakeClient) GetCatalog() (*v2.CatalogResponse, error) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	c.actions = append(c.actions, Action{Type: GetCatalog})

	if c.CatalogReaction != nil {
		return c.CatalogReaction.React()
	}

	return nil, UnexpectedActionError()
}

// ProvisionInstance implements the Client.ProvisionRequest method for the
// FakeClient.
func (c *FakeClient) ProvisionInstance(r *v2.ProvisionRequest) (*v2.ProvisionResponse, error) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	c.actions = append(c.actions, Action{ProvisionInstance, r})

	if c.ProvisionReaction != nil {
		return c.ProvisionReaction.React(r)
	}

	return nil, UnexpectedActionError()
}

// UpdateInstance implements the Client.UpdateInstance method for the
// FakeClient.
func (c *FakeClient) UpdateInstance(r *v2.UpdateInstanceRequest) (*v2.UpdateInstanceResponse, error) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	c.actions = append(c.actions, Action{UpdateInstance, r})

	if c.UpdateInstanceReaction != nil {
		return c.UpdateInstanceReaction.React(r)
	}

	return nil, UnexpectedActionError()
}

// DeprovisionInstance implements the Client.DeprovisionInstance method on the
// FakeClient.
func (c *FakeClient) DeprovisionInstance(r *v2.DeprovisionRequest) (*v2.DeprovisionResponse, error) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	c.actions = append(c.actions, Action{DeprovisionInstance, r})

	if c.DeprovisionReaction != nil {
		return c.DeprovisionReaction.React(r)
	}

	return nil, UnexpectedActionError()
}

// GetInstance implements the Client.GetInstance method for the FakeClient.
func (c *FakeClient) GetInstance(*v2.GetInstanceRequest) (*v2.GetInstanceResponse, error) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	c.actions = append(c.actions, Action{Type: GetInstance})

	if c.GetInstanceReaction != nil {
		return c.GetInstanceReaction.React()
	}

	return nil, UnexpectedActionError()
}

// GetInstances implements the Client.GetInstances method for the FakeClient.
func (c *FakeClient) GetInstances() (*v2.GetInstancesResponse, error) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	c.actions = append(c.actions, Action{Type: GetInstances})

	if c.GetInstancesReaction != nil {
		return c.GetInstancesReaction.React()
	}

	return nil, UnexpectedActionError()
}

// PollLastOperation implements the Client.PollLastOperation method on the
// FakeClient.
func (c *FakeClient) PollLastOperation(r *v2.LastOperationRequest) (*v2.LastOperationResponse, error) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	c.actions = append(c.actions, Action{PollLastOperation, r})

	if r.OperationKey != nil && c.PollLastOperationReactions[*r.OperationKey] != nil {
		return c.PollLastOperationReactions[*r.OperationKey].Response, c.PollLastOperationReactions[*r.OperationKey].Error
	} else if c.PollLastOperationReaction != nil {
		return c.PollLastOperationReaction.React(r)
	}

	return nil, UnexpectedActionError()
}

// PollBindingLastOperation implements the Client.PollBindingLastOperation
// method on the FakeClient.
func (c *FakeClient) PollBindingLastOperation(r *v2.BindingLastOperationRequest) (*v2.LastOperationResponse, error) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	c.actions = append(c.actions, Action{PollBindingLastOperation, r})

	if c.PollBindingLastOperationReaction != nil {
		return c.PollBindingLastOperationReaction.React(r)
	}

	return nil, UnexpectedActionError()
}

// Bind implements the Client.Bind method on the FakeClient.
func (c *FakeClient) Bind(r *v2.BindRequest) (*v2.BindResponse, error) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	c.actions = append(c.actions, Action{Bind, r})

	if c.BindReaction != nil {
		return c.BindReaction.React(r)
	}

	return nil, UnexpectedActionError()
}

// Unbind implements the Client.Unbind method on the FakeClient.
func (c *FakeClient) Unbind(r *v2.UnbindRequest) (*v2.UnbindResponse, error) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	c.actions = append(c.actions, Action{Unbind, r})

	if c.UnbindReaction != nil {
		return c.UnbindReaction.React(r)
	}

	return nil, UnexpectedActionError()
}

// GetBinding implements the Client.GetBinding method for the FakeClient.
func (c *FakeClient) GetBinding(*v2.GetBindingRequest) (*v2.GetBindingResponse, error) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	c.actions = append(c.actions, Action{Type: GetBinding})

	if c.GetBindingReaction != nil {
		return c.GetBindingReaction.React()
	}

	return nil, UnexpectedActionError()
}

// CheckAvailability implements the Client.CheckAvailability method for the FakeClient.
func (c *FakeClient) CheckAvailability() error {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	c.actions = append(c.actions, Action{Type: CheckAvailability})

	if c.CheckAvailabilityReaction != nil {
		return c.CheckAvailabilityReaction.React()
	}

	return UnexpectedActionError()
}

func (c *FakeClient) GetOperation(*v2.GetOperationRequest) (*v2.GetOperationResponse, error) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	c.actions = append(c.actions, Action{Type: GetOperation})

	if c.GetOperationReaction != nil {
		return c.GetOperationReaction.React()
	}

	return nil, UnexpectedActionError()
}

// UnexpectedActionError returns an error message when an action is not found
// in the FakeClient's action array.
func UnexpectedActionError() error {
	return errors.New("Unexpected action")
}

// RequiredFieldsMissingError returns an error message indicating that
// a required field was not set
func RequiredFieldsMissingError() error {
	return errors.New("A required field on the request was not set")
}

// CatalogReactionInterface defines the reaction to GetCatalog requests.
type CatalogReactionInterface interface {
	React() (*v2.CatalogResponse, error)
}

type CatalogReaction struct {
	Response *v2.CatalogResponse
	Error    error
}

func (r *CatalogReaction) React() (*v2.CatalogResponse, error) {
	if r == nil {
		return nil, UnexpectedActionError()
	}
	return r.Response, r.Error
}

type DynamicCatalogReaction func() (*v2.CatalogResponse, error)

func (r DynamicCatalogReaction) React() (*v2.CatalogResponse, error) {
	return r()
}

// ProvisionReactionInterface defines the reaction to ProvisionInstance requests.
type ProvisionReactionInterface interface {
	React(*v2.ProvisionRequest) (*v2.ProvisionResponse, error)
}

type ProvisionReaction struct {
	Response *v2.ProvisionResponse
	Error    error
}

func (r *ProvisionReaction) React(req *v2.ProvisionRequest) (*v2.ProvisionResponse, error) {
	if r == nil {
		return nil, UnexpectedActionError()
	}
	if req.ServiceID == "" || req.PlanID == "" || req.OrganizationGUID == "" || req.SpaceGUID == "" {
		return nil, RequiredFieldsMissingError()
	}
	return r.Response, r.Error
}

type DynamicProvisionReaction func(*v2.ProvisionRequest) (*v2.ProvisionResponse, error)

func (r DynamicProvisionReaction) React(req *v2.ProvisionRequest) (*v2.ProvisionResponse, error) {
	return r(req)
}

// UpdateInstanceReactionInterface defines the reaction to UpdateInstance requests.
type UpdateInstanceReactionInterface interface {
	React(*v2.UpdateInstanceRequest) (*v2.UpdateInstanceResponse, error)
}

type UpdateInstanceReaction struct {
	Response *v2.UpdateInstanceResponse
	Error    error
}

func (r *UpdateInstanceReaction) React(_ *v2.UpdateInstanceRequest) (*v2.UpdateInstanceResponse, error) {
	if r == nil {
		return nil, UnexpectedActionError()
	}
	return r.Response, r.Error
}

type DynamicUpdateInstanceReaction func(*v2.UpdateInstanceRequest) (*v2.UpdateInstanceResponse, error)

func (r DynamicUpdateInstanceReaction) React(req *v2.UpdateInstanceRequest) (*v2.UpdateInstanceResponse, error) {
	return r(req)
}

// DeprovisionReactionInterface defines the reaction to DeprovisionInstance requests.
type DeprovisionReactionInterface interface {
	React(*v2.DeprovisionRequest) (*v2.DeprovisionResponse, error)
}

type DeprovisionReaction struct {
	Response *v2.DeprovisionResponse
	Error    error
}

func (r *DeprovisionReaction) React(_ *v2.DeprovisionRequest) (*v2.DeprovisionResponse, error) {
	if r == nil {
		return nil, UnexpectedActionError()
	}
	return r.Response, r.Error
}

type DynamicDeprovisionReaction func(*v2.DeprovisionRequest) (*v2.DeprovisionResponse, error)

func (r DynamicDeprovisionReaction) React(req *v2.DeprovisionRequest) (*v2.DeprovisionResponse, error) {
	return r(req)
}

// GetInstanceReactionInterface defines the reaction to GetInstance requests.
type GetInstanceReactionInterface interface {
	React() (*v2.GetInstanceResponse, error)
}

// GetInstancesReactionInterface defines the reaction to GetInstances requests.
type GetInstancesReactionInterface interface {
	React() (*v2.GetInstancesResponse, error)
}

type GetInstanceReaction struct {
	Response *v2.GetInstanceResponse
	Error    error
}

func (r *GetInstanceReaction) React() (*v2.GetInstanceResponse, error) {
	if r == nil {
		return nil, UnexpectedActionError()
	}
	return r.Response, r.Error
}

type DynamicGetInstanceReaction func() (*v2.GetInstanceResponse, error)

func (r DynamicGetInstanceReaction) React() (*v2.GetInstanceResponse, error) {
	return r()
}

// PollLastOperationReactionInterface defines the reaction to PollLastOperation
// requests.
type PollLastOperationReactionInterface interface {
	React(*v2.LastOperationRequest) (*v2.LastOperationResponse, error)
}

type PollLastOperationReaction struct {
	Response *v2.LastOperationResponse
	Error    error
}

func (r *PollLastOperationReaction) React(_ *v2.LastOperationRequest) (*v2.LastOperationResponse, error) {
	if r == nil {
		return nil, UnexpectedActionError()
	}
	return r.Response, r.Error
}

type DynamicPollLastOperationReaction func(*v2.LastOperationRequest) (*v2.LastOperationResponse, error)

func (r DynamicPollLastOperationReaction) React(req *v2.LastOperationRequest) (*v2.LastOperationResponse, error) {
	return r(req)
}

// PollBindingLastOperationReactionInterface defines the reaction to PollLastOperation
// requests.
type PollBindingLastOperationReactionInterface interface {
	React(*v2.BindingLastOperationRequest) (*v2.LastOperationResponse, error)
}

type PollBindingLastOperationReaction struct {
	Response *v2.LastOperationResponse
	Error    error
}

func (r *PollBindingLastOperationReaction) React(_ *v2.BindingLastOperationRequest) (*v2.LastOperationResponse, error) {
	if r == nil {
		return nil, UnexpectedActionError()
	}
	return r.Response, r.Error
}

type DynamicPollBindingLastOperationReaction func(*v2.BindingLastOperationRequest) (*v2.LastOperationResponse, error)

func (r DynamicPollBindingLastOperationReaction) React(req *v2.BindingLastOperationRequest) (*v2.LastOperationResponse, error) {
	return r(req)
}

// BindReactionInterface defines the reaction to Bind requests.
type BindReactionInterface interface {
	React(*v2.BindRequest) (*v2.BindResponse, error)
}

type BindReaction struct {
	Response *v2.BindResponse
	Error    error
}

func (r *BindReaction) React(_ *v2.BindRequest) (*v2.BindResponse, error) {
	if r == nil {
		return nil, UnexpectedActionError()
	}
	return r.Response, r.Error
}

type DynamicBindReaction func(*v2.BindRequest) (*v2.BindResponse, error)

func (r DynamicBindReaction) React(req *v2.BindRequest) (*v2.BindResponse, error) {
	return r(req)
}

// UnbindReactionInterface defines the reaction to Unbind requests.
type UnbindReactionInterface interface {
	React(*v2.UnbindRequest) (*v2.UnbindResponse, error)
}

type UnbindReaction struct {
	Response *v2.UnbindResponse
	Error    error
}

func (r *UnbindReaction) React(_ *v2.UnbindRequest) (*v2.UnbindResponse, error) {
	if r == nil {
		return nil, UnexpectedActionError()
	}
	return r.Response, r.Error
}

type DynamicUnbindReaction func(*v2.UnbindRequest) (*v2.UnbindResponse, error)

func (r DynamicUnbindReaction) React(req *v2.UnbindRequest) (*v2.UnbindResponse, error) {
	return r(req)
}

// GetBindingReactionInterface defines the reaction to GetBinding requests.
type GetBindingReactionInterface interface {
	React() (*v2.GetBindingResponse, error)
}

type GetBindingReaction struct {
	Response *v2.GetBindingResponse
	Error    error
}

func (r *GetBindingReaction) React() (*v2.GetBindingResponse, error) {
	if r == nil {
		return nil, UnexpectedActionError()
	}
	return r.Response, r.Error
}

type DynamicGetBindingReaction func() (*v2.GetBindingResponse, error)

func (r DynamicGetBindingReaction) React() (*v2.GetBindingResponse, error) {
	return r()
}

type CheckAvailabilityReactionInterface interface {
	React() error
}

type CheckAvailabilityReaction func() error

func (r CheckAvailabilityReaction) React() error {
	return r()
}

type GetOperationReactionInterface interface {
	React() (*v2.GetOperationResponse, error)
}

type GetOperationReaction struct {
	Response *v2.GetOperationResponse
	Error    error
}

func (r *GetOperationReaction) React() (*v2.GetOperationResponse, error) {
	if r == nil {
		return nil, UnexpectedActionError()
	}
	return r.Response, r.Error
}

func strPtr(s string) *string {
	return &s
}

// AsyncRequiredError returns error for required asynchronous operations.
func AsyncRequiredError() error {
	return v2.HTTPStatusCodeError{
		StatusCode:   http.StatusUnprocessableEntity,
		ErrorMessage: strPtr(v2.AsyncErrorMessage),
		Description:  strPtr(v2.AsyncErrorDescription),
	}
}

// AppGUIDRequiredError returns error for when app GUID is missing from bind
// request.
func AppGUIDRequiredError() error {
	return v2.HTTPStatusCodeError{
		StatusCode:   http.StatusUnprocessableEntity,
		ErrorMessage: strPtr(v2.AppGUIDRequiredErrorMessage),
		Description:  strPtr(v2.AppGUIDRequiredErrorDescription),
	}
}

// ConcurrencyError returns error for when concurrent requests to modify the
// same resource is rejected.
func ConcurrencyError() error {
	return v2.HTTPStatusCodeError{
		StatusCode:   http.StatusUnprocessableEntity,
		ErrorMessage: strPtr(v2.ConcurrencyErrorMessage),
		Description:  strPtr(v2.ConcurrencyErrorDescription),
	}
}
