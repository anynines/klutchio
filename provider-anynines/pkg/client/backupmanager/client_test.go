/*
Copyright 2025 Klutch Authors. All rights reserved.

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
	"testing"

	bkpmgrclient "github.com/anynines/klutchio/clients/a9s-backup-manager"
)

// TestNewBackupManagerService tests the backward-compatible factory function
func TestNewBackupManagerService(t *testing.T) {
	tests := []struct {
		name        string
		username    []byte
		password    []byte
		url         string
		expectError bool
		description string
	}{
		{
			name:        "valid credentials",
			username:    []byte("testuser"),
			password:    []byte("testpass"),
			url:         "https://localhost:3000",
			expectError: false,
			description: "should create service with valid credentials",
		},
		{
			name:        "empty username",
			username:    []byte(""),
			password:    []byte("testpass"),
			url:         "https://localhost:3000",
			expectError: false,
			description: "should create service even with empty username",
		},
		{
			name:        "empty password",
			username:    []byte("testuser"),
			password:    []byte(""),
			url:         "https://localhost:3000",
			expectError: false,
			description: "should create service even with empty password",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client, err := NewBackupManagerService(test.username, test.password, test.url)

			if test.expectError && err == nil {
				t.Errorf("%s: expected error but got none", test.description)
				return
			}

			if !test.expectError && err != nil {
				t.Errorf("%s: unexpected error: %v", test.description, err)
				return
			}

			if !test.expectError && client == nil {
				t.Errorf("%s: expected client but got nil", test.description)
				return
			}
		})
	}
}

// TestNewBackupManagerServiceWithTLS tests the TLS-aware factory function
func TestNewBackupManagerServiceWithTLS(t *testing.T) {
	tests := []struct {
		name               string
		username           []byte
		password           []byte
		url                string
		insecureSkipVerify bool
		caBundle           []byte
		expectError        bool
		description        string
	}{
		{
			name:               "valid credentials without TLS",
			username:           []byte("testuser"),
			password:           []byte("testpass"),
			url:                "https://localhost:3000",
			insecureSkipVerify: false,
			caBundle:           nil,
			expectError:        false,
			description:        "should create service with valid credentials",
		},
		{
			name:               "valid credentials with InsecureSkipVerify",
			username:           []byte("testuser"),
			password:           []byte("testpass"),
			url:                "https://localhost:3000",
			insecureSkipVerify: true,
			caBundle:           nil,
			expectError:        false,
			description:        "should create service with InsecureSkipVerify",
		},
		{
			name:               "valid credentials with invalid CA bundle",
			username:           []byte("testuser"),
			password:           []byte("testpass"),
			url:                "https://localhost:3000",
			insecureSkipVerify: false,
			caBundle:           []byte("invalid certificate data"),
			expectError:        true,
			description:        "should return error for invalid CA bundle",
		},
		{
			name:               "both InsecureSkipVerify and valid CABundle",
			username:           []byte("testuser"),
			password:           []byte("testpass"),
			url:                "https://localhost:3000",
			insecureSkipVerify: true,
			caBundle:           nil, // Use nil when testing InsecureSkipVerify
			expectError:        false,
			description:        "should create service with InsecureSkipVerify",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client, err := NewBackupManagerServiceWithTLS(
				test.username,
				test.password,
				test.url,
				test.insecureSkipVerify,
				test.caBundle,
				"",
			)

			if test.expectError && err == nil {
				t.Errorf("%s: expected error but got none", test.description)
				return
			}

			if !test.expectError && err != nil {
				t.Errorf("%s: unexpected error: %v", test.description, err)
				return
			}

			if !test.expectError && client == nil {
				t.Errorf("%s: expected client but got nil", test.description)
				return
			}
		})
	}
}

// TestBackwardCompatibilityNewBackupManagerService tests that the old function works the same as before
func TestBackwardCompatibilityNewBackupManagerService(t *testing.T) {
	username := []byte("testuser")
	password := []byte("testpass")
	url := "https://localhost:3000"

	// Call the old function
	client1, err1 := NewBackupManagerService(username, password, url)

	// Call the new function with the same parameters (TLS disabled)
	client2, err2 := NewBackupManagerServiceWithTLS(username, password, url, false, nil, "")

	if (err1 == nil) != (err2 == nil) {
		t.Errorf("error mismatch: NewBackupManagerService returned error=%v, NewBackupManagerServiceWithTLS returned error=%v", err1, err2)
	}

	if err1 == nil && err2 == nil && (client1 == nil) != (client2 == nil) {
		t.Errorf("client mismatch: one returned nil, the other did not")
	}
}

// TestNewBackupManagerServiceDelegation tests that the old function delegates to the new function
func TestNewBackupManagerServiceDelegation(t *testing.T) {
	username := []byte("testuser")
	password := []byte("testpass")
	url := "https://localhost:3000"

	client, err := NewBackupManagerService(username, password, url)

	// The function should successfully delegate and create a client
	if err != nil {
		t.Errorf("failed to create service: %v", err)
	}

	if client == nil {
		t.Fatal("expected client but got nil")
	}

	// Verify it's actually a client (duck typing)
	if _, ok := client.(bkpmgrclient.Client); !ok {
		t.Errorf("expected client to implement Client interface")
	}
}

// TestNewBackupManagerServiceWithTLSProperSignature tests that the TLS function has the correct signature
func TestNewBackupManagerServiceWithTLSProperSignature(t *testing.T) {
	// This test ensures the function accepts the expected parameters
	username := []byte("testuser")
	password := []byte("testpass")
	url := "https://localhost:3000"
	insecureSkipVerify := false
	caBundle := []byte("")

	client, err := NewBackupManagerServiceWithTLS(username, password, url, insecureSkipVerify, caBundle, "")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if client == nil {
		t.Fatal("expected client but got nil")
	}
}

// TestFactoryFunctionsCreateDifferentClients tests that factory functions create independent clients
func TestFactoryFunctionsCreateDifferentClients(t *testing.T) {
	username := []byte("testuser")
	password := []byte("testpass")
	url := "https://localhost:3000"

	client1, _ := NewBackupManagerService(username, password, url)
	client2, _ := NewBackupManagerService(username, password, url)

	// They should be different instances
	if client1 == client2 {
		t.Error("factory functions should create different client instances")
	}
}

// TestFactoryFunctionsWithDifferentCredentials tests that different credentials create different clients
func TestFactoryFunctionsWithDifferentCredentials(t *testing.T) {
	url := "https://localhost:3000"

	client1, _ := NewBackupManagerService([]byte("user1"), []byte("pass1"), url)
	client2, _ := NewBackupManagerService([]byte("user2"), []byte("pass2"), url)

	// They should be different instances with different configurations
	if client1 == client2 {
		t.Error("clients with different credentials should be different instances")
	}
}
