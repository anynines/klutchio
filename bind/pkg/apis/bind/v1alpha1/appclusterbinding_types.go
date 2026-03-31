/*
Copyright 2026 The Kube Bind Authors.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	conditionsapi "github.com/anynines/klutchio/bind/pkg/apis/third_party/conditions/apis/conditions/v1alpha1"
)

const (
	// AppClusterBindingConditionSecretValid is set when the kubeconfig secret is valid.
	AppClusterBindingConditionSecretValid conditionsapi.ConditionType = "SecretValid"
	// AppClusterBindingConditionKonnectorDeployed is set when the konnector deployment is created/updated.
	AppClusterBindingConditionKonnectorDeployed conditionsapi.ConditionType = "KonnectorDeployed"
)

// AppClusterBinding represents a binding for an app cluster on the control plane.
//
// +crd
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:scope=Namespaced,categories=kube-bindings
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=`.status.conditions[?(@.type=="Ready")].status`,priority=0
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=`.metadata.creationTimestamp`,priority=0
// +kubebuilder:validation:XValidation:rule="self.metadata.name == oldSelf.metadata.name",message="name is immutable"
type AppClusterBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec represents the desired state of the AppClusterBinding.
	// +required
	// +kubebuilder:validation:Required
	Spec AppClusterBindingSpec `json:"spec"`

	// status contains reconciliation information.
	Status AppClusterBindingStatus `json:"status,omitempty"`
}

func (in *AppClusterBinding) GetConditions() conditionsapi.Conditions {
	return in.Status.Conditions
}

func (in *AppClusterBinding) SetConditions(conditions conditionsapi.Conditions) {
	in.Status.Conditions = conditions
}

// AppClusterBindingSpec represents the desired state of an AppClusterBinding.
type AppClusterBindingSpec struct {
	// kubeconfigSecretRef points to the app cluster kubeconfig.
	//
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="kubeconfigSecretRef is immutable"
	KubeconfigSecretRef ClusterSecretKeyRef `json:"kubeconfigSecretRef"`

	// apiExports is a list of GroupResource entries, where each entry specifies an API group and resource to bind.
	//
	// +optional
	APIExports []GroupResource `json:"apiExports,omitempty"`

	// konnector contains deployment settings for the konnector.
	//
	// +optional
	Konnector *KonnectorSpec `json:"konnector,omitempty"`
}

// KonnectorSpec controls konnector deployment behavior.
type KonnectorSpec struct {
	// deploy enables konnector deployment for this binding.
	//
	// +optional
	Deploy bool `json:"deploy,omitempty"`

	// overrides allow changing the konnector deployment settings.
	//
	// +optional
	Overrides *KonnectorOverrides `json:"overrides,omitempty"`
}

// KonnectorOverrides allows overriding konnector deployment settings.
type KonnectorOverrides struct {
	// image overrides the container image for the konnector.
	//
	// +optional
	Image string `json:"image,omitempty"`

	// containerSettings allow modifying the container spec for the konnector.
	//
	// +optional
	// +kubebuilder:validation:XPreserveUnknownFields
	ContainerSettings runtime.RawExtension `json:"containerSettings,omitempty"`
}

// AppClusterBindingStatus stores status information about an app cluster binding.
type AppClusterBindingStatus struct {
	// conditions is a list of conditions that apply to the AppClusterBinding.
	Conditions conditionsapi.Conditions `json:"conditions,omitempty"`
}

// AppClusterBindingList is a list of AppClusterBindings.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AppClusterBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AppClusterBinding `json:"items"`
}

// DeepCopyInto copies in into out. in must be non-nil.
func (in *AppClusterBinding) DeepCopyInto(out *AppClusterBinding) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy creates a new deep copy of the receiver.
func (in *AppClusterBinding) DeepCopy() *AppClusterBinding {
	if in == nil {
		return nil
	}
	out := new(AppClusterBinding)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies in into out. in must be non-nil.
func (in *AppClusterBindingSpec) DeepCopyInto(out *AppClusterBindingSpec) {
	*out = *in
	out.KubeconfigSecretRef = in.KubeconfigSecretRef
	if in.APIExports != nil {
		out.APIExports = make([]GroupResource, len(in.APIExports))
		copy(out.APIExports, in.APIExports)
	}
	if in.Konnector != nil {
		out.Konnector = new(KonnectorSpec)
		in.Konnector.DeepCopyInto(out.Konnector)
	}
}

// DeepCopy creates a new deep copy of the receiver.
func (in *AppClusterBindingSpec) DeepCopy() *AppClusterBindingSpec {
	if in == nil {
		return nil
	}
	out := new(AppClusterBindingSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies in into out. in must be non-nil.
func (in *KonnectorSpec) DeepCopyInto(out *KonnectorSpec) {
	*out = *in
	if in.Overrides != nil {
		out.Overrides = new(KonnectorOverrides)
		in.Overrides.DeepCopyInto(out.Overrides)
	}
}

// DeepCopy creates a new deep copy of the receiver.
func (in *KonnectorSpec) DeepCopy() *KonnectorSpec {
	if in == nil {
		return nil
	}
	out := new(KonnectorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies in into out. in must be non-nil.
func (in *KonnectorOverrides) DeepCopyInto(out *KonnectorOverrides) {
	*out = *in
	in.ContainerSettings.DeepCopyInto(&out.ContainerSettings)
}

// DeepCopy creates a new deep copy of the receiver.
func (in *KonnectorOverrides) DeepCopy() *KonnectorOverrides {
	if in == nil {
		return nil
	}
	out := new(KonnectorOverrides)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies in into out. in must be non-nil.
func (in *AppClusterBindingStatus) DeepCopyInto(out *AppClusterBindingStatus) {
	*out = *in
	if in.Conditions != nil {
		out.Conditions = make(conditionsapi.Conditions, len(in.Conditions))
		copy(out.Conditions, in.Conditions)
	}
}

// DeepCopy creates a new deep copy of the receiver.
func (in *AppClusterBindingStatus) DeepCopy() *AppClusterBindingStatus {
	if in == nil {
		return nil
	}
	out := new(AppClusterBindingStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies in into out. in must be non-nil.
func (in *AppClusterBindingList) DeepCopyInto(out *AppClusterBindingList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]AppClusterBinding, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

// DeepCopy creates a new deep copy of the receiver.
func (in *AppClusterBindingList) DeepCopy() *AppClusterBindingList {
	if in == nil {
		return nil
	}
	out := new(AppClusterBindingList)
	in.DeepCopyInto(out)
	return out
}
