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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	apisv1 "github.com/anynines/klutch/provider-anynines/apis/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// handleHealthz is the readiness probe endpoint. It always returns success,
// the provider is considered to be healthy as soon as long as it can respond
// to HTTP requests.
func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("ok"))
	if err != nil {
		fmt.Printf("error writing byte: %v", err)
	}
}

// backendHealthHandler reports on healthiness of all ProviderConfigs.
//
// In the future other checks related to the health and reachability of
// dependent services may be added.
//
// The healthiness is not actually evaluated here. That is done by the
// confighealth controller, periodically.
type backendHealthHandler struct {
	client client.Client
}

func (h backendHealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status := map[string]string{}

	providerConfigList := apisv1.ProviderConfigList{}

	if err := h.client.List(ctx, &providerConfigList); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte("Failed to list provider configs"))
		if err != nil {
			fmt.Printf("error writing list provider configs: %v", err)
		}
		return
	}

	for item := range providerConfigList.Items {
		pc := &providerConfigList.Items[item]
		if err := configStatus(pc); err != nil {
			status[pc.Name] = "degraded"
		} else {
			status[pc.Name] = "ok"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(status)
	if err != nil {
		fmt.Printf("error encoding status: %v", err)
	}
}

func configStatus(pc *apisv1.ProviderConfig) error {
	if pc.Status.Health.LastCheckTime == nil {
		// check never ran. We don't report an error yet
		return nil
	}

	if !pc.Status.Health.LastStatus {
		// last check failed, return error
		return errors.New(pc.Status.Health.LastMessage)
	}

	return nil
}
