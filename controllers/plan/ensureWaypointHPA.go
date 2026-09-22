package plan

import (
	"context"

	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/controllers/utils"
	"go.medium.engineering/picchu/plan"

	"github.com/go-logr/logr"
	autoscaling "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	waypointHPAName    = "waypoint"
	waypointDeployment = "waypoint"
	waypointDefaultMin = 2
	waypointDefaultMax = 20
	waypointDefaultCPU = 70

	// Scale-down damping. Without an explicit behavior the Kubernetes defaults
	// let the HPA remove 100% of the waypoint pods in a single 15s step, and
	// every removed pod terminates the gRPC connections routed through it
	// ("upstream connect error ... reset reason: connection termination").
	// Waypoints are shared L7 proxies, so one aggressive step is felt by every
	// caller of the namespace at once.
	waypointScaleDownStabilizationSeconds = 300
	waypointScaleDownPeriodSeconds        = 60
	waypointScaleDownPercent              = 25
	waypointScaleDownPods                 = 1
)

// EnsureWaypointHPA creates or updates an HPA for the waypoint Deployment (min 2, max 20, 70% CPU).
// Call when AmbientMesh is true. The waypoint Deployment is created by Istio from the Gateway.
type EnsureWaypointHPA struct {
	Namespace string
	HPA       *picchuv1alpha1.WaypointHPASpec
}

func (p *EnsureWaypointHPA) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	if p.HPA == nil {
		return nil
	}
	minRep := p.HPA.MinReplicas
	if minRep < waypointDefaultMin {
		minRep = waypointDefaultMin
	}
	maxRep := p.HPA.MaxReplicas
	if maxRep < 1 {
		maxRep = waypointDefaultMax
	}
	if maxRep < minRep {
		minRep = maxRep
	}
	cpuTarget := p.HPA.TargetCPUUtilizationPercentage
	if cpuTarget < 1 {
		cpuTarget = waypointDefaultCPU
	}

	stabilization := int32(waypointScaleDownStabilizationSeconds)
	// Two policies with SelectPolicy: Max. The percent policy does the damping
	// at higher replica counts; the pods policy guarantees the HPA can still
	// make progress at low counts, where 25% rounds down to no pods at all and
	// a percent-only rule would pin the waypoint above its floor forever.
	scaleDown := &autoscaling.HPAScalingRules{
		StabilizationWindowSeconds: &stabilization,
		SelectPolicy:               ptr.To(autoscaling.MaxChangePolicySelect),
		Policies: []autoscaling.HPAScalingPolicy{
			{
				Type:          autoscaling.PercentScalingPolicy,
				Value:         waypointScaleDownPercent,
				PeriodSeconds: waypointScaleDownPeriodSeconds,
			},
			{
				Type:          autoscaling.PodsScalingPolicy,
				Value:         waypointScaleDownPods,
				PeriodSeconds: waypointScaleDownPeriodSeconds,
			},
		},
	}

	hpa := &autoscaling.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      waypointHPAName,
			Namespace: p.Namespace,
		},
		Spec: autoscaling.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscaling.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       waypointDeployment,
			},
			MinReplicas: &minRep,
			MaxReplicas: maxRep,
			// ScaleUp is deliberately left to the Kubernetes defaults. Scaling
			// up does not terminate connections, and damping it would only slow
			// the waypoint's response to a traffic spike.
			Behavior: &autoscaling.HorizontalPodAutoscalerBehavior{
				ScaleDown: scaleDown,
			},
			Metrics: []autoscaling.MetricSpec{
				{
					Type: autoscaling.ResourceMetricSourceType,
					Resource: &autoscaling.ResourceMetricSource{
						Name: "cpu",
						Target: autoscaling.MetricTarget{
							Type:               autoscaling.UtilizationMetricType,
							AverageUtilization: &cpuTarget,
						},
					},
				},
			},
		},
	}
	return plan.CreateOrUpdate(ctx, log, cli, hpa)
}

// DeleteWaypointHPA removes the waypoint HPA when switching off ambient.
type DeleteWaypointHPA struct {
	Namespace string
}

func (p *DeleteWaypointHPA) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	hpa := &autoscaling.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      waypointHPAName,
			Namespace: p.Namespace,
		},
	}
	return utils.DeleteIfExists(ctx, cli, hpa)
}
