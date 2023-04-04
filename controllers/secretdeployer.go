package controllers

import (
	"context"
	"errors"

	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"

	"github.com/go-logr/logr"
	"github.com/prometheus/common/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	s.log.Info("Found secrets", "count", len(s.secretList.Items))
	if err := s.syncNamespace(ctx); err != nil {
		s.log.Info("Failed to sync namespace", "namespace.Name", s.instance.Spec.Target.Namespace)
		return err
	}

	errs := []error{}
	for _, src := range s.secretList.Items {
		s.log.Info("Syncing secret", "secret.Name", src.Name)
		dst := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      src.Name,
				Namespace: s.instance.Spec.Target.Namespace,
			},
		}

		_, err := controllerutil.CreateOrUpdate(ctx, s.client, dst, func() error {
			dst.Labels = s.instance.Spec.Target.Labels
			dst.Annotations = s.instance.Spec.Target.Annotations
			dst.Data = src.Data
			dst.Type = src.Type
			return nil
		})
		if err != nil {
			log.Error(err, "Failed to sync secret", "secret.Name", src.Name)
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.New("Failed to sync all secrets")
	}
	return nil
}

func (s *secretDeployer) syncNamespace(ctx context.Context) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: s.instance.Spec.Target.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, s.client, namespace, func() error {
		return nil
	})
	s.log.Info("Namespace sync'd", "Op", op)
	return err
}
