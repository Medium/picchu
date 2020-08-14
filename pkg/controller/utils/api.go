package utils

import (
	"context"
	"fmt"
	"go.medium.engineering/picchu/pkg/client/scheme"
	"sync"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var cache = map[client.ObjectKey]client.Client{}
var lock = &sync.Mutex{}

func RemoteClient(ctx context.Context, log logr.Logger, reader client.Reader, cluster *picchuv1alpha1.Cluster) (client.Client, error) {
	key, err := client.ObjectKeyFromObject(cluster)
	if err != nil {
		return nil, err
	}
	if client, ok := cache[key]; ok {
		return client, nil
	}
	log.Info("Initializing remote client", "Cluster", cluster.Name)
	secret := &corev1.Secret{}
	if err = reader.Get(ctx, key, secret); err != nil {
		return nil, err
	}
	config, err := cluster.Config(secret)
	if err != nil {
		return nil, err
	}
	if config == nil {
		err := fmt.Errorf("Failed to create k8s client config")
		log.Error(err, "cluster config nil", "Cluster", cluster.Name)
		return nil, err
	}
	cli, err := client.New(config, client.Options{})
	if err != nil {
		return cli, err
	}
	lock.Lock()
	defer lock.Unlock()
	cache[key] = cli
	return cli, nil
}

// UpdateStatus first tries new method of status update, and falls back to old.
func UpdateStatus(ctx context.Context, client client.Client, obj runtime.Object) error {
	return client.Status().Update(ctx, obj)
}

func MustGetKind(obj runtime.Object) schema.GroupVersionKind {
	kinds, _, err := scheme.Scheme.ObjectKinds(obj)
	if err != nil {
		fmt.Printf("Failed to get kind for (%#v)\n", obj)
		panic(err)
	}
	if len(kinds) <= 0 {
		panic("Assertion failed!")
	}
	return kinds[0]
}

func DeleteIfExists(ctx context.Context, cli client.Client, obj runtime.Object) error {
	err := cli.Delete(ctx, obj)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

// MustExtractList panics if the object isn't a list
func MustExtractList(obj runtime.Object) []runtime.Object {
	list, err := meta.ExtractList(obj)
	if err != nil {
		panic(err)
	}
	return list
}
