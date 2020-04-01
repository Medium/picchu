package plan

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Plan interface {
	Apply(ctx context.Context, cli client.Client, options Options, log logr.Logger) error
}

// Options are optional parameters that may be used when applying a plan
type Options struct {
	ClusterName   string
	ScalingFactor float64
}

type compositePlan struct {
	plans []Plan
}

func All(plans ...Plan) Plan {
	return &compositePlan{plans}
}

func (p *compositePlan) Apply(ctx context.Context, cli client.Client, options Options, log logr.Logger) error {
	for i := range p.plans {
		plan := p.plans[i]
		planType := reflect.TypeOf(plan).Elem()

		log.Info("Applying Plan", "Plan", plan)
		if err := plan.Apply(ctx, cli, options, log.WithValues("Applier", planType.Name())); err != nil {
			return err
		}
	}
	return nil
}
