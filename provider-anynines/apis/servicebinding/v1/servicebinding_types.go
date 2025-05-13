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

package v1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

type ServiceBindingParameters struct {
	// InstanceName is the name of the claim owning the instance to bind to.
	InstanceName string `json:"instanceName"`

	// AcceptsIncomplete requires a client API version >= 2.14.
	//
	// AcceptsIncomplete indicates whether the client can accept asynchronous
	// binding. If the broker cannot fulfill a request synchronously and
	// AcceptsIncomplete is set to false, the broker will reject the request. A
	// broker may choose to response to a request with AcceptsIncomplete set to
	// true either synchronously or asynchronously.
	AcceptsIncomplete bool `json:"acceptsIncomplete"`

	// Deprecated; use bind_resource.app_guid to send this value instead.
	// This field will never be used but for completeness reasons we keep it.
	AppGUID *string `json:"appGuid,omitempty"`

	// BindResource holds extra information about a binding. Optional, but
	// it's complicated.
	BindResource *BindResource `json:"bindResource,omitempty"`

	// Parameters is configuration parameters for the binding. Optional.
	// Parameters are currently unsupported.
	Parameters map[string]string `json:"parameters,omitempty"`

	// Context requires a client API version >= 2.13.
	//
	// Context is platform-specific contextual information under which the
	// service binding is to be created.
	Context map[string]string `json:"context,omitempty"`

	// OriginatingIdentity requires a client API version >= 2.13.
	//
	// OriginatingIdentity is the identity on the platform of the user making
	// this request.
	OriginatingIdentity *OriginatingIdentity `json:"originatingIdentity,omitempty"`

	// Credentials is a free-form hash of credentials that can be used by
	// applications or users to access the service.
	Credentials map[string]string `json:"credentials,omitempty"`
}

// BindResource contains data for platform resources associated with a
// binding.
type BindResource struct {
	AppGUID *string `json:"appGuid,omitempty"`

	Route *string `json:"route,omitempty"`
}

// OriginatingIdentity requires a client API version >=2.13.
//
// OriginatingIdentity is used to pass to the broker service an identity from
// the platform
type OriginatingIdentity struct {
	// The name of the platform to which the user belongs
	Platform string `json:"platform,omitempty"`

	// A serialized JSON object that describes the user in a way that makes
	// sense to the platform
	Value string `json:"value,omitempty"`
}

// ServiceBindingObservation are the observable fields of a ServiceBinding.
type ServiceBindingObservation struct {
	// +kubebuilder:default:=Pending
	// +required
	State string `json:"state"`
	// +optional
	ServiceBindingID int `json:"serviceBindingID,omitempty"`

	// InstanceID is the ID of the data service instance to bind.
	InstanceID string `json:"instanceId,omitempty"`

	// PlanID is the Plan ID of the data service instance.
	PlanID string `json:"planID,omitempty"`

	// ServiceID is the Service ID of the data service instance.
	ServiceID string `json:"serviceID,omitempty"`

	// ConnectionDetails is a struct that contains the network details of the data service instance.
	ConnectionDetails []ConnectionDetails `json:"connectionDetails,omitempty"`
}

// ConnectionDetails contains the network details required for connecting to the data service instance.
type ConnectionDetails struct {
	// HostURL is the URL used to connect with the data service instance.
	HostURL string `json:"hostURL,omitempty"`

	// Port is the Port used to connect with the data service instance.
	Port string `json:"port,omitempty"`
}

// A ServiceBindingSpec defines the desired state of a ServiceBinding.
type ServiceBindingSpec struct {
	xpv1.ResourceSpec `json:",inline"`

	// Instance identifies the Data Service Instance that the ServiceBinding binds to.
	ForProvider ServiceBindingParameters `json:"forProvider"`
}

// A ServiceBindingStatus represents the observed state of a ServiceBinding.
type ServiceBindingStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ServiceBindingObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,anynines}
type ServiceBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceBindingSpec   `json:"spec"`
	Status ServiceBindingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServiceBindingList contains a list of ServiceBinding
type ServiceBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceBinding `json:"items"`
}

// ServiceBinding type metadata.
var (
	ServiceBindingKind             = reflect.TypeOf(ServiceBinding{}).Name()
	ServiceBindingGroupKind        = schema.GroupKind{Group: Group, Kind: ServiceBindingKind}.String()
	ServiceBindingKindAPIVersion   = ServiceBindingKind + "." + SchemeGroupVersion.String()
	ServiceBindingGroupVersionKind = SchemeGroupVersion.WithKind(ServiceBindingKind)
)

func init() {
	SchemeBuilder.Register(&ServiceBinding{}, &ServiceBindingList{})
}

func (sbo *ServiceBindingObservation) HasMissingFields() bool {
	return sbo.InstanceID == "" ||
		sbo.ServiceID == "" ||
		sbo.PlanID == ""
}

func (sb *ServiceBinding) AddConnectionDetails(host, port string) {
	sb.Status.AtProvider.ConnectionDetails = append(sb.Status.AtProvider.ConnectionDetails, ConnectionDetails{host, port})
}

func (sb *ServiceBinding) ConnectionDetailsIsNotEmpty() bool {
	return len(sb.Status.AtProvider.ConnectionDetails) > 0
}

func (sb *ServiceBinding) SetDeletionStatusIfNotDeleted(status string) {
	if sb.GetDeletionTimestamp() != nil {
		sb.Status.SetConditions(xpv1.Deleting())
		sb.Status.AtProvider.State = status
	}
}
