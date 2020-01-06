// Copyright © 2019 A Medium Corporation.
// Licensed under the Apache License, Version 2.0; see the NOTICE file.

// Code generated by client. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/client/scheme"
	rest "k8s.io/client-go/rest"
)

type PicchuV1alpha1Interface interface {
	RESTClient() rest.Interface
	ClustersGetter
	ClusterSecretsesGetter
	MirrorsGetter
	ReleaseManagersGetter
	RevisionsGetter
}

// PicchuV1alpha1Client is used to interact with features provided by the picchu.medium.engineering group.
type PicchuV1alpha1Client struct {
	restClient rest.Interface
}

func (c *PicchuV1alpha1Client) Clusters(namespace string) ClusterInterface {
	return newClusters(c, namespace)
}

func (c *PicchuV1alpha1Client) ClusterSecretses(namespace string) ClusterSecretsInterface {
	return newClusterSecretses(c, namespace)
}

func (c *PicchuV1alpha1Client) Mirrors(namespace string) MirrorInterface {
	return newMirrors(c, namespace)
}

func (c *PicchuV1alpha1Client) ReleaseManagers(namespace string) ReleaseManagerInterface {
	return newReleaseManagers(c, namespace)
}

func (c *PicchuV1alpha1Client) Revisions(namespace string) RevisionInterface {
	return newRevisions(c, namespace)
}

// NewForConfig creates a new PicchuV1alpha1Client for the given config.
func NewForConfig(c *rest.Config) (*PicchuV1alpha1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &PicchuV1alpha1Client{client}, nil
}

// NewForConfigOrDie creates a new PicchuV1alpha1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *PicchuV1alpha1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new PicchuV1alpha1Client for the given RESTClient.
func New(c rest.Interface) *PicchuV1alpha1Client {
	return &PicchuV1alpha1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1alpha1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *PicchuV1alpha1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
