// Copyright © 2019 A Medium Corporation.
// Licensed under the Apache License, Version 2.0; see the NOTICE file.

// Code generated by client. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	scheme "go.medium.engineering/picchu/client/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// FaultInjectorsGetter has a method to return a FaultInjectorInterface.
// A group's client should implement this interface.
type FaultInjectorsGetter interface {
	FaultInjectors(namespace string) FaultInjectorInterface
}

// FaultInjectorInterface has methods to work with FaultInjector resources.
type FaultInjectorInterface interface {
	Create(ctx context.Context, faultInjector *v1alpha1.FaultInjector, opts v1.CreateOptions) (*v1alpha1.FaultInjector, error)
	Update(ctx context.Context, faultInjector *v1alpha1.FaultInjector, opts v1.UpdateOptions) (*v1alpha1.FaultInjector, error)
	UpdateStatus(ctx context.Context, faultInjector *v1alpha1.FaultInjector, opts v1.UpdateOptions) (*v1alpha1.FaultInjector, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.FaultInjector, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.FaultInjectorList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.FaultInjector, err error)
	FaultInjectorExpansion
}

// faultInjectors implements FaultInjectorInterface
type faultInjectors struct {
	client rest.Interface
	ns     string
}

// newFaultInjectors returns a FaultInjectors
func newFaultInjectors(c *PicchuV1alpha1Client, namespace string) *faultInjectors {
	return &faultInjectors{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the faultInjector, and returns the corresponding faultInjector object, and an error if there is any.
func (c *faultInjectors) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.FaultInjector, err error) {
	result = &v1alpha1.FaultInjector{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("faultinjectors").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of FaultInjectors that match those selectors.
func (c *faultInjectors) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.FaultInjectorList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.FaultInjectorList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("faultinjectors").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested faultInjectors.
func (c *faultInjectors) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("faultinjectors").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a faultInjector and creates it.  Returns the server's representation of the faultInjector, and an error, if there is any.
func (c *faultInjectors) Create(ctx context.Context, faultInjector *v1alpha1.FaultInjector, opts v1.CreateOptions) (result *v1alpha1.FaultInjector, err error) {
	result = &v1alpha1.FaultInjector{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("faultinjectors").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(faultInjector).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a faultInjector and updates it. Returns the server's representation of the faultInjector, and an error, if there is any.
func (c *faultInjectors) Update(ctx context.Context, faultInjector *v1alpha1.FaultInjector, opts v1.UpdateOptions) (result *v1alpha1.FaultInjector, err error) {
	result = &v1alpha1.FaultInjector{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("faultinjectors").
		Name(faultInjector.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(faultInjector).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *faultInjectors) UpdateStatus(ctx context.Context, faultInjector *v1alpha1.FaultInjector, opts v1.UpdateOptions) (result *v1alpha1.FaultInjector, err error) {
	result = &v1alpha1.FaultInjector{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("faultinjectors").
		Name(faultInjector.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(faultInjector).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the faultInjector and deletes it. Returns an error if one occurs.
func (c *faultInjectors) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("faultinjectors").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *faultInjectors) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("faultinjectors").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched faultInjector.
func (c *faultInjectors) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.FaultInjector, err error) {
	result = &v1alpha1.FaultInjector{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("faultinjectors").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
