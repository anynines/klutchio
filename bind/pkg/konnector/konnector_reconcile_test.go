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

package konnector

import (
	"testing"

	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	bindv1alpha1 "github.com/anynines/klutchio/bind/pkg/apis/bind/v1alpha1"
)

func TestProviderExportNamespaceForBinding_UsesExplicitLabel(t *testing.T) {
	binding := &bindv1alpha1.APIServiceBinding{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				providerNamespaceLabel:          "provider-ns",
				appClusterBindingNamespaceLabel: "appclusterbinding-ns",
			},
		},
		Spec: bindv1alpha1.APIServiceBindingSpec{
			KubeconfigSecretRef: bindv1alpha1.ClusterSecretKeyRef{Namespace: "secret-ns"},
		},
	}

	require.Equal(t, "provider-ns", providerExportNamespaceForBinding(binding))
}

func TestProviderExportNamespaceForBinding_UsesAppClusterBindingNamespace(t *testing.T) {
	binding := &bindv1alpha1.APIServiceBinding{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				appClusterBindingNamespaceLabel: "default",
			},
		},
		Spec: bindv1alpha1.APIServiceBindingSpec{
			KubeconfigSecretRef: bindv1alpha1.ClusterSecretKeyRef{Namespace: "secret-ns"},
		},
	}

	require.Equal(t, "default", providerExportNamespaceForBinding(binding))
}

func TestProviderExportNamespaceForBinding_FallsBackToSecretNamespace(t *testing.T) {
	binding := &bindv1alpha1.APIServiceBinding{
		Spec: bindv1alpha1.APIServiceBindingSpec{
			KubeconfigSecretRef: bindv1alpha1.ClusterSecretKeyRef{Namespace: "secret-ns"},
		},
	}

	require.Equal(t, "secret-ns", providerExportNamespaceForBinding(binding))
}

func TestProviderSecretNamespaceForBinding_UsesSecretNamespace(t *testing.T) {
	binding := &bindv1alpha1.APIServiceBinding{
		Spec: bindv1alpha1.APIServiceBindingSpec{
			KubeconfigSecretRef: bindv1alpha1.ClusterSecretKeyRef{Namespace: "default"},
		},
	}

	require.Equal(t, "default", providerSecretNamespaceForBinding(binding))
}
