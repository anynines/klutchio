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

	v1alpha1 "github.com/anynines/klutchio/bind/contrib/example-backend/apis/examplebackend/v1alpha1"
)

// FakeAPIServiceExportTemplates implements APIServiceExportTemplateInterface
type FakeAPIServiceExportTemplates struct {
	Fake *FakeExampleBackendV1alpha1
	ns   string
}

var apiserviceexporttemplatesResource = schema.GroupVersionResource{Group: "example-backend.klutch.anynines.com", Version: "v1alpha1", Resource: "apiserviceexporttemplates"}

var apiserviceexporttemplatesKind = schema.GroupVersionKind{Group: "example-backend.klutch.anynines.com", Version: "v1alpha1", Kind: "APIServiceExportTemplate"}

// Get takes name of the aPIServiceExportTemplate, and returns the corresponding aPIServiceExportTemplate object, and an error if there is any.
func (c *FakeAPIServiceExportTemplates) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.APIServiceExportTemplate, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(apiserviceexporttemplatesResource, c.ns, name), &v1alpha1.APIServiceExportTemplate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.APIServiceExportTemplate), err
}

// List takes label and field selectors, and returns the list of APIServiceExportTemplates that match those selectors.
func (c *FakeAPIServiceExportTemplates) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.APIServiceExportTemplateList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(apiserviceexporttemplatesResource, apiserviceexporttemplatesKind, c.ns, opts), &v1alpha1.APIServiceExportTemplateList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.APIServiceExportTemplateList{ListMeta: obj.(*v1alpha1.APIServiceExportTemplateList).ListMeta}
	for _, item := range obj.(*v1alpha1.APIServiceExportTemplateList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested aPIServiceExportTemplates.
func (c *FakeAPIServiceExportTemplates) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(apiserviceexporttemplatesResource, c.ns, opts))

}

// Create takes the representation of a aPIServiceExportTemplate and creates it.  Returns the server's representation of the aPIServiceExportTemplate, and an error, if there is any.
func (c *FakeAPIServiceExportTemplates) Create(ctx context.Context, aPIServiceExportTemplate *v1alpha1.APIServiceExportTemplate, opts v1.CreateOptions) (result *v1alpha1.APIServiceExportTemplate, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(apiserviceexporttemplatesResource, c.ns, aPIServiceExportTemplate), &v1alpha1.APIServiceExportTemplate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.APIServiceExportTemplate), err
}

// Update takes the representation of a aPIServiceExportTemplate and updates it. Returns the server's representation of the aPIServiceExportTemplate, and an error, if there is any.
func (c *FakeAPIServiceExportTemplates) Update(ctx context.Context, aPIServiceExportTemplate *v1alpha1.APIServiceExportTemplate, opts v1.UpdateOptions) (result *v1alpha1.APIServiceExportTemplate, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(apiserviceexporttemplatesResource, c.ns, aPIServiceExportTemplate), &v1alpha1.APIServiceExportTemplate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.APIServiceExportTemplate), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeAPIServiceExportTemplates) UpdateStatus(ctx context.Context, aPIServiceExportTemplate *v1alpha1.APIServiceExportTemplate, opts v1.UpdateOptions) (*v1alpha1.APIServiceExportTemplate, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(apiserviceexporttemplatesResource, "status", c.ns, aPIServiceExportTemplate), &v1alpha1.APIServiceExportTemplate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.APIServiceExportTemplate), err
}

// Delete takes name of the aPIServiceExportTemplate and deletes it. Returns an error if one occurs.
func (c *FakeAPIServiceExportTemplates) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(apiserviceexporttemplatesResource, c.ns, name, opts), &v1alpha1.APIServiceExportTemplate{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeAPIServiceExportTemplates) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(apiserviceexporttemplatesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.APIServiceExportTemplateList{})
	return err
}

// Patch applies the patch and returns the patched aPIServiceExportTemplate.
func (c *FakeAPIServiceExportTemplates) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.APIServiceExportTemplate, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(apiserviceexporttemplatesResource, c.ns, name, pt, data, subresources...), &v1alpha1.APIServiceExportTemplate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.APIServiceExportTemplate), err
}
