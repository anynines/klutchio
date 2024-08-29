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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"

	v1alpha1 "github.com/anynines/klutch/bind/pkg/client/clientset/versioned/typed/bind/v1alpha1"
)

type FakeKlutchBindV1alpha1 struct {
	*testing.Fake
}

func (c *FakeKlutchBindV1alpha1) APIServiceBindings() v1alpha1.APIServiceBindingInterface {
	return &FakeAPIServiceBindings{c}
}

func (c *FakeKlutchBindV1alpha1) APIServiceExports(namespace string) v1alpha1.APIServiceExportInterface {
	return &FakeAPIServiceExports{c, namespace}
}

func (c *FakeKlutchBindV1alpha1) APIServiceExportRequests(namespace string) v1alpha1.APIServiceExportRequestInterface {
	return &FakeAPIServiceExportRequests{c, namespace}
}

func (c *FakeKlutchBindV1alpha1) APIServiceNamespaces(namespace string) v1alpha1.APIServiceNamespaceInterface {
	return &FakeAPIServiceNamespaces{c, namespace}
}

func (c *FakeKlutchBindV1alpha1) ClusterBindings(namespace string) v1alpha1.ClusterBindingInterface {
	return &FakeClusterBindings{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeKlutchBindV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}