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

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"

	examplebackendv1alpha1 "github.com/anynines/klutchio/bind/contrib/example-backend/apis/examplebackend/v1alpha1"
	bindv1alpha1 "github.com/anynines/klutchio/bind/pkg/apis/bind/v1alpha1"
	konnectorpkg "github.com/anynines/klutchio/bind/pkg/konnector"
)

func TestNewServiceBindingNamespaced(t *testing.T) {
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
				Version: pointer.String("v1alpha1"),
			},
		},
	}

	got := newServiceBinding(binding, template, "root-ns")
	if got.Namespace != "root-ns" {
		t.Fatalf("expected namespaced APIServiceBinding in root namespace, got %q", got.Namespace)
	}
	if got.Name != "widgets.klutch.anynines.com" {
		t.Fatalf("expected fully qualified binding name, got %q", got.Name)
	}
	if got.Spec.KubeconfigSecretRef.Namespace != "root-ns" {
		t.Fatalf("expected kubeconfig secret namespace root-ns, got %q", got.Spec.KubeconfigSecretRef.Namespace)
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
				Version: pointer.String("v1alpha1"),
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
				Version: pointer.String("v1alpha1"),
			},
			PermissionClaims: []bindv1alpha1.PermissionClaim{{
				GroupResource: bindv1alpha1.GroupResource{Group: "", Resource: "secrets"},
				Version:       "v1",
			}},
		},
	}

	r := &reconciler{}
	req := r.newAPIServiceExportRequest(binding, template)
	if req == nil {
		t.Fatal("expected request to be created")
	}
	expectedNamespace := "klutch-bind-my-ns-my-binding"
	if req.Namespace != expectedNamespace {
		t.Fatalf("expected namespace %s, got %q", expectedNamespace, req.Namespace)
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
			APIExports: []bindv1alpha1.GroupResource{{Group: "db.example.com", Resource: "postgresqls"}},
		},
	}

	created := []*bindv1alpha1.APIServiceExportRequest{}
	r := &reconciler{
		templateFor: func(ctx context.Context, group, resource string) (examplebackendv1alpha1.APIServiceExportTemplate, error) {
			if group != "db.example.com" || resource != "postgresqls" {
				t.Fatalf("unexpected template lookup %s/%s", group, resource)
			}
			return examplebackendv1alpha1.APIServiceExportTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "postgresql"},
				Spec: examplebackendv1alpha1.APIServiceExportTemplateSpec{
					APIServiceSelector: examplebackendv1alpha1.APIServiceSelector{
						GroupResource: bindv1alpha1.GroupResource{Group: "db.example.com", Resource: "postgresqls"},
						Version:       pointer.String("v1alpha1"),
					},
				},
			}, nil
		},
		listAPIServiceExports: func(ctx context.Context, namespace string) (*bindv1alpha1.APIServiceExportList, error) {
			return &bindv1alpha1.APIServiceExportList{}, nil
		},
		deleteAPIServiceExport: func(ctx context.Context, namespace, name string) error {
			return nil
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

func TestRemovingAPIDeletesServiceBindingAndExportRequest(t *testing.T) {
	bindingRootNamespace := "klutch-bind-my-ns-my-binding"

	binding := &bindv1alpha1.AppClusterBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-binding",
			Namespace: "my-ns",
		},
		Spec: bindv1alpha1.AppClusterBindingSpec{
			// APIExports is empty — the postgresql API has been removed.
			APIExports: []bindv1alpha1.GroupResource{},
		},
	}

	var deletedBindings []string
	var deletedExportRequests []string
	var deletedExports []string

	r := &reconciler{
		templateFor: func(ctx context.Context, group, resource string) (examplebackendv1alpha1.APIServiceExportTemplate, error) {
			t.Fatalf("unexpected template lookup for removed API %s/%s", group, resource)
			return examplebackendv1alpha1.APIServiceExportTemplate{}, nil
		},
		// Return an existing APIServiceBinding for the removed API.
		listServiceBindings: func(ctx context.Context, labelSelector string) (*bindv1alpha1.APIServiceBindingList, error) {
			return &bindv1alpha1.APIServiceBindingList{
				Items: []bindv1alpha1.APIServiceBinding{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "postgresqls.db.example.com",
							Namespace: bindingRootNamespace,
							Labels: map[string]string{
								appClusterBindingNameLabel:      binding.Name,
								appClusterBindingNamespaceLabel: binding.Namespace,
							},
						},
					},
				},
			}, nil
		},
		deleteServiceBinding: func(ctx context.Context, namespace, name string) error {
			deletedBindings = append(deletedBindings, namespace+"/"+name)
			return nil
		},
		updateServiceBinding: func(ctx context.Context, binding *bindv1alpha1.APIServiceBinding) (*bindv1alpha1.APIServiceBinding, error) {
			t.Fatal("unexpected update call on service binding")
			return binding, nil
		},
		createServiceBinding: func(ctx context.Context, binding *bindv1alpha1.APIServiceBinding) (*bindv1alpha1.APIServiceBinding, error) {
			t.Fatal("unexpected create call on service binding")
			return binding, nil
		},
		// Return an existing APIServiceExport for the removed API.
		listAPIServiceExports: func(ctx context.Context, namespace string) (*bindv1alpha1.APIServiceExportList, error) {
			return &bindv1alpha1.APIServiceExportList{
				Items: []bindv1alpha1.APIServiceExport{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "postgresqls.db.example.com",
							Namespace: bindingRootNamespace,
						},
					},
				},
			}, nil
		},
		deleteAPIServiceExport: func(ctx context.Context, namespace, name string) error {
			deletedExports = append(deletedExports, namespace+"/"+name)
			return nil
		},
		// Return an existing APIServiceExportRequest for the removed API.
		listAPIServiceExportRequests: func(ctx context.Context, namespace, labelSelector string) (*bindv1alpha1.APIServiceExportRequestList, error) {
			return &bindv1alpha1.APIServiceExportRequestList{
				Items: []bindv1alpha1.APIServiceExportRequest{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "appclusterbinding-my-binding-postgresql-abcd1234",
							Namespace: bindingRootNamespace,
							Labels: map[string]string{
								appClusterBindingNameLabel:      binding.Name,
								appClusterBindingNamespaceLabel: binding.Namespace,
							},
						},
						Spec: bindv1alpha1.APIServiceExportRequestSpec{
							Resources: []bindv1alpha1.APIServiceExportRequestResource{
								{
									GroupResource: bindv1alpha1.GroupResource{
										Group:    "db.example.com",
										Resource: "postgresqls",
									},
								},
							},
						},
					},
				},
			}, nil
		},
		deleteAPIServiceExportRequest: func(ctx context.Context, namespace, name string) error {
			deletedExportRequests = append(deletedExportRequests, namespace+"/"+name)
			return nil
		},
		createAPIServiceExportRequest: func(ctx context.Context, namespace string, req *bindv1alpha1.APIServiceExportRequest) (*bindv1alpha1.APIServiceExportRequest, error) {
			t.Fatal("unexpected create call on export request")
			return req, nil
		},
	}

	if err := r.ensureServiceBindings(context.Background(), binding); err != nil {
		t.Fatalf("ensureServiceBindings failed: %v", err)
	}
	if err := r.ensureServiceExportRequests(context.Background(), binding); err != nil {
		t.Fatalf("ensureServiceExportRequests failed: %v", err)
	}

	if len(deletedBindings) != 1 {
		t.Fatalf("expected 1 deleted service binding, got %d", len(deletedBindings))
	}
	expectedBinding := bindingRootNamespace + "/postgresqls.db.example.com"
	if deletedBindings[0] != expectedBinding {
		t.Fatalf("expected deleted binding %q, got %q", expectedBinding, deletedBindings[0])
	}

	if len(deletedExportRequests) != 1 {
		t.Fatalf("expected 1 deleted export request, got %d", len(deletedExportRequests))
	}
	expectedReq := bindingRootNamespace + "/appclusterbinding-my-binding-postgresql-abcd1234"
	if deletedExportRequests[0] != expectedReq {
		t.Fatalf("expected deleted export request %q, got %q", expectedReq, deletedExportRequests[0])
	}

	if len(deletedExports) != 1 {
		t.Fatalf("expected 1 deleted export, got %d", len(deletedExports))
	}
	expectedExport := bindingRootNamespace + "/postgresqls.db.example.com"
	if deletedExports[0] != expectedExport {
		t.Fatalf("expected deleted export %q, got %q", expectedExport, deletedExports[0])
	}
}

func TestExportExistsSkipsRequestCreation(t *testing.T) {
	binding := &bindv1alpha1.AppClusterBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-binding",
			Namespace: "my-ns",
		},
		Spec: bindv1alpha1.AppClusterBindingSpec{
			APIExports: []bindv1alpha1.GroupResource{{Group: "db.example.com", Resource: "postgresqls"}},
		},
	}

	r := &reconciler{
		templateFor: func(ctx context.Context, group, resource string) (examplebackendv1alpha1.APIServiceExportTemplate, error) {
			return examplebackendv1alpha1.APIServiceExportTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "postgresql"},
				Spec: examplebackendv1alpha1.APIServiceExportTemplateSpec{
					APIServiceSelector: examplebackendv1alpha1.APIServiceSelector{
						GroupResource: bindv1alpha1.GroupResource{Group: "db.example.com", Resource: "postgresqls"},
						Version:       pointer.String("v1alpha1"),
					},
				},
			}, nil
		},
		// Export already exists — no request should be created.
		listAPIServiceExports: func(ctx context.Context, namespace string) (*bindv1alpha1.APIServiceExportList, error) {
			return &bindv1alpha1.APIServiceExportList{
				Items: []bindv1alpha1.APIServiceExport{
					{ObjectMeta: metav1.ObjectMeta{Name: "postgresqls.db.example.com", Namespace: namespace}},
				},
			}, nil
		},
		deleteAPIServiceExport: func(ctx context.Context, namespace, name string) error {
			t.Fatalf("unexpected delete of export %s", name)
			return nil
		},
		listAPIServiceExportRequests: func(ctx context.Context, namespace, labelSelector string) (*bindv1alpha1.APIServiceExportRequestList, error) {
			return &bindv1alpha1.APIServiceExportRequestList{}, nil
		},
		createAPIServiceExportRequest: func(ctx context.Context, namespace string, req *bindv1alpha1.APIServiceExportRequest) (*bindv1alpha1.APIServiceExportRequest, error) {
			t.Fatal("unexpected create of export request — export already exists")
			return req, nil
		},
		deleteAPIServiceExportRequest: func(ctx context.Context, namespace, name string) error {
			return nil
		},
	}

	if err := r.ensureServiceExportRequests(context.Background(), binding); err != nil {
		t.Fatalf("unexpected error: %v", err)
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
		deleteServiceBinding: func(ctx context.Context, namespace, name string) error {
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

func TestEnsureDeletedBlocksUntilClusterRoleGone(t *testing.T) {
	notFound := func(resource, name string) error {
		return apierrors.NewNotFound(schema.GroupResource{Resource: resource}, name)
	}

	r := &reconciler{
		listServiceBindings: func(ctx context.Context, labelSelector string) (*bindv1alpha1.APIServiceBindingList, error) {
			return &bindv1alpha1.APIServiceBindingList{}, nil
		},
		deleteServiceBinding: func(ctx context.Context, namespace, name string) error {
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
			// Simulate the cluster role still existing after delete
			return &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: name}}, nil
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
		t.Fatal("expected strict cleanup to fail while clusterrole still exists")
	}
	if !strings.Contains(err.Error(), "clusterrole") || !strings.Contains(err.Error(), "still exists") {
		t.Fatalf("expected clusterrole still exists error, got %v", err)
	}
}

func TestBuildKonnectorDeploymentConnectorOverrides(t *testing.T) {
	baseBinding := func() *bindv1alpha1.AppClusterBinding {
		return &bindv1alpha1.AppClusterBinding{
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
				Konnector: &bindv1alpha1.KonnectorSpec{
					Deploy: true,
				},
			},
		}
	}

	tests := []struct {
		name      string
		overrides *bindv1alpha1.ContainerOverrides
		validate  func(t *testing.T, containers []corev1.Container)
	}{
		{
			name:      "no overrides uses defaults",
			overrides: nil,
			validate: func(t *testing.T, containers []corev1.Container) {
				if len(containers) != 1 {
					t.Fatalf("expected 1 container, got %d", len(containers))
				}
				expectedImage := konnectorpkg.ImageRepository + ":" + konnectorpkg.Version
				if containers[0].Image != expectedImage {
					t.Fatalf("expected default image %q, got %q", expectedImage, containers[0].Image)
				}
			},
		},
		{
			name: "image override only",
			overrides: &bindv1alpha1.ContainerOverrides{
				Image: "my-registry.example.com/konnector:v2.0.0",
			},
			validate: func(t *testing.T, containers []corev1.Container) {
				if containers[0].Image != "my-registry.example.com/konnector:v2.0.0" {
					t.Fatalf("expected custom image, got %q", containers[0].Image)
				}
			},
		},
		{
			name: "override adds resource limits",
			overrides: &bindv1alpha1.ContainerOverrides{
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
				},
			},
			validate: func(t *testing.T, containers []corev1.Container) {
				limits := containers[0].Resources.Limits
				if limits == nil {
					t.Fatal("expected resource limits to be set")
				}
				if limits.Cpu().Cmp(resource.MustParse("500m")) != 0 {
					t.Fatalf("expected cpu limit 500m, got %v", limits.Cpu())
				}
				if limits.Memory().Cmp(resource.MustParse("256Mi")) != 0 {
					t.Fatalf("expected memory limit 256Mi, got %v", limits.Memory())
				}
			},
		},
		{
			name: "override adds resource requests",
			overrides: &bindv1alpha1.ContainerOverrides{
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("64Mi"),
					},
				},
			},
			validate: func(t *testing.T, containers []corev1.Container) {
				requests := containers[0].Resources.Requests
				if requests == nil {
					t.Fatal("expected resource requests to be set")
				}
				if requests.Cpu().Cmp(resource.MustParse("100m")) != 0 {
					t.Fatalf("expected cpu request 100m, got %v", requests.Cpu())
				}
				if requests.Memory().Cmp(resource.MustParse("64Mi")) != 0 {
					t.Fatalf("expected memory request 64Mi, got %v", requests.Memory())
				}
			},
		},
		{
			name: "override adds environment variable",
			overrides: &bindv1alpha1.ContainerOverrides{
				Env: []corev1.EnvVar{
					{Name: "LOG_LEVEL", Value: "debug"},
				},
			},
			validate: func(t *testing.T, containers []corev1.Container) {
				found := false
				for _, env := range containers[0].Env {
					if env.Name == "LOG_LEVEL" && env.Value == "debug" {
						found = true
						break
					}
				}
				if !found {
					t.Fatal("expected LOG_LEVEL env var to be present")
				}
			},
		},
		{
			name: "override merges env vars with existing",
			overrides: &bindv1alpha1.ContainerOverrides{
				Env: []corev1.EnvVar{
					{Name: "EXTRA_VAR", Value: "extra"},
				},
			},
			validate: func(t *testing.T, containers []corev1.Container) {
				// Base deployment has POD_NAME and POD_NAMESPACE env vars.
				// Strategic merge patch on env (keyed by name) should preserve them.
				envNames := map[string]bool{}
				for _, env := range containers[0].Env {
					envNames[env.Name] = true
				}
				if !envNames["POD_NAME"] {
					t.Fatal("expected POD_NAME env var to be preserved")
				}
				if !envNames["POD_NAMESPACE"] {
					t.Fatal("expected POD_NAMESPACE env var to be preserved")
				}
				if !envNames["EXTRA_VAR"] {
					t.Fatal("expected EXTRA_VAR env var to be added")
				}
			},
		},
		{
			name: "override replaces readiness probe",
			overrides: &bindv1alpha1.ContainerOverrides{
				ReadinessProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: "/ready",
							Port: intstr.FromInt(9090),
						},
					},
				},
			},
			validate: func(t *testing.T, containers []corev1.Container) {
				probe := containers[0].ReadinessProbe
				if probe == nil {
					t.Fatal("expected readiness probe to be set")
				}
				if probe.HTTPGet.Path != "/ready" {
					t.Fatalf("expected readiness probe path /ready, got %q", probe.HTTPGet.Path)
				}
				if probe.HTTPGet.Port.IntValue() != 9090 {
					t.Fatalf("expected readiness probe port 9090, got %d", probe.HTTPGet.Port.IntValue())
				}
			},
		},
		{
			name: "image override combined with resource limits",
			overrides: &bindv1alpha1.ContainerOverrides{
				Image: "custom-image:latest",
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("512Mi"),
					},
				},
			},
			validate: func(t *testing.T, containers []corev1.Container) {
				if containers[0].Image != "custom-image:latest" {
					t.Fatalf("expected custom image, got %q", containers[0].Image)
				}
				limits := containers[0].Resources.Limits
				if limits == nil {
					t.Fatal("expected resource limits")
				}
				if limits.Memory().Cmp(resource.MustParse("512Mi")) != 0 {
					t.Fatalf("expected memory limit 512Mi, got %v", limits.Memory())
				}
			},
		},
		{
			name:      "empty container override is no-op",
			overrides: &bindv1alpha1.ContainerOverrides{},
			validate: func(t *testing.T, containers []corev1.Container) {
				expectedImage := konnectorpkg.ImageRepository + ":" + konnectorpkg.Version
				if containers[0].Image != expectedImage {
					t.Fatalf("expected default image after no-op patch, got %q", containers[0].Image)
				}
			},
		},
		{
			name: "override adds volume mount",
			overrides: &bindv1alpha1.ContainerOverrides{
				VolumeMounts: []corev1.VolumeMount{
					{Name: "config", MountPath: "/etc/config"},
				},
			},
			validate: func(t *testing.T, containers []corev1.Container) {
				found := false
				for _, vm := range containers[0].VolumeMounts {
					if vm.Name == "config" && vm.MountPath == "/etc/config" {
						found = true
						break
					}
				}
				if !found {
					t.Fatal("expected volume mount 'config' at /etc/config")
				}
			},
		},
		{
			name: "image override does not affect args or env vars",
			overrides: &bindv1alpha1.ContainerOverrides{
				Image: "custom-registry.io/konnector:v3.0.0",
			},
			validate: func(t *testing.T, containers []corev1.Container) {
				if containers[0].Image != "custom-registry.io/konnector:v3.0.0" {
					t.Fatalf("expected custom image, got %q", containers[0].Image)
				}
				// Args must still contain control-plane-mode flags.
				if len(containers[0].Args) == 0 {
					t.Fatal("expected args to be preserved")
				}
				found := false
				for _, arg := range containers[0].Args {
					if arg == "--control-plane-mode" {
						found = true
						break
					}
				}
				if !found {
					t.Fatal("expected --control-plane-mode arg to be preserved")
				}
				// Env vars must be preserved.
				envNames := map[string]bool{}
				for _, env := range containers[0].Env {
					envNames[env.Name] = true
				}
				if !envNames["POD_NAME"] || !envNames["POD_NAMESPACE"] {
					t.Fatal("expected base env vars to be preserved")
				}
				// Readiness probe must be preserved.
				if containers[0].ReadinessProbe == nil || containers[0].ReadinessProbe.HTTPGet == nil {
					t.Fatal("expected readiness probe to be preserved")
				}
				// Container name must be preserved.
				if containers[0].Name != konnectorpkg.ContainerName {
					t.Fatalf("expected container name %q, got %q", konnectorpkg.ContainerName, containers[0].Name)
				}
			},
		},
		{
			name: "resource override does not affect image or args",
			overrides: &bindv1alpha1.ContainerOverrides{
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					},
				},
			},
			validate: func(t *testing.T, containers []corev1.Container) {
				// Resources applied.
				if containers[0].Resources.Limits.Memory().Cmp(resource.MustParse("1Gi")) != 0 {
					t.Fatalf("expected memory limit 1Gi, got %v", containers[0].Resources.Limits.Memory())
				}
				// Image must still be the default.
				expectedImage := konnectorpkg.ImageRepository + ":" + konnectorpkg.Version
				if containers[0].Image != expectedImage {
					t.Fatalf("expected default image %q, got %q", expectedImage, containers[0].Image)
				}
				// Args must be unchanged.
				found := false
				for _, arg := range containers[0].Args {
					if arg == "--control-plane-mode" {
						found = true
						break
					}
				}
				if !found {
					t.Fatal("expected --control-plane-mode arg to be preserved")
				}
				// Readiness probe must be preserved.
				if containers[0].ReadinessProbe == nil {
					t.Fatal("expected readiness probe to be preserved")
				}
			},
		},
		{
			name: "env override does not affect image, args, or readiness probe",
			overrides: &bindv1alpha1.ContainerOverrides{
				Env: []corev1.EnvVar{
					{Name: "CUSTOM_VAR", Value: "custom"},
				},
			},
			validate: func(t *testing.T, containers []corev1.Container) {
				// New env var added.
				foundCustom := false
				for _, env := range containers[0].Env {
					if env.Name == "CUSTOM_VAR" && env.Value == "custom" {
						foundCustom = true
					}
				}
				if !foundCustom {
					t.Fatal("expected CUSTOM_VAR to be added")
				}
				// Image must be unchanged.
				expectedImage := konnectorpkg.ImageRepository + ":" + konnectorpkg.Version
				if containers[0].Image != expectedImage {
					t.Fatalf("expected default image %q, got %q", expectedImage, containers[0].Image)
				}
				// Args must be unchanged.
				if len(containers[0].Args) == 0 {
					t.Fatal("expected args to be preserved")
				}
				// Readiness probe must be preserved.
				if containers[0].ReadinessProbe == nil || containers[0].ReadinessProbe.HTTPGet == nil {
					t.Fatal("expected readiness probe to be preserved")
				}
				if containers[0].ReadinessProbe.HTTPGet.Path != konnectorpkg.HealthzPath {
					t.Fatalf("expected readiness probe path %q, got %q", konnectorpkg.HealthzPath, containers[0].ReadinessProbe.HTTPGet.Path)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binding := baseBinding()
			binding.Spec.Konnector.Overrides = tt.overrides

			r := &reconciler{}
			deployment, err := r.buildKonnectorDeployment(binding)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.validate != nil {
				tt.validate(t, deployment.Spec.Template.Spec.Containers)
			}
		})
	}
}
