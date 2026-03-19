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
	"reflect"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclient "k8s.io/client-go/kubernetes"
)

// SecretAccessOptions contains configuration for secret access RBAC.
type SecretAccessOptions struct {
	// SecretNamespace is the namespace containing the secret
	SecretNamespace string
	// SecretName is the name of the secret to grant access to
	SecretName string
	// ServiceAccountNamespace is the namespace of the service account
	ServiceAccountNamespace string
	// ServiceAccountName is the name of the service account
	ServiceAccountName string
	// RoleName is the name for the Role and RoleBinding
	RoleName string
}

// EnsureSecretAccessRBAC creates Role and RoleBinding to grant a service account access to a specific secret.
func EnsureSecretAccessRBAC(ctx context.Context, client kubeclient.Interface, opts SecretAccessOptions) error {
	// Create Role in the secret's namespace
	expectedRole := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.RoleName,
			Namespace: opts.SecretNamespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"secrets"},
				Verbs:         []string{"get"},
				ResourceNames: []string{opts.SecretName},
			},
		},
	}

	role, err := client.RbacV1().Roles(opts.SecretNamespace).Get(ctx, opts.RoleName, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to get role: %w", err)
		}
		if _, err := client.RbacV1().Roles(opts.SecretNamespace).Create(ctx, expectedRole, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("failed to create role %q in namespace %q: %w", opts.RoleName, opts.SecretNamespace, err)
		}
	} else if !reflect.DeepEqual(role.Rules, expectedRole.Rules) {
		role = role.DeepCopy()
		role.Rules = expectedRole.Rules
		if _, err := client.RbacV1().Roles(opts.SecretNamespace).Update(ctx, role, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("failed to update role %q in namespace %q: %w", opts.RoleName, opts.SecretNamespace, err)
		}
	}

	// Create RoleBinding in the secret's namespace
	expectedRoleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.RoleName,
			Namespace: opts.SecretNamespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     opts.RoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      opts.ServiceAccountName,
				Namespace: opts.ServiceAccountNamespace,
			},
		},
	}

	rb, err := client.RbacV1().RoleBindings(opts.SecretNamespace).Get(ctx, opts.RoleName, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to get rolebinding: %w", err)
		}
		if _, err := client.RbacV1().RoleBindings(opts.SecretNamespace).Create(ctx, expectedRoleBinding, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("failed to create rolebinding %q in namespace %q: %w", opts.RoleName, opts.SecretNamespace, err)
		}
	} else if !reflect.DeepEqual(rb.Subjects, expectedRoleBinding.Subjects) {
		rb = rb.DeepCopy()
		rb.Subjects = expectedRoleBinding.Subjects
		// Note: RoleRef is immutable
		if _, err := client.RbacV1().RoleBindings(opts.SecretNamespace).Update(ctx, rb, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("failed to update rolebinding %q in namespace %q: %w", opts.RoleName, opts.SecretNamespace, err)
		}
	}

	return nil
}
