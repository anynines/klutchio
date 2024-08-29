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

package v2

import (
	"fmt"
	"net/http"
)

func (c *client) CheckAvailability() error {
	fullURL := fmt.Sprintf(instancesURLFmt, c.URL)
	response, err := c.prepareAndDo(http.MethodHead, fullURL, nil, nil, nil)

	if err != nil {
		return err
	}

	defer func() {
		_ = drainReader(response.Body)
		response.Body.Close()
	}()

	switch response.StatusCode {
	case http.StatusOK:
		return nil
	default:
		return AvailabilityInvalidStatusError{response.StatusCode}
	}
}