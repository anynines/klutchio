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

package prometheus

import (
	_ "embed"
	"testing"
	"time"

	"github.com/anynines/klutch/test/e2e/funcs"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// ToDo: Add test cases for backups and restores.

// TestPrometheusInstanceLifecycle tests the lifecycle of a single instance, via
// a sequential set of features.
//
// These features must be run in order and to completion, since subsequent features
// rely on the state produced by previous features.
func TestPrometheusInstanceLifecycle(t *testing.T) {
	manifests := "./manifests"

	// This test verifies that a Prometheus XRC is successfully applied, leading
	// to the creation of the associated labels to the Composite Resource.
	provisionPrometheus := features.New("Provision Prometheus Instance").
		WithSetup("ApplyInitialPrometheusClaim", funcs.AllOf(

			// Note: When we have a special or specific use of a function, we
			// can show its logic directly in the testing file instead of making
			// a separate function in the "funcs" package.
			funcs.ApplyResources(fieldManager, manifests, "claim-initial.yaml"),
			funcs.ResourcesCreatedWithin(1*time.Minute, manifests, "claim-initial.yaml"),
		)).
		Assess("ClaimBecomesAvailable",
			funcs.ResourcesHaveConditionWithin(15*time.Minute, manifests, "claim-initial.yaml", xpv1.Available()),
		).Feature()

	// Note: We may consider adding assertions on the labels of the ServiceInstance.
	// This is because there is an implicit dependency on them when the
	// ServiceBinding fetches the instances.
	// As the current test structure involves creating ServiceInstances and
	// ServiceBindings within the same feature, any potential issues related
	// to the dependency on ServiceInstance labels would be caught.

	// This test creates a service binding for the instance, and checks that the
	// corresponding secret is created and contains the expected fields based on
	// the existing service binding secret file. The fields are also checked to
	// ensure that they are not empty.
	createServiceBinding := features.New("Create ServiceBinding for Prometheus").
		WithSetup("ApplyInitialServiceBindingClaim", funcs.AllOf(
			funcs.ApplyResources(fieldManager, manifests, "servicebinding-initial.yaml"),
			funcs.ResourcesCreatedWithin(1*time.Minute, manifests, "servicebinding-initial.yaml"),
		)).
		Assess("A secret is created", funcs.AllOf(
			funcs.ResourcesCreatedWithin(1*time.Minute, manifests, "servicebinding-secret.yaml"),
			funcs.SecretCreatedWithCredentials(manifests, "servicebinding-secret.yaml"),
		),
		).Feature()

	// ToDo: Add actual data to be backed up.
	takeBackup := features.New("Take a Backup").
		WithSetup("ApplyBackupClaim", funcs.AllOf(
			funcs.ApplyResources(fieldManager, manifests, "claim-backup.yaml"),
			funcs.ResourcesCreatedWithin(1*time.Minute, manifests, "claim-backup.yaml"),
		)).
		Assess("ClaimBecomesAvailable",
			funcs.ResourcesHaveConditionWithin(15*time.Minute, manifests, "claim-backup.yaml", xpv1.Available()),
		).
		Feature()

	// TODO: Check actual data is restored.
	restoreBackup := features.New("Restore a backup").
		WithSetup("ApplyRestoreClaim", funcs.AllOf(
			funcs.ApplyResources(fieldManager, manifests, "claim-restore.yaml"),
			funcs.ResourcesCreatedWithin(1*time.Minute, manifests, "claim-restore.yaml"),
		)).
		Assess("ClaimBecomesAvailable",
			funcs.ResourcesHaveConditionWithin(5*time.Minute, manifests, "claim-restore.yaml", xpv1.Available()),
		).
		Feature()

	// Note: Checking the status of MR by directly accessing the Service
	// Broker API is removed. This is something that can be tested at a
	// layer below our end-to-end tests or at the level of the client.

	// ToDo: we should assess the expected secondary k8s resources,
	// such as ConfigMaps and Secrets, to verify the behavior we expect.

	// ToDo: Think about explicit testing of associated MR in future

	// Attempt to apply some invalid changes. These should all be rejected by the API server.
	// This is not a good place to test *all* of the validations, so only two are checked to make
	// sure that validations take effect in general.
	//
	// Individual validations are covered by unit tests in `test/validations/`.
	invalidUpgrades := features.New("Attempt to select invalid upgrades").
		Assess("Change of 'service' is not allowed",
			funcs.ApplyInvalid(fieldManager, manifests, "claim-upgrade-service-not-allowed.yaml",
				"Service is an immutable field"),
		).
		Assess("Change of 'plan' is not allowed",
			funcs.ApplyInvalid(fieldManager, manifests, "claim-upgrade-plan-not-allowed.yaml",
				"Upgrade functionality for Prometheus plans is not supported"),
		).
		Feature()

	deprovisionPrometheusServiceBinding := features.New("Deprovision Prometheus servicebinding").
		Assess("DeleteClaim", funcs.DeleteResources(manifests, "servicebinding-initial.yaml")).
		Assess("ServiceBinding is being deleted",
			funcs.ResourcesDeletedWithin(3*time.Minute, manifests, "servicebinding-initial.yaml"),
		).
		Assess("ServiceBinding secret is deleted",
			funcs.ResourceDoesNotExist("sample-prometheus-servicebinding-creds", "prometheus-lifecycle", "v1", "Secret"),
		).
		Feature()

	deleteBackup := features.New("Delete backup").
		Assess("DeleteClaim", funcs.DeleteResources(manifests, "claim-backup.yaml")).
		Assess("Backup is being deleted",
			funcs.ResourcesDeletedWithin(10*time.Minute, manifests, "claim-backup.yaml"),
		).
		Feature()

	deleteRestore := features.New("Delete restore object").
		Assess("DeleteClaim", funcs.DeleteResources(manifests, "claim-restore.yaml")).
		Assess("Restore is being deleted",
			funcs.ResourcesDeletedWithin(10*time.Minute, manifests, "claim-restore.yaml"),
		).
		Feature()

	// This test deprovisions a Prometheus Service Instance
	deprovisionPrometheus := features.New("Deprovision Prometheus Instance").
		Assess("DeleteClaim", funcs.DeleteResources(manifests, "claim-initial.yaml")).
		Assess("PrometheusInstance is deleted",
			funcs.ResourcesDeletedWithin(20*time.Minute, manifests, "claim-initial.yaml"),
		).
		Feature()

	testenv.Test(t, provisionPrometheus, createServiceBinding, takeBackup, restoreBackup, invalidUpgrades, deprovisionPrometheusServiceBinding, deleteBackup, deleteRestore, deprovisionPrometheus)
}
