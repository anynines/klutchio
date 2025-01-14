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

package utilerr

import (
	"errors"

	backupClient "github.com/anynines/klutchio/clients/a9s-backup-manager"
	osbclient "github.com/anynines/klutchio/clients/a9s-open-service-broker"
)

var _ Userdisplayer = &userError{}
var _ Userdisplayer = PlainUserErr("")

type userError struct {
	message error
	error
}

type UserErrorFactory struct {
	Message error
}

func FromStr(message string) UserErrorFactory {
	return UserErrorFactory{Message: PlainUserErr(message)}
}

func (e UserErrorFactory) WithCause(cause error) *userError {
	return &userError{message: e.Message, error: cause}
}

func (e *userError) UserDisplay() error {
	return e.message
}

// PlainUserErr are error Messages that can be displayed to the user as is.
type PlainUserErr string

func (e PlainUserErr) Error() string {
	return string(e)
}
func (e PlainUserErr) UserDisplay() error {
	return e
}

type PlainErr string

func (e PlainErr) Error() string {
	return string(e)
}

func HandleHttpError(err error) error {
	{
		var httpErr osbclient.HTTPStatusCodeError
		if errors.As(err, &httpErr) &&
			httpErr.StatusCode >= 400 && httpErr.StatusCode < 500 && // codes for invalid user input
			httpErr.Description != nil {
			return &userError{
				message: PlainUserErr(*httpErr.Description),
				error:   err,
			}
		}
	}

	{
		var httpErr backupClient.HTTPStatusCodeError
		if errors.As(err, &httpErr) &&
			httpErr.StatusCode >= 400 && httpErr.StatusCode < 500 && // codes for invalid user input
			httpErr.Description != nil {
			return &userError{
				message: PlainUserErr(*httpErr.Description),
				error:   err,
			}
		}
	}

	return err
}
