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

package v2

import (
	"errors"
	"net/http"
	"testing"
)

func TestIsHTTPError(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		expected bool
		result   *HTTPStatusCodeError
	}{
		{
			name:     "non-http error",
			err:      errors.New("some error"),
			expected: false,
			result:   nil,
		},
		{
			name:     "http error",
			err:      HTTPStatusCodeError{StatusCode: http.StatusGone},
			expected: true,
			result:   &HTTPStatusCodeError{StatusCode: http.StatusGone},
		},
		{
			name:     "http pointer error",
			err:      &HTTPStatusCodeError{StatusCode: http.StatusGone},
			expected: true,
			result:   &HTTPStatusCodeError{StatusCode: http.StatusGone},
		},
		{
			name:     "nil",
			err:      nil,
			expected: false,
			result:   nil,
		},
	}

	for _, tc := range cases {
		err, actual := IsHTTPError(tc.err)
		if tc.expected != actual {
			t.Errorf("%v: expected %v, got %v", tc.name, tc.expected, actual)
		}
		if tc.result != err {
			if *tc.result != *err {
				t.Errorf("%v: expected %v, got %v", tc.name, tc.result, err)
			}
		}
	}
}

func TestIsConflictError(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "non-http error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name: "http non-conflict error",
			err: HTTPStatusCodeError{
				StatusCode: http.StatusForbidden,
			},
			expected: false,
		},
		{
			name: "http conflict error",
			err: HTTPStatusCodeError{
				StatusCode: http.StatusConflict,
			},
			expected: true,
		},
	}

	for _, tc := range cases {
		if e, a := tc.expected, IsConflictError(tc.err); e != a {
			t.Errorf("%v: expected %v, got %v", tc.name, e, a)
		}
	}
}

func TestIsGoneError(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "non-http error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name: "http non-gone error",
			err: HTTPStatusCodeError{
				StatusCode: http.StatusForbidden,
			},
			expected: false,
		},
		{
			name: "http gone error",
			err: HTTPStatusCodeError{
				StatusCode: http.StatusGone,
			},
			expected: true,
		},
	}

	for _, tc := range cases {
		if e, a := tc.expected, IsGoneError(tc.err); e != a {
			t.Errorf("%v: expected %v, got %v", tc.name, e, a)
		}
	}
}

func strPtr(s string) *string {
	return &s
}

func TestIsAsyncRequiredError(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "non-http error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name: "other http error",
			err: HTTPStatusCodeError{
				StatusCode: http.StatusForbidden,
			},
			expected: false,
		},
		{
			name: "async required error",
			err: HTTPStatusCodeError{
				StatusCode:   http.StatusUnprocessableEntity,
				ErrorMessage: strPtr(AsyncErrorMessage),
				Description:  strPtr(AsyncErrorDescription),
			},
			expected: true,
		},
		{
			name: "app guid required error",
			err: HTTPStatusCodeError{
				StatusCode:   http.StatusUnprocessableEntity,
				ErrorMessage: strPtr(AppGUIDRequiredErrorMessage),
				Description:  strPtr(AppGUIDRequiredErrorDescription),
			},
			expected: false,
		},
		{
			name: "concurrency error",
			err: HTTPStatusCodeError{
				StatusCode:   http.StatusUnprocessableEntity,
				ErrorMessage: strPtr(ConcurrencyErrorMessage),
				Description:  strPtr(ConcurrencyErrorDescription),
			},
			expected: false,
		},
		{
			name: "no error message",
			err: HTTPStatusCodeError{
				StatusCode:  http.StatusUnprocessableEntity,
				Description: strPtr(AsyncErrorDescription),
			},
			expected: false,
		},
		{
			name: "no description",
			err: HTTPStatusCodeError{
				StatusCode:   http.StatusUnprocessableEntity,
				ErrorMessage: strPtr(AsyncErrorMessage),
			},
			expected: false,
		},
	}

	for _, tc := range cases {
		if e, a := tc.expected, IsAsyncRequiredError(tc.err); e != a {
			t.Errorf("%v: expected %v, got %v", tc.name, e, a)
		}
	}
}

func TestIsAppGUIDRequiredError(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "non-http error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name: "other http error",
			err: HTTPStatusCodeError{
				StatusCode: http.StatusForbidden,
			},
			expected: false,
		},
		{
			name: "async required error",
			err: HTTPStatusCodeError{
				StatusCode:   http.StatusUnprocessableEntity,
				ErrorMessage: strPtr(AsyncErrorMessage),
				Description:  strPtr(AsyncErrorDescription),
			},
			expected: false,
		},
		{
			name: "app guid required error",
			err: HTTPStatusCodeError{
				StatusCode:   http.StatusUnprocessableEntity,
				ErrorMessage: strPtr(AppGUIDRequiredErrorMessage),
				Description:  strPtr(AppGUIDRequiredErrorDescription),
			},
			expected: true,
		},
		{
			name: "concurrency error",
			err: HTTPStatusCodeError{
				StatusCode:   http.StatusUnprocessableEntity,
				ErrorMessage: strPtr(ConcurrencyErrorMessage),
				Description:  strPtr(ConcurrencyErrorDescription),
			},
			expected: false,
		},
		{
			name: "no error message",
			err: HTTPStatusCodeError{
				StatusCode:  http.StatusUnprocessableEntity,
				Description: strPtr(AppGUIDRequiredErrorDescription),
			},
			expected: false,
		},
		{
			name: "no description",
			err: HTTPStatusCodeError{
				StatusCode:   http.StatusUnprocessableEntity,
				ErrorMessage: strPtr(AppGUIDRequiredErrorMessage),
			},
			expected: false,
		},
	}

	for _, tc := range cases {
		if e, a := tc.expected, IsAppGUIDRequiredError(tc.err); e != a {
			t.Errorf("%v: expected %v, got %v", tc.name, e, a)
		}
	}
}

func TestConcurrencyError(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "non-http error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name: "other http error",
			err: HTTPStatusCodeError{
				StatusCode: http.StatusForbidden,
			},
			expected: false,
		},
		{
			name: "async required error",
			err: HTTPStatusCodeError{
				StatusCode:   http.StatusUnprocessableEntity,
				ErrorMessage: strPtr(AsyncErrorMessage),
				Description:  strPtr(AsyncErrorDescription),
			},
			expected: false,
		},
		{
			name: "app guid required error",
			err: HTTPStatusCodeError{
				StatusCode:   http.StatusUnprocessableEntity,
				ErrorMessage: strPtr(AppGUIDRequiredErrorMessage),
				Description:  strPtr(AppGUIDRequiredErrorDescription),
			},
			expected: false,
		},
		{
			name: "concurrency error",
			err: HTTPStatusCodeError{
				StatusCode:   http.StatusUnprocessableEntity,
				ErrorMessage: strPtr(ConcurrencyErrorMessage),
				Description:  strPtr(ConcurrencyErrorDescription),
			},
			expected: true,
		},
		{
			name: "no error message",
			err: HTTPStatusCodeError{
				StatusCode:  http.StatusUnprocessableEntity,
				Description: strPtr(AppGUIDRequiredErrorDescription),
			},
			expected: false,
		},
		{
			name: "no description",
			err: HTTPStatusCodeError{
				StatusCode:   http.StatusUnprocessableEntity,
				ErrorMessage: strPtr(AppGUIDRequiredErrorMessage),
			},
			expected: false,
		},
	}

	for _, tc := range cases {
		if e, a := tc.expected, IsConcurrencyError(tc.err); e != a {
			t.Errorf("%v: expected %v, got %v", tc.name, e, a)
		}
	}
}

func TestHttpStatusCodeError(t *testing.T) {
	cases := []struct {
		name           string
		err            error
		expectedOutput string
	}{
		{
			name: "async required error",
			err: HTTPStatusCodeError{
				StatusCode:   http.StatusUnprocessableEntity,
				ErrorMessage: strPtr(AsyncErrorMessage),
				Description:  strPtr(AsyncErrorDescription),
			},
			expectedOutput: "Status: 422; ErrorMessage: AsyncRequired; Description: This service plan requires client support for asynchronous service operations.; ResponseError: <nil>",
		},
		{
			name: "app guid required error",
			err: HTTPStatusCodeError{
				StatusCode:   http.StatusUnprocessableEntity,
				ErrorMessage: strPtr(AppGUIDRequiredErrorMessage),
				Description:  strPtr(AppGUIDRequiredErrorDescription),
			},
			expectedOutput: "Status: 422; ErrorMessage: RequiresApp; Description: This service supports generation of credentials through binding an application only.; ResponseError: <nil>",
		},
		{
			name:           "blank error",
			err:            HTTPStatusCodeError{},
			expectedOutput: "Status: 0; ErrorMessage: <nil>; Description: <nil>; ResponseError: <nil>",
		},
	}

	for _, tc := range cases {
		if e, a := tc.expectedOutput, tc.err.Error(); e != a {
			t.Errorf("%v: expected %v, got %v", tc.name, e, a)
		}
	}
}

func TestAsyncBindingOperationsNotAllowedError(t *testing.T) {
	err := AsyncBindingOperationsNotAllowedError{
		reason: "test reason",
	}

	expectedOutput := "Asynchronous binding operations are not allowed: test reason"

	if e, a := expectedOutput, err.Error(); e != a {
		t.Fatalf("expected %v, got %v", e, a)
	}
}

func TestIsAsyncBindingOperationsNotAllowedError(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "async binding operations not allowed error",
			err:      AsyncBindingOperationsNotAllowedError{},
			expected: true,
		},
		{
			name:     "other error",
			err:      errors.New("other error"),
			expected: false,
		},
	}

	for _, tc := range cases {
		if e, a := tc.expected, IsAsyncBindingOperationsNotAllowedError(tc.err); e != a {
			t.Errorf("%v: expected %v, got %v", tc.name, e, a)
		}
	}
}

func TestAvailabilityInvalidStatusError(t *testing.T) {
	err := AvailabilityInvalidStatusError{
		StatusCode: 503,
	}

	expectedOutput := "Received unexpected status 503"

	if e, a := expectedOutput, err.Error(); e != a {
		t.Fatalf("expected %v, got %v", e, a)
	}
}

func TestOperationStateError(t *testing.T) {
	cases := []struct {
		state    string
		expected string
	}{
		{
			state:    "error",
			expected: `Operation is in failed state "error"`,
		},
		{
			state:    "timeout",
			expected: `Operation is in failed state "timeout"`,
		},
		{
			state:    "cancelled",
			expected: `Operation is in failed state "cancelled"`,
		},
	}

	for _, tc := range cases {
		err := OperationStateError{
			State: tc.state,
		}

		if e, a := tc.expected, err.Error(); e != a {
			t.Fatalf("%s: expected %v, got %v", tc.state, e, a)
		}
	}
}
