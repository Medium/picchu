package plan

import (
	"context"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	IstioInjectionLabelName = "istio-injection"
)

type EnsureNamespace struct {
	Name      string
	OwnerName string
	OwnerType string
}

func (p *EnsureNamespace) Apply(ctx context.Context, cli client.Client, log logr.Logger) error {
	om := metav1.ObjectMeta{
		Name: p.Name,
		Labels: map[string]string{
			IstioInjectionLabelName:       "enabled",
			picchuv1alpha1.LabelOwnerType: p.OwnerType,
			picchuv1alpha1.LabelOwnerName: p.OwnerName,
		},
	}

	ns := &corev1.Namespace{ObjectMeta: om}
	op, err := controllerutil.CreateOrUpdate(ctx, cli, ns, func(runtime.Object) error {
		ns.ObjectMeta.Labels = om.Labels
		return nil
	})
	LogSync(log, op, err, ns)
	return err
}
