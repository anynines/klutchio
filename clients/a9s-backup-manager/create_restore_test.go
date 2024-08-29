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

func defaultCreateRestoreRequest() *CreateRestoreRequest {
	return &CreateRestoreRequest{
		InstanceID: "test-instance-id",
		BackupID:   "1",
	}
}

const successCreateRestoreRequestResponseBody = `{
	"id": 1
  }`

const backupNotFoundCreateRestoreResponseBody = `{
	"error": "NotFound",
	"description": "The backup 1 was not found for the instance test-instance."
  }`

const RestoreInProgressCreateRestoreResponseBody = `{
	"error": "RestoreInProgress",
	"description": "Restore of 1 is already in progress"
  }`
const BackupNonRestorableStateCreateRestoreResponseBody = `{
	"error": "BackupNonRestorableState",
	"description": "Backup 1 is in a non-restorable state"
  }`

func TestCreateRestore(t *testing.T) {
	cases := []struct {
		name               string
		request            *CreateRestoreRequest
		httpChecks         httpChecks
		httpReaction       httpReaction
		expectedResponse   *CreateRestoreResponse
		expectedErrMessage string
		expectedErr        error
	}{
		{
			name: "invalid request - missing Instance ID",
			request: &CreateRestoreRequest{
				InstanceID: "",
			},
			expectedErrMessage: "instanceID is required",
		},
		{
			name: "invalid request - missing Backup ID",
			request: &CreateRestoreRequest{
				InstanceID: "1",
				BackupID:   "",
			},
			expectedErrMessage: "backupID is required",
		},
		{
			name: "success - created",
			httpReaction: httpReaction{
				status: http.StatusAccepted,
				body:   successCreateRestoreRequestResponseBody,
			},
			expectedResponse: &CreateRestoreResponse{
				RestoreID: pointer.Int(1),
			},
		},
		{
			name: "backup not found",
			request: &CreateRestoreRequest{
				InstanceID: "test-instance",
				BackupID:   "1",
			},
			httpReaction: httpReaction{
				status: http.StatusNotFound,
				body:   backupNotFoundCreateRestoreResponseBody,
			},
			expectedErrMessage: "backup not found: Status: 404; ErrorMessage: NotFound; Description: The backup 1 was not found for the instance test-instance.; ResponseError: <nil>",
		},
		{
			name: "restore in progress",
			request: &CreateRestoreRequest{
				InstanceID: "test-instance",
				BackupID:   "1",
			},
			httpReaction: httpReaction{
				status: http.StatusConflict,
				body:   RestoreInProgressCreateRestoreResponseBody,
			},
			expectedErrMessage: "restore already in progress: Status: 409; ErrorMessage: RestoreInProgress; Description: Restore of 1 is already in progress; ResponseError: <nil>",
		},
		{
			name: "backup non restorable state",
			request: &CreateRestoreRequest{
				InstanceID: "test-instance",
				BackupID:   "1",
			},
			httpReaction: httpReaction{
				status: http.StatusUnprocessableEntity,
				body:   BackupNonRestorableStateCreateRestoreResponseBody,
			},
			expectedErrMessage: "backup is in a non-restorable state: Status: 422; ErrorMessage: BackupNonRestorableState; Description: Backup 1 is in a non-restorable state; ResponseError: <nil>",
		},
		{
			name: "http error",
			httpReaction: httpReaction{
				err: fmt.Errorf("http error"),
			},
			expectedErrMessage: "http error",
		},
		{
			name: "202 with malformed response",
			httpReaction: httpReaction{
				status: http.StatusAccepted,
				body:   malformedResponse,
			},
			expectedErrMessage: "Status: 202; ErrorMessage: <nil>; Description: <nil>; ResponseError: unexpected end of JSON input",
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
			tc.request = defaultCreateRestoreRequest()
		}

		if tc.httpChecks.URL == "" {
			tc.httpChecks.URL = "/instances/" + tc.request.InstanceID + "/backups/" + tc.request.BackupID + "/restore"
		}

		klient := newTestClient(t, tc.name, tc.httpChecks, tc.httpReaction)

		response, err := klient.CreateRestore(tc.request)

		doResponseChecks(t, tc.name, response, err, tc.expectedResponse, tc.expectedErrMessage, tc.expectedErr)
	}
}

func TestValidateCreateRestore(t *testing.T) {
	cases := []struct {
		name    string
		request *CreateRestoreRequest
		valid   bool
	}{
		{
			name:    "valid",
			request: defaultCreateRestoreRequest(),
			valid:   true,
		},
		{
			name: "missing instance ID",
			request: func() *CreateRestoreRequest {
				r := defaultCreateRestoreRequest()
				r.InstanceID = ""
				return r
			}(),
			valid: false,
		},
		{
			name: "missing backup ID",
			request: func() *CreateRestoreRequest {
				r := defaultCreateRestoreRequest()
				r.BackupID = ""
				return r
			}(),
			valid: false,
		},
	}

	for _, tc := range cases {
		err := validateCreateRestoreRequest(tc.request)
		if err != nil {
			if tc.valid {
				t.Errorf("%v: expected valid, got error: %v", tc.name, err)
			}
		} else if !tc.valid {
			t.Errorf("%v: expected invalid, got valid", tc.name)
		}
	}
}
