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

package test

import (
	"fmt"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"

	sbv1 "github.com/anynines/klutch/provider-anynines/apis/servicebinding/v1"
	dsv1 "github.com/anynines/klutch/provider-anynines/apis/serviceinstance/v1"
	apisv1 "github.com/anynines/klutch/provider-anynines/apis/v1"
	"github.com/go-logr/logr/testr"
	corev1 "k8s.io/api/core/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ProviderConfig(opts ...func(*apisv1.ProviderConfig)) *apisv1.ProviderConfig {
	pc := &apisv1.ProviderConfig{}

	for _, opt := range opts {
		opt(pc)
	}

	return pc
}

func WithProviderConfigSpec(url string,
	username xpv1.CommonCredentialSelectors,
	password xpv1.CommonCredentialSelectors,
	source xpv1.CredentialsSource,
) func(*apisv1.ProviderConfig) {
	pcs := apisv1.ProviderConfigSpec{
		Url: url,
		ProviderCredentials: apisv1.ProviderCredentials{
			Source:   source,
			Username: username,
			Password: password,
		},
	}

	return func(pc *apisv1.ProviderConfig) {
		pc.Spec = pcs
	}
}

func WithProviderConfigHealth(health *apisv1.ProviderConfigHealth) func(*apisv1.ProviderConfig) {
	return func(pc *apisv1.ProviderConfig) {
		pc.Status.Health = *health
	}
}

func SecretRef(name, ns, key string) xpv1.CommonCredentialSelectors {
	sr := xpv1.CommonCredentialSelectors{
		SecretRef: &xpv1.SecretKeySelector{
			SecretReference: xpv1.SecretReference{
				Name:      name,
				Namespace: ns,
			},
			Key: key,
		},
	}

	return sr
}

func Secret(opts ...func(*corev1.Secret)) *corev1.Secret {
	s := &corev1.Secret{}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func WithKey(key, value string) func(*corev1.Secret) {
	return func(s *corev1.Secret) {
		if s.Data == nil {
			s.Data = make(map[string][]byte)
		}

		s.Data[key] = []byte(value)
	}
}

type ProviderConfigHealthOption func(*apisv1.ProviderConfigHealth)

func NewProviderConfigHealth(opts ...ProviderConfigHealthOption) *apisv1.ProviderConfigHealth {
	health := &apisv1.ProviderConfigHealth{}
	for _, opt := range opts {
		opt(health)
	}
	return health
}

func HealthLastCheckTime(t time.Time) ProviderConfigHealthOption {
	return func(h *apisv1.ProviderConfigHealth) {
		metaT := metav1.NewTime(t)
		h.LastCheckTime = &metaT
	}
}

func HealthLastStatus(status bool) ProviderConfigHealthOption {
	return func(h *apisv1.ProviderConfigHealth) {
		h.LastStatus = status
	}
}

func HealthLastMessage(message string) ProviderConfigHealthOption {
	return func(h *apisv1.ProviderConfigHealth) {
		h.LastMessage = message
	}
}

// TODO: This code comes from the ServiceInstance operator tests and could
// potentially be refactored into a shared set of test code
type withObjectMeta interface {
	apisv1.ProviderConfig |
		sbv1.ServiceBinding |
		corev1.Secret |
		dsv1.ServiceInstance
}

func Name[T withObjectMeta](name string) func(*T) {
	return func(obj *T) {
		objMetaOrPanic(obj).SetName(name)
	}
}

func Namespace[T withObjectMeta](namespace string) func(*T) {
	return func(obj *T) {
		objMetaOrPanic(obj).SetNamespace(namespace)
	}
}

func Uid[T withObjectMeta](uid types.UID) func(*T) {
	return func(obj *T) {
		objMetaOrPanic(obj).SetUID(uid)
	}
}

func objMetaOrPanic(obj any) metav1.Object {
	objMeta, err := meta.Accessor(obj)
	if err != nil {
		panic(fmt.Errorf("failed to get metadata for object %#+v: %w", obj, err))
	}
	return objMeta
}

func TestLogger(t *testing.T) logging.Logger {
	return logging.NewLogrLogger(testr.New(t))
}
