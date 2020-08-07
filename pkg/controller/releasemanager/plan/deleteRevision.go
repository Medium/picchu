package plan

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/controller/utils"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeleteRevision struct {
	Labels    map[string]string
	Namespace string
}

func (p *DeleteRevision) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	lists := []List{
		NewSecretList(),
		NewConfigMapList(),
		NewReplicaSetList(),
		NewHorizontalPodAutoscalerList(),
		NewWorkerPodAutoscalerList(),
	}

	opts := &client.ListOptions{
		Namespace:     p.Namespace,
		LabelSelector: labels.SelectorFromSet(p.Labels),
	}

	for _, list := range lists {
		if err := cli.List(ctx, list.GetList(), opts); err != nil {
			switch err.(type) {
			case *meta.NoKindMatchError:
				continue
			default:
				log.Error(err, "Failed to list resource")
				return err
			}
		}
		log.Info("list", "list", list)
		for _, item := range list.GetItems() {
			kind := utils.MustGetKind(item)
			log.Info("Deleting remote resource",
				"Item.Kind", kind,
				"Item", item,
			)
			if err := cli.Delete(ctx, item); err != nil {
				log.Error(err, "Failed to delete resource", "labels", p.Labels, "Resource", list)
				return err
			}
		}
	}
	return nil
}
