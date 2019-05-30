package clustersecrets

import (
	"context"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/controller/utils"

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

func newReconcileRequest(r *ReconcileClusterSecrets, instance *picchuv1alpha1.ClusterSecrets, log logr.Logger) *reconcileRequest {
	return &reconcileRequest{
		client:   r.client,
		scheme:   r.scheme,
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
	for _, cluster := range clusterList.Items {
		r.log.Info("Syncing cluster", "cluster", cluster.Name)
		remoteClient, err := utils.RemoteClient(r.client, &cluster)
		if err != nil {
			return err
		}
		log := r.log.WithValues("Cluster", cluster.Name)
		secretDeployer := newSecretDeployer(remoteClient, log, secretList, r.instance)
		if err = secretDeployer.deploy(ctx); err != nil {
			return err
		}
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

	opts := client.InNamespace(r.instance.Spec.Source.Namespace)
	if labelSelector != "" {
		opts.SetLabelSelector(labelSelector)
	}
	if fieldSelector != "" {
		opts.SetFieldSelector(fieldSelector)
	}

	err := r.client.List(ctx, opts, secretList)
	return secretList, err
}

func (r *reconcileRequest) clusterList(ctx context.Context) (*picchuv1alpha1.ClusterList, error) {
	clusterList := &picchuv1alpha1.ClusterList{}
	labelSelector := r.instance.Spec.Target.LabelSelector
	fieldSelector := r.instance.Spec.Target.FieldSelector

	opts := client.InNamespace(r.instance.Namespace)
	if labelSelector != "" {
		opts.SetLabelSelector(labelSelector)
	}
	if fieldSelector != "" {
		opts.SetFieldSelector(fieldSelector)
		r.log.Info("FieldSelector set", "Selector", opts.FieldSelector)
	}

	err := r.client.List(ctx, opts, clusterList)
	r.scheme.Default(clusterList)
	return clusterList, err
}
