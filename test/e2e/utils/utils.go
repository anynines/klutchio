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
	"fmt"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cr "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// ResourceGetter retrieves arbitrary k8s objects
type ResourceGetter struct {
	ctx    context.Context
	t      *testing.T
	config *envconf.Config
}

// NewResourceGetter returns a new getter that can be used for retrieving k8s objects of arbitrary type
func NewResourceGetter(ctx context.Context, t *testing.T, config *envconf.Config) *ResourceGetter {
	return &ResourceGetter{
		ctx:    ctx,
		t:      t,
		config: config,
	}
}

// Get returns k8s object for the given name, namespace, apiVersion and kind
// if the requested object does not exist, the test fails
func (r *ResourceGetter) Get(name string, namespace string, apiVersion string, kind string) *unstructured.Unstructured {
	client := r.config.Client().Resources().GetControllerRuntimeClient()
	u := &unstructured.Unstructured{}
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		r.t.Fatal(err)
	}
	u.SetGroupVersionKind(gv.WithKind(kind))
	if err := client.Get(r.ctx, cr.ObjectKey{Name: name, Namespace: namespace}, u); err != nil {
		r.t.Fatal("cannot get claim", err)
	}
	return u
}

// GetWithManaged returns a k8s object (claim) for the given name, namespace, apiVersion and kind,
// together with all it's managed resources.
func (r *ResourceGetter) GetWithManaged(name, namespace, apiVersion, kind string) (claim *unstructured.Unstructured, mrs *unstructured.UnstructuredList) {
	claim = r.Get(name, namespace, apiVersion, kind)
	mrs = &unstructured.UnstructuredList{}
	ref := ResourceValue(r.t, claim, "spec", "resourceRef")
	xr := r.Get(ref["name"], "", ref["apiVersion"], ref["kind"])
	mrefs := ResourceSliceValue(r.t, xr, "spec", "resourceRefs")
	for _, mref := range mrefs {
		mr := r.Get(mref["name"], "", mref["apiVersion"], mref["kind"])
		mrs.Items = append(mrs.Items, *mr)
	}
	return
}

// ResourceValue returns the value of field on the given path
// in case of error or if the field does not exist, the test fails
func ResourceValue(t *testing.T, u *unstructured.Unstructured, path ...string) map[string]string {
	f, found, err := unstructured.NestedStringMap(u.Object, path...)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatalf("field not found at path %v", path)
	}
	return f
}

// ResourceStringValue returns the value of field on the given path
// in case of error or if the field does not exist, the test fails
func ResourceStringValue(t *testing.T, u *unstructured.Unstructured, path ...string) string {
	f, found, err := unstructured.NestedString(u.Object, path...)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatalf("field not found at path %v", path)
	}
	return f
}

// ResourceSliceValue returns the slice value of field on the given path
// in case of error or if the field does not exist, the test fails
func ResourceSliceValue(t *testing.T, u *unstructured.Unstructured, path ...string) []map[string]string {
	f, found, err := unstructured.NestedSlice(u.Object, path...)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatalf("field not found at path %v", path)
	}
	var s []map[string]string
	for _, v := range f {
		if vv, ok := v.(map[string]interface{}); ok {
			s = append(s, asMapOfStrings(vv))
		} else {
			t.Fatalf("not a map[string]string: %v type %s", v, reflect.TypeOf(v))
		}
	}
	return s
}

func asMapOfStrings(m map[string]interface{}) map[string]string {
	r := make(map[string]string)
	for k, v := range m {
		r[k] = fmt.Sprintf("%v", v)
	}
	return r
}

// HaveSameFields compares two maps to check if they have the same set of fields.
func HaveSameFields(map1, map2 map[string]interface{}) bool {
	if len(map1) != len(map2) {
		return false
	}
	for key := range map1 {
		if _, exists := map2[key]; !exists {
			return false
		}
	}
	return true
}
