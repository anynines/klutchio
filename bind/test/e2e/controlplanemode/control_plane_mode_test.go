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

package controlplanemode

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"

	bindv1alpha1 "github.com/anynines/klutchio/bind/pkg/apis/bind/v1alpha1"
	conditionsapi "github.com/anynines/klutchio/bind/pkg/apis/third_party/conditions/apis/conditions/v1alpha1"
	bindclient "github.com/anynines/klutchio/bind/pkg/client/clientset/versioned"
	providerfixtures "github.com/anynines/klutchio/bind/test/e2e/bind/fixtures/provider"
	"github.com/anynines/klutchio/bind/test/e2e/framework"
)

func TestControlPlaneModeInstanceSync(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	t.Logf("Creating provider workspace")
	providerConfig, providerKubeconfig := framework.NewWorkspace(t, framework.ClientConfig(t), framework.WithGenerateName("test-control-plane-provider"))

	t.Logf("Starting backend in control plane mode")
	framework.StartBackendWithoutDefaultArgs(t, providerConfig, "--kubeconfig="+providerKubeconfig, "--control-plane-mode")

	t.Logf("Creating CRDs on provider side")
	providerfixtures.Bootstrap(t, framework.DiscoveryClient(t, providerConfig), framework.DynamicClient(t, providerConfig), nil)

	t.Logf("Creating consumer workspace")
	consumerConfig, _ := framework.NewWorkspace(t, framework.ClientConfig(t), framework.WithGenerateName("test-control-plane-consumer"))

	t.Logf("Storing consumer kubeconfig as secret on provider")
	providerKubeClient := framework.KubeClient(t, providerConfig)
	consumerKubeconfigData, err := clientcmd.Write(framework.RestToKubeconfig(consumerConfig, "default"))
	require.NoError(t, err)
	_, err = providerKubeClient.CoreV1().Secrets("default").Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "app-cluster-kubeconfig", Namespace: "default"},
		Data:       map[string][]byte{"kubeconfig": consumerKubeconfigData},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	t.Logf("Creating AppClusterBinding on provider")
	bindClient, err := bindclient.NewForConfig(providerConfig)
	require.NoError(t, err)

	bindingName := "control-plane-test"
	createAppClusterBindingEventually(t, ctx, bindClient, "default", &bindv1alpha1.AppClusterBinding{
		ObjectMeta: metav1.ObjectMeta{Name: bindingName, Namespace: "default"},
		Spec: bindv1alpha1.AppClusterBindingSpec{
			KubeconfigSecretRef: bindv1alpha1.ClusterSecretKeyRef{
				LocalSecretKeyRef: bindv1alpha1.LocalSecretKeyRef{Name: "app-cluster-kubeconfig", Key: "kubeconfig"},
				Namespace:         "default",
			},
			APIExports: []bindv1alpha1.GroupResource{
				{Group: "mangodb.com", Resource: "mangodbs"},
			},
		},
	})

	expectedRootNamespace := "klutch-bind-default-" + bindingName

	t.Logf("Waiting for binding root namespace to be created by backend")
	require.Eventually(t, func() bool {
		_, err := providerKubeClient.CoreV1().Namespaces().Get(ctx, expectedRootNamespace, metav1.GetOptions{})
		return err == nil
	}, wait.ForeverTestTimeout, 100*time.Millisecond, "waiting for binding root namespace")

	t.Logf("Waiting for SecretValid condition to be true")
	require.Eventually(t, func() bool {
		binding, err := bindClient.KlutchBindV1alpha1().AppClusterBindings("default").Get(ctx, bindingName, metav1.GetOptions{})
		if err != nil {
			return false
		}
		status, ok := conditionStatus(binding.Status.Conditions, bindv1alpha1.AppClusterBindingConditionSecretValid)
		return ok && status == corev1.ConditionTrue
	}, wait.ForeverTestTimeout, 100*time.Millisecond, "waiting for SecretValid condition")

	t.Logf("Waiting for APIServiceBinding to be created in binding root namespace")
	require.Eventually(t, func() bool {
		_, err := bindClient.KlutchBindV1alpha1().APIServiceBindings(expectedRootNamespace).Get(ctx, "mangodbs.mangodb.com", metav1.GetOptions{})
		return err == nil
	}, wait.ForeverTestTimeout, 100*time.Millisecond, "waiting for APIServiceBinding")

	t.Logf("Starting konnector in control plane mode")
	framework.StartKonnector(t, providerConfig,
		"--kubeconfig="+providerKubeconfig,
		"--control-plane-mode",
		"--app-cluster-kubeconfig-secret-name=app-cluster-kubeconfig",
		"--app-cluster-kubeconfig-secret-namespace="+expectedRootNamespace,
		"--app-cluster-kubeconfig-secret-key=kubeconfig",
		"--binding-root-namespace="+expectedRootNamespace,
	)

	serviceGVR := schema.GroupVersionResource{Group: "mangodb.com", Version: "v1alpha1", Resource: "mangodbs"}
	consumerClient := framework.DynamicClient(t, consumerConfig).Resource(serviceGVR)
	providerClient := framework.DynamicClient(t, providerConfig).Resource(serviceGVR)

	t.Logf("Waiting for MangoDB CRD to be created on consumer side")
	crdClient := framework.ApiextensionsClient(t, consumerConfig).ApiextensionsV1().CustomResourceDefinitions()
	require.Eventually(t, func() bool {
		_, err := crdClient.Get(ctx, "mangodbs.mangodb.com", metav1.GetOptions{})
		return err == nil
	}, wait.ForeverTestTimeout, 100*time.Millisecond, "waiting for MangoDB CRD on consumer side")

	mangodbInstance := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "mangodb.com/v1alpha1",
			"kind":       "MangoDB",
			"metadata":   map[string]interface{}{"name": "test"},
			"spec":       map[string]interface{}{"tokenSecret": "credentials"},
		},
	}

	t.Logf("Creating MangoDB instance on consumer side")
	require.Eventually(t, func() bool {
		_, err := consumerClient.Namespace("default").Create(ctx, mangodbInstance, metav1.CreateOptions{})
		return err == nil
	}, wait.ForeverTestTimeout, 100*time.Millisecond, "waiting for MangoDB instance creation on consumer side")

	t.Logf("Waiting for MangoDB instance to be synced to provider side")
	require.Eventually(t, func() bool {
		instances, err := providerClient.List(ctx, metav1.ListOptions{})
		return err == nil && len(instances.Items) == 1
	}, wait.ForeverTestTimeout, 100*time.Millisecond, "waiting for MangoDB instance on provider side")
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
}

func createAppClusterBindingEventually(t *testing.T, ctx context.Context, client bindclient.Interface, namespace string, binding *bindv1alpha1.AppClusterBinding) {
	t.Helper()

	require.Eventually(t, func() bool {
		_, err := client.KlutchBindV1alpha1().AppClusterBindings(namespace).Create(ctx, binding, metav1.CreateOptions{})
		if err == nil {
			return true
		}
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
