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

package restore

import (
	"context"
	"fmt"
	"strconv"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	a9sbackupmanager "github.com/anynines/klutchio/clients/a9s-backup-manager"
	bkpv1 "github.com/anynines/klutchio/provider-anynines/apis/backup/v1"
	v1 "github.com/anynines/klutchio/provider-anynines/apis/restore/v1"
	apisv1 "github.com/anynines/klutchio/provider-anynines/apis/v1"
	util "github.com/anynines/klutchio/provider-anynines/internal/controller/utils"
	bkpclient "github.com/anynines/klutchio/provider-anynines/pkg/client/backupmanager"
	"github.com/anynines/klutchio/provider-anynines/pkg/constants"
	utilerr "github.com/anynines/klutchio/provider-anynines/pkg/utilerr"
)

const (
	// AnnotationKeyBackupID is the name of the annotation containing the ID of the Backup object
	// the annotation belongs to.
	AnnotationKeyBackupID = "anynines.crossplane.io/backup-id"
	// AnnotationKeyRestoreID is the name of the annotation containing the ID of the Restore object the
	// annotation belongs to. This has to be done in an annotation because the ID is determined during the
	// Create() function and after executing Create() the crossplane runtime only persists changes in
	// annotations of the reconciled object, not its fields.
	AnnotationKeyRestoreID = "anynines.crossplane.io/restore-id"

	errNotRestore = "something went wrong with crossplane as managed resource reconciled is not a Restore " +
		"custom resource, THIS SHOULD NOT HAPPEN"
	// errUnknownState is the message of the error that is triggered when the status value returned
	// by the a9s Backup Manager does not match any of the possible status values known to this
	// controller.
	errUnknownState = `cannot determine state of restore: a9s Backup Manager returned unknown status "%s"' +
		'. Known statuses are "queued", "running", "done", "failed" and "deleted"`

	// errGetRestore is the message of the error that is triggered while retrieving a Restore
	// from the a9s Backup Manager during the Observe() method of this controller.
	errGetRestore = "cannot get Restore"
	// errNewRestore is the message of the error that is triggered when the provisioning of a new
	// Restore at the a9s Backup Manager fails during the Create() method of this controller.
	errNewRestore = "cannot create new restore"

	errTrackPCUsage         = "cannot track ProviderConfig usage"
	errGetPC                = "cannot get ProviderConfig"
	errRestoreInProgress    = "cannot delete restore that is still %s"
	errRestoreRunning       = utilerr.PlainUserErr("cannot delete restore that is still running")
	errRestoreQueued        = utilerr.PlainUserErr("cannot delete restore that is still queued")
	errNewClient            = "cannot create new client"
	errRestoreStatusIsUnset = "restore status field is unset, setting required values"
)

// Setup adds a controller that reconciles Restore managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1.RestoreGroupKind)

	cps := util.GetConnectionPublisher(mgr, o)
	log := o.Logger.WithValues("controller", name)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1.RestoreGroupVersionKind),
		managed.WithExternalConnecter(&utilerr.ConnectDecorator{
			Connector: &connector{
				kube:         mgr.GetClient(),
				usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1.ProviderConfigUsage{}),
				newServiceFn: bkpclient.NewBackupManagerServiceWithTLS},
			Logger: log,
		}),
		managed.WithLogger(log),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1.Restore{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         k8sclient.Client
	usage        resource.Tracker
	newServiceFn func(username, password []byte, url string, insecureSkipVerify bool, caBundle []byte, overrideServerName string) (a9sbackupmanager.Client, error)
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	rst, ok := mg.(*v1.Restore)
	if !ok {
		return nil, fmt.Errorf(errNotRestore)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, fmt.Errorf("%s: %w", errTrackPCUsage, err)
	}

	pc := &apisv1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: rst.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, fmt.Errorf("%s: %w", errGetPC, err)
	}

	credentials, err := util.GetCredentialsFromProvider(ctx, pc, c.kube)
	if err != nil {
		return nil, err
	}

	svc, err := c.newServiceFn(credentials.Username, credentials.Password, pc.Spec.Url, credentials.InsecureSkipVerify, credentials.CABundle, credentials.OverrideServerName)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errNewClient, err)
	}

	return &external{
		service: svc,
		kube:    c.kube,
	}, nil
}

// An external observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	// A 'client' used to connect to the external resource API. In practice this
	// would be something like an AWS SDK client.
	service a9sbackupmanager.Client

	// A k8s client is used to retrieve backup MRs in order to resolve
	// backup name into instance & backup IDs
	kube k8sclient.Client
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	rst, ok := mg.(*v1.Restore)
	if !ok {
		return managed.ExternalObservation{}, fmt.Errorf(errNotRestore)
	}

	err := c.initializeRestoreStatus(ctx, rst)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	// Initiate create restore request if restoreID is not found in annotation or status
	if (rst.Annotations == nil || rst.Annotations[AnnotationKeyRestoreID] == "") &&
		rst.Status.AtProvider.RestoreID == nil {
		return managed.ExternalObservation{}, nil
	}

	if rst.Status.AtProvider.RestoreID == nil {
		rstID, err := strconv.Atoi(rst.Annotations[AnnotationKeyRestoreID])
		if err != nil {
			return managed.ExternalObservation{}, fmt.Errorf("%s: %w", errGetRestore, err)
		}

		rst.Status.AtProvider.RestoreID = &rstID
	}

	getRestoreResponse, err := c.service.GetRestore(&a9sbackupmanager.GetRestoreRequest{
		InstanceID: rst.Status.AtProvider.InstanceID,
		RestoreID:  strconv.Itoa(*rst.Status.AtProvider.RestoreID),
	})

	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("%s: %w", errGetRestore, utilerr.HandleHttpError(err))
	}

	rst.Status.AtProvider = bkpclient.GenerateBackupRestoreObservation(*getRestoreResponse, *rst)

	setConditions(rst)

	// Restores can't be deleted via a9s-backup-manager because no endpoint exists.
	// Crossplane's original workflow is to call the Delete method to request the deletion
	// of the external resource and then Observe waits till the client returns 'not found',
	// after which the finalizer is removed. In our case, we can't delete restore so we would
	// have to intelligently detect deletion in Observe method.
	deletionPath, err := checkForDeletion(rst)
	if deletionPath {
		return managed.ExternalObservation{}, err
	}

	return managed.ExternalObservation{
		// Return false when the external resource does not exist. This lets
		// the managed resource reconciler know that it needs to call Create to
		// (re)create the resource, or that it has successfully been deleted.
		ResourceExists: true,

		// Return false when the external resource exists, but it not up to date
		// with the desired managed resource state. This lets the managed
		// resource reconciler know that it needs to call Update.
		ResourceUpToDate: true,
	}, nil
}

func checkForDeletion(rst *v1.Restore) (bool, error) {
	if rst.DeletionTimestamp == nil {
		return false, nil
	}

	var returnState error = nil
	switch state := rst.Status.AtProvider.State; state {
	case v1.StatusQueued:
		returnState = errRestoreQueued
	case v1.StatusRunning:
		returnState = errRestoreRunning
	}
	return true, returnState
}

func setConditions(rst *v1.Restore) {
	var conditionValue (xpv1.Condition)
	switch rst.Status.AtProvider.State {
	case v1.StatusQueued:
		conditionValue = xpv1.Creating()
	case v1.StatusRunning:
		conditionValue = xpv1.Creating()
	case v1.StatusDone:
		conditionValue = xpv1.Available().WithMessage("Restore completed successfully")
	case v1.StatusFailed:
		conditionValue = xpv1.Unavailable().WithMessage("Restore has failed")
	case v1.StatusDeleted:
		conditionValue = xpv1.Unavailable().WithMessage("Restore has been deleted")
	default:
		conditionValue = xpv1.ReconcileError(
			fmt.Errorf(fmt.Sprintf(errUnknownState, rst.Status.AtProvider.State)))
	}

	rst.Status.SetConditions(conditionValue)
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	rst, ok := mg.(*v1.Restore)
	if !ok {
		return managed.ExternalCreation{}, fmt.Errorf(errNotRestore)
	}

	response, err := c.service.CreateRestore(&a9sbackupmanager.CreateRestoreRequest{
		InstanceID: rst.Status.AtProvider.InstanceID,
		BackupID:   strconv.Itoa(*rst.Status.AtProvider.BackupID),
	})
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("%s: %w", errNewRestore, utilerr.HandleHttpError(err))
	}

	// If the controller provisions a Restore but crashes before being able to save the
	// RestoreID to the managed resource then the Observe method will think that no Restore has been
	// provisioned yet and call the Create method again, leading to an orphaned Restore that is not
	// associated with any Restore managed resource.

	// Setting the restore-id annotation like this is necessary because Observe and Delete
	// methods are not able to retrieve restore using the InstanceID alone and annotations are the
	// only part of the restore object that Crossplane persists after calling the Create method.
	// We use our own annotation instead of crossplane.io/external-name, because the external-name
	// annotation is initialized with the name of the managed resource and we want an annotation
	// that is initialized with an empty string.
	meta.AddAnnotations(rst, map[string]string{constants.AnnotationKeyRestoreID: strconv.Itoa(*response.RestoreID)})
	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	rst, ok := mg.(*v1.Restore)
	if !ok {
		return managed.ExternalUpdate{}, fmt.Errorf(errNotRestore)
	}

	fmt.Printf("Updating: %+v", rst)

	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	rst, ok := mg.(*v1.Restore)
	if !ok {
		return managed.ExternalDelete{}, fmt.Errorf(errNotRestore)
	}

	switch state := rst.Status.AtProvider.State; state {
	case v1.StatusQueued:
		return managed.ExternalDelete{}, errRestoreQueued
	case v1.StatusRunning:
		return managed.ExternalDelete{}, errRestoreRunning
	default:
		return managed.ExternalDelete{}, nil
	}
}

func (c *external) Disconnect(ctx context.Context) error {
	// Unimplemented, required by newer versions of crossplane-runtime
	return nil
}

// initializeRestoreStatus initializes InstanceID and BackupID values in status if not set.
func (c *external) initializeRestoreStatus(ctx context.Context, rst *v1.Restore) error {
	if rst.Status.AtProvider.InstanceID == "" ||
		rst.Status.AtProvider.BackupID == nil {
		instanceID, backupID, err := c.getBackupDetails(ctx, rst)
		if err != nil {
			return err
		}

		rst.Status.AtProvider.InstanceID = instanceID
		rst.Status.AtProvider.BackupID = backupID

		return fmt.Errorf(errRestoreStatusIsUnset)
	}

	return nil
}

// getBackupDetails returns InstanceID and BackupID from referenced backup MR
func (c *external) getBackupDetails(ctx context.Context, rst *v1.Restore) (string, *int, error) {
	backup, err := c.getBackupManagedResource(ctx, *rst)
	if err != nil {
		return "", nil, err
	}

	// Validate status
	if backup.Status.AtProvider.InstanceID == "" ||
		backup.Status.AtProvider.BackupID == nil {
		return "", nil, fmt.Errorf("backup is not ready")
	}

	return backup.Status.AtProvider.InstanceID, backup.Status.AtProvider.BackupID, nil
}

// getBackupManagedResource tries to retrieve Backup MR using backup claim name and namespace
func (c *external) getBackupManagedResource(ctx context.Context, rst v1.Restore) (*bkpv1.Backup, error) {
	// Current assumption is that restore-claim exists in the same namespace as Backup
	// claim, meaning cross-namespace resource creation is not supported for now.
	labelSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{
			constants.LabelKeyClaimName:      rst.Spec.ForProvider.BackupName,
			constants.LabelKeyClaimNamespace: rst.Labels[constants.LabelKeyClaimNamespace],
		},
	})
	if err != nil {
		return nil, fmt.Errorf(
			"%s, failed to create label selector for "+
				"backup managed resources: %w",
			errNewRestore,
			err,
		)
	}

	backupList := &bkpv1.BackupList{}
	err = c.kube.List(ctx, backupList, &k8sclient.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf(
			"%s, failed to list backup managed resource: %w",
			errNewRestore,
			err,
		)
	}

	return backupList.ToBackup(rst.Name, errNewRestore)
}
