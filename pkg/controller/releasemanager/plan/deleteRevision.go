package plan

import (
	"context"

	"go.medium.engineering/picchu/pkg/controller/utils"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeleteRevision struct {
	Labels    client.MatchingLabels
	Namespace string
}

func (p *DeleteRevision) Apply(ctx context.Context, cli client.Client, scalingFactor float64, log logr.Logger) error {
	lists := []List{
		NewSecretList(),
		NewConfigMapList(),
		NewReplicaSetList(),
		NewHorizontalPodAutoscalerList(),
	}

	opts := []client.ListOption{
		&client.ListOptions{Namespace: p.Namespace},
		p.Labels,
	}

	for _, list := range lists {
		if err := cli.List(ctx, list.GetList(), opts...); err != nil {
			log.Error(err, "Failed to list resource")
			return err
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
