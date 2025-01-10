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

package client

import (
	"errors"
	"strings"

	bkpmgrclient "github.com/anynines/klutchio/clients/a9s-backup-manager"
	v1 "github.com/anynines/klutchio/provider-anynines/apis/backup/v1"
	rstv1 "github.com/anynines/klutchio/provider-anynines/apis/restore/v1"
)

const (
	InstanceNotFound = "InstanceNotFound"
)

// NewBackupManagerService is meant to return a "client" to connext to the external resource
// API.
// TODO: Finish implementation of this function
func NewBackupManagerService(username, password []byte, url string) (bkpmgrclient.Client, error) {
	cfg := bkpmgrclient.DefaultClientConfiguration()
	cfg.Name = "BackupManagerClient"
	cfg.URL = url
	cfg.AuthConfig = &bkpmgrclient.AuthConfig{
		BasicAuthConfig: &bkpmgrclient.BasicAuthConfig{
			Username: strings.TrimSpace(string(username)),
			Password: strings.TrimSpace(string(password)),
		},
	}

	return bkpmgrclient.NewClient(cfg)
}

func GenerateBackupRestoreObservation(in bkpmgrclient.GetRestoreResponse, rst rstv1.Restore) rstv1.RestoreObservation {
	return rstv1.RestoreObservation{
		RestoreID:   in.RestoreID,
		State:       in.Status,
		TriggeredAt: in.TriggeredAt,
		FinishedAt:  in.FinishedAt,
		InstanceID:  rst.Status.AtProvider.InstanceID,
		BackupID:    rst.Status.AtProvider.BackupID,
	}
}

func IsNotFound(err error) bool {
	bkpMgrErr := &bkpmgrclient.HTTPStatusCodeError{}
	if !errors.As(err, bkpMgrErr) || bkpMgrErr.ErrorMessage == nil {
		return false
	}

	if bkpMgrErr.StatusCode == 404 && *bkpMgrErr.ErrorMessage == InstanceNotFound {
		return true
	}

	return false
}

func IsDeleted(err error) bool {
	bkpMgrErr := &bkpmgrclient.HTTPStatusCodeError{}
	if !errors.As(err, bkpMgrErr) || bkpMgrErr.ErrorMessage == nil {
		return false
	}

	if bkpMgrErr.StatusCode == 410 && *bkpMgrErr.ErrorMessage == InstanceNotFound {
		return true
	}

	return false
}

func GenerateObservation(in bkpmgrclient.GetBackupResponse, bkp v1.Backup) v1.BackupObservation {
	return v1.BackupObservation{
		BackupID:     in.BackupID,
		SizeInBytes:  uint64(in.Size),
		Status:       in.Status,
		TriggeredAt:  in.TriggeredAt,
		FinishedAt:   in.FinishedAt,
		Downloadable: in.Downloadable,
		InstanceID:   bkp.Status.AtProvider.InstanceID,
	}
}
