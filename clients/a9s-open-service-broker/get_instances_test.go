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

const okInstancesBytes = `{
  "total_results": 2,
  "total_pages": 1,
  "current_page": 1,
  "prev_url": null,
  "next_url": null,
  "resources": [
    {
      "id": 18,
      "plan_guid": "4241102f-a04e-41b4-851c-8cee51fa9369",
      "service_guid": "d88b827c-3175-4800-9196-c517370eea99",
      "context": {
        "organization_guid": "a1d46b5c-b639-4f43-85c7-e9a0e5f01f75",
        "space_guid": "1bf71cf3-9017-4846-bffc-b9b31872bfaf",
		 "parameters": {
          "client_min_messages": "DEBUG1"
        }
      },
      "dashboard_url": "https://postgresql-ms-1706599150.system.aws-s1.a9s-ops.de/service-instances/261a5e05-7312-4d61-bab8-30be9ff45a13-1706777923",
      "state": "deleted",
      "deployment_name": "ncd6394ee",
      "guid_at_tenant": "261a5e05-7312-4d61-bab8-30be9ff45a13-1706777923",
      "tenant_id": "anynines",
      "provisioned_at": "2024-02-01T10:33:34.026Z",
      "deleted_at": "2024-02-01T10:48:31.795Z",
      "created_at": "2024-02-01T08:58:44.123Z",
      "updated_at": "2024-02-01T10:48:31.807Z",
      "allowed_update_time": {
        "type": "general",
        "allowed": true
      },
      "is_locked": false,
      "credentials": []
    },
    {
      "id": 1,
      "plan_guid": "9b4a988a-134d-4563-bd2e-ce7cd0f850ad",
      "service_guid": "0b87d5f0-c1f3-482b-9050-742dd8280208",
      "context": {
        "organization_guid": "a1d46b5c-b639-4f43-85c7-e9a0e5f01f75",
        "space_guid": "1bf71cf3-9017-4846-bffc-b9b31872bfaf",
        "parameters": {}
      },
      "dashboard_url": "https://postgresql-ms-1706599150.system.aws-s1.a9s-ops.de/service-instances/76c63662-5b66-4d8f-b079-423ce962bc43-1706599907",
      "state": "provisioned",
      "deployment_name": "ncd98aba",
      "guid_at_tenant": "76c63662-5b66-4d8f-b079-423ce962bc43-1706599907",
      "tenant_id": "anynines",
      "provisioned_at": "2024-01-30T08:15:35.202Z",
      "created_at": "2024-01-30T07:31:48.614Z",
      "updated_at": "2024-01-30T08:17:36.190Z",
      "deleted": null,
      "allowed_update_time": {
        "type": "general",
        "allowed": true
      },
      "is_locked": false,
      "credentials": []
    }
  ]
}`

func okGetInstancesResponse() *GetInstancesResponse {
	response := &GetInstancesResponse{
		TotalResults: 2,
		TotalPages:   1,
		CurrentPage:  1,
		PrevURL:      nil,
		NextURL:      nil,
		Resources: []GetInstanceResponse{
			{
				ID:          18,
				PlanGUID:    "4241102f-a04e-41b4-851c-8cee51fa9369",
				ServiceGUID: "d88b827c-3175-4800-9196-c517370eea99",
				Context: Context{
					Parameters: map[string]interface{}{
						"client_min_messages": "DEBUG1",
					},
					OrganizationGUID: "a1d46b5c-b639-4f43-85c7-e9a0e5f01f75",
					SpaceGUID:        "1bf71cf3-9017-4846-bffc-b9b31872bfaf",
				},
				DashboardURL:   "https://postgresql-ms-1706599150.system.aws-s1.a9s-ops.de/service-instances/261a5e05-7312-4d61-bab8-30be9ff45a13-1706777923",
				DeploymentName: "ncd6394ee",
				State:          "deleted",
				GUIDAtTenant:   "261a5e05-7312-4d61-bab8-30be9ff45a13-1706777923",
				TenantID:       "anynines",
				ProvisionedAt:  "2024-02-01T10:33:34.026Z",
				DeletedAt:      "2024-02-01T10:48:31.795Z",
				CreatedAt:      "2024-02-01T08:58:44.123Z",
				UpdatedAt:      "2024-02-01T10:48:31.807Z",
				Credentials:    []Credential{},
			},
			{
				ID:          1,
				PlanGUID:    "9b4a988a-134d-4563-bd2e-ce7cd0f850ad",
				ServiceGUID: "0b87d5f0-c1f3-482b-9050-742dd8280208",
				Context: Context{
					Parameters:       map[string]interface{}{},
					OrganizationGUID: "a1d46b5c-b639-4f43-85c7-e9a0e5f01f75",
					SpaceGUID:        "1bf71cf3-9017-4846-bffc-b9b31872bfaf",
				},
				DashboardURL:   "https://postgresql-ms-1706599150.system.aws-s1.a9s-ops.de/service-instances/76c63662-5b66-4d8f-b079-423ce962bc43-1706599907",
				DeploymentName: "ncd98aba",
				State:          "provisioned",
				GUIDAtTenant:   "76c63662-5b66-4d8f-b079-423ce962bc43-1706599907",
				TenantID:       "anynines",
				ProvisionedAt:  "2024-01-30T08:15:35.202Z",
				CreatedAt:      "2024-01-30T07:31:48.614Z",
				UpdatedAt:      "2024-01-30T08:17:36.190Z",
				Credentials:    []Credential{},
			},
		},
	}
	return response
}

func TestGetInstances(t *testing.T) {
	cases := []struct {
		name               string
		enableAlpha        bool
		APIVersion         APIVersion
		httpReaction       httpReaction
		expectedResponse   *GetInstancesResponse
		expectedErrMessage string
		expectedErr        error
	}{
		{
			name: "success",
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   okInstancesBytes,
			},
			expectedResponse: okGetInstancesResponse(),
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
			expectedErrMessage: "GetInstance not allowed: operation not allowed: must have API version >= 2.14. Current: 2.13",
		},
	}

	for _, tc := range cases {
		httpChecks := httpChecks{
			URL: "/instances",
		}

		if tc.APIVersion.label == "" {
			tc.APIVersion = LatestAPIVersion()
		}

		klient := newTestClient(t, tc.name, tc.APIVersion, tc.enableAlpha, httpChecks, tc.httpReaction)

		response, err := klient.GetInstances()

		doResponseChecks(t, tc.name, response, err, tc.expectedResponse, tc.expectedErrMessage, tc.expectedErr)
	}
}
