/*
Copyright 2022 The Kube Bind Authors.

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

package konnector

import (
	"bytes"
	"context"
	"embed"
	"fmt"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"

	"github.com/anynines/klutchio/bind/pkg/bootstrap"
)

//go:embed *.yaml
var raw embed.FS

func Bootstrap(ctx context.Context, discoveryClient discovery.DiscoveryInterface, dynamicClient dynamic.Interface, image string) error {
	return bootstrap.Bootstrap(ctx, discoveryClient, dynamicClient, sets.NewString(), raw,
		bootstrap.ReplaceOption("IMAGE", image),
	)
}

func Bytes(image string) ([][]byte, error) {
	entries, err := raw.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("readdir: %w", err)
	}

	var manifestBytes [][]byte

	for _, e := range entries {
		if !e.Type().IsRegular() {
			continue
		}
		b, err := raw.ReadFile(e.Name())
		if err != nil {
			return nil, fmt.Errorf("name:%s:%w", e.Name(), err)
		}

		b = bytes.ReplaceAll(b, []byte("IMAGE"), []byte(image))

		manifestBytes = append(manifestBytes, b)
	}

	return manifestBytes, nil
}
