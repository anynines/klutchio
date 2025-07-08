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

package v1

import (
	"fmt"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	utilerr "github.com/anynines/klutchio/provider-anynines/pkg/utilerr"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

const (
	// StatusQueued is the status that the a9s Backup Manager returns when queried about a backup
	// that is queued for execution but hasn't begun yet.
	StatusQueued = "queued"

	// StatusRunning is the status that the a9s Backup Manager returns when queried about a backup
	// that is currently being executed.
	StatusRunning = "running"

	// StatusDone is the status that the a9s Backup Manager returns when queried about a backup that
	// has been successfully executed.
	StatusDone = "done"

	// StatusFailed is the status that the a9s Backup Manager returns when queried about a backup
	// whose execution was not successful.
	StatusFailed = "failed"

	// StatusDeleted is the status that the a9s Backup Manager returns when queried about a backup
	// whose backup file has been deleted.
	// This means the metadata of the backup is still intact on the a9s Backup Manager but the
	// contents of the backup have been removed from the cloud storage where it was hosted.
	StatusDeleted = "deleted"

	errBackupNotFound = utilerr.PlainUserErr("backup was not found")
)

// BackupParameters are the configurable fields of a Backup.
type BackupParameters struct {
	// InstanceName is the name of the data service instance to take a backup from.
	InstanceName string `json:"instanceName"`

	// EncryptionKey is the key used to encrypt backups.
	EncryptionKey string `json:"encryption_key,omitempty"`

	// ExcludeFromAutoBackup indicates whether the data service instance will be
	// excluded from the backup schedule.
	// https://docs.anynines.com/docs/35.0.0/platform-operator/a9s-backup-service/a9s-po-backup-service-backup-process#regular-backup-cycle
	ExcludeFromAutoBackup *bool `json:"exclude_from_auto_backup,omitempty"`

	// CredentialsUpdatedByUser indicates whether credentials are updated by the user.
	CredentialsUpdatedByUser *bool `json:"credentials_updated_by_user,omitempty"`
}

// BackupObservation are the observable fields of a Backup.
type BackupObservation struct {
	// InstanceID is the ID of the data service instance to take a backup from.
	InstanceID string `json:"instanceId,omitempty"`

	// BackupID is the numeric identifier that the a9s Backup Manager assigned to this specific
	// backup, e.g. 123456. It is used when communicating with the a9s Backup Manager.
	BackupID *int `json:"id,omitempty"`

	// SizeInBytes is the size of the backup in bytes. It is passed without a unit identifier, e.g.
	// 1000 is passed for a backup that is one kilobyte big.
	SizeInBytes uint64 `json:"size,omitempty"`

	// Status is the status of the backup as returned by the a9s Backup Manager. Can be "queued",
	// "running", "done", "failed" or "deleted".
	Status string `json:"status,omitempty"`

	// TriggeredAt is the timestamp from when the backup was triggered in the format
	// "YYYY-MM-DDThh:mm:ss.sssZ", e.g. "2023-05-01T01:30:00.742Z"
	TriggeredAt string `json:"triggered_at,omitempty"`

	// FinishedAt is the timestamp from when the backup was finished in the format
	// "YYYY-MM-DDThh:mm:ss.sssZ", e.g. "2023-05-01T01:30:28.300Z"
	FinishedAt string `json:"finished_at,omitempty"`

	// Downloadable indicates whether the the files that constitute this backup can be downloaded
	// from the cloud storage provider where it is hosted or not.
	// This is only true if a user has updated the credentials of the data service instance from
	// which this backup was taken at least once and the backup this observation belongs to was
	// taken after the last credential change for the instance.
	Downloadable bool `json:"downloadable,omitempty"`
}

// A BackupSpec defines the desired state of a Backup.
type BackupSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       BackupParameters `json:"forProvider"`
}

// A BackupStatus represents the observed state of a Backup.
type BackupStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          BackupObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Backup is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,anynines}
type Backup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackupSpec   `json:"spec"`
	Status BackupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BackupList contains a list of Backup
type BackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Backup `json:"items"`
}

// Backup type metadata.
var (
	BackupKind             = reflect.TypeOf(Backup{}).Name()
	BackupGroupKind        = schema.GroupKind{Group: Group, Kind: BackupKind}.String()
	BackupKindAPIVersion   = BackupKind + "." + SchemeGroupVersion.String()
	BackupGroupVersionKind = SchemeGroupVersion.WithKind(BackupKind)
)

func init() {
	SchemeBuilder.Register(&Backup{}, &BackupList{})
}

func (b *BackupList) ToBackup(name string, errMsg string) (*Backup, error) {
	var returnError (error)
	switch {
	case len(b.Items) > 1:
		returnError = fmt.Errorf(
			"%s, expected 1 backup managed resource for backup %s, but got %d",
			errMsg,
			name,
			len(b.Items),
		)
	case len(b.Items) == 0:
		returnError = fmt.Errorf(
			"%s, failed to list backup managed resources for backup %s: %w",
			errMsg,
			name,
			errBackupNotFound,
		)
	default:
		return &b.Items[0], nil
	}

	return nil, returnError
}
