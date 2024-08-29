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

package utilerr

import (
	"context"
	"errors"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

var _ managed.ExternalClient = &Decorator{}
var _ managed.ExternalConnecter = &ConnectDecorator{}
var ErrInternal = errors.New("Internal error in provider")

type Userdisplayer interface {
	error
	UserDisplay() error
}

type ConnectDecorator struct {
	Connector managed.ExternalConnecter
	Logger    logging.Logger
}

func (cd ConnectDecorator) Connect(ctx context.Context, res resource.Managed) (managed.ExternalClient, error) {
	c, err := cd.Connector.Connect(ctx, res)
	if err != nil {
		return nil, err
	}

	return &Decorator{
		ExternalClient: c,
		Logger:         cd.Logger,
	}, nil
}

// Decorator add error handling via logging to an existing ExternalClient.
// If an error returned by an ExternalClient implements UserDisplay() the user
// displayable error will be returned so that crossplane can make it known to the user, and
// the internal error will be logged. If UserDisplay() is not implemented, the Decorator
// logs the error and returns ErrInternal.
type Decorator struct {
	ExternalClient managed.ExternalClient
	Logger         logging.Logger
}

func (cl Decorator) Create(ctx context.Context, res resource.Managed) (managed.ExternalCreation, error) {
	r, err := cl.ExternalClient.Create(ctx, res)
	return r, cl.convertAndLog("Create", err)
}

func (cl Decorator) Delete(ctx context.Context, res resource.Managed) error {
	return cl.convertAndLog("Delete", cl.ExternalClient.Delete(ctx, res))
}

func (cl Decorator) Observe(ctx context.Context, res resource.Managed) (managed.ExternalObservation, error) {
	r, err := cl.ExternalClient.Observe(ctx, res)
	return r, cl.convertAndLog("Observe", err)
}

func (cl Decorator) Update(ctx context.Context, res resource.Managed) (managed.ExternalUpdate, error) {
	r, err := cl.ExternalClient.Update(ctx, res)
	return r, cl.convertAndLog("Update", err)
}

func (cl Decorator) convertAndLog(operation string, err error) error {
	if err != nil {
		var ud Userdisplayer
		cl.Logger.Info("error in "+operation, "error", err)

		if errors.As(err, &ud) {

			return ud.UserDisplay()
		} else if err != nil {
			return ErrInternal
		}
	}

	return nil
}
