// Copyright © 2019 A Medium Corporation.
// Licensed under the Apache License, Version 2.0; see the NOTICE file.

// Code generated by client. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeFaultInjectors implements FaultInjectorInterface
type FakeFaultInjectors struct {
	Fake *FakePicchuV1alpha1
	ns   string
}

var faultinjectorsResource = schema.GroupVersionResource{Group: "picchu.medium.engineering", Version: "v1alpha1", Resource: "faultinjectors"}

var faultinjectorsKind = schema.GroupVersionKind{Group: "picchu.medium.engineering", Version: "v1alpha1", Kind: "FaultInjector"}

// Get takes name of the faultInjector, and returns the corresponding faultInjector object, and an error if there is any.
func (c *FakeFaultInjectors) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.FaultInjector, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(faultinjectorsResource, c.ns, name), &v1alpha1.FaultInjector{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FaultInjector), err
}

// List takes label and field selectors, and returns the list of FaultInjectors that match those selectors.
func (c *FakeFaultInjectors) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.FaultInjectorList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(faultinjectorsResource, faultinjectorsKind, c.ns, opts), &v1alpha1.FaultInjectorList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.FaultInjectorList{ListMeta: obj.(*v1alpha1.FaultInjectorList).ListMeta}
	for _, item := range obj.(*v1alpha1.FaultInjectorList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested faultInjectors.
func (c *FakeFaultInjectors) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(faultinjectorsResource, c.ns, opts))

}

// Create takes the representation of a faultInjector and creates it.  Returns the server's representation of the faultInjector, and an error, if there is any.
func (c *FakeFaultInjectors) Create(ctx context.Context, faultInjector *v1alpha1.FaultInjector, opts v1.CreateOptions) (result *v1alpha1.FaultInjector, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(faultinjectorsResource, c.ns, faultInjector), &v1alpha1.FaultInjector{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FaultInjector), err
}

// Update takes the representation of a faultInjector and updates it. Returns the server's representation of the faultInjector, and an error, if there is any.
func (c *FakeFaultInjectors) Update(ctx context.Context, faultInjector *v1alpha1.FaultInjector, opts v1.UpdateOptions) (result *v1alpha1.FaultInjector, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(faultinjectorsResource, c.ns, faultInjector), &v1alpha1.FaultInjector{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FaultInjector), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeFaultInjectors) UpdateStatus(ctx context.Context, faultInjector *v1alpha1.FaultInjector, opts v1.UpdateOptions) (*v1alpha1.FaultInjector, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(faultinjectorsResource, "status", c.ns, faultInjector), &v1alpha1.FaultInjector{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FaultInjector), err
}

// Delete takes name of the faultInjector and deletes it. Returns an error if one occurs.
func (c *FakeFaultInjectors) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(faultinjectorsResource, c.ns, name), &v1alpha1.FaultInjector{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeFaultInjectors) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(faultinjectorsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.FaultInjectorList{})
	return err
}

// Patch applies the patch and returns the patched faultInjector.
func (c *FakeFaultInjectors) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.FaultInjector, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(faultinjectorsResource, c.ns, name, pt, data, subresources...), &v1alpha1.FaultInjector{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FaultInjector), err
}
