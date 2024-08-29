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
	"testing"
)

func testForDuplicates(list []string, t *testing.T) {
	uuids := map[string]string{}
	for _, name := range list {
		uuid := IDFrom(name)
		if uuids[uuid] != "" {
			t.Fatalf("Found a collision with %s and %s", uuids[uuid], name)
		} else {
			uuids[uuid] = name
		}
	}
}

func TestClassUUID(t *testing.T) {
	g := &Generator{}
	AssignPoolGoT(g)

	testForDuplicates(g.ClassPool, t)
}

func TestPlanUUID(t *testing.T) {
	g := &Generator{}
	AssignPoolGoT(g)

	testForDuplicates(g.PlanPool, t)
}
