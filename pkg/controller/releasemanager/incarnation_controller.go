package releasemanager

import (
	"context"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type IncarnationController struct {
	rrm          *ReconcileReleaseManager
	remoteClient client.Client
	logger       logr.Logger
	rm           *picchuv1alpha1.ReleaseManager
	fs           uint32
}

func (i *IncarnationController) scheme() *runtime.Scheme {
	return i.rrm.scheme
}

func (i *IncarnationController) log() logr.Logger {
	return i.logger
}

func (i *IncarnationController) releaseManager() *picchuv1alpha1.ReleaseManager {
	return i.rm
}

func (i *IncarnationController) client() client.Client {
	return i.remoteClient
}

func (i *IncarnationController) getSecrets(ctx context.Context, opts *client.ListOptions) (*corev1.SecretList, error) {
	secrets := &corev1.SecretList{}
	err := i.rrm.client.List(ctx, opts, secrets)
	return secrets, err
}

func (i *IncarnationController) getConfigMaps(ctx context.Context, opts *client.ListOptions) (*corev1.ConfigMapList, error) {
	configMaps := &corev1.ConfigMapList{}
	err := i.rrm.client.List(ctx, opts, configMaps)
	return configMaps, err
}

func (i *IncarnationController) fleetSize() int32 {
	return int32(i.fs)
}
