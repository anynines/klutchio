/*
Copyright 2022 The Kube Bind Authors.

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
	"time"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apiextensionsinformers "k8s.io/apiextensions-apiserver/pkg/client/informers/externalversions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	kubernetesclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	bindclient "github.com/anynines/klutchio/bind/pkg/client/clientset/versioned"
	bindinformers "github.com/anynines/klutchio/bind/pkg/client/informers/externalversions"
	"github.com/anynines/klutchio/bind/pkg/konnector/options"
)

type Config struct {
	ControlPlaneMode     bool
	BindingRootNamespace string

	// App Cluster: where resources are synced to (the "consumer")
	AppClusterConfig                 *rest.Config
	AppClusterKubeClient             *kubernetesclient.Clientset
	AppClusterApiextensionsClient    *apiextensionsclient.Clientset
	AppClusterKubeInformers          kubeinformers.SharedInformerFactory
	AppClusterApiextensionsInformers apiextensionsinformers.SharedInformerFactory

	// Binding Cluster: where APIServiceBindings and provider kubeconfig secrets are stored
	// - In default mode: same as app cluster
	// - In control plane mode: the control plane cluster
	BindingClusterConfig              *rest.Config
	BindingClusterBindClient          *bindclient.Clientset
	BindingClusterKubeClient          *kubernetesclient.Clientset
	BindingClusterApiextensionsClient *apiextensionsclient.Clientset
	BindingClusterBindInformers       bindinformers.SharedInformerFactory
	BindingClusterKubeInformers       kubeinformers.SharedInformerFactory

	// Control Plane: only used in control plane mode
	// In default mode, these are nil
	ControlPlaneConfig *rest.Config
}

func loadKubeconfig(path string) (*rest.Config, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	rules.ExplicitPath = path
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, nil).ClientConfig()
	if err != nil {
		return nil, err
	}
	cfg = rest.CopyConfig(cfg)
	cfg = rest.AddUserAgent(cfg, "konnector")
	return cfg, nil
}

func (c *Config) initAppCluster(cfg *rest.Config) error {
	c.AppClusterConfig = cfg
	var err error
	if c.AppClusterKubeClient, err = kubernetesclient.NewForConfig(cfg); err != nil {
		return err
	}
	if c.AppClusterApiextensionsClient, err = apiextensionsclient.NewForConfig(cfg); err != nil {
		return err
	}
	c.AppClusterKubeInformers = kubeinformers.NewSharedInformerFactory(c.AppClusterKubeClient, time.Minute*30)
	c.AppClusterApiextensionsInformers = apiextensionsinformers.NewSharedInformerFactory(c.AppClusterApiextensionsClient, time.Minute*30)
	return nil
}

func (c *Config) initBindingCluster(cfg *rest.Config, namespace string) error {
	c.BindingClusterConfig = cfg
	var err error
	if c.BindingClusterBindClient, err = bindclient.NewForConfig(cfg); err != nil {
		return err
	}
	if c.BindingClusterKubeClient, err = kubernetesclient.NewForConfig(cfg); err != nil {
		return err
	}
	if c.BindingClusterApiextensionsClient, err = apiextensionsclient.NewForConfig(cfg); err != nil {
		return err
	}
	// In control plane mode, scope informers to the binding root namespace
	if namespace != "" {
		c.BindingClusterBindInformers = bindinformers.NewSharedInformerFactoryWithOptions(
			c.BindingClusterBindClient,
			time.Minute*30,
			bindinformers.WithNamespace(namespace),
		)
		c.BindingClusterKubeInformers = kubeinformers.NewSharedInformerFactoryWithOptions(
			c.BindingClusterKubeClient,
			time.Minute*30,
			kubeinformers.WithNamespace(namespace),
		)
	} else {
		c.BindingClusterBindInformers = bindinformers.NewSharedInformerFactory(c.BindingClusterBindClient, time.Minute*30)
		c.BindingClusterKubeInformers = kubeinformers.NewSharedInformerFactory(c.BindingClusterKubeClient, time.Minute*30)
	}
	return nil
}

func NewConfig(options *options.CompletedOptions) (*Config, error) {
	config := &Config{
		ControlPlaneMode:     options.ControlPlaneMode,
		BindingRootNamespace: options.BindingRootNamespace,
	}

	if options.ControlPlaneMode {
		// Load control plane config
		cpConfig, err := loadKubeconfig(options.KubeConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load control plane kubeconfig: %w", err)
		}
		config.ControlPlaneConfig = cpConfig

		// Fetch app cluster kubeconfig from secret in control plane
		cpKubeClient, err := kubernetesclient.NewForConfig(cpConfig)
		if err != nil {
			return nil, err
		}
		secret, err := cpKubeClient.CoreV1().Secrets(options.AppClusterKubeconfigSecretNamespace).Get(
			context.Background(),
			options.AppClusterKubeconfigSecretName,
			metav1.GetOptions{},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch app cluster kubeconfig secret %s/%s: %w",
				options.AppClusterKubeconfigSecretNamespace,
				options.AppClusterKubeconfigSecretName,
				err)
		}
		kubeconfigData, ok := secret.Data[options.AppClusterKubeconfigSecretKey]
		if !ok {
			return nil, fmt.Errorf("secret %s/%s does not contain key %s",
				options.AppClusterKubeconfigSecretNamespace,
				options.AppClusterKubeconfigSecretName,
				options.AppClusterKubeconfigSecretKey)
		}
		appConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse app cluster kubeconfig: %w", err)
		}
		appConfig = rest.CopyConfig(appConfig)
		appConfig = rest.AddUserAgent(appConfig, "konnector")

		if err := config.initAppCluster(appConfig); err != nil {
			return nil, err
		}
		// Binding cluster = control plane with namespace scoping
		if err := config.initBindingCluster(cpConfig, options.BindingRootNamespace); err != nil {
			return nil, err
		}
	} else {
		// Default mode: single cluster serves as both app and binding cluster
		appConfig, err := loadKubeconfig(options.KubeConfigPath)
		if err != nil {
			return nil, err
		}
		if err := config.initAppCluster(appConfig); err != nil {
			return nil, err
		}
		// No namespace scoping in default mode
		if err := config.initBindingCluster(appConfig, ""); err != nil {
			return nil, err
		}
	}

	return config, nil
}
