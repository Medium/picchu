package plan

import (
	"context"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Plan interface {
	Apply(ctx context.Context, cli client.Client, scalingFactor float64, log logr.Logger) error
}

type compositePlan struct {
	plans []Plan
}

func All(plans ...Plan) Plan {
	return &compositePlan{plans}
}

func (p *compositePlan) Apply(ctx context.Context, cli client.Client, scalingFactor float64, log logr.Logger) error {
	for _, plan := range p.plans {
		if err := plan.Apply(ctx, cli, scalingFactor, log); err != nil {
			return err
		}
	}
	return nil
}
