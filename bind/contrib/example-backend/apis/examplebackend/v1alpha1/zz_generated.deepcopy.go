//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright The Kube Bind Authors.

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

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"

	bindv1alpha1 "github.com/anynines/klutchio/bind/pkg/apis/bind/v1alpha1"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *APIServiceExportTemplate) DeepCopyInto(out *APIServiceExportTemplate) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new APIServiceExportTemplate.
func (in *APIServiceExportTemplate) DeepCopy() *APIServiceExportTemplate {
	if in == nil {
		return nil
	}
	out := new(APIServiceExportTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *APIServiceExportTemplate) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *APIServiceExportTemplateList) DeepCopyInto(out *APIServiceExportTemplateList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]APIServiceExportTemplate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new APIServiceExportTemplateList.
func (in *APIServiceExportTemplateList) DeepCopy() *APIServiceExportTemplateList {
	if in == nil {
		return nil
	}
	out := new(APIServiceExportTemplateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *APIServiceExportTemplateList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *APIServiceExportTemplateSpec) DeepCopyInto(out *APIServiceExportTemplateSpec) {
	*out = *in
	out.APIServiceSelector = in.APIServiceSelector
	if in.PermissionClaims != nil {
		in, out := &in.PermissionClaims, &out.PermissionClaims
		*out = make([]bindv1alpha1.PermissionClaim, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new APIServiceExportTemplateSpec.
func (in *APIServiceExportTemplateSpec) DeepCopy() *APIServiceExportTemplateSpec {
	if in == nil {
		return nil
	}
	out := new(APIServiceExportTemplateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *APIServiceExportTemplateStatus) DeepCopyInto(out *APIServiceExportTemplateStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new APIServiceExportTemplateStatus.
func (in *APIServiceExportTemplateStatus) DeepCopy() *APIServiceExportTemplateStatus {
	if in == nil {
		return nil
	}
	out := new(APIServiceExportTemplateStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *APIServiceSelector) DeepCopyInto(out *APIServiceSelector) {
	*out = *in
	out.GroupResource = in.GroupResource
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new APIServiceSelector.
func (in *APIServiceSelector) DeepCopy() *APIServiceSelector {
	if in == nil {
		return nil
	}
	out := new(APIServiceSelector)
	in.DeepCopyInto(out)
	return out
}
