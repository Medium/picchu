// Copyright © 2019 A Medium Corporation.
// Licensed under the Apache License, Version 2.0; see the NOTICE file.

// Code generated by client. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeClusterSecretses implements ClusterSecretsInterface
type FakeClusterSecretses struct {
	Fake *FakePicchuV1alpha1
	ns   string
}

var clustersecretsesResource = schema.GroupVersionResource{Group: "picchu.medium.engineering", Version: "v1alpha1", Resource: "clustersecretses"}

var clustersecretsesKind = schema.GroupVersionKind{Group: "picchu.medium.engineering", Version: "v1alpha1", Kind: "ClusterSecrets"}

// Get takes name of the clusterSecrets, and returns the corresponding clusterSecrets object, and an error if there is any.
func (c *FakeClusterSecretses) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.ClusterSecrets, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(clustersecretsesResource, c.ns, name), &v1alpha1.ClusterSecrets{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ClusterSecrets), err
}

// List takes label and field selectors, and returns the list of ClusterSecretses that match those selectors.
func (c *FakeClusterSecretses) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.ClusterSecretsList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(clustersecretsesResource, clustersecretsesKind, c.ns, opts), &v1alpha1.ClusterSecretsList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.ClusterSecretsList{ListMeta: obj.(*v1alpha1.ClusterSecretsList).ListMeta}
	for _, item := range obj.(*v1alpha1.ClusterSecretsList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested clusterSecretses.
func (c *FakeClusterSecretses) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(clustersecretsesResource, c.ns, opts))

}

// Create takes the representation of a clusterSecrets and creates it.  Returns the server's representation of the clusterSecrets, and an error, if there is any.
func (c *FakeClusterSecretses) Create(ctx context.Context, clusterSecrets *v1alpha1.ClusterSecrets, opts v1.CreateOptions) (result *v1alpha1.ClusterSecrets, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(clustersecretsesResource, c.ns, clusterSecrets), &v1alpha1.ClusterSecrets{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ClusterSecrets), err
}

// Update takes the representation of a clusterSecrets and updates it. Returns the server's representation of the clusterSecrets, and an error, if there is any.
func (c *FakeClusterSecretses) Update(ctx context.Context, clusterSecrets *v1alpha1.ClusterSecrets, opts v1.UpdateOptions) (result *v1alpha1.ClusterSecrets, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(clustersecretsesResource, c.ns, clusterSecrets), &v1alpha1.ClusterSecrets{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ClusterSecrets), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeClusterSecretses) UpdateStatus(ctx context.Context, clusterSecrets *v1alpha1.ClusterSecrets, opts v1.UpdateOptions) (*v1alpha1.ClusterSecrets, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(clustersecretsesResource, "status", c.ns, clusterSecrets), &v1alpha1.ClusterSecrets{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ClusterSecrets), err
}

// Delete takes name of the clusterSecrets and deletes it. Returns an error if one occurs.
func (c *FakeClusterSecretses) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(clustersecretsesResource, c.ns, name), &v1alpha1.ClusterSecrets{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeClusterSecretses) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(clustersecretsesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.ClusterSecretsList{})
	return err
}

// Patch applies the patch and returns the patched clusterSecrets.
func (c *FakeClusterSecretses) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ClusterSecrets, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(clustersecretsesResource, c.ns, name, pt, data, subresources...), &v1alpha1.ClusterSecrets{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ClusterSecrets), err
}
