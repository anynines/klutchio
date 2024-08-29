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
	"strconv"
)

func (c *client) CreateRestore(r *CreateRestoreRequest) (*CreateRestoreResponse, error) {
	if err := validateCreateRestoreRequest(r); err != nil {
		return nil, err
	}

	fullURL := fmt.Sprintf(createRestoreURLFmt, c.URL, r.InstanceID, r.BackupID)

	response, err := c.prepareAndDo(http.MethodPost, fullURL, nil, nil)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = drainReader(response.Body)
		_ = response.Body.Close()
	}()

	switch response.StatusCode {
	case http.StatusAccepted:
		userResponse := &CreateRestoreResponse{}
		if err := c.unmarshalResponse(response, userResponse); err != nil {
			return nil, HTTPStatusCodeError{StatusCode: response.StatusCode, ResponseError: err}
		}
		return userResponse, nil
	case http.StatusConflict:
		return nil, RestoreInProgress{
			reason: c.handleFailureResponse(response),
		}
	case http.StatusUnprocessableEntity:
		return nil, BackupNonRestorableState{
			reason: c.handleFailureResponse(response),
		}
	case http.StatusNotFound:
		return nil, BackupNotFound{
			reason: c.handleFailureResponse(response),
		}
	default:
		return nil, c.handleFailureResponse(response)
	}
}

func validateCreateRestoreRequest(request *CreateRestoreRequest) error {
	if request.InstanceID == "" {
		return required("instanceID")
	}
	if request.BackupID == "" {
		return required("backupID")
	}

	if _, err := strconv.Atoi(request.BackupID); err != nil {
		return fmt.Errorf("backupID must be a numerical value")
	}

	return nil
}
