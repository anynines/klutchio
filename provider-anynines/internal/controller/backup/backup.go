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

package backup

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	bkpmgrclient "github.com/anynines/klutch/clients/a9s-backup-manager"
	v1 "github.com/anynines/klutch/provider-anynines/apis/backup/v1"
	dsv1 "github.com/anynines/klutch/provider-anynines/apis/serviceinstance/v1"
	apisv1 "github.com/anynines/klutch/provider-anynines/apis/v1"
	util "github.com/anynines/klutch/provider-anynines/internal/controller/utils"
	bkpclient "github.com/anynines/klutch/provider-anynines/pkg/client/backupmanager"
	"github.com/anynines/klutch/provider-anynines/pkg/constants"
	utilerr "github.com/anynines/klutch/provider-anynines/pkg/utilerr"
)

const (
	// AnnotationKeyBackupID is the name of the annotation containing the ID of the Backup object the annotation belongs
	// to. The ID is determined by the a9s Backup Manager during the provisioning of a backup and then sent to this
	// controller during the Create() function. Since after executing Create() the crossplane runtime only persists
	// changes in annotations of the reconciled object, not its fields, we need to persist the ID in an annotation.
	AnnotationKeyBackupID = "anynines.crossplane.io/backup-id"

	// errNotBackup is the message of the error that is triggered when the managed resource handed
	// to one of the controller's functions is not a Backup custom resource.
	errNotBackup = "something went wrong with crossplane as managed resource reconciled is not a Backup custom resource, THIS SHOULD NOT HAPPEN"
	// errUnknownState is the message of the error that is triggered when the status value returned
	// by the a9s Backup Manager does not match any of the possible status values known to this
	// controller.
	errUnknownState = `cannot determine state of backup: a9s Backup Manager returned unknown status "%s". Known statuses are "queued", "running", "done", "failed" and "deleted"`

	// errNewBackup is the message of the error that is triggered when the creation of a new backup
	// fails during the Create() function of the controller.
	errNewBackup = "cannot create new backup"
	// errGetBackup is the message of the error that is triggered when the controller fails to
	// retrieve a backup it's currently trying to observe from the a9s Backup Manager.
	errGetBackup = "cannot get Backup"
	// errDeleteBackup is the message of the error that is triggered when the deletion of a backup
	// fails during the Delete() function of the controller.
	errDeleteBackup = "cannot delete backup"

	// errTrackPCUsage is the message of the error that is triggered when the controller fails to
	// track that the managed resource is using a ProviderConfig.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is the message of the error that is triggered when the ProviderConfig handed
	// to the controller's Connect() function is not retrievable.
	errGetPC = "cannot get ProviderConfig"
	// errNewClient is the message of the error that is triggered when the creation of a new client
	// fails  during the Connect() function of the controller.
	errNewClient = "cannot create new client"

	// ErrServiceInstanceNotFound is the message of the error that is triggered when service instance referenced in a backup
	// is not found.
	ErrServiceInstanceNotFound = utilerr.PlainUserErr("data service instance was not found")

	// errBackupStatusIsUnset is the message of the error that is triggered when the status field
	// of backup is unset
	errBackupStatusIsUnset = "backup status field is unset, setting required values"
)

// Setup adds a controller that reconciles Backup managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1.BackupGroupKind)
	cps := util.GetConnectionPublisher(mgr, o)

	log := o.Logger.WithValues("controller", name)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1.BackupGroupVersionKind),
		managed.WithExternalConnecter(utilerr.ConnectDecorator{
			Connector: &connector{
				kube:         mgr.GetClient(),
				usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1.ProviderConfigUsage{}),
				newServiceFn: bkpclient.NewBackupManagerService,
			},
			Logger: log,
		}),
		managed.WithLogger(log),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1.Backup{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         k8sclient.Client
	usage        resource.Tracker
	newServiceFn func(username, password []byte, url string) (bkpmgrclient.Client, error)
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	bkp, ok := mg.(*v1.Backup)
	if !ok {
		return nil, errors.New(errNotBackup)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, fmt.Errorf("%s: %w", errTrackPCUsage, err)
	}

	pc := &apisv1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: bkp.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, fmt.Errorf("%s: %w", errGetPC, err)
	}

	credentials, err := util.GetCredentialsFromProvider(ctx, pc, c.kube)
	if err != nil {
		return nil, err
	}

	svc, err := c.newServiceFn(credentials.Username, credentials.Password, pc.Spec.Url)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errNewClient, err)
	}

	return &External{
		Client: svc,
		Kube:   c.kube,
	}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// External resource to ensure it reflects the managed resource's desired state.
type External struct {
	// A 'client' used to connect to the external resource API. In practice this
	// would be something like an AWS SDK client.
	Client bkpmgrclient.Client

	// A k8s client is used to retrieve the MRs for Service Instances in order to resolve the
	// instance names into IDs
	Kube k8sclient.Client
}

// getServiceInstanceID returns instanceID from referenced serviceInstance MR
func (c *External) getServiceInstanceID(ctx context.Context, bkp *v1.Backup) (string, error) {
	serviceInstance, err := c.GetServiceInstanceManagedResource(ctx, *bkp)
	if err != nil {
		return "", err
	}

	// Validate status
	if serviceInstance.Status.AtProvider.InstanceID == "" {
		return "", fmt.Errorf("instance is not ready")
	}

	return serviceInstance.Status.AtProvider.InstanceID, nil
}

// initializeBackupStatus initializes InstanceID value in status if not set.
func (c *External) initializeBackupStatus(ctx context.Context, bkp *v1.Backup) error {
	if bkp.Status.AtProvider.InstanceID == "" {
		instanceID, err := c.getServiceInstanceID(ctx, bkp)
		if err != nil {
			return err
		}

		bkp.Status.AtProvider.InstanceID = instanceID
		return fmt.Errorf(errBackupStatusIsUnset)
	}
	return nil
}

func (c *External) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	bkp, ok := mg.(*v1.Backup)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotBackup)
	}

	err := c.initializeBackupStatus(ctx, bkp)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	if getExternalName(bkp) == "" {
		return managed.ExternalObservation{}, nil
	}

	if bkp.Status.AtProvider.BackupID == nil {
		bkpID, err := strconv.Atoi(bkp.Annotations[AnnotationKeyBackupID])
		if err != nil {
			return managed.ExternalObservation{}, fmt.Errorf("%s: %w", errGetBackup, err)
		}

		bkp.Status.AtProvider.BackupID = &bkpID
	}

	getBackupResponse, err := c.Client.GetBackup(&bkpmgrclient.GetBackupRequest{
		InstanceID: bkp.Status.AtProvider.InstanceID,
		BackupID:   strconv.Itoa(*bkp.Status.AtProvider.BackupID),
	})
	if err != nil {
		return managed.ExternalObservation{},
			fmt.Errorf("%s: %w", errGetBackup, utilerr.HandleHttpError(err))
	}

	bkp.Status.AtProvider = bkpclient.GenerateObservation(*getBackupResponse, *bkp)

	err = setConditions(bkp)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	if bkp.Status.AtProvider.Status == v1.StatusDeleted {
		// When a backup is deleted the object might be still retrievable from
		// the API depending on the deletion method used but it is in a
		// deleted state. We need to return early here so that we don't
		// continually reconcile expecting the instance to be gone.
		return managed.ExternalObservation{}, nil
	}

	return managed.ExternalObservation{
		// Return false when the external resource does not exist. This lets
		// the managed resource reconciler know that it needs to call Create to
		// (re)create the resource, or that it has successfully been deleted.
		ResourceExists: true,
		// Since we do not support updating a backup after creation we never
		// need to call the update method of this controller meaning we never
		// need to return "ResourceUpToDate: false"
		ResourceUpToDate: true,
		// Return any details that may be required to connect to the external
		// resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func setConditions(bkp *v1.Backup) error {
	switch bkp.Status.AtProvider.Status {
	case v1.StatusQueued:
		bkp.SetConditions(xpv1.Creating())
	case v1.StatusRunning:
		bkp.SetConditions(xpv1.Creating())
	case v1.StatusDone:
		bkp.SetConditions(xpv1.Available())
	case v1.StatusFailed:
		bkp.SetConditions(xpv1.Unavailable().WithMessage("Backup has failed"))
	case v1.StatusDeleted:
		bkp.SetConditions(xpv1.Unavailable().WithMessage("Backup has been deleted"))
	default:
		return fmt.Errorf(errUnknownState, bkp.Status.AtProvider.Status)
	}
	return nil
}

func getExternalName(bkp *v1.Backup) string {
	annotations := bkp.GetAnnotations()
	if annotations == nil {
		return ""
	}
	return annotations[AnnotationKeyBackupID]
}

func (c *External) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	bkp, ok := mg.(*v1.Backup)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotBackup)
	}

	// Currently it's possible for the backup controller to request a backup from the Backup Manager
	// and then crash before the BackupID is persisted in backup MR, giving us an orphaned backup
	// belonging to no MR. If we were able to persist the ID in the MR first and then the controller
	// crashes we could easily detect that no backup with that ID was ever provisioned from the
	// Backup Manager and request it.
	// We asked the team maintaining the a9s Backup Manager to expand the Backup Manager's API so
	// that we have the option of providing the BackupID ourselves, if that change has happened we
	// will update this method to use the new API endpoint to prevent the aforementioned behavior.
	response, err := c.Client.CreateBackup(&bkpmgrclient.CreateBackupRequest{
		InstanceID: bkp.Status.AtProvider.InstanceID,
	})
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("%s: %w", errNewBackup, utilerr.HandleHttpError(err))
	}

	// Setting the backup-id annotation like this is necessary because the Observe and Delete
	// methods are not able to retrieve a backup on the InstanceID alone and annotations are the
	// only part of the backup object that Crossplane persists after calling the Create method.
	// We use our own annotation instead of crossplane.io/external-name, because the external-name
	// annotation is initialized with the name of the managed resource and we want an annotation
	// that is initialized with an empty string. We might in the future use the external-name
	// annotation if Crossplane is outfitted with the option to disable the initialization with the
	// name of the managed resource.
	meta.AddAnnotations(bkp, map[string]string{AnnotationKeyBackupID: strconv.Itoa(*response.BackupID)})

	return managed.ExternalCreation{}, nil
}

func (c *External) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	bkp, ok := mg.(*v1.Backup)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotBackup)
	}

	fmt.Printf("Updating: %+v", bkp)

	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *External) Delete(ctx context.Context, mg resource.Managed) error {
	bkp, ok := mg.(*v1.Backup)
	if !ok {
		return errors.New(errNotBackup)
	}

	if bkp.Status.AtProvider.BackupID == nil {
		if bkp.Annotations == nil || bkp.Annotations[AnnotationKeyBackupID] == "" {
			klog.Warning("No BackupID set, assuming no Backup was ever provisioned on the service broker")
			return nil
		} else {
			backupID, err := strconv.Atoi(bkp.Annotations[AnnotationKeyBackupID])
			if err != nil {
				return fmt.Errorf("%s: %w", errDeleteBackup, err)
			}
			bkp.Status.AtProvider.BackupID = ptr.To[int](backupID)
		}
	}

	_, err := c.Client.DeleteBackup(&bkpmgrclient.DeleteBackupRequest{
		InstanceID: bkp.Status.AtProvider.InstanceID,
		BackupID:   bkp.Status.AtProvider.BackupID,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", errDeleteBackup, utilerr.HandleHttpError(err))
	}
	return nil
}

func (c External) GetServiceInstanceManagedResource(ctx context.Context, bkp v1.Backup) (*dsv1.ServiceInstance, error) {
	// Get Service Instance Managed Resource
	instances := &dsv1.ServiceInstanceList{}

	// Current assumption is that backup-claim exists in the same namespace as ServiceInstance
	// claim, hence cross-namespace resource creation is not supported.
	labelSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{
			constants.LabelKeyClaimName:      bkp.Spec.ForProvider.InstanceName,
			constants.LabelKeyClaimNamespace: bkp.Labels[constants.LabelKeyClaimNamespace],
		},
	})
	if err != nil {
		return nil, fmt.Errorf(
			"%s, failed to create label selector for "+
				"ServiceInstance managed resources: %w",
			errNewBackup,
			err,
		)
	}

	err = c.Kube.List(ctx, instances, &k8sclient.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf(
			"%s, failed to list ServiceInstance managed resource: %w",
			errNewBackup,
			err,
		)
	}

	return instances.ToServiceInstance(bkp.Name)
}
