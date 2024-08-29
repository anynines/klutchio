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

package generator

import (
	"encoding/json"
	"testing"

	"k8s.io/klog/v2"
)

func TestCreateGenerator(t *testing.T) {
	g := CreateGenerator(3, Parameters{
		Services: ServiceRanges{
			Plans:                5,
			Tags:                 6,
			Metadata:             4,
			Requires:             2,
			Bindable:             10,
			InstancesRetrievable: 1,
			BindingsRetrievable:  1,
		},
		Plans: PlanRanges{
			Metadata: 4,
			Bindable: 10,
			Free:     4,
		},
	})
	AssignPoolGoT(g)

	catalog, err := g.GetCatalog()
	if err != nil {
		t.Errorf("Got error, %v", err)
	}

	catalogBytes, _ := json.MarshalIndent(catalog, "", "  ")

	catalogJson := string(catalogBytes)

	klog.Info(catalogJson)
}
