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
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclient "k8s.io/client-go/kubernetes"

	kuberesources "github.com/anynines/klutchio/bind/contrib/example-backend/kubernetes/resources"
)

// EnsureBindingRootResources ensures the binding root namespace and service account exist.
func EnsureBindingRootResources(ctx context.Context, client kubeclient.Interface, namespace string) error {
	// Create namespace if it doesn't exist
	_, err := client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to get namespace %q: %w", namespace, err)
		}
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		if _, err := client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("failed to create namespace %q: %w", namespace, err)
		}
	}

	// Create service account if it doesn't exist
	_, err = client.CoreV1().ServiceAccounts(namespace).Get(ctx, kuberesources.ServiceAccountName, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to get service account: %w", err)
		}
		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      kuberesources.ServiceAccountName,
				Namespace: namespace,
			},
		}
		if _, err := client.CoreV1().ServiceAccounts(namespace).Create(ctx, sa, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("failed to create service account %q in namespace %q: %w", kuberesources.ServiceAccountName, namespace, err)
		}
	}

	return nil
}
