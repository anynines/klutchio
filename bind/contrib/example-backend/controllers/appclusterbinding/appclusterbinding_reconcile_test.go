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

package appclusterbinding

import (
	"context"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	examplebackendv1alpha1 "github.com/anynines/klutchio/bind/contrib/example-backend/apis/examplebackend/v1alpha1"
	bindv1alpha1 "github.com/anynines/klutchio/bind/pkg/apis/bind/v1alpha1"
)

func TestNewServiceBindingClusterScoped(t *testing.T) {
	binding := &bindv1alpha1.AppClusterBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-binding",
			Namespace: "my-ns",
		},
		Spec: bindv1alpha1.AppClusterBindingSpec{
			KubeconfigSecretRef: bindv1alpha1.ClusterSecretKeyRef{
				LocalSecretKeyRef: bindv1alpha1.LocalSecretKeyRef{
					Name: "kubeconfig",
					Key:  "kubeconfig",
				},
				Namespace: "source-ns",
			},
		},
	}
	template := &examplebackendv1alpha1.APIServiceExportTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "widgets"},
		Spec: examplebackendv1alpha1.APIServiceExportTemplateSpec{
			APIServiceSelector: examplebackendv1alpha1.APIServiceSelector{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "klutch.anynines.com",
					Resource: "widgets",
				},
				Version: "v1alpha1",
			},
		},
	}

	got := newServiceBinding(binding, template)
	if got.Namespace != "" {
		t.Fatalf("expected cluster-scoped APIServiceBinding with empty namespace, got %q", got.Namespace)
	}
	if got.Name != "widgets.klutch.anynines.com" {
		t.Fatalf("expected fully qualified binding name, got %q", got.Name)
	}
}

func TestAPIServiceBindingName(t *testing.T) {
	template := &examplebackendv1alpha1.APIServiceExportTemplate{
		Spec: examplebackendv1alpha1.APIServiceExportTemplateSpec{
			APIServiceSelector: examplebackendv1alpha1.APIServiceSelector{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "anynines.com",
					Resource: "postgresqlinstances",
				},
				Version: "v1alpha1",
			},
		},
	}

	if got := apiServiceBindingName(template); got != "postgresqlinstances.anynines.com" {
		t.Fatalf("expected fully qualified binding name, got %q", got)
	}
}

func TestNewAPIServiceExportRequest(t *testing.T) {
	binding := &bindv1alpha1.AppClusterBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-binding",
			Namespace: "my-ns",
		},
	}
	template := &examplebackendv1alpha1.APIServiceExportTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "postgresql"},
		Spec: examplebackendv1alpha1.APIServiceExportTemplateSpec{
			APIServiceSelector: examplebackendv1alpha1.APIServiceSelector{
				GroupResource: bindv1alpha1.GroupResource{
					Group:    "db.example.com",
					Resource: "postgresqls",
				},
				Version: "v1alpha1",
			},
			PermissionClaims: []bindv1alpha1.PermissionClaim{{
				GroupResource: bindv1alpha1.GroupResource{Group: "", Resource: "secrets"},
				Version:       "v1",
			}},
		},
	}

	req := newAPIServiceExportRequest(binding, template)
	if req == nil {
		t.Fatal("expected request to be created")
	}
	if req.Namespace != "my-ns" {
		t.Fatalf("expected namespace my-ns, got %q", req.Namespace)
	}
	if req.Spec.Resources[0].Resource != "postgresqls" {
		t.Fatalf("expected resource postgresqls, got %q", req.Spec.Resources[0].Resource)
	}
	if req.Spec.Resources[0].Group != "db.example.com" {
		t.Fatalf("expected group db.example.com, got %q", req.Spec.Resources[0].Group)
	}
	if len(req.Spec.Resources[0].PermissionClaims) != 1 {
		t.Fatalf("expected one permission claim, got %d", len(req.Spec.Resources[0].PermissionClaims))
	}
	if req.Labels[appClusterBindingNameLabel] != "my-binding" {
		t.Fatalf("expected appclusterbinding-name label, got %q", req.Labels[appClusterBindingNameLabel])
	}
}

func TestEnsureServiceExportRequestsUsesTemplateLookup(t *testing.T) {
	binding := &bindv1alpha1.AppClusterBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-binding",
			Namespace: "my-ns",
		},
		Spec: bindv1alpha1.AppClusterBindingSpec{
			APIExports: []string{"postgresql"},
		},
	}

	created := []*bindv1alpha1.APIServiceExportRequest{}
	r := &reconciler{
		getAPIServiceExportTemplate: func(ctx context.Context, namespace, name string) (*examplebackendv1alpha1.APIServiceExportTemplate, error) {
			if namespace != "my-ns" || name != "postgresql" {
				t.Fatalf("unexpected template lookup %s/%s", namespace, name)
			}
			return &examplebackendv1alpha1.APIServiceExportTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "postgresql"},
				Spec: examplebackendv1alpha1.APIServiceExportTemplateSpec{
					APIServiceSelector: examplebackendv1alpha1.APIServiceSelector{
						GroupResource: bindv1alpha1.GroupResource{Group: "db.example.com", Resource: "postgresqls"},
						Version:       "v1alpha1",
					},
				},
			}, nil
		},
		listAPIServiceExportRequests: func(ctx context.Context, namespace, labelSelector string) (*bindv1alpha1.APIServiceExportRequestList, error) {
			return &bindv1alpha1.APIServiceExportRequestList{}, nil
		},
		createAPIServiceExportRequest: func(ctx context.Context, namespace string, req *bindv1alpha1.APIServiceExportRequest) (*bindv1alpha1.APIServiceExportRequest, error) {
			created = append(created, req.DeepCopy())
			return req, nil
		},
		deleteAPIServiceExportRequest: func(ctx context.Context, namespace, name string) error {
			return nil
		},
	}

	if err := r.ensureServiceExportRequests(context.Background(), binding); err != nil {
		t.Fatalf("expected reconcile success, got %v", err)
	}
	if len(created) != 1 {
		t.Fatalf("expected one created request, got %d", len(created))
	}
	if created[0].Spec.Resources[0].Resource != "postgresqls" {
		t.Fatalf("expected template resource postgresqls, got %q", created[0].Spec.Resources[0].Resource)
	}
}

func TestEnsureDeletedBlocksUntilNamespaceGone(t *testing.T) {
	notFound := func(resource, name string) error {
		return apierrors.NewNotFound(schema.GroupResource{Resource: resource}, name)
	}

	r := &reconciler{
		listServiceBindings: func(ctx context.Context, labelSelector string) (*bindv1alpha1.APIServiceBindingList, error) {
			return &bindv1alpha1.APIServiceBindingList{}, nil
		},
		deleteServiceBinding: func(ctx context.Context, name string) error {
			return nil
		},
		listAPIServiceExportRequests: func(ctx context.Context, namespace, labelSelector string) (*bindv1alpha1.APIServiceExportRequestList, error) {
			return &bindv1alpha1.APIServiceExportRequestList{}, nil
		},
		deleteAPIServiceExportRequest: func(ctx context.Context, namespace, name string) error {
			return nil
		},
		deleteClusterRoleBinding: func(ctx context.Context, name string) error {
			return nil
		},
		getClusterRoleBinding: func(ctx context.Context, name string) (*rbacv1.ClusterRoleBinding, error) {
			return nil, notFound("clusterrolebindings", name)
		},
		deleteClusterRole: func(ctx context.Context, name string) error {
			return nil
		},
		getClusterRole: func(ctx context.Context, name string) (*rbacv1.ClusterRole, error) {
			return nil, notFound("clusterroles", name)
		},
		deleteDeployment: func(ctx context.Context, namespace, name string) error {
			return nil
		},
		getDeployment: func(ctx context.Context, namespace, name string) (*appsv1.Deployment, error) {
			return nil, notFound("deployments", name)
		},
		deleteRoleBinding: func(ctx context.Context, namespace, name string) error {
			return nil
		},
		getRoleBinding: func(ctx context.Context, namespace, name string) (*rbacv1.RoleBinding, error) {
			return nil, notFound("rolebindings", name)
		},
		deleteRole: func(ctx context.Context, namespace, name string) error {
			return nil
		},
		getRole: func(ctx context.Context, namespace, name string) (*rbacv1.Role, error) {
			return nil, notFound("roles", name)
		},
		deleteNamespace: func(ctx context.Context, name string) error {
			return nil
		},
		getNamespace: func(ctx context.Context, name string) (*corev1.Namespace, error) {
			return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}, nil
		},
	}

	binding := &bindv1alpha1.AppClusterBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-binding",
			Namespace: "my-ns",
		},
		Spec: bindv1alpha1.AppClusterBindingSpec{
			KubeconfigSecretRef: bindv1alpha1.ClusterSecretKeyRef{
				LocalSecretKeyRef: bindv1alpha1.LocalSecretKeyRef{
					Name: "kubeconfig",
					Key:  "kubeconfig",
				},
				Namespace: "source-ns",
			},
		},
	}

	err := r.ensureDeleted(context.Background(), binding)
	if err == nil {
		t.Fatal("expected strict cleanup to fail while namespace still exists")
	}
	if !strings.Contains(err.Error(), "namespace") || !strings.Contains(err.Error(), "still exists") {
		t.Fatalf("expected namespace still exists error, got %v", err)
	}
}

func TestEnsureDeletedBlocksUntilAPIServiceExportRequestsGone(t *testing.T) {
	notFound := func(resource, name string) error {
		return apierrors.NewNotFound(schema.GroupResource{Resource: resource}, name)
	}

	r := &reconciler{
		listServiceBindings: func(ctx context.Context, labelSelector string) (*bindv1alpha1.APIServiceBindingList, error) {
			return &bindv1alpha1.APIServiceBindingList{}, nil
		},
		deleteServiceBinding: func(ctx context.Context, name string) error {
			return nil
		},
		listAPIServiceExportRequests: func(ctx context.Context, namespace, labelSelector string) (*bindv1alpha1.APIServiceExportRequestList, error) {
			return &bindv1alpha1.APIServiceExportRequestList{
				Items: []bindv1alpha1.APIServiceExportRequest{{ObjectMeta: metav1.ObjectMeta{Name: "req-a", Namespace: namespace}}},
			}, nil
		},
		deleteAPIServiceExportRequest: func(ctx context.Context, namespace, name string) error {
			return nil
		},
		deleteClusterRoleBinding: func(ctx context.Context, name string) error {
			return nil
		},
		getClusterRoleBinding: func(ctx context.Context, name string) (*rbacv1.ClusterRoleBinding, error) {
			return nil, notFound("clusterrolebindings", name)
		},
		deleteClusterRole: func(ctx context.Context, name string) error {
			return nil
		},
		getClusterRole: func(ctx context.Context, name string) (*rbacv1.ClusterRole, error) {
			return nil, notFound("clusterroles", name)
		},
		deleteDeployment: func(ctx context.Context, namespace, name string) error {
			return nil
		},
		getDeployment: func(ctx context.Context, namespace, name string) (*appsv1.Deployment, error) {
			return nil, notFound("deployments", name)
		},
		deleteRoleBinding: func(ctx context.Context, namespace, name string) error {
			return nil
		},
		getRoleBinding: func(ctx context.Context, namespace, name string) (*rbacv1.RoleBinding, error) {
			return nil, notFound("rolebindings", name)
		},
		deleteRole: func(ctx context.Context, namespace, name string) error {
			return nil
		},
		getRole: func(ctx context.Context, namespace, name string) (*rbacv1.Role, error) {
			return nil, notFound("roles", name)
		},
		deleteNamespace: func(ctx context.Context, name string) error {
			return nil
		},
		getNamespace: func(ctx context.Context, name string) (*corev1.Namespace, error) {
			return nil, notFound("namespaces", name)
		},
	}

	binding := &bindv1alpha1.AppClusterBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-binding",
			Namespace: "my-ns",
		},
		Spec: bindv1alpha1.AppClusterBindingSpec{
			KubeconfigSecretRef: bindv1alpha1.ClusterSecretKeyRef{
				LocalSecretKeyRef: bindv1alpha1.LocalSecretKeyRef{
					Name: "kubeconfig",
					Key:  "kubeconfig",
				},
				Namespace: "source-ns",
			},
		},
	}

	err := r.ensureDeleted(context.Background(), binding)
	if err == nil {
		t.Fatal("expected strict cleanup to fail while apiserviceexportrequests still exist")
	}
	if !strings.Contains(err.Error(), "apiserviceexportrequest") {
		t.Fatalf("expected apiserviceexportrequest cleanup error, got %v", err)
	}
}
