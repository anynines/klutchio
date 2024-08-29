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
	"github.com/anynines/klutch/bind/pkg/apis/bind/v1alpha1"
)

const (
	ServiceExportByCustomResourceDefinition = "serviceExportByCustomResourceDefinition"
)

func IndexServiceExportByCustomResourceDefinition(obj interface{}) ([]string, error) {
	export, ok := obj.(*v1alpha1.APIServiceExport)
	if !ok {
		return nil, nil
	}

	return []string{export.Name}, nil
}