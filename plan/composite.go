package plan

import (
	"context"
	"reflect"

	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Plan interface {
	Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error
}

type PrintablePlan interface {
	Printable() interface{}
}

type compositePlan struct {
	plans []Plan
}

func All(plans ...Plan) Plan {
	return &compositePlan{plans}
}

func (p *compositePlan) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	for i := range p.plans {
		plan := p.plans[i]
		planType := reflect.TypeOf(plan).Elem()

		if err := plan.Apply(ctx, cli, cluster, log.WithValues("Applier", planType.Name())); err != nil {
			return err
		}
	}
	return nil
}
