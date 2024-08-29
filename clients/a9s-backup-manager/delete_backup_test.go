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
	"strconv"
	"testing"

	"k8s.io/utils/pointer"
)

func defaultDeleteBackupRequest() *DeleteBackupRequest {
	return &DeleteBackupRequest{
		InstanceID: "test-instance-id",
		BackupID:   pointer.Int(10),
	}
}

const successDeleteBackupRequestResponseBody = `{
	"message": "some msg"
  }`

const instanceNotFoundDeleteBackupResponseBody = `{
	"error": "NotFound",
	"description": "The instance test-instance was not found."
  }`

const backupLockedResponseBody = `{
	"error": "Locked",
	"description": "The backup 1 currently can't be deleted. It is used for a running restore. Please try again later."
}`

const backupNotDeletedBadRequest = `{
	"error": "BadRequest",
	"description": "The backup 1 could not be deleted inside the storage."
  }`

func TestDeleteBackup(t *testing.T) {

	cases := map[string]struct {
		request            *DeleteBackupRequest
		httpChecks         httpChecks
		httpReaction       httpReaction
		expectedResponse   *DeleteBackupResponse
		expectedErrMessage string
		expectedErr        error
	}{
		"invalidRequestMissingInstanceID": {
			request: &DeleteBackupRequest{
				InstanceID: "",
				BackupID:   pointer.Int(1),
			},
			expectedErrMessage: "instanceID is required",
		},
		"invalidRequestMissingBackupID": {
			request: &DeleteBackupRequest{
				InstanceID: "test-id",
				BackupID:   nil,
			},
			expectedErrMessage: "backupID is required",
		},
		"successDeleted": {
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   successDeleteBackupRequestResponseBody,
			},
			expectedResponse: &DeleteBackupResponse{
				Message: pointer.String("some msg"),
			},
		},
		"instanceNotFound": {
			request: &DeleteBackupRequest{
				InstanceID: "test-instance",
				BackupID:   pointer.Int(1),
			},
			httpReaction: httpReaction{
				status: http.StatusNotFound,
				body:   instanceNotFoundDeleteBackupResponseBody,
			},
			expectedErrMessage: "backup not found: Status: 404; ErrorMessage: NotFound; Description: The instance test-instance was not found.; ResponseError: <nil>",
		},
		"emptyBody": {
			httpReaction: httpReaction{
				status: http.StatusNotFound,
			},
			expectedErrMessage: "backup not found: Status: 404; ErrorMessage: <nil>; Description: <nil>; ResponseError: unexpected end of JSON input",
		},
		"restoreInProgress": {
			httpReaction: httpReaction{
				status: http.StatusLocked,
				body:   backupLockedResponseBody,
			},
			expectedErrMessage: "backup is locked because it is currently being restored: Status: 423; ErrorMessage: Locked; Description: The backup 1 currently can't be deleted. It is used for a running restore. Please try again later.; ResponseError: <nil>",
		},
		"delitionFailedBadRequest": {
			httpReaction: httpReaction{
				status: http.StatusBadRequest,
				body:   backupNotDeletedBadRequest,
			},
			expectedErrMessage: "backup file could not be deleted: Status: 400; ErrorMessage: BadRequest; Description: The backup 1 could not be deleted inside the storage.; ResponseError: <nil>",
		},
		"genericHttpError": {
			httpReaction: httpReaction{
				err: fmt.Errorf("http error"),
			},
			expectedErrMessage: "http error",
		},
		"malformedOkResponse": {
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   malformedResponse,
			},
			expectedErrMessage: "Status: 200; ErrorMessage: <nil>; Description: <nil>; ResponseError: unexpected end of JSON input",
		},
		"malformedErrorResponse": {
			httpReaction: httpReaction{
				status: http.StatusInternalServerError,
				body:   malformedResponse,
			},
			expectedErrMessage: "Status: 500; ErrorMessage: <nil>; Description: <nil>; ResponseError: unexpected end of JSON input",
		},
		"conventionalInternalFailureResponse": {
			httpReaction: httpReaction{
				status: http.StatusInternalServerError,
				body:   conventionalFailureResponseBody,
			},
			expectedErr: testHTTPStatusCodeError(),
		},
	}

	for name, tc := range cases {
		tc := tc
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if tc.request == nil {
				tc.request = defaultDeleteBackupRequest()
			}
			backupIdString := ""
			if tc.request.BackupID != nil {
				backupIdString = strconv.Itoa(*tc.request.BackupID)
			}
			if tc.httpChecks.URL == "" {
				tc.httpChecks.URL = "/instances/" + tc.request.InstanceID + "/backups/" + backupIdString
			}

			klient := newTestClient(t, name, tc.httpChecks, tc.httpReaction)
			var response *DeleteBackupResponse
			response, err := klient.DeleteBackup(tc.request)
			doResponseChecks(t, name, response, err, tc.expectedResponse, tc.expectedErrMessage, tc.expectedErr)

		})
		if tc.request == nil {
			tc.request = defaultDeleteBackupRequest()
		}
		backupIdString := ""
		if tc.request.BackupID != nil {
			backupIdString = strconv.Itoa(*tc.request.BackupID)
		}
		if tc.httpChecks.URL == "" {
			tc.httpChecks.URL = "/instances/" + tc.request.InstanceID + "/backups/" + backupIdString
		}

		klient := newTestClient(t, name, tc.httpChecks, tc.httpReaction)
		var response *DeleteBackupResponse
		response, err := klient.DeleteBackup(tc.request)
		doResponseChecks(t, name, response, err, tc.expectedResponse, tc.expectedErrMessage, tc.expectedErr)
	}
}

func TestValidateDeleteBackup(t *testing.T) {
	cases := []struct {
		name    string
		request *DeleteBackupRequest
		valid   bool
	}{
		{
			name:    "valid",
			request: defaultDeleteBackupRequest(),
			valid:   true,
		},
		{
			name: "missing instance ID",
			request: func() *DeleteBackupRequest {
				r := defaultDeleteBackupRequest()
				r.InstanceID = ""
				return r
			}(),
			valid: false,
		},
		{
			name: "missing backup ID",
			request: func() *DeleteBackupRequest {
				r := defaultDeleteBackupRequest()
				r.BackupID = nil
				return r
			}(),
			valid: false,
		},
	}

	for _, tc := range cases {
		err := validateDeleteBackupRequest(tc.request)
		if err != nil {
			if tc.valid {
				t.Errorf("%v: expected valid, got error: %v", tc.name, err)
			}
		} else if !tc.valid {
			t.Errorf("%v: expected invalid, got valid", tc.name)
		}
	}
}
