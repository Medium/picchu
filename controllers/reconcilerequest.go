package controllers

import (
	"context"
	"errors"

	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/controllers/utils"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type reconcileRequest struct {
	client   client.Client
	scheme   *runtime.Scheme
	config   utils.Config
	log      logr.Logger
	instance *picchuv1alpha1.ClusterSecrets
}

func newReconcileRequest(r *ClusterSecretsReconciler, instance *picchuv1alpha1.ClusterSecrets, log logr.Logger) *reconcileRequest {
	return &reconcileRequest{
		client:   r.Client,
		scheme:   r.Scheme,
		config:   r.config,
		instance: instance,
		log:      log,
	}
}

func (r *reconcileRequest) reconcile(ctx context.Context) error {
	secretList, err := r.secretList(ctx)
	if err != nil {
		return err
	}
	clusterList, err := r.clusterList(ctx)
	if err != nil {
		return err
	}
	if len(clusterList.Items) == 0 {
		r.log.Info("No clusters found")
	}
	errs := []error{}
	for _, cluster := range clusterList.Items {
		if !cluster.Spec.Enabled {
			r.log.Info("Skipping disabled cluster", "cluster.Name", cluster.Name)
			continue
		}
		r.log.Info("Syncing cluster", "cluster", cluster.Name)
		remoteClient, err := utils.RemoteClient(ctx, r.log, r.client, &cluster)
		if err != nil {
			r.log.Error(err, "Failed to get remote client", "cluster.Name", cluster.Name)
			errs = append(errs, err)
			continue
		}
		log := r.log.WithValues("Cluster", cluster.Name)
		secretDeployer := newSecretDeployer(remoteClient, log, secretList, r.instance)
		if err = secretDeployer.deploy(ctx); err != nil {
			r.log.Error(err, "Failed to deploy all secrets for cluster", "cluster.Name", cluster.Name)
			errs = append(errs, err)
			continue
		}
	}
	if len(errs) > 0 {
		return errors.New("Failed to reconcile all clusters")
	}
	return nil
}

func (r *reconcileRequest) finalize(ctx context.Context) error {
	// TODO(bob): implement
	return nil
}

func (r *reconcileRequest) secretList(ctx context.Context) (*corev1.SecretList, error) {
	secretList := &corev1.SecretList{}
	labelSelector := r.instance.Spec.Source.LabelSelector
	fieldSelector := r.instance.Spec.Source.FieldSelector

	opts := &client.ListOptions{Namespace: r.instance.Spec.Source.Namespace}
	if labelSelector != "" {
		opts.Raw.LabelSelector = labelSelector
	}
	if fieldSelector != "" {
		opts.Raw.FieldSelector = fieldSelector
	}

	err := r.client.List(ctx, secretList, opts)
	return secretList, err
}

func (r *reconcileRequest) clusterList(ctx context.Context) (*picchuv1alpha1.ClusterList, error) {
	clusterList := &picchuv1alpha1.ClusterList{}
	labelSelector := r.instance.Spec.Target.LabelSelector
	fieldSelector := r.instance.Spec.Target.FieldSelector

	opts := &client.ListOptions{Namespace: r.instance.Namespace}
	if labelSelector != "" {
		opts.Raw.LabelSelector = labelSelector
		r.log.Info("LabelSelector set", "Selector", opts.LabelSelector)
	}
	if fieldSelector != "" {
		opts.Raw.FieldSelector = fieldSelector
		r.log.Info("FieldSelector set", "Selector", opts.FieldSelector)
	}

	err := r.client.List(ctx, clusterList, opts)
	r.scheme.Default(clusterList)
	return clusterList, err
}
