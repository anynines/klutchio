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
	"reflect"
	"testing"
	"time"

	"k8s.io/client-go/tools/cache"
)

const okCatalogBytes = `{
  "services": [{
    "name": "fake-service",
    "id": "acb56d7c-XXXX-XXXX-XXXX-feb140a59a66",
    "description": "fake service",
    "tags": ["tag1", "tag2"],
    "requires": ["route_forwarding"],
    "bindable": true,
    "instances_retrievable": true,
    "bindings_retrievable": true,
    "metadata": {
    	"a": "b",
    	"c": "d"
    },
    "dashboard_client": {
      "id": "398e2f8e-XXXX-XXXX-XXXX-19a71ecbcf64",
      "secret": "277cabb0-XXXX-XXXX-XXXX-7822c0a90e5d",
      "redirect_uri": "http://localhost:1234"
    },
    "plan_updateable": true,
    "plans": [{
      "name": "fake-plan-1",
      "id": "d3031751-XXXX-XXXX-XXXX-a42377d3320e",
      "description": "description1",
      "metadata": {
      	"b": "c",
      	"d": "e"
      }
    }]
  }]
}`

func okCatalogResponse() *CatalogResponse {
	return &CatalogResponse{
		Services: []Service{
			{
				ID:          "acb56d7c-XXXX-XXXX-XXXX-feb140a59a66",
				Name:        "fake-service",
				Description: "fake service",
				Tags: []string{
					"tag1",
					"tag2",
				},
				Requires: []string{
					"route_forwarding",
				},
				Bindable:             true,
				InstancesRetrievable: true,
				BindingsRetrievable:  true,
				PlanUpdatable:        truePtr(),
				Plans: []Plan{
					{
						ID:          "d3031751-XXXX-XXXX-XXXX-a42377d3320e",
						Name:        "fake-plan-1",
						Description: "description1",
						Metadata: map[string]interface{}{
							"b": "c",
							"d": "e",
						},
					},
				},
				DashboardClient: &DashboardClient{
					ID:          "398e2f8e-XXXX-XXXX-XXXX-19a71ecbcf64",
					Secret:      "277cabb0-XXXX-XXXX-XXXX-7822c0a90e5d",
					RedirectURI: "http://localhost:1234",
				},
				Metadata: map[string]interface{}{
					"a": "b",
					"c": "d",
				},
			},
		},
	}
}

const okCatalog2Bytes = `{
  "services": [{
    "name": "fake-service-2",
    "id": "fake-service-2-id",
    "description": "service-description-2",
    "bindable": false,
    "plans": [{
      "name": "fake-plan-2",
      "id": "fake-plan-2-id",
      "description": "description-2",
      "bindable": true
    }]
  }]
}`

func okCatalog2Response() *CatalogResponse {
	return &CatalogResponse{
		Services: []Service{
			{
				ID:          "fake-service-2-id",
				Name:        "fake-service-2",
				Description: "service-description-2",
				Bindable:    false,
				Plans: []Plan{
					{
						ID:          "fake-plan-2-id",
						Name:        "fake-plan-2",
						Description: "description-2",
						Bindable:    truePtr(),
					},
				},
			},
		},
	}
}

const schemaCatalogBytes = `{
  "services": [{
    "name": "fake-service",
    "id": "acb56d7c-XXXX-XXXX-XXXX-feb140a59a66",
    "description": "fake service",
    "tags": ["tag1", "tag2"],
    "requires": ["route_forwarding"],
    "bindable": true,
    "instances_retrievable": true,
    "bindings_retrievable": true,
    "metadata": {
    	"a": "b",
    	"c": "d"
    },
    "dashboard_client": {
      "id": "398e2f8e-XXXX-XXXX-XXXX-19a71ecbcf64",
      "secret": "277cabb0-XXXX-XXXX-XXXX-7822c0a90e5d",
      "redirect_uri": "http://localhost:1234"
    },
    "plan_updateable": true,
    "plans": [{
      "name": "fake-plan-1",
      "id": "d3031751-XXXX-XXXX-XXXX-a42377d3320e",
      "description": "description1",
      "metadata": {
      	"b": "c",
      	"d": "e"
      },
      "schemas": {
      	"service_instance": {
	  	  "create": {
	  	  	"parameters": {
	  		  "foo": "bar"
	  	  	}
	  	  },
	  	  "update": {
	  	  	"parameters": {
	  		  "baz": "zap"
	  	    }
	  	  }
      	},
      	"service_binding": {
      	  "create": {
	  	  	"parameters": {
      	  	  "zoo": "blu"
      	    }
      	  }
      	}
      }
    }]
  }]
}`

func schemaCatalogResponse() *CatalogResponse {
	catalog := okCatalogResponse()
	catalog.Services[0].Plans[0].Schemas = &Schemas{
		ServiceInstance: &ServiceInstanceSchema{
			Create: &InputParametersSchema{
				Parameters: map[string]interface{}{
					"foo": "bar",
				},
			},
			Update: &InputParametersSchema{
				Parameters: map[string]interface{}{
					"baz": "zap",
				},
			},
		},
		ServiceBinding: &ServiceBindingSchema{
			Create: &InputParametersSchema{
				Parameters: map[string]interface{}{
					"zoo": "blu",
				},
			},
		},
	}

	return catalog
}

const okCatalog215Bytes = `{
  "services": [{
    "name": "fake-service-2",
    "id": "fake-service-2-id",
    "description": "service-description-2",
    "bindable": false,
    "plans": [{
      "name": "fake-plan-2",
      "id": "fake-plan-2-id",
      "description": "description-2",
      "bindable": true,
      "plan_updateable": true,
      "maximum_polling_duration": 600,
      "maintenance_info": {
        "version": "1.2.3",
        "description": "Avast! Pieces o' madness are forever clear."
      }
    }]
  }]
}`

func okCatalog215Response() *CatalogResponse {
	response := okCatalog2Response()
	var duration int64 = 600
	response.Services[0].Plans[0].MaximumPollingDuration = &duration

	response.Services[0].Plans[0].PlanUpdateable = truePtr()
	response.Services[0].Plans[0].MaintenanceInfo = &MaintenanceInfo{
		Version:     "1.2.3",
		Description: "Avast! Pieces o' madness are forever clear.",
	}

	return response
}

func TestGetCatalogWithoutCache(t *testing.T) {
	cases := []struct {
		name               string
		version            APIVersion
		enableAlpha        bool
		httpReaction       httpReaction
		expectedResponse   *CatalogResponse
		expectedErrMessage string
		expectedErr        error
	}{
		{
			name: "success 1",
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   okCatalogBytes,
			},
			expectedResponse: okCatalogResponse(),
		},
		{
			name: "success 2",
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   okCatalog2Bytes,
			},
			expectedResponse: okCatalog2Response(),
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
			name:    "schemas included if API version >= 2.13",
			version: Version2_13(),
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   schemaCatalogBytes,
			},
			expectedResponse: schemaCatalogResponse(),
		},
		{
			name:    "schemas not included if API version < 2.13",
			version: Version2_12(),
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   schemaCatalogBytes,
			},
			expectedResponse: okCatalogResponse(),
		},
		{
			name:        "plan has its own updateable attribute, max polling duration and maintenance info",
			version:     LatestAPIVersion(),
			enableAlpha: true,
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   okCatalog215Bytes,
			},
			expectedResponse: okCatalog215Response(),
		},
		{
			name:        "alpha disabled: plan has its own updateable attribute, max polling duration and maintenance info",
			version:     LatestAPIVersion(),
			enableAlpha: false,
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   okCatalog215Bytes,
			},
			expectedResponse: okCatalog2Response(),
		},
	}

	for _, tc := range cases {
		httpChecks := httpChecks{
			URL: "/v2/catalog",
		}

		if tc.version.label == "" {
			tc.version = Version2_11()
		}

		klient := newTestClient(t, tc.name, tc.version, tc.enableAlpha, httpChecks, tc.httpReaction)

		response, err := klient.GetCatalog()

		doResponseChecks(t, tc.name, response, err, tc.expectedResponse, tc.expectedErrMessage, tc.expectedErr)
	}
}

func TestGetCatalogWithCache(t *testing.T) {
	t.Parallel()

	// the fields with the suffix "AfterStaleness" are only used in the cases where "waitForStaleness" is set to true
	cases := []struct {
		alternateHttpReaction            httpReaction
		expectedErrAfterStaleness        error
		expectedErrMessageAfterStaleness string
		expectedResponseAfterStaleness   *CatalogResponse
		enableAlpha                      bool
		expectedErr                      error
		expectedErrMessage               string
		expectedResponse                 *CatalogResponse
		httpReaction                     httpReaction
		name                             string
		version                          APIVersion

		waitForStaleness bool
	}{
		{
			name: "success 1",
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   okCatalogBytes,
			},
			expectedResponse: okCatalogResponse(),
		},
		{
			name: "success 2",
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   okCatalog2Bytes,
			},
			expectedResponse: okCatalog2Response(),
		},
		{
			name:    "schemas included if API version >= 2.13",
			version: Version2_13(),
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   schemaCatalogBytes,
			},
			expectedResponse: schemaCatalogResponse(),
		},
		{
			name:    "schemas not included if API version < 2.13",
			version: Version2_12(),
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   schemaCatalogBytes,
			},
			expectedResponse: okCatalogResponse(),
		},
		{
			name:        "plan has its own updateable attribute, max polling duration and maintenance info",
			version:     LatestAPIVersion(),
			enableAlpha: true,
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   okCatalog215Bytes,
			},
			expectedResponse: okCatalog215Response(),
		},
		{
			name:        "alpha disabled: plan has its own updateable attribute, max polling duration and maintenance info",
			version:     LatestAPIVersion(),
			enableAlpha: false,
			httpReaction: httpReaction{
				status: http.StatusOK,
				body:   okCatalog215Bytes,
			},
			expectedResponse: okCatalog2Response(),
		},
		{
			name:             "stale cache and broker reachable",
			expectedResponse: okCatalogResponse(),
			httpReaction:     httpReaction{status: http.StatusOK, body: okCatalogBytes},
			waitForStaleness: true,
			alternateHttpReaction: httpReaction{
				status: http.StatusOK,
				body:   okCatalog2Bytes,
			},
			expectedResponseAfterStaleness: okCatalog2Response(),
		},
		{
			name:                             "stale cache and broker unreachable",
			expectedErrMessageAfterStaleness: "Status: 0; ErrorMessage: <nil>; Description: <nil>; ResponseError: <nil>",
			expectedResponse:                 okCatalogResponse(),
			httpReaction:                     httpReaction{status: http.StatusOK, body: okCatalogBytes},
			waitForStaleness:                 true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testHttpChecks := httpChecks{
				URL: "/v2/catalog",
			}

			if tc.version.label == "" {
				tc.version = Version2_11()
			}

			if reflect.DeepEqual(tc.alternateHttpReaction, httpReaction{}) {
				tc.alternateHttpReaction = httpReaction{body: "{}"}
			}

			klient := newTestClient(t, tc.name, tc.version, tc.enableAlpha, testHttpChecks, tc.httpReaction)
			klient.cache = cache.NewTTLStore(cacheKeyFunc, 1*time.Second)

			// fill up cache
			response, err := klient.GetCatalog()
			doResponseChecks(t, tc.name, response, err, tc.expectedResponse, tc.expectedErrMessage, tc.expectedErr)

			// disable server response
			klient.doRequestFunc = doHTTP(t, "empty", httpChecks{}, tc.alternateHttpReaction)
			if tc.waitForStaleness {
				time.Sleep(2 * time.Second)

				// read from cache
				response, err = klient.GetCatalog()
				doResponseChecks(t, tc.name, response, err, tc.expectedResponseAfterStaleness, tc.expectedErrMessageAfterStaleness, tc.expectedErrAfterStaleness)
			} else {
				// read from cache
				response, err = klient.GetCatalog()
				doResponseChecks(t, tc.name, response, err, tc.expectedResponse, tc.expectedErrMessage, tc.expectedErr)
			}
		})
	}
}
