/*
Copyright 2023 The Kube Bind Authors.

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

// Package v1alpha1 defines the v1alpha1 version of the Example Backend API
//
// +groupName=example-backend.klutch.anynines.com
// +groupGoName=ExampleBackend
// +k8s:deepcopy-gen=package,register
// +kubebuilder:validation:Optional
package v1alpha1
