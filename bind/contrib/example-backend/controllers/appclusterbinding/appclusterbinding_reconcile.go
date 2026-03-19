/*
Copyright 2026 The Klutch Bind Authors.

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

package appclusterbinding

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"reflect"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	kubernetesclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/pointer"

	examplebackendv1alpha1 "github.com/anynines/klutchio/bind/contrib/example-backend/apis/examplebackend/v1alpha1"
	"github.com/anynines/klutchio/bind/contrib/example-backend/kubernetes/bindings"
	kuberesources "github.com/anynines/klutchio/bind/contrib/example-backend/kubernetes/resources"
	bindv1alpha1 "github.com/anynines/klutchio/bind/pkg/apis/bind/v1alpha1"
	conditionsapi "github.com/anynines/klutchio/bind/pkg/apis/third_party/conditions/apis/conditions/v1alpha1"
	"github.com/anynines/klutchio/bind/pkg/apis/third_party/conditions/util/conditions"
)

const (
	appClusterBindingNameLabel        = "klutch.anynines.com/appclusterbinding-name"
	appClusterBindingNamespaceLabel   = "klutch.anynines.com/appclusterbinding-namespace"
	bindingRootNamespacePrefix        = "klutch-bind"
	apiServiceExportRequestNamePrefix = "appclusterbinding"
)

type reconciler struct {
	kubeClient kubernetesclient.Interface

	getSecret func(ns, name string) (*corev1.Secret, error)

	listServiceBindings           func(ctx context.Context, labelSelector string) (*bindv1alpha1.APIServiceBindingList, error)
	createServiceBinding          func(ctx context.Context, binding *bindv1alpha1.APIServiceBinding) (*bindv1alpha1.APIServiceBinding, error)
	updateServiceBinding          func(ctx context.Context, binding *bindv1alpha1.APIServiceBinding) (*bindv1alpha1.APIServiceBinding, error)
	deleteServiceBinding          func(ctx context.Context, name string) error
	getAPIServiceExportTemplate   func(ctx context.Context, namespace, name string) (*examplebackendv1alpha1.APIServiceExportTemplate, error)
	listAPIServiceExportRequests  func(ctx context.Context, namespace, labelSelector string) (*bindv1alpha1.APIServiceExportRequestList, error)
	createAPIServiceExportRequest func(ctx context.Context, namespace string, req *bindv1alpha1.APIServiceExportRequest) (*bindv1alpha1.APIServiceExportRequest, error)
	deleteAPIServiceExportRequest func(ctx context.Context, namespace, name string) error

	getServiceAccount    func(ctx context.Context, namespace, name string) (*corev1.ServiceAccount, error)
	createServiceAccount func(ctx context.Context, namespace string, sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error)
	deleteServiceAccount func(ctx context.Context, namespace, name string) error

	getRole                  func(ctx context.Context, namespace, name string) (*rbacv1.Role, error)
	createRole               func(ctx context.Context, namespace string, role *rbacv1.Role) (*rbacv1.Role, error)
	updateRole               func(ctx context.Context, namespace string, role *rbacv1.Role) (*rbacv1.Role, error)
	deleteRole               func(ctx context.Context, namespace, name string) error
	getClusterRole           func(ctx context.Context, name string) (*rbacv1.ClusterRole, error)
	createClusterRole        func(ctx context.Context, role *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error)
	updateClusterRole        func(ctx context.Context, role *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error)
	deleteClusterRole        func(ctx context.Context, name string) error
	getClusterRoleBinding    func(ctx context.Context, name string) (*rbacv1.ClusterRoleBinding, error)
	createClusterRoleBinding func(ctx context.Context, crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error)
	updateClusterRoleBinding func(ctx context.Context, crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error)
	deleteClusterRoleBinding func(ctx context.Context, name string) error
	getRoleBinding           func(ctx context.Context, namespace, name string) (*rbacv1.RoleBinding, error)
	createRoleBinding        func(ctx context.Context, namespace string, rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error)
	updateRoleBinding        func(ctx context.Context, namespace string, rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error)
	deleteRoleBinding        func(ctx context.Context, namespace, name string) error

	getNamespace         func(ctx context.Context, name string) (*corev1.Namespace, error)
	createNamespace      func(ctx context.Context, namespace *corev1.Namespace) (*corev1.Namespace, error)
	deleteNamespace      func(ctx context.Context, name string) error
	getClusterBinding    func(ctx context.Context, namespace, name string) (*bindv1alpha1.ClusterBinding, error)
	createClusterBinding func(ctx context.Context, namespace string, binding *bindv1alpha1.ClusterBinding) (*bindv1alpha1.ClusterBinding, error)
	updateClusterBinding func(ctx context.Context, namespace string, binding *bindv1alpha1.ClusterBinding) (*bindv1alpha1.ClusterBinding, error)
	createSecret         func(ctx context.Context, namespace string, secret *corev1.Secret) (*corev1.Secret, error)
	updateSecret         func(ctx context.Context, namespace string, secret *corev1.Secret) (*corev1.Secret, error)

	getDeployment    func(ctx context.Context, namespace, name string) (*appsv1.Deployment, error)
	createDeployment func(ctx context.Context, namespace string, deployment *appsv1.Deployment) (*appsv1.Deployment, error)
	updateDeployment func(ctx context.Context, namespace string, deployment *appsv1.Deployment) (*appsv1.Deployment, error)
	deleteDeployment func(ctx context.Context, namespace, name string) error
}

func (r *reconciler) reconcile(ctx context.Context, binding *bindv1alpha1.AppClusterBinding) error {
	var errs []error

	secretValid, err := r.ensureValidKubeconfigSecret(ctx, binding)
	if err != nil {
		errs = append(errs, err)
	}
	if secretValid {
		if err := r.ensureBindingRootResources(ctx, binding); err != nil {
			errs = append(errs, err)
		}
		if err := r.ensureServiceBindings(ctx, binding); err != nil {
			errs = append(errs, err)
		}
		if err := r.ensureServiceExportRequests(ctx, binding); err != nil {
			errs = append(errs, err)
		}
		if err := r.ensureKonnectorClusterRBAC(ctx, binding); err != nil {
			errs = append(errs, err)
		}
		if err := r.ensureKonnectorNamespacedRBAC(ctx, binding); err != nil {
			errs = append(errs, err)
		}
		if err := r.ensureKonnectorBindingRootRBAC(ctx, binding); err != nil {
			errs = append(errs, err)
		}
		if err := r.ensureKonnectorDeployment(ctx, binding); err != nil {
			errs = append(errs, err)
		}
	}

	conditions.SetSummary(binding)

	return utilerrors.NewAggregate(errs)
}

func (r *reconciler) ensureDeleted(ctx context.Context, binding *bindv1alpha1.AppClusterBinding) error {
	var errs []error

	selector := labels.Set{
		appClusterBindingNameLabel:      binding.Name,
		appClusterBindingNamespaceLabel: binding.Namespace,
	}.AsSelector().String()

	serviceBindings, err := r.listServiceBindings(ctx, selector)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to list apiservicebindings for cleanup: %w", err))
	} else {
		for i := range serviceBindings.Items {
			name := serviceBindings.Items[i].Name
			if err := r.deleteServiceBinding(ctx, name); err != nil && !errors.IsNotFound(err) {
				errs = append(errs, fmt.Errorf("failed to delete apiservicebinding %q: %w", name, err))
			}
		}

		remaining, err := r.listServiceBindings(ctx, selector)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to verify apiservicebinding cleanup: %w", err))
		} else if len(remaining.Items) > 0 {
			errs = append(errs, fmt.Errorf("waiting for %d apiservicebinding resources to be deleted", len(remaining.Items)))
		}
	}

	serviceExportRequests, err := r.listAPIServiceExportRequests(ctx, binding.Namespace, selector)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to list apiserviceexportrequests for cleanup: %w", err))
	} else {
		for i := range serviceExportRequests.Items {
			name := serviceExportRequests.Items[i].Name
			if err := r.deleteAPIServiceExportRequest(ctx, binding.Namespace, name); err != nil && !errors.IsNotFound(err) {
				errs = append(errs, fmt.Errorf("failed to delete apiserviceexportrequest %s/%q: %w", binding.Namespace, name, err))
			}
		}

		remaining, err := r.listAPIServiceExportRequests(ctx, binding.Namespace, selector)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to verify apiserviceexportrequest cleanup: %w", err))
		} else if len(remaining.Items) > 0 {
			errs = append(errs, fmt.Errorf("waiting for %d apiserviceexportrequest resources to be deleted", len(remaining.Items)))
		}
	}

	clusterRoleName := fmt.Sprintf("klutch-konnector-%s-%s", binding.Namespace, binding.Name)
	clusterRoleBindingName := clusterRoleName
	if err := r.deleteClusterRoleBinding(ctx, clusterRoleBindingName); err != nil && !errors.IsNotFound(err) {
		errs = append(errs, fmt.Errorf("failed to delete clusterrolebinding %q: %w", clusterRoleBindingName, err))
	}
	if _, err := r.getClusterRoleBinding(ctx, clusterRoleBindingName); err != nil {
		if !errors.IsNotFound(err) {
			errs = append(errs, fmt.Errorf("failed to verify clusterrolebinding cleanup %q: %w", clusterRoleBindingName, err))
		}
	} else {
		errs = append(errs, fmt.Errorf("clusterrolebinding %q still exists", clusterRoleBindingName))
	}

	if err := r.deleteClusterRole(ctx, clusterRoleName); err != nil && !errors.IsNotFound(err) {
		errs = append(errs, fmt.Errorf("failed to delete clusterrole %q: %w", clusterRoleName, err))
	}
	if _, err := r.getClusterRole(ctx, clusterRoleName); err != nil {
		if !errors.IsNotFound(err) {
			errs = append(errs, fmt.Errorf("failed to verify clusterrole cleanup %q: %w", clusterRoleName, err))
		}
	} else {
		errs = append(errs, fmt.Errorf("clusterrole %q still exists", clusterRoleName))
	}

	deploymentName := fmt.Sprintf("konnector-%s", binding.Name)
	if err := r.deleteDeployment(ctx, binding.Namespace, deploymentName); err != nil && !errors.IsNotFound(err) {
		errs = append(errs, fmt.Errorf("failed to delete deployment %s/%s: %w", binding.Namespace, deploymentName, err))
	}
	if _, err := r.getDeployment(ctx, binding.Namespace, deploymentName); err != nil {
		if !errors.IsNotFound(err) {
			errs = append(errs, fmt.Errorf("failed to verify deployment cleanup %s/%s: %w", binding.Namespace, deploymentName, err))
		}
	} else {
		errs = append(errs, fmt.Errorf("deployment %s/%s still exists", binding.Namespace, deploymentName))
	}

	if err := r.deleteRoleBinding(ctx, binding.Namespace, "konnector-secret-reader"); err != nil && !errors.IsNotFound(err) {
		errs = append(errs, fmt.Errorf("failed to delete rolebinding %s/%s: %w", binding.Namespace, "konnector-secret-reader", err))
	}
	if _, err := r.getRoleBinding(ctx, binding.Namespace, "konnector-secret-reader"); err != nil {
		if !errors.IsNotFound(err) {
			errs = append(errs, fmt.Errorf("failed to verify rolebinding cleanup %s/%s: %w", binding.Namespace, "konnector-secret-reader", err))
		}
	} else {
		errs = append(errs, fmt.Errorf("rolebinding %s/%s still exists", binding.Namespace, "konnector-secret-reader"))
	}

	if err := r.deleteRole(ctx, binding.Namespace, "konnector-secret-reader"); err != nil && !errors.IsNotFound(err) {
		errs = append(errs, fmt.Errorf("failed to delete role %s/%s: %w", binding.Namespace, "konnector-secret-reader", err))
	}
	if _, err := r.getRole(ctx, binding.Namespace, "konnector-secret-reader"); err != nil {
		if !errors.IsNotFound(err) {
			errs = append(errs, fmt.Errorf("failed to verify role cleanup %s/%s: %w", binding.Namespace, "konnector-secret-reader", err))
		}
	} else {
		errs = append(errs, fmt.Errorf("role %s/%s still exists", binding.Namespace, "konnector-secret-reader"))
	}

	secretAccessRoleName := fmt.Sprintf("klutch-bind-read-%s-%s", binding.Namespace, binding.Name)
	secretNamespace := binding.Spec.KubeconfigSecretRef.Namespace
	if err := r.deleteRoleBinding(ctx, secretNamespace, secretAccessRoleName); err != nil && !errors.IsNotFound(err) {
		errs = append(errs, fmt.Errorf("failed to delete rolebinding %s/%s: %w", secretNamespace, secretAccessRoleName, err))
	}
	if _, err := r.getRoleBinding(ctx, secretNamespace, secretAccessRoleName); err != nil {
		if !errors.IsNotFound(err) {
			errs = append(errs, fmt.Errorf("failed to verify rolebinding cleanup %s/%s: %w", secretNamespace, secretAccessRoleName, err))
		}
	} else {
		errs = append(errs, fmt.Errorf("rolebinding %s/%s still exists", secretNamespace, secretAccessRoleName))
	}

	if err := r.deleteRole(ctx, secretNamespace, secretAccessRoleName); err != nil && !errors.IsNotFound(err) {
		errs = append(errs, fmt.Errorf("failed to delete role %s/%s: %w", secretNamespace, secretAccessRoleName, err))
	}
	if _, err := r.getRole(ctx, secretNamespace, secretAccessRoleName); err != nil {
		if !errors.IsNotFound(err) {
			errs = append(errs, fmt.Errorf("failed to verify role cleanup %s/%s: %w", secretNamespace, secretAccessRoleName, err))
		}
	} else {
		errs = append(errs, fmt.Errorf("role %s/%s still exists", secretNamespace, secretAccessRoleName))
	}

	bindingRootNamespace := r.getBindingRootNamespace(binding)
	if err := r.deleteNamespace(ctx, bindingRootNamespace); err != nil && !errors.IsNotFound(err) {
		errs = append(errs, fmt.Errorf("failed to delete binding root namespace %q: %w", bindingRootNamespace, err))
	}
	if _, err := r.getNamespace(ctx, bindingRootNamespace); err != nil {
		if !errors.IsNotFound(err) {
			errs = append(errs, fmt.Errorf("failed to verify namespace cleanup %q: %w", bindingRootNamespace, err))
		}
	} else {
		errs = append(errs, fmt.Errorf("namespace %q still exists", bindingRootNamespace))
	}

	return utilerrors.NewAggregate(errs)
}

func (r *reconciler) ensureValidKubeconfigSecret(ctx context.Context, binding *bindv1alpha1.AppClusterBinding) (bool, error) {
	secret, err := r.getSecret(binding.Spec.KubeconfigSecretRef.Namespace, binding.Spec.KubeconfigSecretRef.Name)
	if err != nil && !errors.IsNotFound(err) {
		return false, err
	} else if errors.IsNotFound(err) {
		conditions.MarkFalse(
			binding,
			bindv1alpha1.AppClusterBindingConditionSecretValid,
			"KubeconfigSecretNotFound",
			conditionsapi.ConditionSeverityError,
			"Kubeconfig secret %s/%s not found.",
			binding.Spec.KubeconfigSecretRef.Namespace,
			binding.Spec.KubeconfigSecretRef.Name,
		)
		return false, nil
	}

	kubeconfig, found := secret.Data[binding.Spec.KubeconfigSecretRef.Key]
	if !found {
		conditions.MarkFalse(
			binding,
			bindv1alpha1.AppClusterBindingConditionSecretValid,
			"KubeconfigSecretInvalid",
			conditionsapi.ConditionSeverityError,
			"Kubeconfig secret %s/%s is missing %q string key.",
			binding.Spec.KubeconfigSecretRef.Namespace,
			binding.Spec.KubeconfigSecretRef.Name,
			binding.Spec.KubeconfigSecretRef.Key,
		)
		return false, nil
	}

	cfg, err := clientcmd.Load(kubeconfig)
	if err != nil {
		conditions.MarkFalse(
			binding,
			bindv1alpha1.AppClusterBindingConditionSecretValid,
			"KubeconfigSecretInvalid",
			conditionsapi.ConditionSeverityError,
			"Kubeconfig secret %s/%s has an invalid kubeconfig: %v",
			binding.Spec.KubeconfigSecretRef.Namespace,
			binding.Spec.KubeconfigSecretRef.Name,
			err,
		)
		return false, nil
	}
	kubeContext, found := cfg.Contexts[cfg.CurrentContext]
	if !found {
		conditions.MarkFalse(
			binding,
			bindv1alpha1.AppClusterBindingConditionSecretValid,
			"KubeconfigSecretInvalid",
			conditionsapi.ConditionSeverityError,
			"Kubeconfig secret %s/%s has an invalid kubeconfig: current context %q not found",
			binding.Spec.KubeconfigSecretRef.Namespace,
			binding.Spec.KubeconfigSecretRef.Name,
			cfg.CurrentContext,
		)
		return false, nil
	}
	if kubeContext.Namespace == "" {
		conditions.MarkFalse(
			binding,
			bindv1alpha1.AppClusterBindingConditionSecretValid,
			"KubeconfigSecretInvalid",
			conditionsapi.ConditionSeverityError,
			"Kubeconfig secret %s/%s has an invalid kubeconfig: current context %q has no namespace set",
			binding.Spec.KubeconfigSecretRef.Namespace,
			binding.Spec.KubeconfigSecretRef.Name,
			cfg.CurrentContext,
		)
		return false, nil
	}
	if _, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig)); err != nil {
		conditions.MarkFalse(
			binding,
			bindv1alpha1.AppClusterBindingConditionSecretValid,
			"KubeconfigSecretInvalid",
			conditionsapi.ConditionSeverityError,
			"Kubeconfig secret %s/%s has an invalid kubeconfig: %v",
			binding.Spec.KubeconfigSecretRef.Namespace,
			binding.Spec.KubeconfigSecretRef.Name,
			err,
		)
		return false, nil
	}

	conditions.MarkTrue(
		binding,
		bindv1alpha1.AppClusterBindingConditionSecretValid,
	)

	return true, nil
}

// getBindingRootNamespace returns the unique namespace name for this AppClusterBinding.
// The namespace is derived from the binding's namespace and name to ensure uniqueness
// across different bindings and konnector deployments.
func (r *reconciler) getBindingRootNamespace(binding *bindv1alpha1.AppClusterBinding) string {
	return fmt.Sprintf("%s-%s-%s", bindingRootNamespacePrefix, binding.Namespace, binding.Name)
}

func (r *reconciler) ensureBindingRootResources(ctx context.Context, binding *bindv1alpha1.AppClusterBinding) error {
	rootNamespace := r.getBindingRootNamespace(binding)

	// Use shared bootstrap helper for namespace + service account
	if err := bindings.EnsureBindingRootResources(ctx, r.kubeClient, rootNamespace); err != nil {
		return fmt.Errorf("failed to ensure binding root resources: %w", err)
	}

	// Verify kubeconfig secret exists in original location
	sourceSecret, err := r.getSecret(binding.Spec.KubeconfigSecretRef.Namespace, binding.Spec.KubeconfigSecretRef.Name)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig secret %s/%s: %w", binding.Spec.KubeconfigSecretRef.Namespace, binding.Spec.KubeconfigSecretRef.Name, err)
	}
	if _, found := sourceSecret.Data[binding.Spec.KubeconfigSecretRef.Key]; !found {
		return fmt.Errorf("kubeconfig secret %s/%s is missing %q string key",
			binding.Spec.KubeconfigSecretRef.Namespace,
			binding.Spec.KubeconfigSecretRef.Name,
			binding.Spec.KubeconfigSecretRef.Key)
	}

	// Use shared RBAC helper for kubeconfig secret access
	roleName := fmt.Sprintf("klutch-bind-read-%s-%s", binding.Namespace, binding.Name)
	if err := bindings.EnsureSecretAccessRBAC(ctx, r.kubeClient, bindings.SecretAccessOptions{
		SecretNamespace:         binding.Spec.KubeconfigSecretRef.Namespace,
		SecretName:              binding.Spec.KubeconfigSecretRef.Name,
		ServiceAccountNamespace: rootNamespace,
		ServiceAccountName:      kuberesources.ServiceAccountName,
		RoleName:                roleName,
	}); err != nil {
		return fmt.Errorf("failed to ensure RBAC for kubeconfig secret: %w", err)
	}

	// Create ClusterBinding in binding root namespace with labels for konnector RBAC
	clusterBindingName := kuberesources.ClusterBindingName
	labels := map[string]string{
		appClusterBindingNameLabel:      binding.Name,
		appClusterBindingNamespaceLabel: binding.Namespace,
	}
	expected := bindings.NewClusterBindingWithLabels(
		rootNamespace,
		binding.Name,
		bindv1alpha1.LocalSecretKeyRef{
			Name: binding.Spec.KubeconfigSecretRef.Name,
			Key:  binding.Spec.KubeconfigSecretRef.Key,
		},
		labels,
	)

	cb, err := r.getClusterBinding(ctx, rootNamespace, clusterBindingName)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get clusterbinding: %w", err)
	}
	// Treat zero-value or nil objects as "not found"
	if cb == nil || cb.Name == "" {
		if _, err := r.createClusterBinding(ctx, rootNamespace, expected); err != nil {
			return fmt.Errorf("failed to create clusterbinding %q in namespace %q: %w", clusterBindingName, rootNamespace, err)
		}
	} else if !reflect.DeepEqual(cb.Spec, expected.Spec) || !reflect.DeepEqual(cb.Labels, expected.Labels) {
		cb = cb.DeepCopy()
		cb.Name = clusterBindingName // Ensure Name is set
		cb.Spec = expected.Spec
		cb.Labels = expected.Labels
		if _, err := r.updateClusterBinding(ctx, rootNamespace, cb); err != nil {
			return fmt.Errorf("failed to update clusterbinding %q in namespace %q: %w", clusterBindingName, rootNamespace, err)
		}
	}

	return nil
}

// ensureKonnectorClusterRBAC creates ClusterRole and ClusterRoleBinding for konnector SA
func (r *reconciler) ensureKonnectorClusterRBAC(ctx context.Context, binding *bindv1alpha1.AppClusterBinding) error {
	clusterRoleName := fmt.Sprintf("klutch-konnector-%s-%s", binding.Namespace, binding.Name)

	// Create ClusterRole for konnector permissions
	expectedClusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleName,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"klutch.anynines.com"},
				Resources: []string{"apiservicebindings", "clusterbindings"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"klutch.anynines.com"},
				Resources: []string{"apiservicebindings/status"},
				Verbs:     []string{"patch", "update"},
			},
			{
				APIGroups: []string{"coordination.k8s.io"},
				Resources: []string{"leases"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets", "namespaces"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"apiextensions.k8s.io"},
				Resources: []string{"customresourcedefinitions"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}

	cr, err := r.getClusterRole(ctx, clusterRoleName)
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to get clusterrole: %w", err)
		}
		if _, err := r.createClusterRole(ctx, expectedClusterRole); err != nil {
			return fmt.Errorf("failed to create clusterrole %q: %w", clusterRoleName, err)
		}
	} else if !reflect.DeepEqual(cr.Rules, expectedClusterRole.Rules) {
		cr = cr.DeepCopy()
		cr.Rules = expectedClusterRole.Rules
		if _, err := r.updateClusterRole(ctx, cr); err != nil {
			return fmt.Errorf("failed to update clusterrole %q: %w", clusterRoleName, err)
		}
	}

	// Create ClusterRoleBinding to bind konnector SA to the ClusterRole
	expectedClusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("klutch-konnector-%s-%s", binding.Namespace, binding.Name),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "konnector",
				Namespace: binding.Namespace,
			},
		},
	}

	crb, err := r.getClusterRoleBinding(ctx, fmt.Sprintf("klutch-konnector-%s-%s", binding.Namespace, binding.Name))
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to get clusterrolebinding: %w", err)
		}
		if _, err := r.createClusterRoleBinding(ctx, expectedClusterRoleBinding); err != nil {
			return fmt.Errorf("failed to create clusterrolebinding: %w", err)
		}
	} else if !reflect.DeepEqual(crb.Subjects, expectedClusterRoleBinding.Subjects) || !reflect.DeepEqual(crb.RoleRef, expectedClusterRoleBinding.RoleRef) {
		crb = crb.DeepCopy()
		crb.Subjects = expectedClusterRoleBinding.Subjects
		crb.RoleRef = expectedClusterRoleBinding.RoleRef
		if _, err := r.updateClusterRoleBinding(ctx, crb); err != nil {
			return fmt.Errorf("failed to update clusterrolebinding: %w", err)
		}
	}

	return nil
}

// ensureKonnectorNamespacedRBAC creates Role and RoleBinding in the AppClusterBinding namespace for secrets
func (r *reconciler) ensureKonnectorNamespacedRBAC(ctx context.Context, binding *bindv1alpha1.AppClusterBinding) error {
	ownerRef := ownerReferenceForAppClusterBinding(binding)

	expectedRole := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "konnector-secret-reader",
			Namespace: binding.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				ownerRef,
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get"},
			},
		},
	}

	role, err := r.getRole(ctx, binding.Namespace, "konnector-secret-reader")
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to get role: %w", err)
		}
		if _, err := r.createRole(ctx, binding.Namespace, expectedRole); err != nil {
			return fmt.Errorf("failed to create role: %w", err)
		}
	} else if !reflect.DeepEqual(role.Rules, expectedRole.Rules) || !reflect.DeepEqual(role.OwnerReferences, expectedRole.OwnerReferences) {
		role = role.DeepCopy()
		role.Rules = expectedRole.Rules
		role.OwnerReferences = expectedRole.OwnerReferences
		if _, err := r.updateRole(ctx, binding.Namespace, role); err != nil {
			return fmt.Errorf("failed to update role: %w", err)
		}
	}

	expectedRB := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "konnector-secret-reader",
			Namespace: binding.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				ownerRef,
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "konnector",
				Namespace: binding.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "konnector-secret-reader",
		},
	}

	rb, err := r.getRoleBinding(ctx, binding.Namespace, "konnector-secret-reader")
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to get rolebinding: %w", err)
		}
		if _, err := r.createRoleBinding(ctx, binding.Namespace, expectedRB); err != nil {
			return fmt.Errorf("failed to create rolebinding: %w", err)
		}
	} else if !reflect.DeepEqual(rb.Subjects, expectedRB.Subjects) || !reflect.DeepEqual(rb.OwnerReferences, expectedRB.OwnerReferences) {
		rb = rb.DeepCopy()
		rb.Subjects = expectedRB.Subjects
		rb.OwnerReferences = expectedRB.OwnerReferences
		if _, err := r.updateRoleBinding(ctx, binding.Namespace, rb); err != nil {
			return fmt.Errorf("failed to update rolebinding: %w", err)
		}
	}

	return nil
}

// ensureKonnectorBindingRootRBAC creates Role and RoleBinding in the binding root namespace
func (r *reconciler) ensureKonnectorBindingRootRBAC(ctx context.Context, binding *bindv1alpha1.AppClusterBinding) error {
	bindingRootNamespace := r.getBindingRootNamespace(binding)

	expectedRole := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "konnector-binding-root-access",
			Namespace: bindingRootNamespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"bind.kube.anynines.com"},
				Resources: []string{"apiservicebindings"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
		},
	}

	role, err := r.getRole(ctx, bindingRootNamespace, "konnector-binding-root-access")
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to get role: %w", err)
		}
		if _, err := r.createRole(ctx, bindingRootNamespace, expectedRole); err != nil {
			return fmt.Errorf("failed to create role: %w", err)
		}
	} else if !reflect.DeepEqual(role.Rules, expectedRole.Rules) {
		role = role.DeepCopy()
		role.Rules = expectedRole.Rules
		if _, err := r.updateRole(ctx, bindingRootNamespace, role); err != nil {
			return fmt.Errorf("failed to update role: %w", err)
		}
	}

	expectedRB := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "konnector-binding-root-access",
			Namespace: bindingRootNamespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "konnector",
				Namespace: binding.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "konnector-binding-root-access",
		},
	}

	rb, err := r.getRoleBinding(ctx, bindingRootNamespace, "konnector-binding-root-access")
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to get rolebinding: %w", err)
		}
		if _, err := r.createRoleBinding(ctx, bindingRootNamespace, expectedRB); err != nil {
			return fmt.Errorf("failed to create rolebinding: %w", err)
		}
	} else if !reflect.DeepEqual(rb.Subjects, expectedRB.Subjects) {
		rb = rb.DeepCopy()
		rb.Subjects = expectedRB.Subjects
		if _, err := r.updateRoleBinding(ctx, bindingRootNamespace, rb); err != nil {
			return fmt.Errorf("failed to update rolebinding: %w", err)
		}
	}

	return nil
}

func (r *reconciler) ensureServiceBindings(ctx context.Context, binding *bindv1alpha1.AppClusterBinding) error {
	desired := map[string]*bindv1alpha1.APIServiceBinding{}
	for _, apiExport := range binding.Spec.APIExports {
		templateName := strings.TrimSpace(apiExport)
		if templateName == "" {
			continue
		}

		template, err := r.getAPIServiceExportTemplate(ctx, binding.Namespace, templateName)
		if err != nil {
			return fmt.Errorf("failed to get apiserviceexporttemplate %s/%s: %w", binding.Namespace, templateName, err)
		}

		bindingObj := newServiceBinding(binding, template)
		if bindingObj == nil {
			continue
		}
		desired[bindingObj.Name] = bindingObj
	}

	selector := labels.Set{
		appClusterBindingNameLabel:      binding.Name,
		appClusterBindingNamespaceLabel: binding.Namespace,
	}.AsSelector().String()

	existingList, err := r.listServiceBindings(ctx, selector)
	if err != nil {
		return err
	}

	var errs []error
	for i := range existingList.Items {
		existing := existingList.Items[i]
		desiredBinding, shouldExist := desired[existing.Name]
		if !shouldExist {
			if err := r.deleteServiceBinding(ctx, existing.Name); err != nil && !errors.IsNotFound(err) {
				errs = append(errs, err)
			}
			continue
		}

		delete(desired, existing.Name)
		updated := existing.DeepCopy()
		updated.Labels = desiredBinding.Labels
		updated.Spec = bindv1alpha1.APIServiceBindingSpec{
			KubeconfigSecretRef: desiredBinding.Spec.KubeconfigSecretRef,
			PermissionClaims:    existing.Spec.PermissionClaims,
		}

		if !reflect.DeepEqual(existing.Spec, updated.Spec) || !reflect.DeepEqual(existing.Labels, updated.Labels) {
			if _, err := r.updateServiceBinding(ctx, updated); err != nil {
				errs = append(errs, err)
			}
		}
	}

	for _, bindingObj := range desired {
		if _, err := r.createServiceBinding(ctx, bindingObj); err != nil && !errors.IsAlreadyExists(err) {
			errs = append(errs, err)
		}
	}

	return utilerrors.NewAggregate(errs)
}

func (r *reconciler) ensureServiceExportRequests(ctx context.Context, binding *bindv1alpha1.AppClusterBinding) error {
	desired := map[string]*bindv1alpha1.APIServiceExportRequest{}
	var errs []error
	for _, apiExport := range binding.Spec.APIExports {
		templateName := strings.TrimSpace(apiExport)
		if templateName == "" {
			continue
		}

		template, err := r.getAPIServiceExportTemplate(ctx, binding.Namespace, templateName)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to get apiserviceexporttemplate %s/%s: %w", binding.Namespace, templateName, err))
			continue
		}

		req := newAPIServiceExportRequest(binding, template)
		if req == nil {
			continue
		}
		desired[req.Name] = req
	}

	selector := labels.Set{
		appClusterBindingNameLabel:      binding.Name,
		appClusterBindingNamespaceLabel: binding.Namespace,
	}.AsSelector().String()

	existingList, err := r.listAPIServiceExportRequests(ctx, binding.Namespace, selector)
	if err != nil {
		errs = append(errs, err)
		return utilerrors.NewAggregate(errs)
	}

	for i := range existingList.Items {
		existing := existingList.Items[i]
		desiredReq, shouldExist := desired[existing.Name]
		if !shouldExist {
			if err := r.deleteAPIServiceExportRequest(ctx, binding.Namespace, existing.Name); err != nil && !errors.IsNotFound(err) {
				errs = append(errs, err)
			}
			continue
		}

		delete(desired, existing.Name)
		if !reflect.DeepEqual(existing.Spec.Resources, desiredReq.Spec.Resources) {
			if err := r.deleteAPIServiceExportRequest(ctx, binding.Namespace, existing.Name); err != nil && !errors.IsNotFound(err) {
				errs = append(errs, err)
				continue
			}
			if _, err := r.createAPIServiceExportRequest(ctx, binding.Namespace, desiredReq); err != nil && !errors.IsAlreadyExists(err) {
				errs = append(errs, err)
			}
		}
	}

	for _, desiredReq := range desired {
		if _, err := r.createAPIServiceExportRequest(ctx, binding.Namespace, desiredReq); err != nil && !errors.IsAlreadyExists(err) {
			errs = append(errs, err)
		}
	}

	return utilerrors.NewAggregate(errs)
}

func newAPIServiceExportRequest(binding *bindv1alpha1.AppClusterBinding, template *examplebackendv1alpha1.APIServiceExportTemplate) *bindv1alpha1.APIServiceExportRequest {
	if template == nil {
		return nil
	}

	resource := strings.TrimSpace(template.Spec.APIServiceSelector.Resource)
	if resource == "" {
		return nil
	}
	group := strings.TrimSpace(template.Spec.APIServiceSelector.Group)

	return &bindv1alpha1.APIServiceExportRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apiServiceExportRequestName(binding.Name, template.Name),
			Namespace: binding.Namespace,
			Labels:    ensureBindingLabels(map[string]string{}, binding),
		},
		Spec: bindv1alpha1.APIServiceExportRequestSpec{
			Resources: []bindv1alpha1.APIServiceExportRequestResource{
				{
					GroupResource: bindv1alpha1.GroupResource{
						Group:    group,
						Resource: resource,
					},
					PermissionClaims: template.Spec.PermissionClaims,
				},
			},
		},
	}
}

func apiServiceExportRequestName(bindingName, apiExport string) string {
	bindingPart := sanitizeNamePart(bindingName)
	exportPart := sanitizeNamePart(apiExport)
	if bindingPart == "" {
		bindingPart = "binding"
	}
	if exportPart == "" {
		exportPart = "export"
	}

	h := fnv.New32a()
	_, _ = h.Write([]byte(apiExport))
	hash := fmt.Sprintf("%08x", h.Sum32())

	name := fmt.Sprintf("%s-%s-%s-%s", apiServiceExportRequestNamePrefix, bindingPart, exportPart, hash)
	if len(name) <= 63 {
		return name
	}

	overhead := len(apiServiceExportRequestNamePrefix) + len(bindingPart) + len(hash) + 3
	maxExportLen := 63 - overhead
	if maxExportLen < 1 {
		maxExportLen = 1
	}
	if len(exportPart) > maxExportLen {
		exportPart = exportPart[:maxExportLen]
	}

	return fmt.Sprintf("%s-%s-%s-%s", apiServiceExportRequestNamePrefix, bindingPart, exportPart, hash)
}

func sanitizeNamePart(input string) string {
	lower := strings.ToLower(strings.TrimSpace(input))
	var b strings.Builder
	b.Grow(len(lower))
	lastDash := false
	for _, r := range lower {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}

	sanitized := strings.Trim(b.String(), "-")
	if sanitized == "" {
		return ""
	}
	if len(sanitized) > 24 {
		return sanitized[:24]
	}

	return sanitized
}

func newServiceBinding(binding *bindv1alpha1.AppClusterBinding, template *examplebackendv1alpha1.APIServiceExportTemplate) *bindv1alpha1.APIServiceBinding {
	if template == nil {
		return nil
	}

	name := apiServiceBindingName(template)
	if name == "" {
		return nil
	}

	labels := ensureBindingLabels(nil, binding)

	return &bindv1alpha1.APIServiceBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: bindv1alpha1.APIServiceBindingSpec{
			KubeconfigSecretRef: binding.Spec.KubeconfigSecretRef,
		},
	}
}

func apiServiceBindingName(template *examplebackendv1alpha1.APIServiceExportTemplate) string {
	if template == nil {
		return ""
	}

	resource := strings.TrimSpace(template.Spec.APIServiceSelector.Resource)
	if resource == "" {
		return ""
	}

	group := strings.TrimSpace(template.Spec.APIServiceSelector.Group)
	if group == "" {
		return resource
	}

	return resource + "." + group
}

func ownerReferenceForAppClusterBinding(binding *bindv1alpha1.AppClusterBinding) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: bindv1alpha1.SchemeGroupVersion.String(),
		Kind:       "AppClusterBinding",
		Name:       binding.Name,
		UID:        binding.UID,
		Controller: pointer.Bool(true),
	}
}

func ensureBindingLabels(target map[string]string, binding *bindv1alpha1.AppClusterBinding) map[string]string {
	if target == nil {
		target = map[string]string{}
	}
	target[appClusterBindingNameLabel] = binding.Name
	target[appClusterBindingNamespaceLabel] = binding.Namespace
	return target
}

func (r *reconciler) ensureKonnectorDeployment(ctx context.Context, binding *bindv1alpha1.AppClusterBinding) error {
	if binding.Spec.Konnector == nil || !binding.Spec.Konnector.Deploy {
		conditions.MarkTrue(
			binding,
			bindv1alpha1.AppClusterBindingConditionKonnectorDeployed,
		)
		return nil
	}

	// Deploy konnector to control plane cluster in the binding's namespace
	deployment := r.buildKonnectorDeployment(binding)
	deploymentName := fmt.Sprintf("konnector-%s", binding.Name)

	existing, err := r.getDeployment(ctx, binding.Namespace, deploymentName)
	if err != nil && !errors.IsNotFound(err) {
		conditions.MarkFalse(
			binding,
			bindv1alpha1.AppClusterBindingConditionKonnectorDeployed,
			"KonnectorDeploymentFailed",
			conditionsapi.ConditionSeverityError,
			"Failed to get konnector deployment: %v",
			err,
		)
		return err
	} else if errors.IsNotFound(err) {
		if _, err := r.createDeployment(ctx, binding.Namespace, deployment); err != nil {
			conditions.MarkFalse(
				binding,
				bindv1alpha1.AppClusterBindingConditionKonnectorDeployed,
				"KonnectorDeploymentFailed",
				conditionsapi.ConditionSeverityError,
				"Failed to create konnector deployment: %v",
				err,
			)
			return err
		}
		conditions.MarkTrue(
			binding,
			bindv1alpha1.AppClusterBindingConditionKonnectorDeployed,
		)
		return nil
	}

	if !reflect.DeepEqual(existing.Spec, deployment.Spec) {
		updated := existing.DeepCopy()
		updated.Spec = deployment.Spec
		if _, err := r.updateDeployment(ctx, binding.Namespace, updated); err != nil {
			conditions.MarkFalse(
				binding,
				bindv1alpha1.AppClusterBindingConditionKonnectorDeployed,
				"KonnectorDeploymentFailed",
				conditionsapi.ConditionSeverityError,
				"Failed to update konnector deployment: %v",
				err,
			)
			return err
		}
	}

	conditions.MarkTrue(
		binding,
		bindv1alpha1.AppClusterBindingConditionKonnectorDeployed,
	)
	return nil
}

func (r *reconciler) buildKonnectorDeployment(binding *bindv1alpha1.AppClusterBinding) *appsv1.Deployment {
	replicas := int32(1)
	//TODO: change source of the image to a stable location - for now we hardcode a specific digest to ensure immutability
	image := "ghcr.io/lhaendler/konnector-1c3773bcc98f06a77d017340f46f82e4@sha256:854199289848019e1bac5d44a61002964b7ba3056166339d663085135f66196a"

	if binding.Spec.Konnector.Overrides != nil && binding.Spec.Konnector.Overrides.Image != "" {
		image = binding.Spec.Konnector.Overrides.Image
	}

	// Build args for control plane mode
	bindingRootNamespace := r.getBindingRootNamespace(binding)
	args := []string{
		"--control-plane-mode",
		fmt.Sprintf("--app-cluster-kubeconfig-secret-name=%s", binding.Spec.KubeconfigSecretRef.Name),
		fmt.Sprintf("--app-cluster-kubeconfig-secret-namespace=%s", binding.Spec.KubeconfigSecretRef.Namespace),
		fmt.Sprintf("--app-cluster-kubeconfig-secret-key=%s", binding.Spec.KubeconfigSecretRef.Key),
		fmt.Sprintf("--binding-root-namespace=%s", bindingRootNamespace),
		"--lease-namespace=$(POD_NAMESPACE)",
	}

	container := corev1.Container{
		Name:  "konnector",
		Image: image,
		Args:  args,
		Env: []corev1.EnvVar{
			{
				Name: "POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
			{
				Name: "POD_NAMESPACE",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.namespace",
					},
				},
			},
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/healthz",
					Port: intstr.FromInt(8080),
				},
			},
		},
	}

	if binding.Spec.Konnector.Overrides != nil && len(binding.Spec.Konnector.Overrides.ContainerSettings.Raw) > 0 {
		container = r.applyContainerOverrides(container, binding.Spec.Konnector.Overrides.ContainerSettings.Raw)
	}

	labels := map[string]string{
		"app": "konnector",
	}
	labels = ensureBindingLabels(labels, binding)

	deploymentName := fmt.Sprintf("konnector-%s", binding.Name)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: binding.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: bindv1alpha1.SchemeGroupVersion.String(),
					Kind:       "AppClusterBinding",
					Name:       binding.Name,
					UID:        binding.UID,
					Controller: pointer.Bool(true),
				},
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					RestartPolicy:      corev1.RestartPolicyAlways,
					ServiceAccountName: "konnector",
					Containers:         []corev1.Container{container},
				},
			},
		},
	}
}

func (r *reconciler) applyContainerOverrides(base corev1.Container, overrides []byte) corev1.Container {
	baseJSON, err := json.Marshal(base)
	if err != nil {
		return base
	}

	merged, err := strategicpatch.StrategicMergePatch(baseJSON, overrides, corev1.Container{})
	if err != nil {
		return base
	}

	var result corev1.Container
	if err := json.Unmarshal(merged, &result); err != nil {
		return base
	}

	return result
}
