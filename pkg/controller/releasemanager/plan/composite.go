package plan

import (
	"context"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Plan interface {
	Apply(context.Context, client.Client, logr.Logger) error
}

type compositePlan struct {
	plans []Plan
}

func All(plans ...Plan) Plan {
	return &compositePlan{plans}
}

func (p *compositePlan) Apply(ctx context.Context, cli client.Client, log logr.Logger) error {
	log.Info("Applying Plan", "Plan", p)
	for _, plan := range p.plans {
		if err := plan.Apply(ctx, cli, log); err != nil {
			return err
		}
	}
	return nil
}
