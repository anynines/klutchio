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

func (c *client) GetRestore(r *GetRestoreRequest) (*GetRestoreResponse, error) {
	if err := validateGetRestoreRequest(r); err != nil {
		return nil, err
	}

	fullURL := fmt.Sprintf(restoreURLFmt, c.URL, r.InstanceID, r.RestoreID)

	response, err := c.prepareAndDo(http.MethodGet, fullURL, nil, nil)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = drainReader(response.Body)
		response.Body.Close()
	}()

	switch response.StatusCode {
	case http.StatusOK:
		userResponse := &GetRestoreResponse{}
		if err := c.unmarshalResponse(response, userResponse); err != nil {
			return nil, HTTPStatusCodeError{StatusCode: response.StatusCode, ResponseError: err}
		}

		return userResponse, nil
	default:
		return nil, c.handleFailureResponse(response)
	}
}

func validateGetRestoreRequest(request *GetRestoreRequest) error {
	if request.InstanceID == "" {
		return required("instanceID")
	}

	if request.RestoreID == "" {
		return required("restoreID")
	}

	if _, err := strconv.Atoi(request.RestoreID); err != nil {
		return fmt.Errorf("restoreID must be a numerical value")
	}

	return nil
}
