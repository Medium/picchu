package utils

import (
	"context"
	"fmt"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

// UpdateStatus first tries new method of status update, and falls back to old.
func UpdateStatus(ctx context.Context, client client.Client, obj runtime.Object) error {
	return client.Status().Update(ctx, obj)
}

func MustGetKind(scheme *runtime.Scheme, obj runtime.Object) schema.GroupVersionKind {
	kinds, _, err := scheme.ObjectKinds(obj)
	if err != nil {
		panic(fmt.Sprintf("Failed to get kind for (%#v)", obj))
	}
	if len(kinds) <= 0 {
		panic("Assertion failed!")
	}
	return kinds[0]
}
