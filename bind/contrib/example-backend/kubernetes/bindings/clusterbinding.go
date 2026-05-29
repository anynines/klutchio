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

package bindings

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	bindv1alpha1 "github.com/anynines/klutchio/bind/pkg/apis/bind/v1alpha1"
)

const (
	clusterBindingName = "cluster"
)

// NewClusterBinding constructs a ClusterBinding resource.
func NewClusterBinding(namespace, providerPrettyName string, secretRef bindv1alpha1.LocalSecretKeyRef) *bindv1alpha1.ClusterBinding {
	return &bindv1alpha1.ClusterBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterBindingName,
			Namespace: namespace,
		},
		Spec: bindv1alpha1.ClusterBindingSpec{
			ProviderPrettyName:  providerPrettyName,
			KubeconfigSecretRef: secretRef,
		},
	}
}

// NewClusterBindingWithLabels constructs a ClusterBinding resource with additional labels.
func NewClusterBindingWithLabels(namespace, providerPrettyName string, secretRef bindv1alpha1.LocalSecretKeyRef, labels map[string]string) *bindv1alpha1.ClusterBinding {
	cb := NewClusterBinding(namespace, providerPrettyName, secretRef)
	if cb.Labels == nil {
		cb.Labels = make(map[string]string)
	}
	for k, v := range labels {
		cb.Labels[k] = v
	}
	return cb
}
