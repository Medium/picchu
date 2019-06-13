package plan

import (
	"context"

	"go.medium.engineering/picchu/pkg/controller/utils"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeleteRevision struct {
	Labels    map[string]string
	Namespace string
}

func (p *DeleteRevision) Apply(ctx context.Context, cli client.Client, log logr.Logger) error {
	log.Info("Applying Plan", "Plan", p)
	lists := []List{
		NewSecretList(),
		NewConfigMapList(),
		NewReplicaSetList(),
		NewHorizontalPodAutoscalerList(),
	}
	opts := client.
		MatchingLabels(p.Labels).
		InNamespace(p.Namespace)

	for _, list := range lists {
		if err := cli.List(ctx, opts, list.GetList()); err != nil {
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
