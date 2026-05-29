/*
Copyright 2022 The Kube Bind Authors.

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

package clusterbinding

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	bindv1alpha1 "github.com/anynines/klutchio/bind/pkg/apis/bind/v1alpha1"
)

func TestEnsureConsumerSecretCreatesSecretWithCorrectNamespaceAndName(t *testing.T) {
	var created *corev1.Secret

	r := &reconciler{
		consumerSecretRefKey: "target-namespace/target-secret",
		providerNamespace:    "provider-namespace",
		getProviderSecret: func() (*corev1.Secret, error) {
			return &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "provider-secret", Namespace: "provider-namespace"},
				Data:       map[string][]byte{"kubeconfig": []byte("config")},
				Type:       corev1.SecretTypeOpaque,
			}, nil
		},
		getConsumerSecret: func() (*corev1.Secret, error) {
			return nil, nil
		},
		createConsumerSecret: func(ctx context.Context, secret *corev1.Secret) (*corev1.Secret, error) {
			created = secret.DeepCopy()
			return secret, nil
		},
		updateConsumerSecret: func(ctx context.Context, secret *corev1.Secret) (*corev1.Secret, error) {
			t.Fatalf("did not expect update path")
			return nil, nil
		},
	}

	binding := &bindv1alpha1.ClusterBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "provider-namespace"},
		Spec: bindv1alpha1.ClusterBindingSpec{
			KubeconfigSecretRef: bindv1alpha1.LocalSecretKeyRef{
				Name: "provider-secret",
				Key:  "kubeconfig",
			},
		},
	}

	if err := r.ensureConsumerSecret(context.Background(), binding); err != nil {
		t.Fatalf("expected ensureConsumerSecret success, got %v", err)
	}
	if created == nil {
		t.Fatal("expected consumer secret to be created")
	}
	if created.Namespace != "target-namespace" {
		t.Fatalf("expected consumer secret namespace target-namespace, got %q", created.Namespace)
	}
	if created.Name != "target-secret" {
		t.Fatalf("expected consumer secret name target-secret, got %q", created.Name)
	}
}
