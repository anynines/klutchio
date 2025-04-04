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

package v1alpha1

import (
	"context"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"

	v1alpha1 "github.com/anynines/klutchio/bind/pkg/apis/bind/v1alpha1"
	scheme "github.com/anynines/klutchio/bind/pkg/client/clientset/versioned/scheme"
)

// APIServiceBindingsGetter has a method to return a APIServiceBindingInterface.
// A group's client should implement this interface.
type APIServiceBindingsGetter interface {
	APIServiceBindings() APIServiceBindingInterface
}

// APIServiceBindingInterface has methods to work with APIServiceBinding resources.
type APIServiceBindingInterface interface {
	Create(ctx context.Context, aPIServiceBinding *v1alpha1.APIServiceBinding, opts v1.CreateOptions) (*v1alpha1.APIServiceBinding, error)
	Update(ctx context.Context, aPIServiceBinding *v1alpha1.APIServiceBinding, opts v1.UpdateOptions) (*v1alpha1.APIServiceBinding, error)
	UpdateStatus(ctx context.Context, aPIServiceBinding *v1alpha1.APIServiceBinding, opts v1.UpdateOptions) (*v1alpha1.APIServiceBinding, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.APIServiceBinding, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.APIServiceBindingList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.APIServiceBinding, err error)
	APIServiceBindingExpansion
}

// aPIServiceBindings implements APIServiceBindingInterface
type aPIServiceBindings struct {
	client rest.Interface
}

// newAPIServiceBindings returns a APIServiceBindings
func newAPIServiceBindings(c *KlutchBindV1alpha1Client) *aPIServiceBindings {
	return &aPIServiceBindings{
		client: c.RESTClient(),
	}
}

// Get takes name of the aPIServiceBinding, and returns the corresponding aPIServiceBinding object, and an error if there is any.
func (c *aPIServiceBindings) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.APIServiceBinding, err error) {
	result = &v1alpha1.APIServiceBinding{}
	err = c.client.Get().
		Resource("apiservicebindings").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of APIServiceBindings that match those selectors.
func (c *aPIServiceBindings) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.APIServiceBindingList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.APIServiceBindingList{}
	err = c.client.Get().
		Resource("apiservicebindings").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested aPIServiceBindings.
func (c *aPIServiceBindings) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("apiservicebindings").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a aPIServiceBinding and creates it.  Returns the server's representation of the aPIServiceBinding, and an error, if there is any.
func (c *aPIServiceBindings) Create(ctx context.Context, aPIServiceBinding *v1alpha1.APIServiceBinding, opts v1.CreateOptions) (result *v1alpha1.APIServiceBinding, err error) {
	result = &v1alpha1.APIServiceBinding{}
	err = c.client.Post().
		Resource("apiservicebindings").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(aPIServiceBinding).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a aPIServiceBinding and updates it. Returns the server's representation of the aPIServiceBinding, and an error, if there is any.
func (c *aPIServiceBindings) Update(ctx context.Context, aPIServiceBinding *v1alpha1.APIServiceBinding, opts v1.UpdateOptions) (result *v1alpha1.APIServiceBinding, err error) {
	result = &v1alpha1.APIServiceBinding{}
	err = c.client.Put().
		Resource("apiservicebindings").
		Name(aPIServiceBinding.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(aPIServiceBinding).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *aPIServiceBindings) UpdateStatus(ctx context.Context, aPIServiceBinding *v1alpha1.APIServiceBinding, opts v1.UpdateOptions) (result *v1alpha1.APIServiceBinding, err error) {
	result = &v1alpha1.APIServiceBinding{}
	err = c.client.Put().
		Resource("apiservicebindings").
		Name(aPIServiceBinding.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(aPIServiceBinding).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the aPIServiceBinding and deletes it. Returns an error if one occurs.
func (c *aPIServiceBindings) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("apiservicebindings").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *aPIServiceBindings) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("apiservicebindings").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched aPIServiceBinding.
func (c *aPIServiceBindings) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.APIServiceBinding, err error) {
	result = &v1alpha1.APIServiceBinding{}
	err = c.client.Patch(pt).
		Resource("apiservicebindings").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
