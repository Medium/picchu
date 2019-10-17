package plan

import (
	"context"
	"math"

	"go.medium.engineering/picchu/pkg/controller/utils"
	"go.medium.engineering/picchu/pkg/plan"

	"github.com/go-logr/logr"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ScaleRevision struct {
	Tag       string
	Namespace string
	Min       int32
	Max       int32
	Labels    map[string]string
	CPUTarget *int32
}

func (p *ScaleRevision) Apply(ctx context.Context, cli client.Client, scalingFactor float64, log logr.Logger) error {
	var cpuTarget *int32
	if p.CPUTarget != nil {
		t := *p.CPUTarget
		cpuTarget = &t
	}
	if p.Min > p.Max {
		p.Max = p.Min
	}
	if cpuTarget != nil && *cpuTarget == 0 {
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

	scaledMin := int32(math.Ceil(float64(p.Min) * scalingFactor))
	scaledMax := int32(math.Ceil(float64(p.Max) * scalingFactor))

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
			MinReplicas:                    &scaledMin,
			MaxReplicas:                    scaledMax,
			TargetCPUUtilizationPercentage: cpuTarget,
		},
	}

	return plan.CreateOrUpdate(ctx, log, cli, hpa)
}
