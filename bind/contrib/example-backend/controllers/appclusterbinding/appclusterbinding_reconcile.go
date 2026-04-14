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
	konnectorServiceAccountName       = "klutch-binder"
	controlPlaneConnectorRoleName     = "control-plane-connector"
)

type reconciler struct {
	kubeClient kubernetesclient.Interface

	getSecret func(ns, name string) (*corev1.Secret, error)

	listServiceBindings           func(ctx context.Context, labelSelector string) (*bindv1alpha1.APIServiceBindingList, error)
	createServiceBinding          func(ctx context.Context, binding *bindv1alpha1.APIServiceBinding) (*bindv1alpha1.APIServiceBinding, error)
	updateServiceBinding          func(ctx context.Context, binding *bindv1alpha1.APIServiceBinding) (*bindv1alpha1.APIServiceBinding, error)
	deleteServiceBinding          func(ctx context.Context, namespace, name string) error
	templateFor                   func(ctx context.Context, group, resource string) (examplebackendv1alpha1.APIServiceExportTemplate, error)
	listAPIServiceExportRequests  func(ctx context.Context, namespace, labelSelector string) (*bindv1alpha1.APIServiceExportRequestList, error)
	createAPIServiceExportRequest func(ctx context.Context, namespace string, req *bindv1alpha1.APIServiceExportRequest) (*bindv1alpha1.APIServiceExportRequest, error)
	deleteAPIServiceExportRequest func(ctx context.Context, namespace, name string) error

	getRole                  func(ctx context.Context, namespace, name string) (*rbacv1.Role, error)
	createRole               func(ctx context.Context, namespace string, role *rbacv1.Role) (*rbacv1.Role, error)
	updateRole               func(ctx context.Context, namespace string, role *rbacv1.Role) (*rbacv1.Role, error)
	deleteRole               func(ctx context.Context, namespace, name string) error
	getClusterRole           func(ctx context.Context, name string) (*rbacv1.ClusterRole, error)
	deleteClusterRole        func(ctx context.Context, name string) error
	getClusterRoleBinding    func(ctx context.Context, name string) (*rbacv1.ClusterRoleBinding, error)
	deleteClusterRoleBinding func(ctx context.Context, name string) error
	getRoleBinding           func(ctx context.Context, namespace, name string) (*rbacv1.RoleBinding, error)
	createRoleBinding        func(ctx context.Context, namespace string, rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error)
	updateRoleBinding        func(ctx context.Context, namespace string, rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error)
	deleteRoleBinding        func(ctx context.Context, namespace, name string) error

	getNamespace         func(ctx context.Context, name string) (*corev1.Namespace, error)
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
		if err := r.ensureControlPlaneConnectorRBAC(ctx, binding); err != nil {
			errs = append(errs, err)
		}
		if err := r.ensureServiceBindings(ctx, binding); err != nil {
			errs = append(errs, err)
		}
		if err := r.ensureServiceExportRequests(ctx, binding); err != nil {
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
			namespace := serviceBindings.Items[i].Namespace
			if err := r.deleteServiceBinding(ctx, namespace, name); err != nil && !errors.IsNotFound(err) {
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
	bindingRootNamespace := r.getBindingRootNamespace(binding)
	if err := r.deleteDeployment(ctx, bindingRootNamespace, deploymentName); err != nil && !errors.IsNotFound(err) {
		errs = append(errs, fmt.Errorf("failed to delete deployment %s/%s: %w", bindingRootNamespace, deploymentName, err))
	}
	if _, err := r.getDeployment(ctx, bindingRootNamespace, deploymentName); err != nil {
		if !errors.IsNotFound(err) {
			errs = append(errs, fmt.Errorf("failed to verify deployment cleanup %s/%s: %w", bindingRootNamespace, deploymentName, err))
		}
	} else {
		errs = append(errs, fmt.Errorf("deployment %s/%s still exists", bindingRootNamespace, deploymentName))
	}

	if err := r.deleteRoleBinding(ctx, bindingRootNamespace, controlPlaneConnectorRoleName); err != nil && !errors.IsNotFound(err) {
		errs = append(errs, fmt.Errorf("failed to delete rolebinding %s/%s: %w", bindingRootNamespace, controlPlaneConnectorRoleName, err))
	}
	if _, err := r.getRoleBinding(ctx, bindingRootNamespace, controlPlaneConnectorRoleName); err != nil {
		if !errors.IsNotFound(err) {
			errs = append(errs, fmt.Errorf("failed to verify rolebinding cleanup %s/%s: %w", bindingRootNamespace, controlPlaneConnectorRoleName, err))
		}
	} else {
		errs = append(errs, fmt.Errorf("rolebinding %s/%s still exists", bindingRootNamespace, controlPlaneConnectorRoleName))
	}

	if err := r.deleteRole(ctx, bindingRootNamespace, controlPlaneConnectorRoleName); err != nil && !errors.IsNotFound(err) {
		errs = append(errs, fmt.Errorf("failed to delete role %s/%s: %w", bindingRootNamespace, controlPlaneConnectorRoleName, err))
	}
	if _, err := r.getRole(ctx, bindingRootNamespace, controlPlaneConnectorRoleName); err != nil {
		if !errors.IsNotFound(err) {
			errs = append(errs, fmt.Errorf("failed to verify role cleanup %s/%s: %w", bindingRootNamespace, controlPlaneConnectorRoleName, err))
		}
	} else {
		errs = append(errs, fmt.Errorf("role %s/%s still exists", bindingRootNamespace, controlPlaneConnectorRoleName))
	}

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

	if err := r.ensureBindingRootKubeconfigSecret(ctx, binding, sourceSecret, rootNamespace); err != nil {
		return err
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

func (r *reconciler) ensureBindingRootKubeconfigSecret(ctx context.Context, binding *bindv1alpha1.AppClusterBinding, sourceSecret *corev1.Secret, rootNamespace string) error {
	key := binding.Spec.KubeconfigSecretRef.Key
	kubeconfigData, found := sourceSecret.Data[key]
	if !found {
		return fmt.Errorf("kubeconfig secret %s/%s is missing %q string key",
			binding.Spec.KubeconfigSecretRef.Namespace,
			binding.Spec.KubeconfigSecretRef.Name,
			key)
	}

	labels := ensureBindingLabels(nil, binding)
	expected := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      binding.Spec.KubeconfigSecretRef.Name,
			Namespace: rootNamespace,
			Labels:    labels,
		},
		Type: sourceSecret.Type,
		Data: map[string][]byte{
			key: kubeconfigData,
		},
	}

	existing, err := r.getSecret(rootNamespace, expected.Name)
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to get mirrored kubeconfig secret %s/%s: %w", rootNamespace, expected.Name, err)
		}
		if _, err := r.createSecret(ctx, rootNamespace, expected); err != nil {
			return fmt.Errorf("failed to create mirrored kubeconfig secret %s/%s: %w", rootNamespace, expected.Name, err)
		}
		return nil
	}

	updated := existing.DeepCopy()
	updated.Labels = ensureBindingLabels(updated.Labels, binding)
	updated.Type = expected.Type
	updated.Data = expected.Data
	if !reflect.DeepEqual(existing.Labels, updated.Labels) ||
		!reflect.DeepEqual(existing.Data, updated.Data) ||
		existing.Type != updated.Type {
		if _, err := r.updateSecret(ctx, rootNamespace, updated); err != nil {
			return fmt.Errorf("failed to update mirrored kubeconfig secret %s/%s: %w", rootNamespace, expected.Name, err)
		}
	}

	return nil
}

func (r *reconciler) resolveTemplate(ctx context.Context, apiExport bindv1alpha1.GroupResource) (bindv1alpha1.GroupResource, *examplebackendv1alpha1.APIServiceExportTemplate, error) {
	templateRef := bindv1alpha1.GroupResource{
		Group:    strings.TrimSpace(apiExport.Group),
		Resource: strings.TrimSpace(apiExport.Resource),
	}
	if templateRef.Resource == "" {
		return templateRef, nil, nil
	}

	template, err := r.templateFor(ctx, templateRef.Group, templateRef.Resource)
	if err != nil {
		return templateRef, nil, err
	}

	return templateRef, &template, nil
}

func (r *reconciler) ensureControlPlaneConnectorRBAC(ctx context.Context, binding *bindv1alpha1.AppClusterBinding) error {
	bindingRootNamespace := r.getBindingRootNamespace(binding)

	expectedRole := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      controlPlaneConnectorRoleName,
			Namespace: bindingRootNamespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"klutch.anynines.com"},
				Resources: []string{"apiservicebindings"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
			{
				APIGroups: []string{"klutch.anynines.com"},
				Resources: []string{"apiservicebindings/status"},
				Verbs:     []string{"patch", "update"},
			},
			{
				APIGroups: []string{"coordination.k8s.io"},
				Resources: []string{"leases"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
		},
	}

	role, err := r.getRole(ctx, bindingRootNamespace, controlPlaneConnectorRoleName)
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
			Name:      controlPlaneConnectorRoleName,
			Namespace: bindingRootNamespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      konnectorServiceAccountName,
				Namespace: bindingRootNamespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     controlPlaneConnectorRoleName,
		},
	}

	rb, err := r.getRoleBinding(ctx, bindingRootNamespace, controlPlaneConnectorRoleName)
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to get rolebinding: %w", err)
		}
		if _, err := r.createRoleBinding(ctx, bindingRootNamespace, expectedRB); err != nil {
			return fmt.Errorf("failed to create rolebinding: %w", err)
		}
	} else if !reflect.DeepEqual(rb.Subjects, expectedRB.Subjects) || !reflect.DeepEqual(rb.RoleRef, expectedRB.RoleRef) {
		rb = rb.DeepCopy()
		rb.Subjects = expectedRB.Subjects
		rb.RoleRef = expectedRB.RoleRef
		if _, err := r.updateRoleBinding(ctx, bindingRootNamespace, rb); err != nil {
			return fmt.Errorf("failed to update rolebinding: %w", err)
		}
	}

	return nil
}

func (r *reconciler) ensureServiceBindings(ctx context.Context, binding *bindv1alpha1.AppClusterBinding) error {
	desired := map[string]*bindv1alpha1.APIServiceBinding{}
	for _, apiExport := range binding.Spec.APIExports {
		templateRef, template, err := r.resolveTemplate(ctx, apiExport)
		if err != nil {
			return fmt.Errorf("failed to get apiserviceexporttemplate for %s/%s: %w", templateRef.Group, templateRef.Resource, err)
		}
		if template == nil {
			continue
		}

		bindingObj := newServiceBinding(binding, template, r.getBindingRootNamespace(binding))
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
			if err := r.deleteServiceBinding(ctx, existing.Namespace, existing.Name); err != nil && !errors.IsNotFound(err) {
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
		templateRef, template, err := r.resolveTemplate(ctx, apiExport)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to get apiserviceexporttemplate for %s/%s: %w", templateRef.Group, templateRef.Resource, err))
			continue
		}
		if template == nil {
			continue
		}

		req := r.newAPIServiceExportRequest(binding, template)
		if req == nil {
			continue
		}
		desired[req.Name] = req
	}

	selector := labels.Set{
		appClusterBindingNameLabel:      binding.Name,
		appClusterBindingNamespaceLabel: binding.Namespace,
	}.AsSelector().String()

	bindingRootNamespace := r.getBindingRootNamespace(binding)
	existingList, err := r.listAPIServiceExportRequests(ctx, bindingRootNamespace, selector)
	if err != nil {
		errs = append(errs, err)
		return utilerrors.NewAggregate(errs)
	}

	for i := range existingList.Items {
		existing := existingList.Items[i]
		desiredReq, shouldExist := desired[existing.Name]
		if !shouldExist {
			if err := r.deleteAPIServiceExportRequest(ctx, bindingRootNamespace, existing.Name); err != nil && !errors.IsNotFound(err) {
				errs = append(errs, err)
			}
			continue
		}

		delete(desired, existing.Name)
		if !reflect.DeepEqual(existing.Spec.Resources, desiredReq.Spec.Resources) {
			if err := r.deleteAPIServiceExportRequest(ctx, bindingRootNamespace, existing.Name); err != nil && !errors.IsNotFound(err) {
				errs = append(errs, err)
				continue
			}
			if _, err := r.createAPIServiceExportRequest(ctx, bindingRootNamespace, desiredReq); err != nil && !errors.IsAlreadyExists(err) {
				errs = append(errs, err)
			}
		}
	}

	for _, desiredReq := range desired {
		if _, err := r.createAPIServiceExportRequest(ctx, bindingRootNamespace, desiredReq); err != nil && !errors.IsAlreadyExists(err) {
			errs = append(errs, err)
		}
	}

	return utilerrors.NewAggregate(errs)
}

func (r *reconciler) newAPIServiceExportRequest(binding *bindv1alpha1.AppClusterBinding, template *examplebackendv1alpha1.APIServiceExportTemplate) *bindv1alpha1.APIServiceExportRequest {
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
			Namespace: r.getBindingRootNamespace(binding),
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

func newServiceBinding(binding *bindv1alpha1.AppClusterBinding, template *examplebackendv1alpha1.APIServiceExportTemplate, namespace string) *bindv1alpha1.APIServiceBinding {
	if template == nil {
		return nil
	}

	name := apiServiceBindingName(template)
	if name == "" {
		return nil
	}

	labels := ensureBindingLabels(nil, binding)
	secretRef := binding.Spec.KubeconfigSecretRef
	secretRef.Namespace = namespace

	return &bindv1alpha1.APIServiceBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: bindv1alpha1.APIServiceBindingSpec{
			KubeconfigSecretRef: secretRef,
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

	// Deploy konnector to control plane cluster in the binding root namespace
	bindingRootNamespace := r.getBindingRootNamespace(binding)
	deployment := r.buildKonnectorDeployment(binding)
	deploymentName := fmt.Sprintf("konnector-%s", binding.Name)

	existing, err := r.getDeployment(ctx, bindingRootNamespace, deploymentName)
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
		if _, err := r.createDeployment(ctx, bindingRootNamespace, deployment); err != nil {
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
		if _, err := r.updateDeployment(ctx, bindingRootNamespace, updated); err != nil {
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
	image := "ghcr.io/lhaendler/konnector-1c3773bcc98f06a77d017340f46f82e4@sha256:7fa40d94dcc58fdee883ada75db6ae53844edc92ec0d90d5b515430b45c884f8"

	if binding.Spec.Konnector.Overrides != nil && binding.Spec.Konnector.Overrides.Image != "" {
		image = binding.Spec.Konnector.Overrides.Image
	}

	// Build args for control plane mode
	bindingRootNamespace := r.getBindingRootNamespace(binding)
	args := []string{
		"--control-plane-mode",
		fmt.Sprintf("--app-cluster-kubeconfig-secret-name=%s", binding.Spec.KubeconfigSecretRef.Name),
		fmt.Sprintf("--app-cluster-kubeconfig-secret-namespace=%s", bindingRootNamespace),
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
			Namespace: bindingRootNamespace,
			Labels:    labels,
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
					ServiceAccountName: konnectorServiceAccountName,
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
