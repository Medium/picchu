package plan

import (
	"context"

	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/plan"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	IstioInjectionLabelName   = "istio-injection"
	IstioDataplaneModeLabel   = "istio.io/dataplane-mode"
	IstioDataplaneModeAmbient = "ambient"
)

type EnsureNamespace struct {
	Name        string
	OwnerName   string
	OwnerType   string
	AmbientMesh bool
}

func (p *EnsureNamespace) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	labels := map[string]string{
		picchuv1alpha1.LabelOwnerType: p.OwnerType,
		picchuv1alpha1.LabelOwnerName: p.OwnerName,
	}
	if p.AmbientMesh {
		labels[IstioDataplaneModeLabel] = IstioDataplaneModeAmbient
		// Do not set istio-injection; ambient mode does not use sidecars.
	} else {
		labels[IstioInjectionLabelName] = "enabled"
	}

	om := metav1.ObjectMeta{
		Name:   p.Name,
		Labels: labels,
	}

	return plan.CreateOrUpdate(ctx, log, cli, &corev1.Namespace{ObjectMeta: om})
}
