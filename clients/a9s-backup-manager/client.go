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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

const (
	instanceURLFmt        = "%s/instances/%s"
	instanceBackupURLFmt  = "%s/instances/%s/backups"
	backupURLFmt          = "%s/instances/%s/backups/%s"
	instanceConfigURLFmt  = "%s/instances/%s/config"
	instanceRestoreURLFmt = "%s/instances/%s/restores"
	restoreURLFmt         = "%s/instances/%s/restores/%s"
	createRestoreURLFmt   = "%s/instances/%s/backups/%s/restore"
	deleteBackupURLFmt    = "%s/instances/%s/backups/%d"
)

// NewClient is a CreateFunc for creating a new functional Client and
// implements the CreateFunc interface.
func NewClient(config *ClientConfiguration) (Client, error) {
	httpClient := &http.Client{
		Timeout: time.Duration(config.TimeoutSeconds) * time.Second,
	}

	// use default values lifted from DefaultTransport
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	httpClient.Transport = transport

	c := &client{
		Name:       config.Name,
		URL:        strings.TrimRight(config.URL, "/"),
		Verbose:    config.Verbose,
		httpClient: httpClient,
	}
	c.doRequestFunc = c.doRequest

	if config.AuthConfig != nil {
		if config.AuthConfig.BasicAuthConfig == nil {
			return nil, errors.New("non-nil AuthConfig cannot be empty")
		}

		c.AuthConfig = config.AuthConfig
	}

	return c, nil
}

var _ CreateFunc = NewClient

type doRequestFunc func(request *http.Request) (*http.Response, error)

// client provides a functional implementation of the Client interface.
type client struct {
	Name       string
	URL        string
	AuthConfig *AuthConfig
	Verbose    bool

	httpClient    *http.Client
	doRequestFunc doRequestFunc
}

var _ Client = &client{}

// This file contains shared methods used by each interface method of the
// Client interface. Individual interface methods are in the following files:
//
// CreateBackup: create_backup.go

const (
	contentType = "Content-Type"
	jsonType    = "application/json"
)

// prepareAndDo prepares a request for the given method, URL, and
// message body, and executes the request, returning an http.Response or an
// error.  Errors returned from this function represent http-layer errors and
// not errors in the Backup Manager API.
func (c *client) prepareAndDo(method, url string, params map[string]string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader

	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}

		bodyReader = bytes.NewReader(bodyBytes)
	}

	request, err := http.NewRequestWithContext(context.Background(), method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	if bodyReader != nil {
		request.Header.Set(contentType, jsonType)
	}

	if c.AuthConfig != nil {
		if c.AuthConfig.BasicAuthConfig != nil {
			basicAuth := c.AuthConfig.BasicAuthConfig
			request.SetBasicAuth(basicAuth.Username, basicAuth.Password)
		}
	}

	if params != nil {
		q := request.URL.Query()

		for k, v := range params {
			q.Set(k, v)
		}

		request.URL.RawQuery = q.Encode()
	}

	if c.Verbose {
		klog.Infof("backup-manager: doing request to %q", url)
	}

	return c.doRequestFunc(request)
}

func (c *client) doRequest(request *http.Request) (*http.Response, error) {
	return c.httpClient.Do(request)
}

// unmarshalResponse unmarshals the response body of the given response into
// the given object or returns an error.
func (c *client) unmarshalResponse(response *http.Response, obj interface{}) error {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	if c.Verbose {
		klog.Infof("backup-manager: response body: %v, type: %T", string(body), obj)
	}

	err = json.Unmarshal(body, obj)
	if err != nil {
		return err
	}

	return nil
}

// handleFailureResponse returns an HTTPStatusCodeError for the given
// response.
func (c *client) handleFailureResponse(response *http.Response) error {
	klog.Info("handling failure responses")

	httpErr := HTTPStatusCodeError{
		StatusCode: response.StatusCode,
	}

	backupManagerResponse := make(map[string]interface{})
	if err := c.unmarshalResponse(response, &backupManagerResponse); err != nil {
		httpErr.ResponseError = err
		return httpErr
	}

	if errorMessage, ok := backupManagerResponse["error"].(string); ok {
		httpErr.ErrorMessage = &errorMessage
	}

	if description, ok := backupManagerResponse["description"].(string); ok {
		httpErr.Description = &description
	}

	return httpErr
}

// drainReader reads and discards the remaining data in reader (for example
// response body data) For HTTP this ensures that the http connection
// could be reused for another request if the keepalive is enabled.
// see https://gist.github.com/mholt/eba0f2cc96658be0f717#gistcomment-2605879
func drainReader(reader io.Reader) error {
	if reader == nil {
		return nil
	}
	_, drainError := io.Copy(io.Discard, io.LimitReader(reader, 4096))
	return drainError
}
