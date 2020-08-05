package plan

import (
	"context"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"

	"github.com/go-logr/logr"
	wpav1 "github.com/practo/k8s-worker-pod-autoscaler/pkg/apis/workerpodautoscaler/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RetireRevision struct {
	Tag       string
	Namespace string
}

func (p *RetireRevision) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) (err error) {
	namespacedName := types.NamespacedName{Namespace: p.Namespace, Name: p.Tag}
	wpa := &wpav1.WorkerPodAutoScaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Tag,
			Namespace: p.Namespace,
		},
	}
	err = cli.Delete(ctx, wpa)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "Failed to delete WPA while retiring revision",
				"namespace", p.Namespace,
				"tag", p.Tag,
			)
			return
		}
	}

	rs := &appsv1.ReplicaSet{}
	err = cli.Get(ctx, namespacedName, rs)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "Failed to update replicaset to 0 replicas, while retiring revision",
				"namespace", p.Namespace,
				"tag", p.Tag,
			)
			return
		}
	} else if *rs.Spec.Replicas != 0 {
		var r int32 = 0
		rs.Spec.Replicas = &r
		err = cli.Update(ctx, rs)
		log.Info("ReplicaSet sync'd", "Type", "ReplicaSet", "Audit", true, "Content", rs, "Op", "updated")
		if err != nil {
			return
		}
	}

	return nil
}
