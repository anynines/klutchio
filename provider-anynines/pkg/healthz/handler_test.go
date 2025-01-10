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

package healthz

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	apisv1 "github.com/anynines/klutchio/provider-anynines/apis/v1"
	a9stest "github.com/anynines/klutchio/provider-anynines/internal/controller/test"
)

func TestMain(m *testing.M) {
	if err := apisv1.AddToScheme(scheme.Scheme); err != nil {
		panic("failed to add API github.com/anynines/klutchio/provider-anynines/apis/v1 to scheme")
	}

	os.Exit(m.Run())
}

func TestHandleHealthz(t *testing.T) {
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	handleHealthz(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	expectedStatus := http.StatusOK
	expectedBody := "ok"

	if resp.StatusCode != expectedStatus {
		t.Fatalf("Expected status %v, got %v", expectedStatus, resp.StatusCode)
	}
	if string(body) != expectedBody {
		t.Fatalf("Expected body %v, got %v", expectedBody, string(body))
	}
}

func TestBackendHealthHandler(t *testing.T) {
	cases := map[string]struct {
		configs        []runtime.Object
		expectedStatus int
		expectedBody   string
	}{
		"no configs": {
			configs:        []runtime.Object{},
			expectedStatus: http.StatusOK,
			expectedBody:   "{}",
		},
		"single config (success)": {
			configs: []runtime.Object{
				a9stest.ProviderConfig(a9stest.Name[apisv1.ProviderConfig]("test-provider"),
					a9stest.WithProviderConfigHealth(
						a9stest.NewProviderConfigHealth(
							a9stest.HealthLastCheckTime(time.Now()),
							a9stest.HealthLastStatus(true),
							a9stest.HealthLastMessage("Available"),
						)),
				),
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"test-provider":"ok"}`,
		},
		"single config (failure)": {
			configs: []runtime.Object{
				a9stest.ProviderConfig(a9stest.Name[apisv1.ProviderConfig]("test-provider"),
					a9stest.WithProviderConfigHealth(
						a9stest.NewProviderConfigHealth(
							a9stest.HealthLastCheckTime(time.Now()),
							a9stest.HealthLastStatus(false),
							a9stest.HealthLastMessage("Something went wrong"),
						)),
				),
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"test-provider":"degraded"}`,
		},
		"one success, one failure": {
			configs: []runtime.Object{
				a9stest.ProviderConfig(a9stest.Name[apisv1.ProviderConfig]("a"),
					a9stest.WithProviderConfigHealth(
						a9stest.NewProviderConfigHealth(
							a9stest.HealthLastCheckTime(time.Now()),
							a9stest.HealthLastStatus(true),
							a9stest.HealthLastMessage("All good"),
						)),
				),
				a9stest.ProviderConfig(a9stest.Name[apisv1.ProviderConfig]("b"),
					a9stest.WithProviderConfigHealth(
						a9stest.NewProviderConfigHealth(
							a9stest.HealthLastCheckTime(time.Now()),
							a9stest.HealthLastStatus(false),
							a9stest.HealthLastMessage("Something went wrong"),
						)),
				),
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"a":"ok","b":"degraded"}`,
		},
	}

	for name, tc := range cases {
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			handler := backendHealthHandler{
				client: fake.NewClientBuilder().
					WithRuntimeObjects(tc.configs...).
					WithScheme(scheme.Scheme).
					Build(),
			}

			req := httptest.NewRequest("GET", "/backend-health", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			resp := w.Result()
			bodyBytes, _ := io.ReadAll(resp.Body)
			defer resp.Body.Close()

			body := strings.TrimSpace(string(bodyBytes))

			if resp.StatusCode != tc.expectedStatus {
				t.Fatalf("Expected status %v, got %v", tc.expectedStatus, resp.StatusCode)
			}
			if body != tc.expectedBody {
				t.Fatalf("Expected body %q, got %q", tc.expectedBody, body)
			}
		})
	}
}
