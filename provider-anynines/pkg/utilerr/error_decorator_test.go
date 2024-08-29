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

package utilerr_test

import (
	"context"
	"errors"
	"testing"

	crossplaneLogging "github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/go-logr/logr"

	logging "github.com/anynines/klutch/provider-anynines/pkg/utilerr"
)

func TestDecorated(t *testing.T) {
	dec := logging.Decorator{
		ExternalClient: &managed.NopClient{},
		Logger:         crossplaneLogging.NewNopLogger(),
	}

	_, err := dec.Create(context.Background(), nil)
	if err != nil {
		t.Fatal("expected nil error", err)
	}
}

func TestInternalError(t *testing.T) {
	t.Parallel()

	var internal error = errors.New("frobber is kalooning")

	dec := logging.Decorator{
		ExternalClient: managed.ExternalClientFns{
			CreateFn: func(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
				return managed.ExternalCreation{}, internal
			}},
		Logger: crossplaneLogging.NewLogrLogger(logr.Discard()),
	}

	_, err := dec.Create(context.Background(), nil)
	if !errors.Is(err, logging.ErrInternal) {
		t.Fatal(err)
	}
}

type withError struct {
	error
	message error
}

func (e withError) UserDisplay() error {
	return e.message
}

func TestUserDisplayErr(t *testing.T) {
	t.Parallel()

	var internal error = errors.New("frobber is kalooning")
	var userMessage = errors.New("Could not create resource")

	dec := logging.Decorator{
		ExternalClient: managed.ExternalClientFns{
			CreateFn: func(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
				return managed.ExternalCreation{}, withError{
					error:   internal,
					message: userMessage,
				}
			}},
		Logger: crossplaneLogging.NewLogrLogger(logr.Discard()),
	}

	_, err := dec.Create(context.Background(), nil)
	if !errors.Is(err, userMessage) {
		t.Fatal(err)
	}
}
