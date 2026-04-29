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

package konnector

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	bindv1alpha1 "github.com/anynines/klutchio/bind/pkg/apis/bind/v1alpha1"
	conditionsapi "github.com/anynines/klutchio/bind/pkg/apis/third_party/conditions/apis/conditions/v1alpha1"
	bindclient "github.com/anynines/klutchio/bind/pkg/client/clientset/versioned"
	"github.com/anynines/klutchio/bind/test/e2e/framework"
)

func TestControlPlaneModeKonnectorDeployment(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	t.Logf("Creating provider workspace")
	providerConfig, providerKubeconfig := framework.NewWorkspace(t, framework.ClientConfig(t), framework.WithGenerateName("test-control-plane-provider"))

	t.Logf("Starting backend in control plane mode")
	framework.StartBackendWithoutDefaultArgs(t, providerConfig, "--kubeconfig="+providerKubeconfig, "--control-plane-mode")

	t.Logf("Creating app-cluster workspace to provide kubeconfig secret")
	_, appClusterKubeconfig := framework.NewWorkspace(t, framework.ClientConfig(t), framework.WithGenerateName("test-control-plane-app-cluster"))
	kubeconfigData, err := os.ReadFile(appClusterKubeconfig)
	require.NoError(t, err)

	providerKubeClient := framework.KubeClient(t, providerConfig)
	_, err = providerKubeClient.CoreV1().Secrets("default").Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "app-cluster-kubeconfig", Namespace: "default"},
		Data:       map[string][]byte{"kubeconfig": kubeconfigData},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	bindClient, err := bindclient.NewForConfig(providerConfig)
	require.NoError(t, err)

	bindingName := "control-plane-valid"
	createAppClusterBindingEventually(t, ctx, bindClient, "default", &bindv1alpha1.AppClusterBinding{
		ObjectMeta: metav1.ObjectMeta{Name: bindingName, Namespace: "default"},
		Spec: bindv1alpha1.AppClusterBindingSpec{
			KubeconfigSecretRef: bindv1alpha1.ClusterSecretKeyRef{
				LocalSecretKeyRef: bindv1alpha1.LocalSecretKeyRef{Name: "app-cluster-kubeconfig", Key: "kubeconfig"},
				Namespace:         "default",
			},
			Konnector: &bindv1alpha1.KonnectorSpec{Deploy: true},
		},
	})

	expectedDeploymentName := "konnector-" + bindingName
	expectedRootNamespace := "klutch-bind-default-" + bindingName

	var deployment *appsv1.Deployment
	require.Eventually(t, func() bool {
		deployment, err = providerKubeClient.AppsV1().Deployments("default").Get(ctx, expectedDeploymentName, metav1.GetOptions{})
		return err == nil
	}, wait.ForeverTestTimeout, 100*time.Millisecond, "waiting for control-plane konnector deployment to be created")

	require.NotEmpty(t, deployment.Spec.Template.Spec.Containers)
	args := deployment.Spec.Template.Spec.Containers[0].Args
	require.Contains(t, args, "--control-plane-mode")
	require.Contains(t, args, "--app-cluster-kubeconfig-secret-name=app-cluster-kubeconfig")
	require.Contains(t, args, "--app-cluster-kubeconfig-secret-namespace=default")
	require.Contains(t, args, "--app-cluster-kubeconfig-secret-key=kubeconfig")
	require.Contains(t, args, "--binding-root-namespace="+expectedRootNamespace)

	require.Eventually(t, func() bool {
		_, err := providerKubeClient.CoreV1().Namespaces().Get(ctx, expectedRootNamespace, metav1.GetOptions{})
		return err == nil
	}, wait.ForeverTestTimeout, 100*time.Millisecond, "waiting for binding root namespace to be created")

	require.Eventually(t, func() bool {
		binding, err := bindClient.KlutchBindV1alpha1().AppClusterBindings("default").Get(ctx, bindingName, metav1.GetOptions{})
		if err != nil {
			return false
		}
		status, ok := conditionStatus(binding.Status.Conditions, bindv1alpha1.AppClusterBindingConditionSecretValid)
		return ok && status == corev1.ConditionTrue
	}, wait.ForeverTestTimeout, 100*time.Millisecond, "waiting for SecretValid condition to become true")
}

func TestControlPlaneModeInvalidKubeconfigCondition(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	t.Logf("Creating provider workspace")
	providerConfig, providerKubeconfig := framework.NewWorkspace(t, framework.ClientConfig(t), framework.WithGenerateName("test-control-plane-invalid-provider"))

	t.Logf("Starting backend in control plane mode")
	framework.StartBackendWithoutDefaultArgs(t, providerConfig, "--kubeconfig="+providerKubeconfig, "--control-plane-mode")

	providerKubeClient := framework.KubeClient(t, providerConfig)
	_, err := providerKubeClient.CoreV1().Secrets("default").Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "invalid-kubeconfig", Namespace: "default"},
		Data:       map[string][]byte{"kubeconfig": []byte("not-a-valid-kubeconfig")},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	bindClient, err := bindclient.NewForConfig(providerConfig)
	require.NoError(t, err)

	bindingName := "control-plane-invalid"
	createAppClusterBindingEventually(t, ctx, bindClient, "default", &bindv1alpha1.AppClusterBinding{
		ObjectMeta: metav1.ObjectMeta{Name: bindingName, Namespace: "default"},
		Spec: bindv1alpha1.AppClusterBindingSpec{
			KubeconfigSecretRef: bindv1alpha1.ClusterSecretKeyRef{
				LocalSecretKeyRef: bindv1alpha1.LocalSecretKeyRef{Name: "invalid-kubeconfig", Key: "kubeconfig"},
				Namespace:         "default",
			},
			Konnector: &bindv1alpha1.KonnectorSpec{Deploy: true},
		},
	})

	require.Eventually(t, func() bool {
		binding, err := bindClient.KlutchBindV1alpha1().AppClusterBindings("default").Get(ctx, bindingName, metav1.GetOptions{})
		if err != nil {
			return false
		}
		status, ok := conditionStatus(binding.Status.Conditions, bindv1alpha1.AppClusterBindingConditionSecretValid)
		return ok && status == corev1.ConditionFalse
	}, wait.ForeverTestTimeout, 100*time.Millisecond, "waiting for SecretValid condition to become false")

	err = wait.PollUntilContextTimeout(ctx, 100*time.Millisecond, 10*time.Second, true, func(context.Context) (bool, error) {
		_, depErr := providerKubeClient.AppsV1().Deployments("default").Get(ctx, "konnector-"+bindingName, metav1.GetOptions{})
		if apierrors.IsNotFound(depErr) {
			return true, nil
		}
		if depErr != nil {
			return false, depErr
		}
		return false, nil
	})
	require.NoError(t, err, "deployment should not be created while kubeconfig secret is invalid")
}

func createAppClusterBindingEventually(t *testing.T, ctx context.Context, client bindclient.Interface, namespace string, binding *bindv1alpha1.AppClusterBinding) {
	t.Helper()

	require.Eventually(t, func() bool {
		_, err := client.KlutchBindV1alpha1().AppClusterBindings(namespace).Create(ctx, binding, metav1.CreateOptions{})
		if err == nil || apierrors.IsAlreadyExists(err) {
			return true
		}
		if apierrors.IsNotFound(err) {
			return false
		}
		require.NoError(t, err)
		return false
	}, wait.ForeverTestTimeout, 100*time.Millisecond, "waiting for AppClusterBinding CRD to be available")
}

func conditionStatus(conditions conditionsapi.Conditions, conditionType conditionsapi.ConditionType) (corev1.ConditionStatus, bool) {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return conditions[i].Status, true
		}
	}
	return corev1.ConditionUnknown, false
}
