package plan

import (
	"context"

	"go.medium.engineering/picchu/pkg/controller/utils"

	"github.com/go-logr/logr"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type ScaleRevision struct {
	Tag       string
	Namespace string
	Min       int32
	Max       int32
	Labels    map[string]string
	CPUTarget *int32
}

func (p *ScaleRevision) Apply(ctx context.Context, cli client.Client, log logr.Logger) error {
	if p.Min > p.Max {
		p.Max = p.Min
	}
	if p.CPUTarget != nil && *p.CPUTarget == 0 {
		hpa := &autoscalingv1.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      p.Tag,
				Namespace: p.Namespace,
			},
		}
		if err := utils.DeleteIfExists(ctx, cli, hpa); err != nil {
			return err
		}
		return nil
	}

	copyMin := p.Min

	hpa := &autoscalingv1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Tag,
			Namespace: p.Namespace,
			Labels:    p.Labels,
		},
		Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
				Kind:       "ReplicaSet",
				Name:       p.Tag,
				APIVersion: "apps/v1",
			},
			MinReplicas:                    &copyMin,
			MaxReplicas:                    p.Max,
			TargetCPUUtilizationPercentage: p.CPUTarget,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, cli, hpa, func(runtime.Object) error {
		min := p.Min
		hpa.Spec.MinReplicas = &min
		hpa.Spec.MaxReplicas = p.Max
		return nil
	})
	LogSync(log, op, err, hpa)
	return err
}
