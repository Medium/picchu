package releasemanager

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ConfigFetcher struct {
	Client client.Client
}

func (c *ConfigFetcher) GetSecrets(ctx context.Context, opts *client.ListOptions) (*corev1.SecretList, error) {
	secrets := &corev1.SecretList{}
	err := c.Client.List(ctx, opts, secrets)
	return secrets, err
}

func (c *ConfigFetcher) GetConfigMaps(ctx context.Context, opts *client.ListOptions) (*corev1.ConfigMapList, error) {
	configMaps := &corev1.ConfigMapList{}
	err := c.Client.List(ctx, opts, configMaps)
	return configMaps, err
}
