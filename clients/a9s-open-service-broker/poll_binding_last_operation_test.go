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
	"fmt"
	"net/http"
	"testing"
	"time"
)

func defaultBindingLastOperationRequest() *BindingLastOperationRequest {
	return &BindingLastOperationRequest{
		InstanceID: testInstanceID,
		BindingID:  testBindingID,
		ServiceID:  strPtr(testServiceID),
		PlanID:     strPtr(testPlanID),
	}
}

func TestPollBindingLastOperation(t *testing.T) {
	cases := []struct {
		name                string
		enableAlpha         bool
		originatingIdentity *OriginatingIdentity
		request             *BindingLastOperationRequest
		APIVersion          APIVersion
		httpChecks          httpChecks
		httpReaction        httpReaction
		expectedResponse    *LastOperationResponse
		expectedErrMessage  string
		expectedErr         error
	}{
		{
			name: "op succeeded",
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   successLastOperationResponseBody,
			},
			expectedResponse: successLastOperationResponse(),
		},
		{
			name: "op in progress",
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   inProgressLastOperationResponseBody,
			},
			expectedResponse: inProgressLastOperationResponse(),
		},
		{
			name: "op failed",
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   failedLastOperationResponseBody,
			},
			expectedResponse: failedLastOperationResponse(),
		},
		{
			name: "http error",
			httpReaction: httpReaction{
				err: fmt.Errorf("http error"),
			},
			expectedErrMessage: "http error",
		},
		{
			name: "200 with malformed response",
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   malformedResponse,
			},
			expectedErrMessage: "Status: 200; ErrorMessage: <nil>; Description: <nil>; ResponseError: unexpected end of JSON input",
		},
		{
			name: "500 with malformed response",
			httpReaction: httpReaction{
				status: http.StatusInternalServerError,
				body:   malformedResponse,
			},
			expectedErrMessage: "Status: 500; ErrorMessage: <nil>; Description: <nil>; ResponseError: unexpected end of JSON input",
		},
		{
			name: "500 with conventional response",
			httpReaction: httpReaction{
				status: http.StatusInternalServerError,
				body:   conventionalFailureResponseBody,
			},
			expectedErr: testHTTPStatusCodeError(),
		},
		{
			name: "op succeeded",
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   successLastOperationResponseBody,
			},
			expectedResponse: successLastOperationResponse(),
		},
		{
			name:                "originating identity included",
			originatingIdentity: testOriginatingIdentity,
			httpChecks:          httpChecks{headers: map[string]string{OriginatingIdentityHeader: testOriginatingIdentityHeaderValue}},
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   successLastOperationResponseBody,
			},
			expectedResponse: successLastOperationResponse(),
		},
		{
			name:                "originating identity excluded",
			originatingIdentity: nil,
			httpChecks:          httpChecks{headers: map[string]string{OriginatingIdentityHeader: ""}},
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   successLastOperationResponseBody,
			},
			expectedResponse: successLastOperationResponse(),
		},
		{
			name:               "unsupported API version",
			APIVersion:         Version2_13(),
			expectedErrMessage: "Asynchronous binding operations are not allowed: operation not allowed: must have API version >= 2.14. Current: 2.13",
		},
		{
			name: "polling delay header is decoded",
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   inProgressLastOperationResponseBody,
				header: map[string][]string{PollingDelayHeader: {"300"}},
			},
			enableAlpha: true,
			expectedResponse: &LastOperationResponse{
				State:       StateInProgress,
				Description: strPtr("test description"),
				PollDelay:   durationPtr(300 * time.Second),
			},
		},
		{
			name:       "polling delay header is ignored when alpha feature is disabled",
			APIVersion: LatestAPIVersion(),
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   inProgressLastOperationResponseBody,
				header: map[string][]string{PollingDelayHeader: {"300"}},
			},
			expectedResponse: &LastOperationResponse{
				State:       StateInProgress,
				Description: strPtr("test description"),
				PollDelay:   nil,
			},
		},
		{
			name:        "polling delay header as timestamp is ignored",
			enableAlpha: true,
			APIVersion:  LatestAPIVersion(),
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   inProgressLastOperationResponseBody,
				header: map[string][]string{PollingDelayHeader: {"2021-01-23T23:12:00Z"}},
			},
			expectedResponse: &LastOperationResponse{
				State:       StateInProgress,
				Description: strPtr("test description"),
				PollDelay:   nil,
			},
		},
	}

	for _, tc := range cases {
		if tc.request == nil {
			tc.request = defaultBindingLastOperationRequest()
		}

		tc.request.OriginatingIdentity = tc.originatingIdentity

		if tc.httpChecks.URL == "" {
			tc.httpChecks.URL = "/v2/service_instances/test-instance-id/service_bindings/test-binding-id/last_operation"
		}

		if len(tc.httpChecks.params) == 0 {
			tc.httpChecks.params = map[string]string{}
			tc.httpChecks.params[VarKeyServiceID] = testServiceID
			tc.httpChecks.params[VarKeyPlanID] = testPlanID
		}

		if tc.APIVersion.label == "" {
			tc.APIVersion = LatestAPIVersion()
		}

		klient := newTestClient(t, tc.name, tc.APIVersion, tc.enableAlpha, tc.httpChecks, tc.httpReaction)

		response, err := klient.PollBindingLastOperation(tc.request)

		doResponseChecks(t, tc.name, response, err, tc.expectedResponse, tc.expectedErrMessage, tc.expectedErr)
	}
}

func TestValidateBindingLastOperationRequest(t *testing.T) {
	cases := []struct {
		name    string
		request *BindingLastOperationRequest
		valid   bool
	}{
		{
			name:    "valid",
			request: defaultBindingLastOperationRequest(),
			valid:   true,
		},
		{
			name: "missing instance ID",
			request: func() *BindingLastOperationRequest {
				r := defaultBindingLastOperationRequest()
				r.InstanceID = ""
				return r
			}(),
			valid: false,
		},
		{
			name: "missing binding ID",
			request: func() *BindingLastOperationRequest {
				r := defaultBindingLastOperationRequest()
				r.BindingID = ""
				return r
			}(),
			valid: false,
		},
	}

	for _, tc := range cases {
		err := validateBindingLastOperationRequest(tc.request)
		if err != nil {
			if tc.valid {
				t.Errorf("%v: expected valid, got error: %v", tc.name, err)
			}
		} else if !tc.valid {
			t.Errorf("%v: expected invalid, got valid", tc.name)
		}
	}
}
