package observe

import (
	"context"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"

	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Observer returns an Observation for a namespace
type Observer interface {
	Observe(context.Context, string) (*Observation, error)
}

// ClusterObserver observers a single target cluster
type ClusterObserver struct {
	cluster Cluster
	cli     client.Client
	log     logr.Logger
}

// NewClusterObserver creates a ClusterObserver
func NewClusterObserver(cluster Cluster, cli client.Client, log logr.Logger) Observer {
	return &ClusterObserver{cluster, cli, log}
}

// Observe observes a single target cluster
func (o *ClusterObserver) Observe(ctx context.Context, namespace string) (*Observation, error) {
	replicaSetList := &appsv1.ReplicaSetList{}
	listOpts := &client.ListOptions{Namespace: namespace}
	err := o.cli.List(ctx, listOpts, replicaSetList)
	if err != nil {
		return nil, err
	}

	replicaSets := []replicaSet{}
	for _, rs := range replicaSetList.Items {
		tag, ok := rs.Labels[picchuv1alpha1.LabelTag]
		if !ok {
			continue
		}
		replicaSets = append(replicaSets, replicaSet{
			Cluster: o.cluster,
			Tag:     tag,
			Desired: *rs.Spec.Replicas,
			Current: rs.Status.AvailableReplicas,
		})
	}

	obs := &Observation{
		ReplicaSets: replicaSets,
	}
	o.log.Info("Cluster state", "Observation", obs)
	return obs, nil
}

// ConcurrentObserver combines results from multiple Observers, run concurrently. Work abandoned on first known error.
type ConcurrentObserver struct {
	observers []Observer
	log       logr.Logger
}

// NewConcurrentObserver creates a ConcurrentObserver
func NewConcurrentObserver(observers []Observer, log logr.Logger) Observer {
	return &ConcurrentObserver{observers, log}
}

// Observe calls all child observers concurrently. It bails out when an error is encountered.
func (o *ConcurrentObserver) Observe(ctx context.Context, namespace string) (*Observation, error) {
	g, ctx := errgroup.WithContext(ctx)
	observations := []Observation{}

	for i := range o.observers {
		i := i
		g.Go(func() error {
			obs, err := o.observers[i].Observe(ctx, namespace)
			if err != nil {
				return err
			}
			observations = append(observations, *obs)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	obs := &Observation{}
	for i := range observations {
		o := observations[i]
		obs = obs.Combine(&o)
	}
	// TODO(bob): this is too big to log, maybe only display replicasets with >0 values.
	// o.log.Info("Concurrent observations collected", "Observation", obs)
	return obs, nil
}
