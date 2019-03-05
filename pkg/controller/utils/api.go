package utils

import (
	"context"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RemoteClient(reader client.Reader, cluster *picchuv1alpha1.Cluster) (client.Client, error) {
	key, err := client.ObjectKeyFromObject(cluster)
	if err != nil {
		return nil, err
	}
	secret := &corev1.Secret{}
	if err = reader.Get(context.TODO(), key, secret); err != nil {
		return nil, err
	}
	config, err := cluster.Config(secret)
	if err != nil {
		return nil, err
	}
	if config != nil {
		return client.New(config, client.Options{})
	}
	return nil, nil
}
