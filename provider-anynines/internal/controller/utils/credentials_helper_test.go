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

package utils

import (
	"context"
	"testing"

	apisv1 "github.com/anynines/klutchio/provider-anynines/apis/v1"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newTestSecret(name, namespace, key string, data []byte) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			key: data,
		},
	}
}

func newTestProviderConfig(name string) *apisv1.ProviderConfig {
	return &apisv1.ProviderConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: apisv1.ProviderConfigSpec{
			Url: "https://localhost:3000",
			ProviderCredentials: apisv1.ProviderCredentials{
				Source: xpv1.CredentialsSourceSecret,
				Username: xpv1.CommonCredentialSelectors{
					SecretRef: &xpv1.SecretKeySelector{
						SecretReference: xpv1.SecretReference{
							Namespace: "default",
							Name:      "creds",
						},
						Key: "username",
					},
				},
				Password: xpv1.CommonCredentialSelectors{
					SecretRef: &xpv1.SecretKeySelector{
						SecretReference: xpv1.SecretReference{
							Namespace: "default",
							Name:      "creds",
						},
						Key: "password",
					},
				},
			},
		},
	}
}

// TestGetCredentialsFromProviderBasic tests basic credentials extraction without TLS
func TestGetCredentialsFromProviderBasic(t *testing.T) {
	ctx := context.Background()

	scheme := runtime.NewScheme()
	_ = apisv1.AddToScheme(scheme)
	_ = v1.AddToScheme(scheme)

	credsSecret := newTestSecret("creds", "default", "username", []byte("testuser"))
	credsSecret.Data["password"] = []byte("testpass")

	kube := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(credsSecret).
		Build()

	pc := newTestProviderConfig("test-pc")

	creds, err := GetCredentialsFromProvider(ctx, pc, kube)
	if err != nil {
		t.Fatalf("failed to get credentials: %v", err)
	}

	if string(creds.Username) != "testuser" {
		t.Errorf("expected username 'testuser', got %s", string(creds.Username))
	}

	if string(creds.Password) != "testpass" {
		t.Errorf("expected password 'testpass', got %s", string(creds.Password))
	}

	if creds.InsecureSkipVerify != false {
		t.Errorf("expected InsecureSkipVerify to be false, got %v", creds.InsecureSkipVerify)
	}

	if creds.CABundle != nil {
		t.Errorf("expected CABundle to be nil, got %v", creds.CABundle)
	}
}

// TestGetCredentialsFromProviderWithTLS tests credentials extraction with TLS configuration
func TestGetCredentialsFromProviderWithTLS(t *testing.T) {
	ctx := context.Background()

	scheme := runtime.NewScheme()
	_ = apisv1.AddToScheme(scheme)
	_ = v1.AddToScheme(scheme)

	credsSecret := newTestSecret("creds", "default", "username", []byte("testuser"))
	credsSecret.Data["password"] = []byte("testpass")

	caSecret := newTestSecret("ca-secret", "default", "caCert", []byte("mock-ca-cert-data"))

	kube := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(credsSecret, caSecret).
		Build()

	pc := newTestProviderConfig("test-pc-tls")
	pc.Spec.TLS = &apisv1.ProviderConfigTLS{
		InsecureSkipVerify: false,
		CABundleSecretRef: &xpv1.SecretKeySelector{
			SecretReference: xpv1.SecretReference{
				Namespace: "default",
				Name:      "ca-secret",
			},
			Key: "caCert",
		},
	}

	creds, err := GetCredentialsFromProvider(ctx, pc, kube)
	if err != nil {
		t.Fatalf("failed to get credentials with TLS: %v", err)
	}

	if string(creds.Username) != "testuser" {
		t.Errorf("expected username 'testuser', got %s", string(creds.Username))
	}

	if string(creds.Password) != "testpass" {
		t.Errorf("expected password 'testpass', got %s", string(creds.Password))
	}

	if creds.InsecureSkipVerify != false {
		t.Errorf("expected InsecureSkipVerify to be false, got %v", creds.InsecureSkipVerify)
	}

	if string(creds.CABundle) != "mock-ca-cert-data" {
		t.Errorf("expected CABundle to be 'mock-ca-cert-data', got %s", string(creds.CABundle))
	}
}

// TestGetCredentialsFromProviderWithInsecureSkipVerify tests credentials extraction with InsecureSkipVerify
func TestGetCredentialsFromProviderWithInsecureSkipVerify(t *testing.T) {
	ctx := context.Background()

	scheme := runtime.NewScheme()
	_ = apisv1.AddToScheme(scheme)
	_ = v1.AddToScheme(scheme)

	credsSecret := newTestSecret("creds", "default", "username", []byte("testuser"))
	credsSecret.Data["password"] = []byte("testpass")

	kube := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(credsSecret).
		Build()

	pc := newTestProviderConfig("test-pc-insecure")
	pc.Spec.TLS = &apisv1.ProviderConfigTLS{
		InsecureSkipVerify: true,
	}

	creds, err := GetCredentialsFromProvider(ctx, pc, kube)
	if err != nil {
		t.Fatalf("failed to get credentials: %v", err)
	}

	if creds.InsecureSkipVerify != true {
		t.Errorf("expected InsecureSkipVerify to be true, got %v", creds.InsecureSkipVerify)
	}

	if creds.CABundle != nil {
		t.Errorf("expected CABundle to be nil when InsecureSkipVerify is true, got %v", creds.CABundle)
	}
}

// TestGetCredentialsFromProviderMissingSecret tests error handling when credentials secret is missing
func TestGetCredentialsFromProviderMissingSecret(t *testing.T) {
	ctx := context.Background()

	scheme := runtime.NewScheme()
	_ = apisv1.AddToScheme(scheme)
	_ = v1.AddToScheme(scheme)

	kube := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	pc := newTestProviderConfig("test-pc-missing")

	_, err := GetCredentialsFromProvider(ctx, pc, kube)
	if err == nil {
		t.Fatal("expected error for missing credentials secret")
	}
}

// TestGetCredentialsFromProviderMissingCASecret tests error handling when CA secret is missing
func TestGetCredentialsFromProviderMissingCASecret(t *testing.T) {
	ctx := context.Background()

	scheme := runtime.NewScheme()
	_ = apisv1.AddToScheme(scheme)
	_ = v1.AddToScheme(scheme)

	credsSecret := newTestSecret("creds", "default", "username", []byte("testuser"))
	credsSecret.Data["password"] = []byte("testpass")

	kube := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(credsSecret).
		Build()

	pc := newTestProviderConfig("test-pc-missing-ca")
	pc.Spec.TLS = &apisv1.ProviderConfigTLS{
		CABundleSecretRef: &xpv1.SecretKeySelector{
			SecretReference: xpv1.SecretReference{
				Namespace: "default",
				Name:      "missing-ca-secret",
			},
			Key: "ca.crt",
		},
	}

	_, err := GetCredentialsFromProvider(ctx, pc, kube)
	if err == nil {
		t.Fatal("expected error for missing CA secret")
	}
}

// TestGetCredentialsFromProviderWithDefaultCAKey tests that default key "ca.crt" is used when not specified
func TestGetCredentialsFromProviderWithDefaultCAKey(t *testing.T) {
	ctx := context.Background()

	scheme := runtime.NewScheme()
	_ = apisv1.AddToScheme(scheme)
	_ = v1.AddToScheme(scheme)

	credsSecret := newTestSecret("creds", "default", "username", []byte("testuser"))
	credsSecret.Data["password"] = []byte("testpass")

	caSecret := newTestSecret("ca-secret", "default", "ca.crt", []byte("mock-ca-cert-data"))

	kube := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(credsSecret, caSecret).
		Build()

	pc := newTestProviderConfig("test-pc-default-key")
	pc.Spec.TLS = &apisv1.ProviderConfigTLS{
		CABundleSecretRef: &xpv1.SecretKeySelector{
			SecretReference: xpv1.SecretReference{
				Namespace: "default",
				Name:      "ca-secret",
			},
		},
	}

	creds, err := GetCredentialsFromProvider(ctx, pc, kube)
	if err != nil {
		t.Fatalf("failed to get credentials with default CA key: %v", err)
	}

	if string(creds.CABundle) != "mock-ca-cert-data" {
		t.Errorf("expected CABundle to be 'mock-ca-cert-data', got %s", string(creds.CABundle))
	}
}

// TestCredentialsStructure tests that Credentials struct has all required fields
func TestCredentialsStructure(t *testing.T) {
	creds := Credentials{
		Username:           []byte("user"),
		Password:           []byte("pass"),
		CABundle:           []byte("ca-data"),
		InsecureSkipVerify: true,
	}

	if string(creds.Username) != "user" {
		t.Errorf("expected username 'user', got %s", string(creds.Username))
	}

	if string(creds.Password) != "pass" {
		t.Errorf("expected password 'pass', got %s", string(creds.Password))
	}

	if string(creds.CABundle) != "ca-data" {
		t.Errorf("expected CABundle 'ca-data', got %s", string(creds.CABundle))
	}

	if creds.InsecureSkipVerify != true {
		t.Errorf("expected InsecureSkipVerify to be true, got %v", creds.InsecureSkipVerify)
	}
}
