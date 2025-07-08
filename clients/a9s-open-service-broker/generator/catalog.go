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
	"fmt"
	"math/rand"
	"sort"

	"k8s.io/klog/v2"

	v2 "github.com/anynines/klutchio/clients/a9s-open-service-broker"
)

// GetCatalog will produce a valid GetCatalog response based on the generator settings.
func (g *Generator) GetCatalog() (*v2.CatalogResponse, error) {
	if len(g.Services) == 0 {
		return nil, fmt.Errorf("no services defined")
	}

	services := make([]v2.Service, len(g.Services))

	for s, gs := range g.Services {
		services[s].Plans = make([]v2.Plan, len(gs.Plans))
		service := &services[s]
		service.Name = g.ClassPool[s+g.ClassPoolOffset]
		service.Description = g.description(s)
		service.ID = IDFrom(g.ClassPool[s])
		service.DashboardClient = g.dashboardClient(service.Name)

		for property, count := range gs.FromPool {
			switch property {
			case Tags:
				service.Tags = g.tagNames(s, count)
			case Metadata:
				service.Metadata = g.metaNames(s, count)
			case Bindable:
				service.Bindable = count > 0
			case InstancesRetrievable:
				service.InstancesRetrievable = count > 0
			case BindingsRetrievable:
				service.BindingsRetrievable = count > 0
			case Requires:
				service.Requires = g.requiresNames(s, count)
			}
		}

		planNames := g.planNames(s, len(service.Plans))
		for p, gp := range gs.Plans {
			plan := &service.Plans[p]
			plan.Name = planNames[p]
			plan.Description = g.description(1000 + 1000*s*p)
			plan.ID = IDFrom(planNames[p])

			for property, count := range gp.FromPool {
				switch property {
				case Metadata:
					plan.Metadata = g.metaNames(1000+1000*s*p, count)
				case Free:
					isFree := count > 0
					plan.Free = &isFree
				}
			}
		}
	}

	return &v2.CatalogResponse{
		Services: services,
	}, nil
}

func getSliceWithoutDuplicates(count int, seed int64, list []string) []string {

	if len(list) < count {
		klog.Error("not enough items in list")
		return []string{""}
	}

	rand.Seed(seed)

	set := map[string]int32{}

	// Get strings from list without duplicates
	for len(set) < count {
		x := rand.Int31n(int32(len(list)))
		set[list[x]] = x
	}

	keys := []string(nil)
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (g *Generator) description(seed int) string {
	return getSliceWithoutDuplicates(1, int64(seed), g.DescriptionPool)[0]
}

func (g *Generator) planNames(seed, count int) []string {
	return getSliceWithoutDuplicates(count, int64(seed), g.PlanPool)
}

func (g *Generator) tagNames(seed, count int) []string {
	return getSliceWithoutDuplicates(count, int64(seed*1000+1000), g.TagPool)
}

func (g *Generator) requiresNames(seed, count int) []string {
	return getSliceWithoutDuplicates(count, int64(seed*1000+2000), g.RequiresPool)
}

func (g *Generator) metaNames(seed, count int) map[string]interface{} {
	key := getSliceWithoutDuplicates(count, int64(seed*1000+3000), g.MetadataPool)
	value := getSliceWithoutDuplicates(count, int64(seed*3000+4000), g.MetadataPool)
	meta := make(map[string]interface{}, count)
	for i := 0; i < len(key); i++ {
		meta[key[i]] = value[i]
	}
	return meta
}

func (g *Generator) dashboardClient(name string) *v2.DashboardClient {
	return &v2.DashboardClient{
		ID:          IDFrom(fmt.Sprintf("%s%s", name, "id")),
		Secret:      IDFrom(fmt.Sprintf("%s%s", name, "secret")),
		RedirectURI: "http://localhost:1234",
	}
}
