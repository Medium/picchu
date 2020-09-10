package plan

import (
	"context"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"

	"github.com/go-logr/logr"
	wpav1 "github.com/practo/k8s-worker-pod-autoscaler/pkg/apis/workerpodautoscaler/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RetireRevision struct {
	Tag       string
	Namespace string
}

func (p *RetireRevision) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	if err := deleteWPA(ctx, cli, p.Namespace, p.Tag); err != nil {
		log.Error(err, "Failed to delete WPA while retiring revision",
			"tag", p.Tag,
			"name", p.Namespace,
		)
		return err
	}

	namespacedName := types.NamespacedName{Namespace: p.Namespace, Name: p.Tag}
	rs := &appsv1.ReplicaSet{}
	if err := cli.Get(ctx, namespacedName, rs); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "Failed to get ReplicaSet while retiring revision",
				"namespace", p.Namespace,
				"tag", p.Tag,
			)
			return err
		}
		rs = nil
	}
	if rs != nil && *rs.Spec.Replicas != 0 {
		var zero int32 = 0
		rs.Spec.Replicas = &zero
		if err := cli.Update(ctx, rs); err != nil {
			log.Error(err, "Failed to update ReplicaSet to 0 replicas while retiring revision",
				"namespace", p.Namespace,
				"tag", p.Tag,
			)
			return err
		}

		log.Info("ReplicaSet sync'd",
			"Type", "ReplicaSet",
			"Audit", true,
			"Content", rs,
			"Op", "updated",
		)
	}

	return nil
}

// Delete any WorkerPodAutoScaler for the deployed revision
func deleteWPA(ctx context.Context, cli client.Client, namespace, name string) error {
	wpa := &wpav1.WorkerPodAutoScaler{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
	if err := cli.Delete(ctx, wpa); err != nil {
		switch err.(type) {
		case *meta.NoKindMatchError:
			break
		default:
			if !errors.IsNotFound(err) {
				return err
			}
		}
	}
	return nil
}
