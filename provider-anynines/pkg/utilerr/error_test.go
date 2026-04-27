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
	"errors"
	"testing"

	"github.com/anynines/klutchio/provider-anynines/pkg/utilerr"
)

// TestWithCause_SurfacesUnderlyingError documents whether WithCause makes
// the underlying cause reachable via standard Go error-chain APIs.
//
// The user-visible message is set via FromStr; the technical root cause is
// passed to WithCause. Callers frequently use errors.Is / errors.As to inspect
// error chains, so it matters whether the cause is accessible that way.
func TestWithCause_SurfacesUnderlyingError(t *testing.T) {
	t.Parallel()

	cause := errors.New("underlying cause")
	wrapped := utilerr.FromStr("user-facing message").WithCause(cause)

	if errors.Unwrap(wrapped) != cause {
		t.Error("expected errors.Unwrap to return the cause, but it did not")
	}

	if !errors.Is(wrapped, cause) {
		t.Error("expected errors.Is(wrapped, cause) to be true, but it was false")
	}
}
