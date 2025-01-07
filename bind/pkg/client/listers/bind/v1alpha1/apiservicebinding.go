/*
Copyright The Kube Bind Authors.

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

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	v1alpha1 "github.com/anynines/klutchio/bind/pkg/apis/bind/v1alpha1"
)

// APIServiceBindingLister helps list APIServiceBindings.
// All objects returned here must be treated as read-only.
type APIServiceBindingLister interface {
	// List lists all APIServiceBindings in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.APIServiceBinding, err error)
	// Get retrieves the APIServiceBinding from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.APIServiceBinding, error)
	APIServiceBindingListerExpansion
}

// aPIServiceBindingLister implements the APIServiceBindingLister interface.
type aPIServiceBindingLister struct {
	indexer cache.Indexer
}

// NewAPIServiceBindingLister returns a new APIServiceBindingLister.
func NewAPIServiceBindingLister(indexer cache.Indexer) APIServiceBindingLister {
	return &aPIServiceBindingLister{indexer: indexer}
}

// List lists all APIServiceBindings in the indexer.
func (s *aPIServiceBindingLister) List(selector labels.Selector) (ret []*v1alpha1.APIServiceBinding, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.APIServiceBinding))
	})
	return ret, err
}

// Get retrieves the APIServiceBinding from the index for a given name.
func (s *aPIServiceBindingLister) Get(name string) (*v1alpha1.APIServiceBinding, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("apiservicebinding"), name)
	}
	return obj.(*v1alpha1.APIServiceBinding), nil
}
