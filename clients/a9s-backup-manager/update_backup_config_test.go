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

func defaultUpdateBackupConfigRequest() *UpdateBackupConfigRequest {
	return &UpdateBackupConfigRequest{
		InstanceID:            "test-instance-id",
		ExcludeFromAutoBackup: pointer.Bool(true),
	}
}

func TestUpdateBackupConfig(t *testing.T) {
	cases := []struct {
		name               string
		request            *UpdateBackupConfigRequest
		httpChecks         httpChecks
		httpReaction       httpReaction
		expectedResponse   *UpdateBackupConfigResponse
		expectedErrMessage string
		expectedErr        error
	}{
		{
			name: "invalid request",
			request: &UpdateBackupConfigRequest{
				InstanceID: "",
			},
			expectedErrMessage: "instanceID is required",
		},
		{
			name: "success - update EncryptionKey",
			request: &UpdateBackupConfigRequest{
				InstanceID:    "test-instance-id",
				EncryptionKey: pointer.String("test"),
			},
			httpChecks: httpChecks{
				body: `{"encryption_key":"test"}`,
			},
			httpReaction: httpReaction{
				status: http.StatusCreated,
				body:   `{"message": "instance updated"}`,
			},
			expectedResponse: &UpdateBackupConfigResponse{
				Message: pointer.String("instance updated"),
			},
		},
		{
			name: "success - update ExcludeFromAutoBackup",
			request: &UpdateBackupConfigRequest{
				InstanceID:            "test-instance-id",
				ExcludeFromAutoBackup: pointer.Bool(true),
			},
			httpChecks: httpChecks{
				body: `{"exclude_from_auto_backup":true}`,
			},
			httpReaction: httpReaction{
				status: http.StatusCreated,
				body:   `{"message": "instance updated"}`,
			},
			expectedResponse: &UpdateBackupConfigResponse{
				Message: pointer.String("instance updated"),
			},
		},
		{
			name: "success - update CredentialsUpdatedByUser",
			request: &UpdateBackupConfigRequest{
				InstanceID:               "test-instance-id",
				CredentialsUpdatedByUser: pointer.Bool(true),
			},
			httpChecks: httpChecks{
				body: `{"credentials_updated_by_user":true}`,
			},
			httpReaction: httpReaction{
				status: http.StatusCreated,
				body:   `{"message": "instance updated"}`,
			},
			expectedResponse: &UpdateBackupConfigResponse{
				Message: pointer.String("instance updated"),
			},
		},
		{
			name: "success - update ExcludeFromAutoBackup & CredentialsUpdatedByUser",
			request: &UpdateBackupConfigRequest{
				InstanceID:               "test-instance-id",
				ExcludeFromAutoBackup:    pointer.Bool(true),
				CredentialsUpdatedByUser: pointer.Bool(true),
			},
			httpChecks: httpChecks{
				body: `{"exclude_from_auto_backup":true,"credentials_updated_by_user":true}`,
			},
			httpReaction: httpReaction{
				status: http.StatusCreated,
				body:   `{"message": "instance updated"}`,
			},
			expectedResponse: &UpdateBackupConfigResponse{
				Message: pointer.String("instance updated"),
			},
		},
		{
			name: "success - update ExcludeFromAutoBackup & EncryptionKey",
			request: &UpdateBackupConfigRequest{
				InstanceID:            "test-instance-id",
				ExcludeFromAutoBackup: pointer.Bool(true),
				EncryptionKey:         pointer.String("test"),
			},
			httpChecks: httpChecks{
				body: `{"encryption_key":"test","exclude_from_auto_backup":true}`,
			},
			httpReaction: httpReaction{
				status: http.StatusCreated,
				body:   `{"message": "instance updated"}`,
			},
			expectedResponse: &UpdateBackupConfigResponse{
				Message: pointer.String("instance updated"),
			},
		},
		{
			name: "success - update CredentialsUpdatedByUser & EncryptionKey",
			request: &UpdateBackupConfigRequest{
				InstanceID:               "test-instance-id",
				CredentialsUpdatedByUser: pointer.Bool(true),
				EncryptionKey:            pointer.String("test"),
			},
			httpChecks: httpChecks{
				body: `{"encryption_key":"test","credentials_updated_by_user":true}`,
			},
			httpReaction: httpReaction{
				status: http.StatusCreated,
				body:   `{"message": "instance updated"}`,
			},
			expectedResponse: &UpdateBackupConfigResponse{
				Message: pointer.String("instance updated"),
			},
		},
		{
			name: "success - update ExcludeFromAutoBackup & CredentialsUpdatedByUser & EncryptionKey",
			request: &UpdateBackupConfigRequest{
				InstanceID:               "test-instance-id",
				ExcludeFromAutoBackup:    pointer.Bool(true),
				CredentialsUpdatedByUser: pointer.Bool(true),
				EncryptionKey:            pointer.String("test"),
			},
			httpChecks: httpChecks{
				body: `{"encryption_key":"test","exclude_from_auto_backup":true,"credentials_updated_by_user":true}`,
			},
			httpReaction: httpReaction{
				status: http.StatusCreated,
				body:   `{"message": "instance updated"}`,
			},
			expectedResponse: &UpdateBackupConfigResponse{
				Message: pointer.String("instance updated"),
			},
		},
		{
			name: "instance not found",
			request: &UpdateBackupConfigRequest{
				InstanceID:            "test-instance",
				ExcludeFromAutoBackup: pointer.Bool(true),
			},
			httpChecks: httpChecks{
				body: `{"exclude_from_auto_backup":true}`,
			},
			httpReaction: httpReaction{
				status: http.StatusNotFound,
				body:   instanceNotFoundCreateBackupResponseBody,
			},
			expectedErrMessage: "instance not found: Status: 404; ErrorMessage: NotFound; Description: The instance test-instance was not found.; ResponseError: <nil>",
		},
		{
			name: "empty body",
			request: &UpdateBackupConfigRequest{
				InstanceID: "test-instance",
			},
			httpReaction: httpReaction{
				status: http.StatusNotFound,
				body:   instanceNotFoundCreateBackupResponseBody,
			},
			expectedErrMessage: "at least one property must be set",
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
			tc.request = defaultUpdateBackupConfigRequest()
			tc.httpChecks.body = `{"exclude_from_auto_backup":true}`
		}

		if tc.httpChecks.URL == "" {
			tc.httpChecks.URL = "/instances/" + tc.request.InstanceID
		}

		klient := newTestClient(t, tc.name, tc.httpChecks, tc.httpReaction)

		response, err := klient.UpdateBackupConfig(tc.request)

		doResponseChecks(t, tc.name, response, err, tc.expectedResponse, tc.expectedErrMessage, tc.expectedErr)
	}
}

func TestValidateUpdateBackupConfig(t *testing.T) {
	cases := []struct {
		name    string
		request *UpdateBackupConfigRequest
		valid   bool
	}{
		{
			name: "with ExcludeFromAutoBackup value set",
			request: func() *UpdateBackupConfigRequest {
				return &UpdateBackupConfigRequest{
					InstanceID:            "test-instance-id",
					ExcludeFromAutoBackup: pointer.Bool(true),
				}
			}(),
			valid: true,
		},
		{
			name: "with CredentialsUpdatedByUser value set",
			request: func() *UpdateBackupConfigRequest {
				return &UpdateBackupConfigRequest{
					InstanceID:               "test-instance-id",
					CredentialsUpdatedByUser: pointer.Bool(true),
				}
			}(),
			valid: true,
		},
		{
			name: "with EncryptionKey value set",
			request: func() *UpdateBackupConfigRequest {
				return &UpdateBackupConfigRequest{
					InstanceID:    "test-instance-id",
					EncryptionKey: pointer.String("asd"),
				}
			}(),
			valid: true,
		},
		{
			name: "missing instance ID",
			request: func() *UpdateBackupConfigRequest {
				return &UpdateBackupConfigRequest{
					InstanceID: "",
				}
			}(),
			valid: false,
		},
		{
			name: "no field updated",
			request: func() *UpdateBackupConfigRequest {
				return &UpdateBackupConfigRequest{
					InstanceID: "test-instance-id",
				}
			}(),
			valid: false,
		},
	}

	for _, tc := range cases {
		err := validateUpdateBackupConfigRequest(tc.request)
		if err != nil {
			if tc.valid {
				t.Errorf("%v: expected valid, got error: %v", tc.name, err)
			}
		} else if !tc.valid {
			t.Errorf("%v: expected invalid, got valid", tc.name)
		}
	}
}
