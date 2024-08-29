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

func defaultGetBackupRequest() *GetBackupRequest {
	return &GetBackupRequest{
		InstanceID: "test-instance-id",
		BackupID:   "10",
	}
}

const successGetBackupRequestResponseBody = `{
	"id": 5,
	"size": 1184,
	"status": "done",
	"triggered_at": "2023-04-11T08:52:48.209Z",
	"finished_at": "2023-04-11T08:53:16.411Z",
	"downloadable": false
  }`

const instanceNotFoundGetBackupResponseBody = `{
	"error": "NotFound",
	"description": "The instance test-instance was not found."
  }`

const backupNotFoundGetBackupResponseBody = `{
	"error": "NotFound",
	"description": "The backup test-backup was not found."
  }`

func TestGetBackup(t *testing.T) {
	cases := []struct {
		name               string
		request            *GetBackupRequest
		httpChecks         httpChecks
		httpReaction       httpReaction
		expectedResponse   *GetBackupResponse
		expectedErrMessage string
		expectedErr        error
	}{
		{
			name: "invalid request instanceID missing",
			request: &GetBackupRequest{
				InstanceID: "",
				BackupID:   "1",
			},
			expectedErrMessage: "instanceID is required",
		},
		{
			name: "invalid request backupID missing",
			request: &GetBackupRequest{
				InstanceID: "test",
				BackupID:   "",
			},
			expectedErrMessage: "backupID is required",
		},
		{
			name: "success",
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   successGetBackupRequestResponseBody,
			},
			expectedResponse: &GetBackupResponse{
				BackupID:     pointer.Int(5),
				Size:         1184,
				Status:       "done",
				TriggeredAt:  "2023-04-11T08:52:48.209Z",
				FinishedAt:   "2023-04-11T08:53:16.411Z",
				Downloadable: false,
			},
		},
		{
			name: "instance not found",
			request: &GetBackupRequest{
				InstanceID: "test-instance",
				BackupID:   "3",
			},
			httpReaction: httpReaction{
				status: http.StatusNotFound,
				body:   instanceNotFoundGetBackupResponseBody,
			},
			expectedErrMessage: "Status: 404; ErrorMessage: NotFound; Description: The instance test-instance was not found.; ResponseError: <nil>",
		},
		{
			name: "backup not found",
			request: &GetBackupRequest{
				InstanceID: "test-instance",
				BackupID:   "4",
			},
			httpReaction: httpReaction{
				status: http.StatusNotFound,
				body:   backupNotFoundGetBackupResponseBody,
			},
			expectedErrMessage: "Status: 404; ErrorMessage: NotFound; Description: The backup test-backup was not found.; ResponseError: <nil>",
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
			tc.request = defaultGetBackupRequest()
		}

		if tc.httpChecks.URL == "" {
			tc.httpChecks.URL = "/instances/" + tc.request.InstanceID + "/backups/" + tc.request.BackupID
		}

		klient := newTestClient(t, tc.name, tc.httpChecks, tc.httpReaction)

		response, err := klient.GetBackup(tc.request)
		doResponseChecks(t, tc.name, response, err, tc.expectedResponse, tc.expectedErrMessage, tc.expectedErr)
	}
}

func TestValidateGetBackup(t *testing.T) {
	cases := []struct {
		name    string
		request *GetBackupRequest
		valid   bool
	}{
		{
			name:    "valid",
			request: defaultGetBackupRequest(),
			valid:   true,
		},
		{
			name: "missing instance ID",
			request: func() *GetBackupRequest {
				r := defaultGetBackupRequest()
				r.InstanceID = ""
				return r
			}(),
			valid: false,
		},
		{
			name: "missing backup ID",
			request: func() *GetBackupRequest {
				r := defaultGetBackupRequest()
				r.InstanceID = "test-instance-id"
				r.BackupID = ""
				return r
			}(),
			valid: false,
		},
	}

	for _, tc := range cases {
		err := validateGetBackupRequest(tc.request)
		if err != nil {
			if tc.valid {
				t.Errorf("%v: expected valid, got error: %v", tc.name, err)
			}
		} else if !tc.valid {
			t.Errorf("%v: expected invalid, got valid", tc.name)
		}
	}
}
