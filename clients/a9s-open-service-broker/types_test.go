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

import "testing"

func TestGetOperationResponseIsDone(t *testing.T) {
	cases := []struct {
		state    string
		expected bool
	}{
		{
			state:    "queued",
			expected: false,
		},
		{
			state:    "processing",
			expected: false,
		},
		{
			state:    "pending",
			expected: false,
		},
		{
			state:    "done",
			expected: true,
		},
		{
			state:    "error",
			expected: false,
		},
		{
			state:    "cancelled",
			expected: false,
		},
		{
			state:    "timeout",
			expected: false,
		},
	}

	for _, tc := range cases {
		resp := &GetOperationResponse{
			State: tc.state,
		}

		if e, a := tc.expected, resp.IsDone(); e != a {
			t.Fatalf("state=%s: expected %v, got %v", tc.state, e, a)
		}
	}
}

func TestGetOperationResponseIsFailure(t *testing.T) {
	cases := []struct {
		state       string
		expected    bool
		expectedErr error
	}{
		{
			state:    "queued",
			expected: false,
		},
		{
			state:    "processing",
			expected: false,
		},
		{
			state:    "pending",
			expected: false,
		},
		{
			state:    "done",
			expected: false,
		},
		{
			state:       "error",
			expected:    true,
			expectedErr: OperationStateError{"error"},
		},
		{
			state:       "cancelled",
			expected:    true,
			expectedErr: OperationStateError{"cancelled"},
		},
		{
			state:       "timeout",
			expected:    true,
			expectedErr: OperationStateError{"timeout"},
		},
	}

	for _, tc := range cases {
		resp := &GetOperationResponse{
			State: tc.state,
		}

		actual, err := resp.IsFailure()

		if err != tc.expectedErr {
			t.Fatalf("Expected error %v, got %v", tc.expectedErr, err)
		}

		if e, a := tc.expected, actual; e != a {
			t.Fatalf("state=%s: expected %v, got %v", tc.state, e, a)
		}
	}
}
