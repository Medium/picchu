package plan

import (
	"context"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/plan"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	IstioInjectionLabelName = "istio-injection"
)

type EnsureNamespace struct {
	Name      string
	OwnerName string
	OwnerType string
}

func (p *EnsureNamespace) Apply(ctx context.Context, cli client.Client, scalingFactor float64, log logr.Logger) error {
	om := metav1.ObjectMeta{
		Name: p.Name,
		Labels: map[string]string{
			IstioInjectionLabelName:       "enabled",
			picchuv1alpha1.LabelOwnerType: p.OwnerType,
			picchuv1alpha1.LabelOwnerName: p.OwnerName,
		},
	}

	return plan.CreateOrUpdate(ctx, log, cli, &corev1.Namespace{ObjectMeta: om})
}
