/*
Copyright 2024 Klutch Authors. All rights reserved.

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

package logme2

import (
	"os"
	"testing"

	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
)

// fieldManager is the server-side apply field manager used when applying
// manifests.
const fieldManager = "crossplane-e2e-tests"

var testenv env.Environment

// We assume that a Kubernetes environment is already set up and has the
// necessary components installed, including access to the a9s Service Broker
// and Backup Manager. At the current stage, we assume that SB and BU Manager
// are accessible on localhost. However, this will probably change as the
// implementation progresses.

func TestMain(m *testing.M) {

	testenv = env.New()
	// Note: Additional flags could be used for further configuration

	// ResolveKubeConfigFile() function is called to get kubeconfig loaded, it
	// uses either --kubeconfig flag, KUBECONFIG env or by default
	// $HOME/.kube/config path.
	path := conf.ResolveKubeConfigFile()
	cfg := envconf.NewWithKubeConfig(path)
	testenv = env.NewWithConfig(cfg)

	testenv.Setup(
		envfuncs.CreateNamespace("logme2-lifecycle"),
	)

	testenv.Finish(
		envfuncs.DeleteNamespace("logme2-lifecycle"),
	)

	os.Exit(testenv.Run(m))
}
