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
	resource "github.com/crossplane/crossplane-runtime/pkg/resource"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	errGetCreds = utilerr.FromStr("cannot get credentials")
)

type Credentials struct {
	Username []byte
	Password []byte
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

	return Credentials{usernameData, passwordData}, nil
}
