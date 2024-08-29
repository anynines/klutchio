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

func defaultLastOperationRequest() *LastOperationRequest {
	return &LastOperationRequest{
		InstanceID:   testInstanceID,
		ServiceID:    strPtr(testServiceID),
		PlanID:       strPtr(testPlanID),
		OperationKey: &testOperation,
	}
}

func successLastOperationResponse() *LastOperationResponse {
	return &LastOperationResponse{
		State:       StateSucceeded,
		Description: strPtr("test description"),
	}
}

const successLastOperationResponseBody = `{"state":"succeeded","description":"test description"}`

func inProgressLastOperationResponse() *LastOperationResponse {
	return &LastOperationResponse{
		State:       StateInProgress,
		Description: strPtr("test description"),
	}
}

const inProgressLastOperationResponseBody = `{"state":"in progress","description":"test description"}`

func failedLastOperationResponse() *LastOperationResponse {
	return &LastOperationResponse{
		State:       StateFailed,
		Description: strPtr("test description"),
	}
}

const failedLastOperationResponseBody = `{"state":"failed","description":"test description"}`

func TestPollLastOperation(t *testing.T) {
	cases := []struct {
		name                string
		version             APIVersion
		enableAlpha         bool
		originatingIdentity *OriginatingIdentity
		request             *LastOperationRequest
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
			version:             Version2_13(),
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
			version:             Version2_13(),
			originatingIdentity: nil,
			httpChecks:          httpChecks{headers: map[string]string{OriginatingIdentityHeader: ""}},
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   successLastOperationResponseBody,
			},
			expectedResponse: successLastOperationResponse(),
		},
		{
			name:                "originating identity not sent unless API version >= 2.13",
			version:             Version2_12(),
			originatingIdentity: testOriginatingIdentity,
			httpChecks:          httpChecks{headers: map[string]string{OriginatingIdentityHeader: ""}},
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   successLastOperationResponseBody,
			},
			expectedResponse: successLastOperationResponse(),
		},
		{
			name:        "retry delay header supplied with alpha features enabled",
			version:     LatestAPIVersion(),
			enableAlpha: true,
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   inProgressLastOperationResponseBody,
				header: map[string][]string{PollingDelayHeader: {"600"}},
			},
			expectedResponse: &LastOperationResponse{
				State:       StateInProgress,
				Description: strPtr("test description"),
				PollDelay:   durationPtr(600 * time.Second),
			},
		},
		{
			name:        "retry delay header supplied with alpha features disabled",
			version:     LatestAPIVersion(),
			enableAlpha: false,
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   inProgressLastOperationResponseBody,
				header: map[string][]string{PollingDelayHeader: {"600"}},
			},
			expectedResponse: &LastOperationResponse{
				State:       StateInProgress,
				Description: strPtr("test description"),
			},
		},
		{
			name:        "retry delay header with a timestamp is ignored",
			version:     LatestAPIVersion(),
			enableAlpha: true,
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   inProgressLastOperationResponseBody,
				header: map[string][]string{PollingDelayHeader: {"2020-12-31T23:59:60Z"}},
			},
			expectedResponse: &LastOperationResponse{
				State:       StateInProgress,
				Description: strPtr("test description"),
			},
		},
	}

	for _, tc := range cases {
		if tc.request == nil {
			tc.request = defaultLastOperationRequest()
		}

		tc.request.OriginatingIdentity = tc.originatingIdentity

		if tc.httpChecks.URL == "" {
			tc.httpChecks.URL = "/v2/service_instances/test-instance-id/last_operation"
		}

		if len(tc.httpChecks.params) == 0 {
			tc.httpChecks.params = map[string]string{}
			tc.httpChecks.params[VarKeyServiceID] = testServiceID
			tc.httpChecks.params[VarKeyPlanID] = testPlanID
			tc.httpChecks.params[VarKeyOperation] = "test-operation-key"
		}

		if tc.version.label == "" {
			tc.version = Version2_11()
		}

		klient := newTestClient(t, tc.name, tc.version, tc.enableAlpha, tc.httpChecks, tc.httpReaction)

		response, err := klient.PollLastOperation(tc.request)

		doResponseChecks(t, tc.name, response, err, tc.expectedResponse, tc.expectedErrMessage, tc.expectedErr)
	}
}

func durationPtr(d time.Duration) *time.Duration {
	return &d
}

func TestValidateLastOperationRequest(t *testing.T) {
	cases := []struct {
		name    string
		request *LastOperationRequest
		valid   bool
	}{
		{
			name:    "valid",
			request: defaultLastOperationRequest(),
			valid:   true,
		},
		{
			name: "missing instance ID",
			request: func() *LastOperationRequest {
				r := defaultLastOperationRequest()
				r.InstanceID = ""
				return r
			}(),
			valid: false,
		},
	}

	for _, tc := range cases {
		err := validateLastOperationRequest(tc.request)
		if err != nil {
			if tc.valid {
				t.Errorf("%v: expected valid, got error: %v", tc.name, err)
			}
		} else if !tc.valid {
			t.Errorf("%v: expected invalid, got valid", tc.name)
		}
	}
}
