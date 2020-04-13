package plan

import (
	"context"
	"github.com/go-logr/logr"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RetireRevision struct {
	Tag       string
	Namespace string
}

func (p *RetireRevision) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	rs := &appsv1.ReplicaSet{}
	s := types.NamespacedName{p.Namespace, p.Tag}
	err := cli.Get(ctx, s, rs)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		log.Error(err, "Failed to update replicaset to 0 replicas")
		return err
	}
	if *rs.Spec.Replicas != 0 {
		var r int32 = 0
		rs.Spec.Replicas = &r
		err = cli.Update(ctx, rs)
		log.Info("ReplicaSet sync'd", "Type", "ReplicaSet", "Audit", true, "Content", rs, "Op", "updated")
		if err != nil {
			return err
		}
	}
	return nil
}
