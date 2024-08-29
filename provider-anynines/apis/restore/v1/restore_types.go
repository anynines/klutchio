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
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

const (
	// StatusQueued is the status that the a9s Backup Manager returns when queried about a restore
	// that is queued for execution but hasn't begun yet.
	StatusQueued = "queued"
	// StatusRunning is the status that the a9s Backup Manager returns when queried about a restore
	// that is currently being executed.
	StatusRunning = "running"
	// StatusDone is the status that the a9s Backup Manager returns when queried about a restore that
	// has been successfully executed.
	StatusDone = "done"
	// StatusFailed is the status that the a9s Backup Manager returns when queried about a restore
	// whose execution was not successful.
	StatusFailed = "failed"
	// StatusDeleted is the status that the a9s Backup Manager returns when queried about a restore
	// which was deleted.
	StatusDeleted = "deleted"
)

type RestoreParameters struct {
	// BackupName is the claim name of a data service instance backup to use for the restore.
	BackupName string `json:"backupName"`
}

// RestoreObservation represents the observed state of a restore from the
// backup manager api.
type RestoreObservation struct {
	// InstanceID is the ID of a data service instance on which
	// restore should be performed.
	InstanceID string `json:"instanceId,omitempty"`
	// BackupID is the ID of a data service instance backup to use for the restore.
	BackupID *int `json:"backupId,omitempty"`
	// RestoreID represents the restore ID of a data service instance.
	RestoreID *int `json:"restoreId,omitempty"`
	// State represents the status of a restore on a data service instance
	// e.g. queued, running, done, failed, deleted.
	// +kubebuilder:validation:Enum:=queued;running;done;failed;deleted
	State string `json:"state,omitempty"`
	// TriggeredAt represents the timestamp of when the restore was
	// initiated(YYYY-MM-DDThh:mm:ss.sssZ), e.g. 2023-05-01T01:30:28.300Z.
	TriggeredAt string `json:"triggeredAt,omitempty"`
	// FinishedAt represents the timestamp of when the restore was
	// completed(YYYY-MM-DDThh:mm:ss.sssZ), e.g. 2023-05-01T01:30:28.300Z.
	FinishedAt string `json:"finishedAt,omitempty"`
}

// A RestoreSpec defines the desired state of a Restore.
type RestoreSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       RestoreParameters `json:"forProvider"`
}

// A RestoreStatus represents the observed state of a Restore.
type RestoreStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          RestoreObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Restore is a request to use a previously taken Backup to restore a Data Service
// Instance to the state contained in the Backup.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,anynines}
type Restore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RestoreSpec   `json:"spec"`
	Status RestoreStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RestoreList contains a list of Restore
type RestoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Restore `json:"items"`
}

// Restore type metadata.
var (
	RestoreKind             = reflect.TypeOf(Restore{}).Name()
	RestoreGroupKind        = schema.GroupKind{Group: Group, Kind: RestoreKind}.String()
	RestoreKindAPIVersion   = RestoreKind + "." + SchemeGroupVersion.String()
	RestoreGroupVersionKind = SchemeGroupVersion.WithKind(RestoreKind)
)

func init() {
	SchemeBuilder.Register(&Restore{}, &RestoreList{})
}
