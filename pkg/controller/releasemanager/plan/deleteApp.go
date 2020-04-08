package plan

import (
	"context"
	"github.com/go-logr/logr"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeleteApp struct {
	Namespace string
}

func (p *DeleteApp) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	item := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: p.Namespace,
		},
	}
	log.Info("Deleting remote resource",
		"Item.Kind", "Namespace",
		"Item", item,
	)
	if err := cli.Delete(ctx, item); err != nil && !errors.IsNotFound(err) {
		log.Error(err, "Failed to delete resource", "Resource", item)
		return err
	}
	return nil
}
