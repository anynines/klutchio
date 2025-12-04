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

package utils

import (
	"context"

	apisv1 "github.com/anynines/klutchio/provider-anynines/apis/v1"
	utilerr "github.com/anynines/klutchio/provider-anynines/pkg/utilerr"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	resource "github.com/crossplane/crossplane-runtime/pkg/resource"
	v1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	errGetCreds = utilerr.FromStr("cannot get credentials")
)

type Credentials struct {
	Username           []byte
	Password           []byte
	CABundle           []byte // TLS CA certificate(s)
	InsecureSkipVerify bool
	OverrideServerName string // Override server name for TLS certificate verification
}

func GetCredentialsFromProvider(ctx context.Context, pc *apisv1.ProviderConfig, kube k8sclient.Client) (Credentials, error) {
	username := pc.Spec.ProviderCredentials.Username
	password := pc.Spec.ProviderCredentials.Password

	usernameData, err := resource.CommonCredentialExtractor(ctx, pc.Spec.ProviderCredentials.Source, kube, username)
	if err != nil {
		return Credentials{}, errGetCreds.WithCause(err)
	}

	passwordData, err := resource.CommonCredentialExtractor(ctx, pc.Spec.ProviderCredentials.Source, kube, password)
	if err != nil {
		return Credentials{}, errGetCreds.WithCause(err)
	}

	creds := Credentials{usernameData, passwordData, nil, false, ""}

	// Extract TLS configuration if provided
	if pc.Spec.TLS != nil {
		creds.InsecureSkipVerify = pc.Spec.TLS.InsecureSkipVerify
		creds.OverrideServerName = pc.Spec.TLS.OverrideServerName

		// Extract CA bundle from secret if referenced
		if pc.Spec.TLS.CABundleSecretRef != nil {
			caBundle, err := extractSecretData(ctx, kube, pc.Spec.TLS.CABundleSecretRef)
			if err != nil {
				return Credentials{}, err
			}
			creds.CABundle = caBundle
		}
	}

	return creds, nil
}

// extractSecretData retrieves the value of a key from a Kubernetes secret
func extractSecretData(ctx context.Context, kube k8sclient.Client, ref *xpv1.SecretKeySelector) ([]byte, error) {
	if ref == nil || ref.Name == "" {
		return nil, nil
	}

	secret := &v1.Secret{}
	err := kube.Get(ctx, k8sclient.ObjectKey{
		Namespace: ref.Namespace,
		Name:      ref.Name,
	}, secret)
	if err != nil {
		return nil, utilerr.FromStr("cannot get secret").WithCause(err)
	}

	key := ref.Key
	if key == "" {
		key = "ca.crt" // default key for CA certificates
	}

	data, ok := secret.Data[key]
	if !ok {
		return nil, utilerr.PlainUserErr("secret key not found in CA certificate secret: " + key)
	}

	return data, nil
}
