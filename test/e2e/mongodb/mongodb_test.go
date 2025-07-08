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

package mongodb

import (
	"testing"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"github.com/anynines/klutchio/test/e2e/funcs"
)

func TestMongoDBInstanceLifecycle(t *testing.T) {
	manifests := "./manifests"

	provisionMongoDB := features.New("Provision MongoDB Instance").
		WithSetup("ApplyInitialMongoDBClaim", funcs.AllOf(
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

	// This test creates a service binding for the instance, and checks that the corresponding secret is created
	createServiceBinding := features.New("Create ServiceBinding for MongoDB").
		WithSetup("ApplyInitialServiceBindingClaim", funcs.AllOf(
			funcs.ApplyResources(fieldManager, manifests, "servicebinding-initial.yaml"),
			funcs.ResourcesCreatedWithin(1*time.Minute, manifests, "servicebinding-initial.yaml"),
		)).
		Assess("A secret is created", funcs.AllOf(
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

	upgradeMongoDB := features.New("Upgrade MongoDB Instance").
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

	deprovisionMongoDBServiceBinding := features.New("Deprovision MongoDB servicebinding").
		Assess("DeleteClaim", funcs.DeleteResources(manifests, "servicebinding-initial.yaml")).
		Assess("ServiceBinding is being deleted",
			funcs.ResourcesDeletedWithin(3*time.Minute, manifests, "servicebinding-initial.yaml"),
		).
		Assess("ServiceBinding secret is deleted",
			funcs.ResourceDoesNotExist("sample-mongodb-sb-creds", "mongodb-lifecycle", "v1", "Secret"),
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

	deprovisionMongoDB := features.New("Deprovision MongoDB Instance").
		Assess("DeleteClaim", funcs.DeleteResources(manifests, "claim-upgrade-plan.yaml")).
		Assess("MongoDBInstance is eventually deleted",
			funcs.ResourcesDeletedWithin(10*time.Minute, manifests, "claim-upgrade-plan.yaml"),
		).
		Feature()

	testenv.Test(t, provisionMongoDB, createServiceBinding, takeBackup, restoreBackup, upgradeMongoDB, invalidUpgrades, deprovisionMongoDBServiceBinding, deleteBackup, deleteRestore, deprovisionMongoDB)
}
