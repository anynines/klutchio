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
	"net/http"
	"testing"

	"k8s.io/utils/pointer"
)

func TestCheckAvailability(t *testing.T) {
	cases := []struct {
		name               string
		httpReaction       httpReaction
		expectedErrMessage *string
	}{
		{
			name: "success 200",
			httpReaction: httpReaction{
				status: http.StatusOK,
			},
			expectedErrMessage: nil,
		},
		{
			name: "failure 401",
			httpReaction: httpReaction{
				status: http.StatusUnauthorized,
			},
			expectedErrMessage: pointer.String("Received unexpected status 401"),
		},
		{
			name: "failure 500",
			httpReaction: httpReaction{
				status: http.StatusInternalServerError,
			},
			expectedErrMessage: pointer.String("Received unexpected status 500"),
		},
	}

	for _, tc := range cases {
		httpChecks := httpChecks{URL: "/instances"}

		klient := newTestClient(t, tc.name, LatestAPIVersion(), false, httpChecks, tc.httpReaction)

		err := klient.CheckAvailability()

		if err == nil && tc.expectedErrMessage != nil {
			t.Fatalf("Expected check to fail with %v, but it did not", tc.expectedErrMessage)
		} else if err != nil && tc.expectedErrMessage == nil {
			t.Fatalf("Expected check to succeed, but it failed with %v", err)
		} else if err != nil && err.Error() != *tc.expectedErrMessage {
			t.Fatalf("Expected error %v, got %v", *tc.expectedErrMessage, err.Error())
		}
	}
}
