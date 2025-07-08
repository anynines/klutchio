/*
Copyright 2023 The Kube Bind Authors.

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

package adopt

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/anynines/klutchio/bind/pkg/apis/bind/v1alpha1"
)

func TestAnnotationsAdded(t *testing.T) {

	var consumerUpdate, providerUpdate *unstructured.Unstructured
	x := reconciler{
		getConsumerObject: func(ns, name string) (*unstructured.Unstructured, error) {
			obj := unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name":      "dummy",
						"namespace": "default",
					},
				},
			}
			return &obj, nil
		},
		getServiceNamespace: defaultNamespace,
		updateConsumerObject: func(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			consumerUpdate = obj
			return obj, nil
		},
		updateProviderObject: func(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			providerUpdate = obj
			return obj, nil
		},
	}

	err := x.reconcile(context.Background(), &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "dummy",
				"namespace": "cluster-x-default",
			},
		}})

	require.NoError(t, err, "reconcile should succeed")
	assertAnnotation(t, consumerUpdate, "klutch.anynines.com/bound", "true")
	assertAnnotation(t, providerUpdate, "klutch.anynines.com/bound", "true")
}

func TestCreate(t *testing.T) {
	var createdObj *unstructured.Unstructured
	x := reconciler{
		getConsumerObject: func(ns, name string) (*unstructured.Unstructured, error) {
			return nil, errors.NewNotFound(v1alpha1.Resource("MangoDB"), name)
		},
		getServiceNamespace: defaultNamespace,

		updateProviderObject: func(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			return obj, nil
		},
		createConsumerObject: func(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			createdObj = obj
			return obj, nil
		},
	}

	err := x.reconcile(context.Background(), &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "dummy",
				"namespace": "cluster-x-default",
			},
		}})

	assertAnnotation(t, createdObj, "klutch.anynines.com/bound", "true")
	require.NoError(t, err, "reconcile should succeed")
}

func assertAnnotation(t *testing.T, obj *unstructured.Unstructured, key, value string) {
	t.Helper()

	if obj == nil {
		t.Fatal("obj is nil")
	}

	ann := obj.GetAnnotations()
	if ann == nil {
		t.Fatal("annotations are nil")
	}
	if v, ok := ann[key]; !ok || v != value {
		t.Fatalf("expected %v but got %v", value, v)
	}
}

func defaultNamespace(upstreamNamespace string) (*v1alpha1.APIServiceNamespace, error) {
	return &v1alpha1.APIServiceNamespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "kube-bind",
		},
		Spec: v1alpha1.APIServiceNamespaceSpec{},
		Status: v1alpha1.APIServiceNamespaceStatus{
			Namespace: "cluster-x-default",
		},
	}, nil
}
