// Copyright © 2019 A Medium Corporation.
// Licensed under the Apache License, Version 2.0; see the NOTICE file.

// Code generated by client. DO NOT EDIT.

package fake

import (
	v1alpha1 "go.medium.engineering/picchu/client/typed/picchu/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakePicchuV1alpha1 struct {
	*testing.Fake
}

func (c *FakePicchuV1alpha1) Clusters(namespace string) v1alpha1.ClusterInterface {
	return &FakeClusters{c, namespace}
}

func (c *FakePicchuV1alpha1) ClusterSecretses(namespace string) v1alpha1.ClusterSecretsInterface {
	return &FakeClusterSecretses{c, namespace}
}

func (c *FakePicchuV1alpha1) FaultInjectors(namespace string) v1alpha1.FaultInjectorInterface {
	return &FakeFaultInjectors{c, namespace}
}

func (c *FakePicchuV1alpha1) Mirrors(namespace string) v1alpha1.MirrorInterface {
	return &FakeMirrors{c, namespace}
}

func (c *FakePicchuV1alpha1) ReleaseManagers(namespace string) v1alpha1.ReleaseManagerInterface {
	return &FakeReleaseManagers{c, namespace}
}

func (c *FakePicchuV1alpha1) Revisions(namespace string) v1alpha1.RevisionInterface {
	return &FakeRevisions{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakePicchuV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
