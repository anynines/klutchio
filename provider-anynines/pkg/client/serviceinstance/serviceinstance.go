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

package serviceinstance

import (
	"bytes"
	"encoding/json"
	"fmt"

	osbclient "github.com/anynines/klutchio/clients/a9s-open-service-broker"
	v1 "github.com/anynines/klutchio/provider-anynines/apis/serviceinstance/v1"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"k8s.io/utils/ptr"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// The Service Broker API provides a generic abstraction for all Data Services so much of the logic
// could be reused for other Data Services. We would still likely want to separate controllers
// reconciling their respective managed resources for each Data Service due to each Data Service
// having a separate Service Broker which the provider will connect to.

// LateInitialize fills the empty fields of ServiceInstanceParameters if the corresponding
// fields are given in GetInstanceResponse.
func LateInitialize(spec *v1.ServiceInstanceParameters, meta osbclient.Metadata) error {
	// AcceptsIncomplete must always be set since we always require asynchronous service operations.
	spec.AcceptsIncomplete = ptr.To(true)
	spec.OrganizationGUID = LateInitializeString(spec.OrganizationGUID, meta.OrganizationGUID)
	spec.SpaceGUID = LateInitializeString(spec.SpaceGUID, meta.SpaceGUID)

	params, err := ServiceBrokerParamsToKubernetes(meta.Parameters)
	if err != nil {
		return err
	}

	spec.Parameters = LateInitializeJsonMap(spec.Parameters, params)

	return nil
}

// LateInitializeString implements late initialization for string type.
func LateInitializeString(s *string, from string) *string {
	if s != nil || from == "" {
		return s
	}
	return &from
}

// LateInitializeString implements late initialization for bool type.
func LateInitializeBool(b *bool, from bool) *bool {
	if b != nil || !from {
		return b
	}
	return &from
}

// LateInitializeIntOrStringMap implements late initialization for a RawExtension map type
func LateInitializeJsonMap(s map[string]apiextv1.JSON, from map[string]apiextv1.JSON) map[string]apiextv1.JSON {
	if len(s) != 0 || len(from) == 0 {
		return s
	}
	return from
}

// GenerateObservation is used to produce an observation object from a Service Broker
// GetInstanceResponse
func GenerateObservation(in osbclient.GetInstanceResponse) (v1.ServiceInstanceObservation, error) {
	params, err := ServiceBrokerParamsToKubernetes(in.Metadata.Parameters)
	if err != nil {
		return v1.ServiceInstanceObservation{}, fmt.Errorf("cannot generate observation: %w", err)
	}

	return v1.ServiceInstanceObservation{
		State:         in.State,
		ProvisionedAt: in.ProvisionedAt,
		DeletedAt:     in.DeletedAt,
		CreatedAt:     in.CreatedAt,
		UpdatedAt:     in.UpdatedAt,
		PlanID:        in.PlanGUID,
		ServiceID:     in.ServiceGUID,
		InstanceID:    in.GUIDAtTenant,
		Parameters:    params,
	}, nil
}

// SpecMatchesObservedState checks whether current state is up-to-date compared to the given set of parameters.
func SpecMatchesObservedState(spec v1.ServiceInstanceParameters, in osbclient.GetInstanceResponse) (bool, string) {
	// We pre-fill these values for the ServiceName and the PlanName into the struct because they
	// are not part of the observation response we get from the Service broker and therefore these
	// field would be nil in the variable "observed". This would in turn lead the provider to assume
	// that the k8s object and the service instance at the provider are out of sync when in reality
	// they might not be.
	observed := &v1.ServiceInstanceParameters{
		ServiceName: spec.ServiceName,
		PlanName:    spec.PlanName,
	}

	LateInitialize(observed, in.Metadata)

	if spec.Parameters != nil && observed.Parameters == nil {
		observed.Parameters = map[string]apiextv1.JSON{}
	}

	return cmp.Equal(*observed, spec), cmp.Diff(*observed, spec)
}

// ServiceBrokerParamsToKubernetes converts parameters from the format used by the service broker to
// the format used by our internal ServiceInstance API.
func ServiceBrokerParamsToKubernetes(parameters map[string]interface{}) (map[string]apiextv1.JSON, error) {
	if parameters == nil {
		return nil, nil
	}

	result := map[string]apiextv1.JSON{}

	for key, value := range parameters {
		b, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal key %q: %w", key, err)
		}

		result[key] = apiextv1.JSON{Raw: b}
	}

	return result, nil
}

// KubernetesParamsToServiceBroker is the inverse of serviceBrokerParamsToKubernetes
func KubernetesParamsToServiceBroker(parameters map[string]apiextv1.JSON) (map[string]interface{}, error) {
	if parameters == nil {
		return nil, nil
	}

	result := map[string]interface{}{}
	for key, value := range parameters {
		var v interface{}
		dec := json.NewDecoder(bytes.NewReader(value.Raw))
		err := dec.Decode(&v)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal key %q: %w", key, err)
		}

		result[key] = v
	}

	return result, nil
}

// ParameterUpdateForBroker takes two sets of broker parameters, compares them and computes
// an update to be sent to the service broker to turn `observed` parameters into `expected`.
//
// If observed and expected is equal, returns `nil`.
// If a key is present in `observed`, but not in `observed`, sets the key to `nil` in the result.
// This signals to the service broker that the parameter is supposed to be removed (or reset to default).
func ParameterUpdateForBroker(observed map[string]interface{}, expected map[string]interface{}) map[string]interface{} {
	update := map[string]interface{}{}
	// update parameters that were removed by setting them to `null`
	for key := range observed {
		if _, ok := expected[key]; !ok {
			update[key] = nil
		}
	}

	// we want to ignore the order of elements when comparing arrays, in case the broker returns
	// parameters in a different order during an observation.
	var diffOpts = []cmp.Option{
		cmpopts.SortSlices(func(a, b interface{}) bool {
			aj, _ := json.Marshal(a)
			bj, _ := json.Marshal(b)
			return string(aj) < string(bj)
		}),
		cmpopts.EquateEmpty(),
	}

	// set parameters that are wanted
	for key, value := range expected {
		if gotValue, ok := observed[key]; ok && cmp.Equal(gotValue, value, diffOpts...) {
			// ... unless they are already set to the same value
			continue
		}
		update[key] = value
	}
	if len(update) == 0 {
		return nil
	}
	return update
}
