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

package backupmanager

import (
	"fmt"
	"net/http"
)

// HTTPStatusCodeError is an error type that provides additional information
// based on the Backup Manager API conventions for returning information
// about errors. If the response body provided by the manager to any client
// operation is malformed, an error of this type will be returned with the
// ResponseError field set to the unmarshalling error.
//
// These errors may optionally provide a machine-readable error message and
// human-readable description.
//
// The IsHTTPError method checks whether an error is of this type.
type HTTPStatusCodeError struct {
	// StatusCode is the HTTP status code returned by the backup manager api.
	StatusCode int
	// ErrorMessage is a machine-readable error string that may be returned by
	// the backup manager api.
	ErrorMessage *string
	// Description is a human-readable description of the error that may be
	// returned by the backup manager api.
	Description *string
	// ResponseError is set to the error that occurred when unmarshalling a
	// response body from the backup manager api.
	ResponseError error
}

func (e HTTPStatusCodeError) Error() string {
	errorMessage := "<nil>"
	description := "<nil>"

	if e.ErrorMessage != nil {
		errorMessage = *e.ErrorMessage
	}
	if e.Description != nil {
		description = *e.Description
	}
	return fmt.Sprintf("Status: %v; ErrorMessage: %v; Description: %v; ResponseError: %v", e.StatusCode, errorMessage, description, e.ResponseError)
}

// IsHTTPError returns whether the error represents an HTTPStatusCodeError.  A
// client method returning an HTTP error indicates that the backup manager api
// returned an error code and a correctly formed response body.
func IsHTTPError(err error) (*HTTPStatusCodeError, bool) {
	statusCodeError, ok := err.(HTTPStatusCodeError)
	if ok {
		return &statusCodeError, ok
	}

	statusCodeErrorPointer, ok := err.(*HTTPStatusCodeError)
	if ok {
		return statusCodeErrorPointer, ok
	}

	return nil, ok
}

// IsGoneError returns whether the error represents an HTTP GONE status.
func IsGoneError(err error) bool {
	statusCodeError, ok := err.(HTTPStatusCodeError)
	if !ok {
		return false
	}

	return statusCodeError.StatusCode == http.StatusGone
}

// OperationNotAllowedError is an error type signifying that an operation
// is not allowed for this client.
type OperationNotAllowedError struct {
	reason string
}

func (e OperationNotAllowedError) Error() string {
	return fmt.Sprintf(
		"operation not allowed: %s",
		e.reason,
	)
}

// InstanceNotFoundError is an error type signifying that an instance
// was not found by the client.
type InstanceNotFoundError struct {
	Reason error
}

func (e InstanceNotFoundError) Error() string {
	return fmt.Sprintf(
		"instance not found: %s",
		e.Reason,
	)
}
func (e InstanceNotFoundError) Unwrap() error {
	return e.Reason
}

// BackupNotFoundError is an error type signifying that a backup can
// not be deleted because it has neither failed nor been completed yet
type BackupNotFoundError struct {
	Reason error
}

func (e BackupNotFoundError) Error() string {
	return fmt.Sprintf(
		"backup not found: %s",
		e.Reason,
	)
}
func (e BackupNotFoundError) Unwrap() error {
	return e.Reason
}

// BackupLockedError is an error type signifying that a backup can not
// be deleted because it is currently being restored.
type BackupLockedError struct {
	Reason string
}

func (e BackupLockedError) Error() string {
	return fmt.Sprintf(
		"backup is locked because it is currently being restored: %s",
		e.Reason,
	)
}

// BackupFileDeletionFailed is an error type signifying that the backup manager
// requested the deletion of a backup's file on the cloud provider hosting the
// file but the deletion failed.
type BackupFileDeletionFailed struct {
	reason string
}

func (e BackupFileDeletionFailed) Error() string {
	return fmt.Sprintf(
		"backup file could not be deleted: %s",
		e.reason,
	)
}

// BackupNonRestorableState is an error type signifying that a backup
// is in a non-restorable state.
type BackupNonRestorableState struct {
	reason error
}

func (e BackupNonRestorableState) Error() string {
	return fmt.Sprintf(
		"backup is in a non-restorable state: %s",
		e.reason,
	)
}
func (e BackupNonRestorableState) Unwrap() error {
	return e.reason
}

// BackupNotFound is an error type signifying that no backup
// was found for given ID.
type BackupNotFound struct {
	reason error
}

func (e BackupNotFound) Error() string {
	return fmt.Sprintf(
		"backup not found: %s",
		e.reason,
	)
}
func (e BackupNotFound) Unwrap() error {
	return e.reason
}

// RestoreInProgress is an error type signifying that a restore
// is already in progress.
type RestoreInProgress struct {
	reason error
}

func (e RestoreInProgress) Unwrap() error {
	return e.reason
}

func (e RestoreInProgress) Error() string {
	return fmt.Sprintf(
		"restore already in progress: %s",
		e.reason,
	)
}
