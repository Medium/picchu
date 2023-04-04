package utils

import (
	"context"
	"fmt"
	"sync"

	"go.medium.engineering/picchu/client/scheme"

	errorsGen "errors"

	"github.com/go-logr/logr"
	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var cache = map[client.ObjectKey]client.Client{}
var lock = &sync.RWMutex{}

func checkCache(key client.ObjectKey) (client.Client, bool) {
	lock.RLock()
	defer lock.RUnlock()
	if client, ok := cache[client.ObjectKey{}]; ok {
		return client, true
	}
	return nil, false
}

// RemoteClient creates a k8s client from a cluster object.
func RemoteClient(ctx context.Context, log logr.Logger, reader client.Client, cluster *picchuv1alpha1.Cluster) (client.Client, error) {
	key := client.ObjectKeyFromObject(cluster)
	if key == (types.NamespacedName{}) {
		return nil, errorsGen.New("NamespacedName is empty")
	}
	if client, ok := checkCache(key); ok {
		return client, nil
	}
	secret := &corev1.Secret{}
	if err := reader.Get(ctx, key, secret); err != nil {
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
	cli, err := client.New(config, client.Options{Scheme: reader.Scheme()})
	if err != nil {
		return cli, err
	}
	lock.Lock()
	defer lock.Unlock()
	cache[key] = cli
	return cli, nil
}

// UpdateStatus first tries new method of status update, and falls back to old.
func UpdateStatus(ctx context.Context, client client.Client, obj client.Object) error {
	return client.Status().Update(ctx, obj)
}
func MustGetKindViaScheme(obj runtime.Object, remoteScheme *runtime.Scheme) schema.GroupVersionKind {
	kinds, _, err := remoteScheme.ObjectKinds(obj)
	if err != nil {
		fmt.Printf("Failed to get kind for (%#v)\n", obj)
		panic(err)
	}
	if len(kinds) <= 0 {
		panic("Assertion failed!")
	}
	return kinds[0]
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

func DeleteIfExists(ctx context.Context, cli client.Client, obj client.Object) error {
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
