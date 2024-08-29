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
	"fmt"
	"net/http"
	"testing"

	"k8s.io/utils/pointer"
)

func defaultGetRestoreRequest() *GetRestoreRequest {
	return &GetRestoreRequest{
		InstanceID: "test-instance-id",
		RestoreID:  "10",
	}
}

const successGetRestoreRequestResponseBody = `{
	"id": 1,
	"backup_id": 10,
	"status": "done",
	"triggered_at": "2023-04-11T08:52:48.209Z",
	"finished_at": "2023-04-11T08:53:16.411Z"
  }`

const instanceNotFoundGetRestoreResponseBody = `{
	"error": "NotFound",
	"description": "The instance test-instance was not found."
  }`

const restoreNotFoundGetRestoreResponseBody = `{
	"error": "NotFound",
	"description": "The restore 4 was not found."
  }`

func TestGetRestore(t *testing.T) {
	cases := []struct {
		name               string
		request            *GetRestoreRequest
		httpChecks         httpChecks
		httpReaction       httpReaction
		expectedResponse   *GetRestoreResponse
		expectedErrMessage string
		expectedErr        error
	}{
		{
			name: "invalid request instanceID missing",
			request: &GetRestoreRequest{
				InstanceID: "",
				RestoreID:  "1",
			},
			expectedErrMessage: "instanceID is required",
		},
		{
			name: "invalid request restoreID missing",
			request: &GetRestoreRequest{
				InstanceID: "test",
				RestoreID:  "",
			},
			expectedErrMessage: "restoreID is required",
		},
		{
			name: "success",
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   successGetRestoreRequestResponseBody,
			},
			expectedResponse: &GetRestoreResponse{
				RestoreID:   pointer.Int(1),
				BackupID:    pointer.Int(10),
				Status:      "done",
				TriggeredAt: "2023-04-11T08:52:48.209Z",
				FinishedAt:  "2023-04-11T08:53:16.411Z",
			},
		},
		{
			name: "instance not found",
			request: &GetRestoreRequest{
				InstanceID: "test-instance",
				RestoreID:  "3",
			},
			httpReaction: httpReaction{
				status: http.StatusNotFound,
				body:   instanceNotFoundGetRestoreResponseBody,
			},
			expectedErrMessage: "Status: 404; ErrorMessage: NotFound; Description: The instance test-instance was not found.; ResponseError: <nil>",
		},
		{
			name: "restore not found",
			request: &GetRestoreRequest{
				InstanceID: "test-instance",
				RestoreID:  "4",
			},
			httpReaction: httpReaction{
				status: http.StatusNotFound,
				body:   restoreNotFoundGetRestoreResponseBody,
			},
			expectedErrMessage: "Status: 404; ErrorMessage: NotFound; Description: The restore 4 was not found.; ResponseError: <nil>",
		},
		{
			name: "http error",
			httpReaction: httpReaction{
				err: fmt.Errorf("http error"),
			},
			expectedErrMessage: "http error",
		},
		{
			name: "201 with malformed response",
			httpReaction: httpReaction{
				status: http.StatusCreated,
				body:   malformedResponse,
			},
			expectedErrMessage: "Status: 201; ErrorMessage: <nil>; Description: <nil>; ResponseError: unexpected end of JSON input",
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
			name: "500 with conventional failure response",
			httpReaction: httpReaction{
				status: http.StatusInternalServerError,
				body:   conventionalFailureResponseBody,
			},
			expectedErr: testHTTPStatusCodeError(),
		},
	}

	for _, tc := range cases {
		if tc.request == nil {
			tc.request = defaultGetRestoreRequest()
		}

		if tc.httpChecks.URL == "" {
			tc.httpChecks.URL = "/instances/" + tc.request.InstanceID + "/restores/" + tc.request.RestoreID
		}

		klient := newTestClient(t, tc.name, tc.httpChecks, tc.httpReaction)

		response, err := klient.GetRestore(tc.request)
		doResponseChecks(t, tc.name, response, err, tc.expectedResponse, tc.expectedErrMessage, tc.expectedErr)
	}
}

func TestValidateGetRestore(t *testing.T) {
	cases := []struct {
		name    string
		request *GetRestoreRequest
		valid   bool
	}{
		{
			name:    "valid",
			request: defaultGetRestoreRequest(),
			valid:   true,
		},
		{
			name: "missing instance ID",
			request: func() *GetRestoreRequest {
				r := defaultGetRestoreRequest()
				r.InstanceID = ""
				return r
			}(),
			valid: false,
		},
		{
			name: "missing restore ID",
			request: func() *GetRestoreRequest {
				r := defaultGetRestoreRequest()
				r.InstanceID = "test-instance-id"
				r.RestoreID = ""
				return r
			}(),
			valid: false,
		},
	}

	for _, tc := range cases {
		err := validateGetRestoreRequest(tc.request)
		if err != nil {
			if tc.valid {
				t.Errorf("%v: expected valid, got error: %v", tc.name, err)
			}
		} else if !tc.valid {
			t.Errorf("%v: expected invalid, got valid", tc.name)
		}
	}
}
