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

package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/support/kind"
)

const fieldManager = "crossplane-e2e-tests"

var testenv env.Environment

func TestMain(m *testing.M) {
	testenv = env.New()
	// When changing the `cb-e2e` prefix to something else, make sure to also modify
	// `test/e2e/Makefile` accordingly.
	kindClusterName := envconf.RandomName("cb-e2e", 16)
	namespace := envconf.RandomName("myns", 16)

	// Use pre-defined environment funcs to create a kind cluster prior to test run
	testenv.Setup(
		envfuncs.CreateCluster(kind.NewProvider(), kindClusterName),
	)

	testenv.Setup(
		envfuncs.CreateNamespace(namespace),
	)

	testenv.Finish(
		envfuncs.DeleteNamespace(namespace),
	)

	if os.Getenv("KEEP_CLUSTER") == "true" {
		testenv.Finish(
			func(ctx context.Context, c *envconf.Config) (context.Context, error) {
				fmt.Printf(
					"KEEP_CLUSTER=true: not deleting cluster \"kind-%s\". Have fun with it! 🚀",
					kindClusterName)
				return ctx, nil
			},
		)
	} else {
		testenv.Finish(
			envfuncs.DestroyCluster(kindClusterName),
		)
	}

	os.Exit(testenv.Run(m))
}
