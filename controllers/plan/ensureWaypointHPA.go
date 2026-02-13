package plan

import (
	"context"

	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/controllers/utils"
	"go.medium.engineering/picchu/plan"

	"github.com/go-logr/logr"
	autoscaling "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	waypointHPAName    = "waypoint"
	waypointDeployment = "waypoint"
	waypointDefaultMin = 2
	waypointDefaultMax = 20
	waypointDefaultCPU = 70
)

// EnsureWaypointHPA creates or updates an HPA for the waypoint Deployment (min 2, max 20, 70% CPU).
// Call when AmbientMesh is true. The waypoint Deployment is created by Istio from the Gateway.
type EnsureWaypointHPA struct {
	Namespace string
	HPA       *picchuv1alpha1.WaypointHPASpec
}

func (p *EnsureWaypointHPA) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	if p.HPA == nil || p.HPA.MaxReplicas < 1 {
		return nil
	}
	minRep := p.HPA.MinReplicas
	if minRep < 1 {
		minRep = 1
	}
	if p.HPA.MaxReplicas < minRep {
		minRep = p.HPA.MaxReplicas
	}
	maxRep := p.HPA.MaxReplicas
	cpuTarget := p.HPA.TargetCPUUtilizationPercentage
	if cpuTarget < 1 {
		cpuTarget = waypointDefaultCPU
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
