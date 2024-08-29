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

func defaultGetBackupsRequest() *GetBackupsRequest {
	return &GetBackupsRequest{
		InstanceID: "test-instance-id",
	}
}

const successGetBackupsRequestResponseBody = `[
	{
		"id": 1,
		"size": 1184,
		"status": "done",
		"triggered_at": "2023-04-11T08:52:48.209Z",
		"finished_at": "2023-04-11T08:53:16.411Z",
		"downloadable": false
	},
	{
		"id": 2,
		"size": 1234,
		"status": "failed",
		"triggered_at": "2023-04-12T08:52:51.209Z",
		"finished_at": "2023-04-12T08:53:53.411Z",
		"downloadable": true
	}
]`

const instanceNotFoundGetBackupsResponseBody = `{
	"error": "NotFound",
	"description": "The instance test-instance was not found."
  }`

func TestGetBackups(t *testing.T) {
	cases := []struct {
		name               string
		request            *GetBackupsRequest
		httpChecks         httpChecks
		httpReaction       httpReaction
		expectedResponse   *GetBackupsResponse
		expectedErrMessage string
		expectedErr        error
	}{
		{
			name: "invalid request instanceID missing",
			request: &GetBackupsRequest{
				InstanceID: "",
			},
			expectedErrMessage: "instanceID is required",
		},
		{
			name: "success",
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   successGetBackupsRequestResponseBody,
			},
			expectedResponse: &GetBackupsResponse{
				Backups: []GetBackupResponse{
					{
						BackupID:     pointer.Int(1),
						Size:         1184,
						Status:       "done",
						TriggeredAt:  "2023-04-11T08:52:48.209Z",
						FinishedAt:   "2023-04-11T08:53:16.411Z",
						Downloadable: false,
					},
					{
						BackupID:     pointer.Int(2),
						Size:         1234,
						Status:       "failed",
						TriggeredAt:  "2023-04-12T08:52:51.209Z",
						FinishedAt:   "2023-04-12T08:53:53.411Z",
						Downloadable: true,
					},
				},
			},
		},
		{
			name: "instance not found",
			request: &GetBackupsRequest{
				InstanceID: "test-instance",
			},
			httpReaction: httpReaction{
				status: http.StatusNotFound,
				body:   instanceNotFoundGetBackupsResponseBody,
			},
			expectedErrMessage: "Status: 404; ErrorMessage: NotFound; Description: The instance test-instance was not found.; ResponseError: <nil>",
		},
		{
			name: "backup not found",
			request: &GetBackupsRequest{
				InstanceID: "test-instance",
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
			tc.request = defaultGetBackupsRequest()
		}

		if tc.httpChecks.URL == "" {
			tc.httpChecks.URL = "/instances/" + tc.request.InstanceID + "/backups"
		}

		klient := newTestClient(t, tc.name, tc.httpChecks, tc.httpReaction)

		response, err := klient.GetBackups(tc.request)

		doResponseChecks(t, tc.name, response, err, tc.expectedResponse, tc.expectedErrMessage, tc.expectedErr)
	}
}

func TestValidateGetBackups(t *testing.T) {
	cases := []struct {
		name    string
		request *GetBackupsRequest
		valid   bool
	}{
		{
			name:    "valid",
			request: defaultGetBackupsRequest(),
			valid:   true,
		},
		{
			name: "missing instance ID",
			request: func() *GetBackupsRequest {
				r := defaultGetBackupsRequest()
				r.InstanceID = ""
				return r
			}(),
			valid: false,
		},
	}

	for _, tc := range cases {
		err := validateGetBackupsRequest(tc.request)
		if err != nil {
			if tc.valid {
				t.Errorf("%v: expected valid, got error: %v", tc.name, err)
			}
		} else if !tc.valid {
			t.Errorf("%v: expected invalid, got valid", tc.name)
		}
	}
}
