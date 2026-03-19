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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/anynines/klutchio/bind/deploy/crd"
	bindv1alpha1 "github.com/anynines/klutchio/bind/pkg/apis/bind/v1alpha1"
	healthz "github.com/anynines/klutchio/bind/pkg/konnector/healthz"
)

type Server struct {
	Config     *Config
	Controller *Controller

	webServer *healthz.Server
}

func NewServer(config *Config) (*Server, error) {
	consumerConfig := config.AppClusterConfig
	namespaceInformer := config.AppClusterKubeInformers.Core().V1().Namespaces()
	crdInformer := config.AppClusterApiextensionsInformers.Apiextensions().V1().CustomResourceDefinitions()

	k, err := New(
		consumerConfig,
		config.BindingClusterConfig,
		config.BindingRootNamespace,
		config.ControlPlaneMode,
		config.BindingClusterBindInformers.KlutchBind().V1alpha1().APIServiceBindings(),
		config.BindingClusterKubeInformers.Core().V1().Secrets(),
		namespaceInformer,
		crdInformer,
	)
	if err != nil {
		return nil, err
	}

	s := &Server{
		Config:     config,
		Controller: k,
	}

	s.webServer, err = healthz.NewServer()
	if err != nil {
		return nil, fmt.Errorf("error setting up HTTP Server: %w", err)
	}

	return s, nil
}

func (s *Server) StartHealthCheck(ctx context.Context) {
	s.webServer.Start(ctx)
}

func (s *Server) AddCheck(check healthz.HealthChecker) {
	s.webServer.Checker.AddCheck(check)
}

type prepared struct {
	Server
}

type Prepared struct {
	*prepared
}

func (s *Server) PrepareRun(ctx context.Context) (Prepared, error) {
	// If app and binding clusters differ, CRD bootstrap happens in the app cluster.
	if s.Config.ControlPlaneMode || (s.Config.AppClusterConfig != nil && s.Config.BindingClusterConfig != nil && s.Config.AppClusterConfig.Host != s.Config.BindingClusterConfig.Host) {
		return Prepared{
			prepared: &prepared{
				Server: *s,
			},
		}, nil
	}

	// Install/upgrade the APIServiceBinding CRD in the binding cluster
	// (where APIServiceBinding CRs are stored)
	if err := crd.Create(ctx,
		s.Config.BindingClusterApiextensionsClient.ApiextensionsV1().CustomResourceDefinitions(),
		metav1.GroupResource{Group: bindv1alpha1.GroupName, Resource: "apiservicebindings"},
	); err != nil {
		return Prepared{}, err
	}
	return Prepared{
		prepared: &prepared{
			Server: *s,
		},
	}, nil
}

func (s *Prepared) OptionallyStartInformers(ctx context.Context) {
	logger := klog.FromContext(ctx)

	// Start informers
	logger.Info("starting informers")

	// App cluster informers (always present)
	s.Config.AppClusterKubeInformers.Start(ctx.Done())
	s.Config.AppClusterApiextensionsInformers.Start(ctx.Done())

	// Binding cluster informers (APIServiceBindings and secrets)
	s.Config.BindingClusterBindInformers.Start(ctx.Done())
	s.Config.BindingClusterKubeInformers.Start(ctx.Done())

	// Wait for sync
	appClusterKubeSynced := s.Config.AppClusterKubeInformers.WaitForCacheSync(ctx.Done())
	appClusterApiextensionsSynced := s.Config.AppClusterApiextensionsInformers.WaitForCacheSync(ctx.Done())
	bindingClusterBindSynced := s.Config.BindingClusterBindInformers.WaitForCacheSync(ctx.Done())
	bindingClusterKubeSynced := s.Config.BindingClusterKubeInformers.WaitForCacheSync(ctx.Done())

	logger.Info("informers are synced",
		"appClusterKubeSynced", fmt.Sprintf("%v", appClusterKubeSynced),
		"appClusterApiextensionsSynced", fmt.Sprintf("%v", appClusterApiextensionsSynced),
		"bindingClusterBindSynced", fmt.Sprintf("%v", bindingClusterBindSynced),
		"bindingClusterKubeSynced", fmt.Sprintf("%v", bindingClusterKubeSynced),
	)
}

func (s Prepared) Run(ctx context.Context) error {
	s.Controller.Start(ctx, 2)
	return nil
}
