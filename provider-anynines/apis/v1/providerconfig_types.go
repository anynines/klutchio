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

// A ProviderConfigSpec defines the desired state of a ProviderConfig.
type ProviderConfigSpec struct {
	Url string `json:"url"`
	// Endpoint to use for broker health checks. If not set, the endpoint /instances is used.
	// +kubebuilder:validation:Optional
	HealthCheckEndpoint string `json:"healthCheckEndpoint,omitempty"`
	// Credentials required to authenticate to this provider.
	ProviderCredentials ProviderCredentials `json:"providerCredentials"`
}

// ProviderCredentials required to authenticate.
type ProviderCredentials struct {
	// Source of the provider credentials.
	// +kubebuilder:validation:Enum=None;Secret;InjectedIdentity;Environment;Filesystem
	Source xpv1.CredentialsSource `json:"source"`

	Username xpv1.CommonCredentialSelectors `json:"username"`
	Password xpv1.CommonCredentialSelectors `json:"password"`
}

// A ProviderConfigStatus reflects the observed state of a ProviderConfig.
type ProviderConfigStatus struct {
	xpv1.ProviderConfigStatus `json:",inline"`

	// Health contains indications of the provider's health.
	// The health of each ProviderConfig is evaluated periodically,
	// by trying to reach the configured URL.
	// +kubebuilder:validation:Optional
	Health ProviderConfigHealth `json:"health"`
}

type ProviderConfigHealth struct {
	// LastCheckTime is the last time at which the check was performed.
	// It is used to determine when to run the next check.
	// To trigger an immediate health check, this can be manually set to null.
	// +kubebuilder:validation:Optional
	LastCheckTime *metav1.Time `json:"lastCheckTime"`
	// LastStatus indicates if the health check was successful.
	// +kubebuilder:validation:Optional
	LastStatus bool `json:"lastStatus"`
	// LastMessage contains a human-readable message with details
	// aobut the health check result.
	// +kubebuilder:validation:Optional
	LastMessage string `json:"lastMessage"`
}

// +kubebuilder:object:root=true

// A ProviderConfig configures a anynines provider.
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="SECRET-NAME",type="string",JSONPath=".spec.credentials.secretRef.name",priority=1
// +kubebuilder:printcolumn:name="HEALTHY",type="boolean",JSONPath=".status.health.lastStatus"
// +kubebuilder:resource:scope=Cluster
type ProviderConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProviderConfigSpec   `json:"spec"`
	Status ProviderConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProviderConfigList contains a list of ProviderConfig.
type ProviderConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProviderConfig `json:"items"`
}

// ProviderConfig type metadata.
var (
	ProviderConfigKind             = reflect.TypeOf(ProviderConfig{}).Name()
	ProviderConfigGroupKind        = schema.GroupKind{Group: Group, Kind: ProviderConfigKind}.String()
	ProviderConfigKindAPIVersion   = ProviderConfigKind + "." + SchemeGroupVersion.String()
	ProviderConfigGroupVersionKind = SchemeGroupVersion.WithKind(ProviderConfigKind)
)

func init() {
	SchemeBuilder.Register(&ProviderConfig{}, &ProviderConfigList{})
}
