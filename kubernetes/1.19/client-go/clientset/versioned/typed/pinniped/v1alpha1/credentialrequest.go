/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/suzerain-io/pinniped/kubernetes/1.19/api/apis/pinniped/v1alpha1"
	scheme "github.com/suzerain-io/pinniped/kubernetes/1.19/client-go/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// CredentialRequestsGetter has a method to return a CredentialRequestInterface.
// A group's client should implement this interface.
type CredentialRequestsGetter interface {
	CredentialRequests() CredentialRequestInterface
}

// CredentialRequestInterface has methods to work with CredentialRequest resources.
type CredentialRequestInterface interface {
	Create(ctx context.Context, credentialRequest *v1alpha1.CredentialRequest, opts v1.CreateOptions) (*v1alpha1.CredentialRequest, error)
	Update(ctx context.Context, credentialRequest *v1alpha1.CredentialRequest, opts v1.UpdateOptions) (*v1alpha1.CredentialRequest, error)
	UpdateStatus(ctx context.Context, credentialRequest *v1alpha1.CredentialRequest, opts v1.UpdateOptions) (*v1alpha1.CredentialRequest, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.CredentialRequest, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.CredentialRequestList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.CredentialRequest, err error)
	CredentialRequestExpansion
}

// credentialRequests implements CredentialRequestInterface
type credentialRequests struct {
	client rest.Interface
}

// newCredentialRequests returns a CredentialRequests
func newCredentialRequests(c *PinnipedV1alpha1Client) *credentialRequests {
	return &credentialRequests{
		client: c.RESTClient(),
	}
}

// Get takes name of the credentialRequest, and returns the corresponding credentialRequest object, and an error if there is any.
func (c *credentialRequests) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.CredentialRequest, err error) {
	result = &v1alpha1.CredentialRequest{}
	err = c.client.Get().
		Resource("credentialrequests").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of CredentialRequests that match those selectors.
func (c *credentialRequests) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.CredentialRequestList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.CredentialRequestList{}
	err = c.client.Get().
		Resource("credentialrequests").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested credentialRequests.
func (c *credentialRequests) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("credentialrequests").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a credentialRequest and creates it.  Returns the server's representation of the credentialRequest, and an error, if there is any.
func (c *credentialRequests) Create(ctx context.Context, credentialRequest *v1alpha1.CredentialRequest, opts v1.CreateOptions) (result *v1alpha1.CredentialRequest, err error) {
	result = &v1alpha1.CredentialRequest{}
	err = c.client.Post().
		Resource("credentialrequests").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(credentialRequest).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a credentialRequest and updates it. Returns the server's representation of the credentialRequest, and an error, if there is any.
func (c *credentialRequests) Update(ctx context.Context, credentialRequest *v1alpha1.CredentialRequest, opts v1.UpdateOptions) (result *v1alpha1.CredentialRequest, err error) {
	result = &v1alpha1.CredentialRequest{}
	err = c.client.Put().
		Resource("credentialrequests").
		Name(credentialRequest.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(credentialRequest).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *credentialRequests) UpdateStatus(ctx context.Context, credentialRequest *v1alpha1.CredentialRequest, opts v1.UpdateOptions) (result *v1alpha1.CredentialRequest, err error) {
	result = &v1alpha1.CredentialRequest{}
	err = c.client.Put().
		Resource("credentialrequests").
		Name(credentialRequest.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(credentialRequest).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the credentialRequest and deletes it. Returns an error if one occurs.
func (c *credentialRequests) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("credentialrequests").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *credentialRequests) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("credentialrequests").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched credentialRequest.
func (c *credentialRequests) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.CredentialRequest, err error) {
	result = &v1alpha1.CredentialRequest{}
	err = c.client.Patch(pt).
		Resource("credentialrequests").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
