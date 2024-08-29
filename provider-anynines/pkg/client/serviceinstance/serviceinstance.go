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
	osbclient "github.com/anynines/klutch/clients/a9s-open-service-broker"
	v1 "github.com/anynines/klutch/provider-anynines/apis/serviceinstance/v1"
	"github.com/anynines/klutch/provider-anynines/pkg/utils"
	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

// The Service Broker API provides a generic abstraction for all Data Services so much of the logic
// could be reused for other Data Services. We would still likely want to separate controllers
// reconciling their respective managed resources for each Data Service due to each Data Service
// having a separate Service Broker which the provider will connect to.

// LateInitialize fills the empty fields of ServiceInstanceParameters if the corresponding
// fields are given in GetInstanceResponse.
func LateInitialize(spec *v1.ServiceInstanceParameters, meta osbclient.Metadata) {
	// AcceptsIncomplete must always be set since we always require asynchronous service operations.
	spec.AcceptsIncomplete = ptr.To[bool](true)
	spec.OrganizationGUID = LateInitializeString(spec.OrganizationGUID, meta.OrganizationGUID)
	spec.SpaceGUID = LateInitializeString(spec.SpaceGUID, meta.SpaceGUID)
	spec.Parameters = LateInitializeIntOrStringMap(spec.Parameters, ServiceBrokerParamsToKubernetes(meta.Parameters))
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

// LateInitializeIntOrStringMap implements late initialization for an IntOrString map type
func LateInitializeIntOrStringMap(s map[string]intstr.IntOrString, from map[string]intstr.IntOrString) map[string]intstr.IntOrString {
	if len(s) != 0 || len(from) == 0 {
		return s
	}
	return from
}

// GenerateObservation is used to produce an observation object from a Service Broker
// GetInstanceResponse
func GenerateObservation(in osbclient.GetInstanceResponse) v1.ServiceInstanceObservation {
	return v1.ServiceInstanceObservation{
		State:         in.State,
		ProvisionedAt: in.ProvisionedAt,
		DeletedAt:     in.DeletedAt,
		CreatedAt:     in.CreatedAt,
		UpdatedAt:     in.UpdatedAt,
		PlanID:        in.PlanGUID,
		ServiceID:     in.ServiceGUID,
		InstanceID:    in.GUIDAtTenant,
		Parameters:    ServiceBrokerParamsToKubernetes(in.Metadata.Parameters),
	}
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
		observed.Parameters = map[string]intstr.IntOrString{}
	}

	return cmp.Equal(*observed, spec), cmp.Diff(*observed, spec)
}

// ServiceBrokerParamsToKubernetes converts parameters from the format used by the service broker to
// the format used by our internal ServiceInstance API.
//
// The differences are:
//   - Keys on the service broker side are in underscore format (e.g. "max_connections"),
//     while in the ServiceInstance API we use camel case ("maxConnections").
//   - The service broker can in theory return parameters of any JSON type (expressed as `interface{}`),
//     while the ServiceInstance API only supports integers or strings.
func ServiceBrokerParamsToKubernetes(parameters map[string]interface{}) map[string]intstr.IntOrString {
	if parameters == nil {
		return nil
	}
	result := map[string]intstr.IntOrString{}
	for key, value := range parameters {
		key := utils.UnderscoreToCamelCase(key)
		if floatValue, ok := value.(float64); ok {
			// all JSON numbers are interpreted as float64
			intValue := int(floatValue)
			result[key] = intstr.FromInt(intValue)
		} else if stringValue, ok := value.(string); ok {
			result[key] = intstr.FromString(stringValue)
		}
		// discard other values.
	}
	return result
}

// KubernetesParamsToServiceBroker is the inverse of serviceBrokerParamsToKubernetes
func KubernetesParamsToServiceBroker(parameters map[string]intstr.IntOrString) map[string]interface{} {
	if parameters == nil {
		return nil
	}
	result := map[string]interface{}{}
	for key, value := range parameters {
		key := utils.CamelCaseToUnderscore(key)
		if value.Type == intstr.Int {
			result[key] = value.IntValue()
		} else {
			result[key] = value.StrVal
		}
	}
	return result
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
	// set parameters that are wanted
	for key, value := range expected {
		if gotValue, ok := observed[key]; ok && gotValue == value {
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
