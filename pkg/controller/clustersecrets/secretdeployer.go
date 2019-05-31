package clustersecrets

import (
	"context"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type secretDeployer struct {
	client     client.Client
	log        logr.Logger
	secretList *corev1.SecretList
	instance   *picchuv1alpha1.ClusterSecrets
}

func newSecretDeployer(client client.Client, log logr.Logger, secretList *corev1.SecretList, instance *picchuv1alpha1.ClusterSecrets) *secretDeployer {
	return &secretDeployer{
		client:     client,
		log:        log,
		secretList: secretList,
		instance:   instance,
	}
}

func (s *secretDeployer) deploy(ctx context.Context) error {
	if len(s.secretList.Items) == 0 {
		s.log.Info("No secrets found")
	}
	for _, src := range s.secretList.Items {
		s.log.Info("Syncing secret", "secret.Name", src.Name)
		dst := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      src.Name,
				Namespace: s.instance.Spec.Target.Namespace,
			},
		}

		_, err := controllerutil.CreateOrUpdate(ctx, s.client, dst, func(runtime.Object) error {
			dst.Labels = s.instance.Spec.Target.Labels
			dst.Annotations = s.instance.Spec.Target.Annotations
			dst.Data = src.Data
			dst.Type = src.Type
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}
