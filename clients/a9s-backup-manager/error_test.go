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
