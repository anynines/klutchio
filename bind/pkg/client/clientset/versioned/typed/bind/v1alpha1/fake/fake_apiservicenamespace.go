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
	"context"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"

	v1alpha1 "github.com/anynines/klutchio/bind/pkg/apis/bind/v1alpha1"
)

// FakeAPIServiceNamespaces implements APIServiceNamespaceInterface
type FakeAPIServiceNamespaces struct {
	Fake *FakeKlutchBindV1alpha1
	ns   string
}

var apiservicenamespacesResource = schema.GroupVersionResource{Group: "klutch.anynines.com", Version: "v1alpha1", Resource: "apiservicenamespaces"}

var apiservicenamespacesKind = schema.GroupVersionKind{Group: "klutch.anynines.com", Version: "v1alpha1", Kind: "APIServiceNamespace"}

// Get takes name of the aPIServiceNamespace, and returns the corresponding aPIServiceNamespace object, and an error if there is any.
func (c *FakeAPIServiceNamespaces) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.APIServiceNamespace, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(apiservicenamespacesResource, c.ns, name), &v1alpha1.APIServiceNamespace{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.APIServiceNamespace), err
}

// List takes label and field selectors, and returns the list of APIServiceNamespaces that match those selectors.
func (c *FakeAPIServiceNamespaces) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.APIServiceNamespaceList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(apiservicenamespacesResource, apiservicenamespacesKind, c.ns, opts), &v1alpha1.APIServiceNamespaceList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.APIServiceNamespaceList{ListMeta: obj.(*v1alpha1.APIServiceNamespaceList).ListMeta}
	for _, item := range obj.(*v1alpha1.APIServiceNamespaceList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested aPIServiceNamespaces.
func (c *FakeAPIServiceNamespaces) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(apiservicenamespacesResource, c.ns, opts))

}

// Create takes the representation of a aPIServiceNamespace and creates it.  Returns the server's representation of the aPIServiceNamespace, and an error, if there is any.
func (c *FakeAPIServiceNamespaces) Create(ctx context.Context, aPIServiceNamespace *v1alpha1.APIServiceNamespace, opts v1.CreateOptions) (result *v1alpha1.APIServiceNamespace, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(apiservicenamespacesResource, c.ns, aPIServiceNamespace), &v1alpha1.APIServiceNamespace{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.APIServiceNamespace), err
}

// Update takes the representation of a aPIServiceNamespace and updates it. Returns the server's representation of the aPIServiceNamespace, and an error, if there is any.
func (c *FakeAPIServiceNamespaces) Update(ctx context.Context, aPIServiceNamespace *v1alpha1.APIServiceNamespace, opts v1.UpdateOptions) (result *v1alpha1.APIServiceNamespace, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(apiservicenamespacesResource, c.ns, aPIServiceNamespace), &v1alpha1.APIServiceNamespace{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.APIServiceNamespace), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeAPIServiceNamespaces) UpdateStatus(ctx context.Context, aPIServiceNamespace *v1alpha1.APIServiceNamespace, opts v1.UpdateOptions) (*v1alpha1.APIServiceNamespace, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(apiservicenamespacesResource, "status", c.ns, aPIServiceNamespace), &v1alpha1.APIServiceNamespace{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.APIServiceNamespace), err
}

// Delete takes name of the aPIServiceNamespace and deletes it. Returns an error if one occurs.
func (c *FakeAPIServiceNamespaces) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(apiservicenamespacesResource, c.ns, name, opts), &v1alpha1.APIServiceNamespace{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeAPIServiceNamespaces) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(apiservicenamespacesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.APIServiceNamespaceList{})
	return err
}

// Patch applies the patch and returns the patched aPIServiceNamespace.
func (c *FakeAPIServiceNamespaces) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.APIServiceNamespace, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(apiservicenamespacesResource, c.ns, name, pt, data, subresources...), &v1alpha1.APIServiceNamespace{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.APIServiceNamespace), err
}
