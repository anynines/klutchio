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
	"testing"
	"time"

	"github.com/anynines/klutch/test/e2e/funcs"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	// Must equal metadata.name in ./manifests/install/provider.yaml
	crossplaneProviderName = "anynines-provider"
	// Must equal metadata.name in ./manifests/configuration.yaml
	crossplaneConfigurationName = "anynines-dataservices"
)

func TestProviderInstallAndConfigure(t *testing.T) {
	manifests := "./manifests"

	expectedXRDs := []string{
		"xbackups.anynines.com",
		"xlogme2instances.anynines.com",
		"xmariadbinstances.anynines.com",
		"xmessaginginstances.anynines.com",
		"xmongodbinstances.anynines.com",
		"xpostgresqlinstances.anynines.com",
		"xprometheusinstances.anynines.com",
		"xrestores.anynines.com",
		"xsearchinstances.anynines.com",
		"xservicebindings.anynines.com",
	}

	installProvider := features.New("Install anynines Provider").
		WithSetup("Install crossplane using Helm", funcs.InstallCrossplane()).
		WithSetup("Install anynines crossplane provider", funcs.AllOf(
			funcs.ApplyResources(fieldManager, manifests, "install/*.yaml"),
			funcs.ResourcesCreatedWithin(10*time.Second, manifests, "install/*.yaml"),
		)).
		Assess("Provider becomes installed and healthy", funcs.ResourceHasConditionWithin(
			time.Minute, crossplaneProviderName, "", "pkg.crossplane.io/v1", "Provider",
			xpv1.Condition{Type: "Installed", Status: "True"},
			xpv1.Condition{Type: "Healthy", Status: "True"},
		)).Feature()

	installConfigurations := features.New("Install anynines crossplane configuration").
		WithSetup("Install configuration package", funcs.AllOf(
			funcs.ApplyResources(fieldManager, manifests, "configuration.yaml"),
			funcs.ResourcesCreatedWithin(10*time.Second, manifests, "configuration.yaml"),
		)).
		Assess("Configuration becomes installed and healthy", funcs.ResourceHasConditionWithin(
			time.Minute, crossplaneConfigurationName, "", "pkg.crossplane.io/v1", "Configuration",
			xpv1.Condition{Type: "Installed", Status: "True"},
			xpv1.Condition{Type: "Healthy", Status: "True"},
		)).
		Assess("Expected XRDs become established and offered",
			funcs.ForEach(expectedXRDs, funcs.CompositeResourceEstablishedAndOffered),
		).Feature()

	testenv.Test(t, installProvider, installConfigurations)
}
