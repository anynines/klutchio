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
)

const (
	okBindingBytes = `{
  "credentials": {
    "test-key": "foo"
  }
}`
	okBindingEndpointBytes = `{
  "credentials": {
    "test-key": "foo"
  },
  "endpoints": [
    {"host": "c-beam.alpha.city", "ports": [8080, 8443], "protocol": "tcp"}
  ]
}`
)

func defaultGetBindingRequest() *GetBindingRequest {
	return &GetBindingRequest{
		InstanceID: testInstanceID,
		BindingID:  testBindingID,
	}
}

func okGetBindingResponse() *GetBindingResponse {
	response := &GetBindingResponse{}
	response.Credentials = map[string]interface{}{
		"test-key": "foo",
	}
	return response
}
func okGetBindingEndpointResponse() *GetBindingResponse {
	response := okGetBindingResponse()
	response.Endpoints = &[]Endpoint{
		{
			Host:     "c-beam.alpha.city",
			Ports:    []uint16{8080, 8443},
			Protocol: (*EndpointProtocol)(strPtr("tcp")),
		},
	}
	return response
}

func TestGetBinding(t *testing.T) {
	cases := []struct {
		name               string
		enableAlpha        bool
		request            *GetBindingRequest
		APIVersion         APIVersion
		httpReaction       httpReaction
		expectedResponse   *GetBindingResponse
		expectedErrMessage string
		expectedErr        error
	}{
		{
			name: "success",
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   okBindingBytes,
			},
			expectedResponse: okGetBindingResponse(),
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
			name:               "unsupported API version",
			APIVersion:         Version2_13(),
			expectedErrMessage: "GetBinding not allowed: operation not allowed: must have API version >= 2.14. Current: 2.13",
		},
		{
			name:        "binding with endpoints",
			APIVersion:  LatestAPIVersion(),
			enableAlpha: true,
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   okBindingEndpointBytes,
			},
			expectedResponse: okGetBindingEndpointResponse(),
		},
		{
			name:        "alpha features disabled: binding with endpoints",
			APIVersion:  LatestAPIVersion(),
			enableAlpha: false,
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   okBindingEndpointBytes,
			},
			expectedResponse: okGetBindingResponse(),
		},
	}

	for _, tc := range cases {
		if tc.request == nil {
			tc.request = defaultGetBindingRequest()
		}

		httpChecks := httpChecks{
			URL: "/v2/service_instances/test-instance-id/service_bindings/test-binding-id",
		}

		if tc.APIVersion.label == "" {
			tc.APIVersion = LatestAPIVersion()
		}

		klient := newTestClient(t, tc.name, tc.APIVersion, tc.enableAlpha, httpChecks, tc.httpReaction)

		response, err := klient.GetBinding(tc.request)

		doResponseChecks(t, tc.name, response, err, tc.expectedResponse, tc.expectedErrMessage, tc.expectedErr)
	}
}
