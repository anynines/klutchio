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

package servicebinding

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	apiextensionsfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	apiextensionsinformers "k8s.io/apiextensions-apiserver/pkg/client/informers/externalversions"
	apiextensionslisters "k8s.io/apiextensions-apiserver/pkg/client/listers/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	bindv1alpha1 "github.com/anynines/klutchio/bind/pkg/apis/bind/v1alpha1"
	bindfake "github.com/anynines/klutchio/bind/pkg/client/clientset/versioned/fake"
	bindinformers "github.com/anynines/klutchio/bind/pkg/client/informers/externalversions"
	bindlisters "github.com/anynines/klutchio/bind/pkg/client/listers/bind/v1alpha1"
	"github.com/anynines/klutchio/bind/pkg/konnector/controllers/dynamic"
)

func newTrackingServer(t *testing.T) (*httptest.Server, *atomic.Int32, *atomic.Value) {
	t.Helper()

	patchCount := &atomic.Int32{}
	lastPath := &atomic.Value{}
	lastPath.Store("")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			patchCount.Add(1)
			lastPath.Store(r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"apiVersion":"klutch.anynines.com/v1alpha1","kind":"APIServiceBinding","metadata":{"name":"test","resourceVersion":"2"}}`)
	}))
	t.Cleanup(server.Close)

	return server, patchCount, lastPath
}

func TestNewController_CommitTargetsBindingCluster(t *testing.T) {
	bindingServer, bindingPatches, bindingPath := newTrackingServer(t)
	consumerServer, consumerPatches, _ := newTrackingServer(t)
	providerServer, _, _ := newTrackingServer(t)

	bindingClient := bindfake.NewSimpleClientset()
	providerClient := bindfake.NewSimpleClientset()
	apiextensionsClient := apiextensionsfake.NewSimpleClientset()

	bindingInformers := bindinformers.NewSharedInformerFactory(bindingClient, 0)
	providerInformers := bindinformers.NewSharedInformerFactoryWithOptions(providerClient, 0, bindinformers.WithNamespace("provider"))
	apiextensionsInformers := apiextensionsinformers.NewSharedInformerFactory(apiextensionsClient, 0)

	c, err := NewController(
		"default/app-cluster-kubeconfig",
		"provider",
		func(*bindv1alpha1.APIServiceBinding) bool { return true },
		&rest.Config{Host: bindingServer.URL},
		&rest.Config{Host: consumerServer.URL},
		&rest.Config{Host: providerServer.URL},
		dynamic.NewDynamicInformer[bindlisters.APIServiceBindingLister](bindingInformers.KlutchBind().V1alpha1().APIServiceBindings()),
		providerInformers.KlutchBind().V1alpha1().APIServiceExports(),
		dynamic.NewDynamicInformer[apiextensionslisters.CustomResourceDefinitionLister](apiextensionsInformers.Apiextensions().V1().CustomResourceDefinitions()),
	)
	require.NoError(t, err)

	old := &Resource{
		ObjectMeta: metav1.ObjectMeta{Name: "test", ResourceVersion: "1"},
		Spec:       &bindv1alpha1.APIServiceBindingSpec{},
		Status:     &bindv1alpha1.APIServiceBindingStatus{},
	}
	updated := &Resource{
		ObjectMeta: metav1.ObjectMeta{Name: "test", ResourceVersion: "1"},
		Spec:       &bindv1alpha1.APIServiceBindingSpec{},
		Status: &bindv1alpha1.APIServiceBindingStatus{
			ProviderPrettyName: "provider",
		},
	}

	err = c.commit(context.Background(), old, updated)
	require.NoError(t, err)

	require.Equal(t, int32(1), bindingPatches.Load())
	require.Equal(t, int32(0), consumerPatches.Load())
	require.Equal(t, "/apis/klutch.anynines.com/v1alpha1/apiservicebindings/test/status", bindingPath.Load())
}
