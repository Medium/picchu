package plan

import (
	"context"
	"errors"
	"math"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"

	"go.medium.engineering/picchu/pkg/controller/utils"
	"go.medium.engineering/picchu/pkg/plan"

	"github.com/go-logr/logr"
	wpav1 "github.com/practo/k8s-worker-pod-autoscaler/pkg/apis/workerpodautoscaler/v1"
	autoscaling "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ScaleRevision struct {
	Tag                string
	Namespace          string
	Min                int32
	Max                int32
	Labels             map[string]string
	CPUTarget          *int32
	RequestsRateMetric string
	RequestsRateTarget *resource.Quantity
	Worker             *picchuv1alpha1.WorkerScaleInfo
}

func (p *ScaleRevision) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	if p.Min > p.Max {
		p.Max = p.Min
	}

	scalingFactor := cluster.Spec.ScalingFactor
	if scalingFactor == nil {
		e := errors.New("cluster scalingFactor can't be nil")
		log.Error(e, "Cluster scalingFactor nil")
		return e
	}

	scaledMin := int32(math.Ceil(float64(p.Min) * *scalingFactor))
	scaledMax := int32(math.Ceil(float64(p.Max) * *scalingFactor))

	if p.Worker != nil {
		return p.applyWPA(ctx, cli, log, scaledMin, scaledMax)
	}

	return p.applyHPA(ctx, cli, log, scaledMin, scaledMax)
}

func (p *ScaleRevision) applyHPA(ctx context.Context, cli client.Client, log logr.Logger, scaledMin int32, scaledMax int32) error {
	var metrics []autoscaling.MetricSpec

	if p.CPUTarget != nil {
		cpuTarget := *p.CPUTarget
		metrics = append(metrics, autoscaling.MetricSpec{
			Type: autoscaling.ResourceMetricSourceType,
			Resource: &autoscaling.ResourceMetricSource{
				Name: "cpu",
				Target: autoscaling.MetricTarget{
					AverageUtilization: &cpuTarget,
					Type:               autoscaling.UtilizationMetricType,
				},
			},
		})
	}

	if p.RequestsRateTarget != nil {
		rateTarget := *p.RequestsRateTarget
		metrics = append(metrics, autoscaling.MetricSpec{
			Type: autoscaling.PodsMetricSourceType,
			Pods: &autoscaling.PodsMetricSource{
				Metric: autoscaling.MetricIdentifier{
					Name: p.RequestsRateMetric,
				},
				Target: autoscaling.MetricTarget{
					AverageValue: &rateTarget,
					Type:         autoscaling.AverageValueMetricType,
				},
			},
		})
	}

	if len(metrics) <= 0 {
		hpa := &autoscaling.HorizontalPodAutoscaler{
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

	hpa := &autoscaling.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Tag,
			Namespace: p.Namespace,
			Labels:    p.Labels,
		},
		Spec: autoscaling.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscaling.CrossVersionObjectReference{
				Kind:       "ReplicaSet",
				Name:       p.Tag,
				APIVersion: "apps/v1",
			},
			MinReplicas: &scaledMin,
			MaxReplicas: scaledMax,
			Metrics:     metrics,
		},
	}

	return plan.CreateOrUpdate(ctx, log, cli, hpa)
}

func (p *ScaleRevision) applyWPA(ctx context.Context, cli client.Client, log logr.Logger, scaledMin int32, scaledMax int32) error {
	wpa := &wpav1.WorkerPodAutoScaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Tag,
			Namespace: p.Namespace,
			Labels:    p.Labels,
		},
		Spec: wpav1.WorkerPodAutoScalerSpec{
			ReplicaSetName:          p.Tag,
			MinReplicas:             &scaledMin,
			MaxReplicas:             &scaledMax,
			QueueURI:                p.Worker.QueueURI,
			TargetMessagesPerWorker: p.Worker.TargetMessagesPerWorker,
			SecondsToProcessOneJob:  p.Worker.SecondsToProcessOneJob,
			MaxDisruption:           p.Worker.MaxDisruption,
		},
	}

	return plan.CreateOrUpdate(ctx, log, cli, wpa)
}
