/*
Copyright 2023 The Kube Bind Authors.

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
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/yaml"

	bindv1alpha1 "github.com/anynines/klutch/bind/pkg/apis/bind/v1alpha1"
	providerfixtures "github.com/anynines/klutch/bind/test/e2e/bind/fixtures/provider"
	"github.com/anynines/klutch/bind/test/e2e/framework"
)

func TestKonnectorAdopt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	t.Logf("Creating provider workspace")
	providerConfig, providerKubeconfig := framework.NewWorkspace(t, framework.ClientConfig(t), framework.WithGenerateName("test-konnector-adopt"))

	t.Logf("Starting backend with random port")
	addr, _ := framework.StartBackend(t, providerConfig, "--kubeconfig="+providerKubeconfig, "--listen-port=0", "--consumer-scope="+string(bindv1alpha1.NamespacedScope))

	t.Logf("Creating MangoDB CRD on provider side")
	providerfixtures.Bootstrap(t, framework.DiscoveryClient(t, providerConfig), framework.DynamicClient(t, providerConfig), nil)

	t.Logf("Creating consumer workspace and starting konnector")
	consumerConfig, consumerKubeconfig := framework.NewWorkspace(t, framework.ClientConfig(t), framework.WithGenerateName("test-konnector-adopt"))
	framework.StartKonnector(t, consumerConfig, "--kubeconfig="+consumerKubeconfig)

	consumerClient := framework.DynamicClient(t, consumerConfig).Resource(
		schema.GroupVersionResource{Group: "mangodb.com", Version: "v1alpha1", Resource: "mangodbs"},
	).Namespace("default")
	providerClient := framework.DynamicClient(t, providerConfig).Resource(
		schema.GroupVersionResource{Group: "mangodb.com", Version: "v1alpha1", Resource: "mangodbs"},
	)

	upstreamNS := "unknown"
	downstreamNS := "unknown"

	for _, tc := range []struct {
		name string
		step func(t *testing.T)
	}{
		{
			name: "MangoDB is bound dry run",
			step: func(t *testing.T) {
				iostreams, _, bufOut, _ := genericclioptions.NewTestIOStreams()
				authURLDryRunCh := make(chan string, 1)
				go simulateBrowser(t, authURLDryRunCh, "mangodbs")
				framework.Bind(t, iostreams, authURLDryRunCh, nil, fmt.Sprintf("http://%s/export", addr.String()), "--kubeconfig", consumerKubeconfig, "--skip-konnector", "--dry-run")
				_, err := yaml.YAMLToJSON(bufOut.Bytes())
				require.NoError(t, err)
			},
		},
		{
			name: "MangoDB is bound",
			step: func(t *testing.T) {
				in := bytes.NewBufferString("y\n")
				iostreams, _, _, _ := genericclioptions.NewTestIOStreams()
				authURLCh := make(chan string, 1)
				go simulateBrowser(t, authURLCh, "mangodbs")
				invocations := make(chan framework.SubCommandInvocation, 1)
				framework.Bind(t, iostreams, authURLCh, invocations, fmt.Sprintf("http://%s/export", addr.String()), "--kubeconfig", consumerKubeconfig, "--skip-konnector")
				inv := <-invocations
				requireEqualSlicePattern(t, []string{"apiservice", "--remote-kubeconfig-namespace", "*", "--remote-kubeconfig-name", "*", "-f", "*", "--kubeconfig=" + consumerKubeconfig, "--skip-konnector=true", "--no-banner"}, inv.Args)
				framework.BindAPIService(t, in, "", inv.Args...)

				t.Logf("Waiting for MangoDB CRD to be created on consumer side")
				crdClient := framework.ApiextensionsClient(t, consumerConfig).ApiextensionsV1().CustomResourceDefinitions()
				require.Eventually(t, func() bool {
					_, err := crdClient.Get(ctx, "mangodbs.mangodb.com", metav1.GetOptions{})
					return err == nil
				}, wait.ForeverTestTimeout, time.Millisecond*100, "waiting for MangoDB CRD to be created on consumer side")
			},
		},
		{
			name: "instances are synced",
			step: func(t *testing.T) {
				t.Logf("Trying to create MangoDB on consumer side")

				require.Eventually(t, func() bool {
					_, err := consumerClient.Create(ctx, toUnstructured(t, `
apiVersion: mangodb.com/v1alpha1
kind: MangoDB
metadata:
  name: test
spec:
  tokenSecret: credentials
`), metav1.CreateOptions{})
					return err == nil
				}, wait.ForeverTestTimeout, time.Millisecond*100, "waiting for MangoDB CRD to be created on consumer side")

				t.Logf("Waiting for the MangoDB instance to be created on consumer side")
				var consumerMangos *unstructured.UnstructuredList
				require.Eventually(t, func() bool {
					var err error
					consumerMangos, err = consumerClient.List(ctx, metav1.ListOptions{})
					return err == nil && len(consumerMangos.Items) == 1
				}, wait.ForeverTestTimeout, time.Millisecond*100, "waiting for the MangoDB instance to be created on consumer side")

				// this is used everywhere further down
				downstreamNS = consumerMangos.Items[0].GetNamespace()

				t.Logf("Waiting for the MangoDB instance to be created on provider side")
				var mangos *unstructured.UnstructuredList
				require.Eventually(t, func() bool {
					var err error
					mangos, err = providerClient.List(ctx, metav1.ListOptions{})
					return err == nil && len(mangos.Items) == 1
				}, wait.ForeverTestTimeout, time.Millisecond*100, "waiting for the MangoDB instance to be created on provider side")

				// this is used everywhere further down
				upstreamNS = mangos.Items[0].GetNamespace()
			},
		},
		{
			name: "Different MangoDB is created at provider",
			step: func(t *testing.T) {
				t.Logf("Trying to create MangoDB on provider side")

				require.Eventually(t, func() bool {
					_, err := providerClient.Namespace(upstreamNS).Create(ctx, toUnstructured(t, `
apiVersion: mangodb.com/v1alpha1
kind: MangoDB
metadata:
  name: test2
spec:
  tokenSecret: credentials
`), metav1.CreateOptions{})
					return err == nil
				}, wait.ForeverTestTimeout, time.Millisecond*100, "waiting for MangoDB CRD to be created on provider side")

				t.Logf("Waiting for the MangoDB instance to be created on provider side")
				var providerMangos *unstructured.UnstructuredList
				require.Eventually(t, func() bool {
					var err error
					providerMangos, err = providerClient.List(ctx, metav1.ListOptions{})
					return err == nil && len(providerMangos.Items) == 2
				}, wait.ForeverTestTimeout, time.Millisecond*100, "waiting for the MangoDB instance to be created on provider side")

				t.Logf("Waiting for the MangoDB instance to be created on consumer side")
				var mangos *unstructured.UnstructuredList
				require.Eventually(t, func() bool {
					var err error
					mangos, err = consumerClient.List(ctx, metav1.ListOptions{})
					if err != nil {
						return false
					}

					t.Logf("got items: n=%d i=%+v", len(mangos.Items), mangos)
					var item *unstructured.Unstructured
					for _, i := range mangos.Items {
						if i.GetName() == "test2" {
							item = i.DeepCopy()
						}
					}
					if item == nil {
						return false
					}

					annotations := item.GetAnnotations()
					if annotations == nil {
						return false
					}
					if v, ok := annotations["klutch.anynines.com/bound"]; !ok || v != "true" {
						return false
					}
					return true
				}, wait.ForeverTestTimeout, time.Millisecond*100, "waiting for the MangoDB instance to be created on consumer side")
			},
		},
		{
			name: "deletion of objects on consumer cluster",
			step: func(t *testing.T) {
				require.NoError(t, consumerClient.Delete(ctx, "test", metav1.DeleteOptions{}))

				require.NoError(t, consumerClient.Delete(ctx, "test2", metav1.DeleteOptions{}))
				require.Eventually(t, func() bool {
					l, err := providerClient.List(ctx, metav1.ListOptions{})
					if err != nil {
						return false
					}
					return len(l.Items) == 0
				}, wait.ForeverTestTimeout, time.Millisecond*100, "waiting for provider objects to be deleted")
			},
		},
	} {
		tc.step(t)
	}

	t.Log(upstreamNS, downstreamNS)
}
