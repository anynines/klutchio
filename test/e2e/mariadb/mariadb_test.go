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

package mariadb

import (
	_ "embed"
	"testing"
	"time"

	"github.com/anynines/klutchio/test/e2e/funcs"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"

	"sigs.k8s.io/e2e-framework/pkg/features"
)

// TestMariaDBInstanceLifecycle tests the lifecycle of a single instance, via
// a sequential set of features.
//
// These features must be run in order and to completion, since subsequent features
// rely on the state produced by previous features.
func TestMariaDBInstanceLifecycle(t *testing.T) {
	manifests := "./manifests"

	// This test verifies that a MariaDB XRC is successfully applied, leading
	// to the creation of the associated MR and the addition of labels to the claim.
	provisionMariaDB := features.New("Provision MariaDB Instance").
		WithSetup("ApplyInitialMariaDBClaim", funcs.AllOf(
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
	createServiceBinding := features.New("Create ServiceBinding for MariaDB").
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

	// This test ensures that applying a new plan to a Service Instance
	// results in corresponding changes to the underlying Service Instance.
	upgradeMariaDB := features.New("Upgrade MariaDB Instance").
		Assess("UpgradePlan", funcs.AllOf(
			funcs.ApplyResources(fieldManager, manifests, "claim-upgrade-plan.yaml"),
			funcs.ResourcesCreatedWithin(1*time.Minute, manifests, "claim-upgrade-plan.yaml"),
		)).
		// Confirm that the Plan of the Service Instance Object matches the
		// the one specified in the provided yaml file.
		Assess("PlanNameChanged",
			funcs.CheckFieldValueMatch(manifests, "claim-upgrade-plan.yaml", "spec", "plan"),
		).
		Assess("UpgradedPlanClaimBecomesAvailable", funcs.AllOf(
			// When applying an update of plan, the service broker immediately
			// returns a false positive that the ServiceInstance is ready.
			// To address this, before assessing that the updated Service Instance
			// is ready, we first check if it is non-ready due to being updated.
			funcs.ResourcesHaveConditionWithin(5*time.Minute, manifests, "claim-upgrade-plan.yaml", xpv1.ReconcileSuccess(), xpv1.Creating()),
			funcs.ResourcesHaveConditionWithin(30*time.Minute, manifests, "claim-upgrade-plan.yaml", xpv1.ReconcileSuccess(), xpv1.Available()),
		)).
		Feature()

	// Attempt to apply some invalid changes. These should all be rejected by the API server.
	// This is not a good place to test *all* of the validations, so only two are checked to make
	// sure that validations take effect in general.
	//
	// Individual validations are covered by unit tests in `test/validations/`.
	//
	// Currently, only one MariaDB service is supported, making it impossible
	// to test changes to another service.
	invalidUpgrades := features.New("Attempt to select invalid upgrades").
		Assess("Change of 'service' is not allowed",
			funcs.ApplyInvalid(fieldManager, manifests, "claim-upgrade-service-not-allowed.yaml",
				"Service is an immutable field"),
		).
		Assess("Downgrade of 'plan' is not allowed",
			funcs.ApplyInvalid(fieldManager, manifests, "claim-downgrade-plan-not-allowed.yaml",
				"Transition from bigger to smaller plan size is not supported."),
		).
		Feature()

	deprovisionMariaDBServiceBinding := features.New("Deprovision MariaDB servicebinding").
		Assess("DeleteClaim", funcs.DeleteResources(manifests, "servicebinding-initial.yaml")).
		Assess("ServiceBinding is being deleted",
			funcs.ResourcesDeletedWithin(3*time.Minute, manifests, "servicebinding-initial.yaml"),
		).
		Assess("ServiceBinding secret is deleted",
			funcs.ResourceDoesNotExist("sample-mariadb-instance-sb-creds", "mariadb-lifecycle", "v1", "Secret"),
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

		// This test deprovisions a MariaDB Service Instance and verifies the
		// corresponding MR's deletion.
	deprovisionMariaDB := features.New("Deprovision MariaDB Instance").
		Assess("DeleteClaim", funcs.DeleteResources(manifests, "claim-upgrade-plan.yaml")).
		Assess("MariaDBInstance is deleted",
			funcs.ResourcesDeletedWithin(20*time.Minute, manifests, "claim-upgrade-plan.yaml"),
		).
		Feature()

	testenv.Test(t, provisionMariaDB, createServiceBinding, takeBackup, restoreBackup, upgradeMariaDB, invalidUpgrades, deprovisionMariaDBServiceBinding, deleteBackup, deleteRestore, deprovisionMariaDB)
}
