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

func (c *client) CreateBackup(r *CreateBackupRequest) (*CreateBackupResponse, error) {
	if err := validateCreateBackupRequest(r); err != nil {
		return nil, err
	}

	fullURL := fmt.Sprintf(instanceBackupURLFmt, c.URL, r.InstanceID)

	response, err := c.prepareAndDo(http.MethodPost, fullURL, nil, nil)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = drainReader(response.Body)
		_ = response.Body.Close()
	}()

	switch response.StatusCode {
	case http.StatusCreated, http.StatusOK:
		userResponse := &CreateBackupResponse{}
		if err := c.unmarshalResponse(response, userResponse); err != nil {
			return nil, HTTPStatusCodeError{StatusCode: response.StatusCode, ResponseError: err}
		}

		return userResponse, nil
	case http.StatusNotFound:
		return nil, InstanceNotFoundError{
			Reason: c.handleFailureResponse(response),
		}
	default:
		return nil, c.handleFailureResponse(response)
	}
}

func required(name string) error {
	return fmt.Errorf("%v is required", name)
}

func validateCreateBackupRequest(request *CreateBackupRequest) error {
	if request.InstanceID == "" {
		return required("instanceID")
	}

	return nil
}
