package plan

import (
	"context"
	"reflect"

	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"

	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Applier interface {
	Apply(context.Context, Plan) error
}

type ClusterApplier struct {
	cli     client.Client
	cluster *picchuv1alpha1.Cluster
	log     logr.Logger
}

func NewClusterApplier(cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) Applier {
	return &ClusterApplier{cli, cluster, log}
}

func (a *ClusterApplier) Apply(ctx context.Context, plan Plan) error {
	planType := reflect.TypeOf(plan).Elem()
	return plan.Apply(ctx, a.cli, a.cluster, a.log.WithValues("Applier", planType.Name()))
}

type ConcurrentApplier struct {
	appliers []Applier
	log      logr.Logger
}

func NewConcurrentApplier(appliers []Applier, log logr.Logger) Applier {
	return &ConcurrentApplier{appliers, log}
}

func (a *ConcurrentApplier) Apply(ctx context.Context, plan Plan) error {
	var g errgroup.Group

	for i := range a.appliers {
		applier := a.appliers[i]
		g.Go(func() error {
			return applier.Apply(ctx, plan)
		})
	}

	return g.Wait()
}
