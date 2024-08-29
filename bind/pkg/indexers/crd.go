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

package indexers

import (
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	bindv1alpha1 "github.com/anynines/klutch/bind/pkg/apis/bind/v1alpha1"
)

const (
	CRDByServiceBinding = "CRDByServiceBinding"
)

func IndexCRDByServiceBinding(obj interface{}) ([]string, error) {
	crd, ok := obj.(*apiextensionsv1.CustomResourceDefinition)
	if !ok {
		return nil, nil
	}

	bindings := []string{}
	for _, ref := range crd.OwnerReferences {
		parts := strings.SplitN(ref.APIVersion, "/", 2)
		if parts[0] != bindv1alpha1.SchemeGroupVersion.Group || ref.Kind != "APIServiceBinding" {
			continue
		}
		bindings = append(bindings, ref.Name)
	}
	return bindings, nil
}
