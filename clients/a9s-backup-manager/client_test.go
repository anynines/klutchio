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
	"fmt"
	"io"
	"net/http"
	"reflect"
	"testing"
)

const malformedResponse = `{`

const conventionalFailureResponseBody = `{
	"error": "TestError",
	"description": "test error description"
}`

func testHTTPStatusCodeError() error {
	errorMessage := "TestError"
	description := "test error description"
	return HTTPStatusCodeError{
		StatusCode:   http.StatusInternalServerError,
		ErrorMessage: &errorMessage,
		Description:  &description,
	}
}

func closer(s string) io.ReadCloser {
	return nopCloser{bytes.NewBufferString(s)}
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

type httpChecks struct {
	URL     string
	body    string
	params  map[string]string
	headers map[string]string
}

type httpReaction struct {
	status int
	body   string
	err    error
	header http.Header
}

func newTestClient(t *testing.T, name string, httpChecks httpChecks, httpReaction httpReaction) *client {
	return &client{
		Name:          "test client",
		Verbose:       true,
		URL:           "https://example.com",
		doRequestFunc: doHTTP(t, name, httpChecks, httpReaction),
	}
}

var errWalkingGhost = fmt.Errorf("test has already failed")

func doHTTP(t *testing.T, name string, checks httpChecks, reaction httpReaction) func(*http.Request) (*http.Response, error) {
	return func(request *http.Request) (*http.Response, error) {
		if len(checks.URL) > 0 && checks.URL != request.URL.Path {
			t.Errorf("%v: unexpected URL; expected %v, got %v", name, checks.URL, request.URL.Path)
			return nil, errWalkingGhost
		}

		for k, v := range checks.headers {
			actualValue := request.Header.Get(k)
			if e, a := v, actualValue; e != a {
				t.Errorf("%v: unexpected header value for key %q; expected %v, got %v", name, k, e, a)
				return nil, errWalkingGhost
			}
		}

		for k, v := range checks.params {
			actualValue := request.URL.Query().Get(k)
			if e, a := v, actualValue; e != a {
				t.Errorf("%v: unexpected parameter value for key %q; expected %v, got %v", name, k, e, a)
				return nil, errWalkingGhost
			}
		}

		var bodyBytes []byte
		if request.Body != nil {
			var err error
			bodyBytes, err = io.ReadAll(request.Body)
			if err != nil {
				t.Errorf("%v: error reading request body bytes: %v", name, err)
				return nil, errWalkingGhost
			}
		}

		if e, a := checks.body, string(bodyBytes); e != a {
			t.Errorf("%v: unexpected request body: expected %v, got %v", name, e, a)
			return nil, errWalkingGhost
		}

		return &http.Response{
			StatusCode: reaction.status,
			Header:     reaction.header,
			Body:       closer(reaction.body),
		}, reaction.err
	}
}

func doResponseChecks(t *testing.T, name string, response interface{}, err error, expectedResponse interface{}, expectedErrMessage string, expectedErr error) {
	if err != nil && expectedErrMessage == "" && expectedErr == nil {
		t.Errorf("%v: error performing request: %v", name, err)
		return
	} else if err != nil && expectedErrMessage != "" && expectedErrMessage != err.Error() {
		t.Errorf("%v: unexpected error message: expected %v, got %v", name, expectedErrMessage, err)
		return
	} else if err != nil && expectedErr != nil && !reflect.DeepEqual(expectedErr, err) {
		t.Errorf("%v: unexpected error:\n\nexpected: %+v\n\ngot:      %+v", name, expectedErr, err)
		return
	}

	if e, a := expectedResponse, response; !reflect.DeepEqual(e, a) {
		t.Errorf("%v: unexpected diff in response;\n\nexpected: %+v\n\ngot:      %+v", name, e, a)
		return
	}
}

const justDescriptionErr = `{
  "description": "test description"
}`

const justErrorErr = `{
  "error": "test error"
}`

const fullErr = `{
  "description": "test description",
  "error": "test error"
}`

const invalidErrorErr = `{
  "error": {
    "foo": "bar"
  }
}`

const invalidErrorValidDescriptionErr = `{
  "description": "test description",
  "error": {
    "foo": "bar"
  }
}`

const invalidJSONErr = `{`

func TestHandleFailureResponse(t *testing.T) {
	cases := []struct {
		name               string
		errBody            string
		expectedErrMessage string
	}{
		{
			name:               "error with just description",
			errBody:            justDescriptionErr,
			expectedErrMessage: "Status: 500; ErrorMessage: <nil>; Description: test description; ResponseError: <nil>",
		},
		{
			name:               "error with just error message",
			errBody:            justErrorErr,
			expectedErrMessage: "Status: 500; ErrorMessage: test error; Description: <nil>; ResponseError: <nil>",
		},
		{
			name:               "error with error message and description",
			errBody:            fullErr,
			expectedErrMessage: "Status: 500; ErrorMessage: test error; Description: test description; ResponseError: <nil>",
		},
		{
			name:               "error with invalid error message",
			errBody:            invalidErrorErr,
			expectedErrMessage: "Status: 500; ErrorMessage: <nil>; Description: <nil>; ResponseError: <nil>",
		},
		{
			name:               "error with invalid error message and valid description",
			errBody:            invalidErrorValidDescriptionErr,
			expectedErrMessage: "Status: 500; ErrorMessage: <nil>; Description: test description; ResponseError: <nil>",
		},
		{
			name:               "invalid error",
			errBody:            invalidJSONErr,
			expectedErrMessage: "Status: 500; ErrorMessage: <nil>; Description: <nil>; ResponseError: unexpected end of JSON input",
		},
	}
	for _, tc := range cases {
		klient := newTestClient(t, tc.name, httpChecks{}, httpReaction{})

		testResponse := &http.Response{
			StatusCode: 500,
			Body:       closer(tc.errBody),
		}
		err := klient.handleFailureResponse(testResponse)

		if e, a := tc.expectedErrMessage, err.Error(); e != a {
			t.Errorf("%v: expected %v, got %v", tc.name, e, a)
		}
	}
}
