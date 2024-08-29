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

func defaultCreateBackupRequest() *CreateBackupRequest {
	return &CreateBackupRequest{
		InstanceID: "test-instance-id",
	}
}

const successCreateBackupRequestResponseBody = `{
	"id": 1,
	"message": "some msg"
  }`

const instanceNotFoundCreateBackupResponseBody = `{
	"error": "NotFound",
	"description": "The instance test-instance was not found."
  }`

func TestCreateBackup(t *testing.T) {
	cases := []struct {
		name               string
		request            *CreateBackupRequest
		httpChecks         httpChecks
		httpReaction       httpReaction
		expectedResponse   *CreateBackupResponse
		expectedErrMessage string
		expectedErr        error
	}{
		{
			name: "invalid request",
			request: &CreateBackupRequest{
				InstanceID: "",
			},
			expectedErrMessage: "instanceID is required",
		},
		{
			name: "success - created",
			httpReaction: httpReaction{
				status: http.StatusCreated,
				body:   successCreateBackupRequestResponseBody,
			},
			expectedResponse: &CreateBackupResponse{
				BackupID: pointer.Int(1),
				Message:  pointer.String("some msg"),
			},
		},
		{
			name: "instance not found",
			request: &CreateBackupRequest{
				InstanceID: "test-instance",
			},
			httpReaction: httpReaction{
				status: http.StatusNotFound,
				body:   instanceNotFoundCreateBackupResponseBody,
			},
			expectedErrMessage: "instance not found: Status: 404; ErrorMessage: NotFound; Description: The instance test-instance was not found.; ResponseError: <nil>",
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
			tc.request = defaultCreateBackupRequest()
		}

		if tc.httpChecks.URL == "" {
			tc.httpChecks.URL = "/instances/" + tc.request.InstanceID + "/backups"
		}

		klient := newTestClient(t, tc.name, tc.httpChecks, tc.httpReaction)

		response, err := klient.CreateBackup(tc.request)

		doResponseChecks(t, tc.name, response, err, tc.expectedResponse, tc.expectedErrMessage, tc.expectedErr)
	}
}

func TestValidateCreateBackup(t *testing.T) {
	cases := []struct {
		name    string
		request *CreateBackupRequest
		valid   bool
	}{
		{
			name:    "valid",
			request: defaultCreateBackupRequest(),
			valid:   true,
		},
		{
			name: "missing instance ID",
			request: func() *CreateBackupRequest {
				r := defaultCreateBackupRequest()
				r.InstanceID = ""
				return r
			}(),
			valid: false,
		},
	}

	for _, tc := range cases {
		err := validateCreateBackupRequest(tc.request)
		if err != nil {
			if tc.valid {
				t.Errorf("%v: expected valid, got error: %v", tc.name, err)
			}
		} else if !tc.valid {
			t.Errorf("%v: expected invalid, got valid", tc.name)
		}
	}
}
