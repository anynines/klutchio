/*
Copyright 2024 The Kube Bind Authors.

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
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apiextensionsfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	apiextensionsinformers "k8s.io/apiextensions-apiserver/pkg/client/informers/externalversions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	bindv1alpha1 "github.com/anynines/klutchio/bind/pkg/apis/bind/v1alpha1"
	conditionsapi "github.com/anynines/klutchio/bind/pkg/apis/third_party/conditions/apis/conditions/v1alpha1"
	bindfake "github.com/anynines/klutchio/bind/pkg/client/clientset/versioned/fake"
	bindinformers "github.com/anynines/klutchio/bind/pkg/client/informers/externalversions"
)

// newTrackingServer creates an httptest server that counts PATCH requests
// and returns a valid APIServiceBinding JSON response for all requests.
func newTrackingServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	patchCount := &atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			patchCount.Add(1)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"apiVersion":"klutch.anynines.com/v1alpha1","kind":"APIServiceBinding","metadata":{"name":"test","resourceVersion":"2"}}`)
	}))
	t.Cleanup(server.Close)
	return server, patchCount
}

// createTestInformers creates fake informers suitable for passing to New().
func createTestInformers(t *testing.T) (
	bindinformers.SharedInformerFactory,
	kubeinformers.SharedInformerFactory,
	kubeinformers.SharedInformerFactory,
	apiextensionsinformers.SharedInformerFactory,
) {
	t.Helper()
	bindClient := bindfake.NewSimpleClientset()
	kubeClient := kubefake.NewSimpleClientset()
	apiextClient := apiextensionsfake.NewSimpleClientset()

	bindInformers := bindinformers.NewSharedInformerFactory(bindClient, 0)
	bindingKubeInformers := kubeinformers.NewSharedInformerFactory(kubeClient, 0)
	appKubeInformers := kubeinformers.NewSharedInformerFactoryWithOptions(kubeClient, 0)
	apiextInformers := apiextensionsinformers.NewSharedInformerFactory(apiextClient, 0)

	return bindInformers, bindingKubeInformers, appKubeInformers, apiextInformers
}

// newController creates a Controller via New() with the given configs and fake informers.
func newController(t *testing.T, consumerConfig, bindingConfig *rest.Config) *Controller {
	t.Helper()
	bindInformers, bindingKubeInformers, appKubeInformers, apiextInformers := createTestInformers(t)

	c, err := New(
		consumerConfig,
		bindingConfig,
		"",
		false,
		bindInformers.KlutchBind().V1alpha1().APIServiceBindings(),
		bindingKubeInformers.Core().V1().Secrets(),
		appKubeInformers.Core().V1().Namespaces(),
		apiextInformers.Apiextensions().V1().CustomResourceDefinitions(),
	)
	require.NoError(t, err)
	return c
}

func TestNew_ConsumerConfig_PointsToAppCluster(t *testing.T) {
	appServer, _ := newTrackingServer(t)
	bindingServer, _ := newTrackingServer(t)

	appConfig := &rest.Config{Host: appServer.URL}
	bindingConfig := &rest.Config{Host: bindingServer.URL}

	c := newController(t, appConfig, bindingConfig)

	// consumerConfig should point to the app cluster, not the binding cluster.
	// This config is captured by newClusterController for resource syncing.
	require.True(t, strings.HasPrefix(c.consumerConfig.Host, appServer.URL),
		"consumerConfig.Host should point to app cluster, got %s", c.consumerConfig.Host)
}

func TestNew_DefaultMode_SameCluster(t *testing.T) {
	// In default mode, both configs point to the same cluster.
	server, _ := newTrackingServer(t)
	config := &rest.Config{Host: server.URL}

	c := newController(t, config, config)

	require.True(t, strings.HasPrefix(c.consumerConfig.Host, server.URL),
		"consumerConfig should point to the single cluster")
}

func TestNew_ControlPlaneMode_DifferentClusters(t *testing.T) {
	appServer, _ := newTrackingServer(t)
	cpServer, _ := newTrackingServer(t)

	appConfig := &rest.Config{Host: appServer.URL}
	cpConfig := &rest.Config{Host: cpServer.URL}

	c := newController(t, appConfig, cpConfig)

	// consumerConfig must point to app cluster for resource syncing
	require.True(t, strings.HasPrefix(c.consumerConfig.Host, appServer.URL),
		"consumerConfig should be app cluster")
	// consumerConfig must NOT point to control plane
	require.False(t, strings.HasPrefix(c.consumerConfig.Host, cpServer.URL),
		"consumerConfig should not be control plane")
}

func TestNew_CommitTargetsBindingCluster(t *testing.T) {
	appServer, appPatches := newTrackingServer(t)
	bindingServer, bindingPatches := newTrackingServer(t)

	appConfig := &rest.Config{Host: appServer.URL}
	bindingConfig := &rest.Config{Host: bindingServer.URL}

	c := newController(t, appConfig, bindingConfig)

	// Trigger a commit with a status change.
	// The committer should PATCH the APIServiceBinding on the binding cluster.
	old := &Resource{
		ObjectMeta: metav1.ObjectMeta{Name: "test", ResourceVersion: "1"},
		Spec:       &bindv1alpha1.APIServiceBindingSpec{},
		Status:     &bindv1alpha1.APIServiceBindingStatus{},
	}
	updated := &Resource{
		ObjectMeta: metav1.ObjectMeta{Name: "test", ResourceVersion: "1"},
		Spec:       &bindv1alpha1.APIServiceBindingSpec{},
		Status: &bindv1alpha1.APIServiceBindingStatus{
			ProviderPrettyName: "changed",
		},
	}

	err := c.commit(context.Background(), old, updated)
	require.NoError(t, err)

	require.Equal(t, int32(1), bindingPatches.Load(),
		"PATCH should go to the binding cluster")
	require.Equal(t, int32(0), appPatches.Load(),
		"PATCH should NOT go to the app cluster")
}

func TestNew_CommitTargetsBindingCluster_StatusConditionChange(t *testing.T) {
	appServer, appPatches := newTrackingServer(t)
	bindingServer, bindingPatches := newTrackingServer(t)

	appConfig := &rest.Config{Host: appServer.URL}
	bindingConfig := &rest.Config{Host: bindingServer.URL}

	c := newController(t, appConfig, bindingConfig)

	old := &Resource{
		ObjectMeta: metav1.ObjectMeta{Name: "test-binding", ResourceVersion: "1"},
		Spec:       &bindv1alpha1.APIServiceBindingSpec{},
		Status:     &bindv1alpha1.APIServiceBindingStatus{},
	}
	updated := &Resource{
		ObjectMeta: metav1.ObjectMeta{Name: "test-binding", ResourceVersion: "1"},
		Spec:       &bindv1alpha1.APIServiceBindingSpec{},
		Status: &bindv1alpha1.APIServiceBindingStatus{
			Conditions: conditionsapi.Conditions{
				{
					Type:               "Connected",
					Status:             "True",
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	err := c.commit(context.Background(), old, updated)
	require.NoError(t, err)

	require.Equal(t, int32(1), bindingPatches.Load(),
		"status PATCH should go to binding cluster")
	require.Equal(t, int32(0), appPatches.Load(),
		"status PATCH should NOT go to app cluster")
}

func TestNew_NoChangeNoCommit(t *testing.T) {
	appServer, appPatches := newTrackingServer(t)
	bindingServer, bindingPatches := newTrackingServer(t)

	appConfig := &rest.Config{Host: appServer.URL}
	bindingConfig := &rest.Config{Host: bindingServer.URL}

	c := newController(t, appConfig, bindingConfig)

	// Same old and new — no commit should happen.
	resource := &Resource{
		ObjectMeta: metav1.ObjectMeta{Name: "test", ResourceVersion: "1"},
		Spec:       &bindv1alpha1.APIServiceBindingSpec{},
		Status:     &bindv1alpha1.APIServiceBindingStatus{},
	}

	err := c.commit(context.Background(), resource, resource)
	require.NoError(t, err)

	require.Equal(t, int32(0), bindingPatches.Load(), "no PATCH expected")
	require.Equal(t, int32(0), appPatches.Load(), "no PATCH expected")
}

func TestNewServer_ConsumerConfig_IsAppCluster(t *testing.T) {
	appServer, _ := newTrackingServer(t)
	cpServer, _ := newTrackingServer(t)

	appConfig := &rest.Config{Host: appServer.URL}
	cpConfig := &rest.Config{Host: cpServer.URL}

	// Build a minimal Config simulating control plane mode
	config := &Config{
		ControlPlaneMode:   true,
		ControlPlaneConfig: cpConfig,
	}
	if err := config.initAppCluster(appConfig); err != nil {
		t.Fatal(err)
	}
	if err := config.initBindingCluster(cpConfig); err != nil {
		t.Fatal(err)
	}

	server, err := NewServer(config)
	require.NoError(t, err)
	require.NotNil(t, server)

	// Verify the controller's consumerConfig points to app cluster
	require.True(t, strings.HasPrefix(server.Controller.consumerConfig.Host, appServer.URL),
		"controller consumerConfig should point to app cluster")
}

func TestNewServer_DefaultMode_SameConfigs(t *testing.T) {
	singleServer, _ := newTrackingServer(t)
	singleConfig := &rest.Config{Host: singleServer.URL}

	config := &Config{
		ControlPlaneMode: false,
	}
	if err := config.initAppCluster(singleConfig); err != nil {
		t.Fatal(err)
	}
	if err := config.initBindingCluster(singleConfig); err != nil {
		t.Fatal(err)
	}

	server, err := NewServer(config)
	require.NoError(t, err)
	require.NotNil(t, server)

	require.True(t, strings.HasPrefix(server.Controller.consumerConfig.Host, singleServer.URL),
		"controller consumerConfig should point to the single cluster")
}

func TestNewServer_ControlPlaneMode_CommitTargetsControlPlane(t *testing.T) {
	appServer, appPatches := newTrackingServer(t)
	cpServer, cpPatches := newTrackingServer(t)

	appConfig := &rest.Config{Host: appServer.URL}
	cpConfig := &rest.Config{Host: cpServer.URL}

	config := &Config{
		ControlPlaneMode:   true,
		ControlPlaneConfig: cpConfig,
	}
	if err := config.initAppCluster(appConfig); err != nil {
		t.Fatal(err)
	}
	if err := config.initBindingCluster(cpConfig); err != nil {
		t.Fatal(err)
	}

	server, err := NewServer(config)
	require.NoError(t, err)

	// Trigger a commit to verify it targets the control plane (binding cluster)
	old := &Resource{
		ObjectMeta: metav1.ObjectMeta{Name: "test", ResourceVersion: "1"},
		Spec:       &bindv1alpha1.APIServiceBindingSpec{},
		Status:     &bindv1alpha1.APIServiceBindingStatus{},
	}
	updated := &Resource{
		ObjectMeta: metav1.ObjectMeta{Name: "test", ResourceVersion: "1"},
		Spec:       &bindv1alpha1.APIServiceBindingSpec{},
		Status: &bindv1alpha1.APIServiceBindingStatus{
			ProviderPrettyName: "my-provider",
		},
	}

	err = server.Controller.commit(context.Background(), old, updated)
	require.NoError(t, err)

	require.Equal(t, int32(1), cpPatches.Load(),
		"PATCH should go to control plane (binding cluster)")
	require.Equal(t, int32(0), appPatches.Load(),
		"PATCH should NOT go to app cluster")
}

func TestNewServer_DefaultMode_CommitTargetsSameCluster(t *testing.T) {
	server, patches := newTrackingServer(t)
	singleConfig := &rest.Config{Host: server.URL}

	config := &Config{
		ControlPlaneMode: false,
	}
	if err := config.initAppCluster(singleConfig); err != nil {
		t.Fatal(err)
	}
	if err := config.initBindingCluster(singleConfig); err != nil {
		t.Fatal(err)
	}

	s, err := NewServer(config)
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
			ProviderPrettyName: "my-provider",
		},
	}

	err = s.Controller.commit(context.Background(), old, updated)
	require.NoError(t, err)

	require.Equal(t, int32(1), patches.Load(),
		"PATCH should go to the single cluster")
}

func TestNewClusterController_CapturesConsumerConfig(t *testing.T) {
	appServer, _ := newTrackingServer(t)
	bindingServer, _ := newTrackingServer(t)

	appConfig := &rest.Config{Host: appServer.URL}
	bindingConfig := &rest.Config{Host: bindingServer.URL}

	c := newController(t, appConfig, bindingConfig)

	// The newClusterController closure captures consumerConfig.
	// We verify this by checking that the closure is set and calling it
	// would use the consumerConfig (app cluster), not bindingConfig.
	//
	// We can't easily call newClusterController without a valid provider
	// kubeconfig, but we can verify the consumerConfig it would capture.
	require.True(t, strings.HasPrefix(c.consumerConfig.Host, appServer.URL),
		"newClusterController should capture consumerConfig pointing to app cluster, got %s", c.consumerConfig.Host)
	require.NotNil(t, c.reconciler.newClusterController,
		"newClusterController closure should be set")
}

func TestConfig_DefaultMode_SameConfigs(t *testing.T) {
	server, _ := newTrackingServer(t)
	cfg := &rest.Config{Host: server.URL}

	config := &Config{}
	require.NoError(t, config.initAppCluster(cfg))
	require.NoError(t, config.initBindingCluster(cfg))

	// In default mode, AppCluster and BindingCluster configs share the same host
	require.Equal(t, config.AppClusterConfig.Host, config.BindingClusterConfig.Host,
		"in default mode, app and binding cluster should be the same")
}

func TestConfig_ControlPlaneMode_DifferentConfigs(t *testing.T) {
	appServer, _ := newTrackingServer(t)
	cpServer, _ := newTrackingServer(t)

	appConfig := &rest.Config{Host: appServer.URL}
	cpConfig := &rest.Config{Host: cpServer.URL}

	config := &Config{
		ControlPlaneMode:   true,
		ControlPlaneConfig: cpConfig,
	}
	require.NoError(t, config.initAppCluster(appConfig))
	require.NoError(t, config.initBindingCluster(cpConfig))

	require.NotEqual(t, config.AppClusterConfig.Host, config.BindingClusterConfig.Host,
		"in control plane mode, app and binding cluster should differ")
	require.Equal(t, config.BindingClusterConfig.Host, config.ControlPlaneConfig.Host,
		"binding cluster should point to control plane in CP mode")
}

func TestConfig_InitAppCluster_PopulatesAllFields(t *testing.T) {
	server, _ := newTrackingServer(t)
	cfg := &rest.Config{Host: server.URL}

	config := &Config{}
	require.NoError(t, config.initAppCluster(cfg))

	require.NotNil(t, config.AppClusterConfig)
	require.NotNil(t, config.AppClusterKubeClient)
	require.NotNil(t, config.AppClusterApiextensionsClient)
	require.NotNil(t, config.AppClusterKubeInformers)
	require.NotNil(t, config.AppClusterApiextensionsInformers)
}

func TestConfig_InitBindingCluster_PopulatesAllFields(t *testing.T) {
	server, _ := newTrackingServer(t)
	cfg := &rest.Config{Host: server.URL}

	config := &Config{}
	require.NoError(t, config.initBindingCluster(cfg))

	require.NotNil(t, config.BindingClusterConfig)
	require.NotNil(t, config.BindingClusterBindClient)
	require.NotNil(t, config.BindingClusterKubeClient)
	require.NotNil(t, config.BindingClusterApiextensionsClient)
	require.NotNil(t, config.BindingClusterBindInformers)
	require.NotNil(t, config.BindingClusterKubeInformers)
}

func TestConfig_InformersUseDifferentClients(t *testing.T) {
	appServer, _ := newTrackingServer(t)
	cpServer, _ := newTrackingServer(t)

	appConfig := &rest.Config{Host: appServer.URL}
	cpConfig := &rest.Config{Host: cpServer.URL}

	config := &Config{
		ControlPlaneMode:   true,
		ControlPlaneConfig: cpConfig,
	}
	require.NoError(t, config.initAppCluster(appConfig))
	require.NoError(t, config.initBindingCluster(cpConfig))

	// Verify that informer factories are distinct objects in CP mode
	// (they watch different clusters)
	require.NotSame(t, config.AppClusterKubeInformers, config.BindingClusterKubeInformers,
		"app and binding kube informers should be different in CP mode")
}

func TestNewServer_InformersStartCorrectly(t *testing.T) {
	appServer, _ := newTrackingServer(t)
	cpServer, _ := newTrackingServer(t)

	appConfig := &rest.Config{Host: appServer.URL}
	cpConfig := &rest.Config{Host: cpServer.URL}

	config := &Config{
		ControlPlaneMode:   true,
		ControlPlaneConfig: cpConfig,
	}
	require.NoError(t, config.initAppCluster(appConfig))
	require.NoError(t, config.initBindingCluster(cpConfig))

	s, err := NewServer(config)
	require.NoError(t, err)
	require.NotNil(t, s)

	// OptionallyStartInformers should not panic
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start informers — this verifies all four informer factories are properly initialized
	s.Config.AppClusterKubeInformers.Start(ctx.Done())
	s.Config.AppClusterApiextensionsInformers.Start(ctx.Done())
	s.Config.BindingClusterBindInformers.Start(ctx.Done())
	s.Config.BindingClusterKubeInformers.Start(ctx.Done())
}
