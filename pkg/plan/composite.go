package plan

import (
	"context"
	"reflect"

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
	for i := range p.plans {
		plan := p.plans[i]
		planType := reflect.TypeOf(plan).Elem()

		log.Info("Applying Plan", "Plan", plan)
		if err := plan.Apply(ctx, cli, scalingFactor, log.WithValues("Applier", planType.Name())); err != nil {
			return err
		}
	}
	return nil
}
