package plan

import (
	"context"

	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/plan"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type EnsureServiceAccount struct {
	Name      string
	OwnerName string
	OwnerType string
	Namespace string
}

func (p *EnsureServiceAccount) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	sa := &corev1.ServiceAccount{}
	annotations := map[string]string{}
	err := cli.Get(ctx, client.ObjectKey{Namespace: p.Namespace, Name: p.Name}, sa)

	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if err == nil && sa.Annotations != nil {
		for k, v := range sa.Annotations {
			annotations[k] = v
		}
	}
	om := metav1.ObjectMeta{
		Name: p.Name,
		Labels: map[string]string{
			picchuv1alpha1.LabelOwnerType: p.OwnerType,
			picchuv1alpha1.LabelOwnerName: p.OwnerName,
		},
		Annotations: annotations,
	}

	return plan.CreateOrUpdate(ctx, log, cli, &corev1.ServiceAccount{ObjectMeta: om})
}
