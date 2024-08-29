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

	osbclient "github.com/anynines/klutch/clients/a9s-open-service-broker"
)

const (
	InstanceNotFound = "InstanceNotFound"
)

// NewSbService is meant to return a "client" to connext to the external resource
// API.
// TODO: Finish implementation of this function
func NewOsbService(username, password []byte, url string) (osbclient.Client, error) {
	cfg := osbclient.DefaultClientConfiguration()
	cfg.Name = "OSBClient"
	cfg.URL = url
	cfg.AuthConfig = &osbclient.AuthConfig{
		BasicAuthConfig: &osbclient.BasicAuthConfig{
			Username: strings.TrimSpace(string(username)),
			Password: strings.TrimSpace(string(password)),
		},
	}

	return osbclient.NewClient(cfg)
}

func IsNotFound(err error) bool {
	OSBErr := &osbclient.HTTPStatusCodeError{}
	if !errors.As(err, OSBErr) || OSBErr.ErrorMessage == nil {
		return false
	}

	if OSBErr.StatusCode == 404 && *OSBErr.ErrorMessage == InstanceNotFound {
		return true
	}

	return false
}

func IsDeleted(err error) bool {
	OSBErr := &osbclient.HTTPStatusCodeError{}
	if !errors.As(err, OSBErr) || OSBErr.ErrorMessage == nil {
		return false
	}

	if OSBErr.StatusCode == 410 && *OSBErr.ErrorMessage == InstanceNotFound {
		return true
	}

	return false
}
