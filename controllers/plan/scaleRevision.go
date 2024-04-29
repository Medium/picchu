package plan

import (
	"context"
	"errors"
	"math"
	"strconv"

	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"

	"go.medium.engineering/picchu/controllers/utils"
	"go.medium.engineering/picchu/plan"

	"github.com/go-logr/logr"
	kedav1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	wpav1 "github.com/practo/k8s-worker-pod-autoscaler/pkg/apis/workerpodautoscaler/v1"
	autoscaling "k8s.io/api/autoscaling/v2"
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
	MemoryTarget       *int32
	RequestsRateMetric string
	RequestsRateTarget *resource.Quantity
	Worker             *picchuv1alpha1.WorkerScaleInfo
	KedaWorker         *picchuv1alpha1.KedaScaleInfo
}

func (p *ScaleRevision) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	if p.Min > p.Max {
		p.Max = p.Min
	}

	var scalingFactor *float64
	if cluster.Spec.ScalingFactorString != nil {
		f, err := strconv.ParseFloat(*cluster.Spec.ScalingFactorString, 64)
		if err != nil {
			log.Error(err, "Could not parse %v to float", *cluster.Spec.ScalingFactorString)
		} else {
			scalingFactor = &f
		}
	}
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

	if p.KedaWorker != nil {
		if err := p.applyKedaTriggerAuth(ctx, cli, log); err != nil {
			return err
		}
		return p.applyKeda(ctx, cli, log, scaledMin, scaledMax)
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

	if p.MemoryTarget != nil {
		memoryTarget := *p.MemoryTarget
		metrics = append(metrics, autoscaling.MetricSpec{
			Type: autoscaling.ResourceMetricSourceType,
			Resource: &autoscaling.ResourceMetricSource{
				Name: "memory",
				Target: autoscaling.MetricTarget{
					AverageUtilization: &memoryTarget,
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
	var secondsToProcessOneJob *float64
	if p.Worker.SecondsToProcessOneJobString != nil {
		f, err := strconv.ParseFloat(*p.Worker.SecondsToProcessOneJobString, 64)
		if err != nil {
			log.Error(err, "Could not parse %v to float", *p.Worker.SecondsToProcessOneJobString)
		} else {
			secondsToProcessOneJob = &f
		}
	}

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
			SecondsToProcessOneJob:  secondsToProcessOneJob,
			MaxDisruption:           p.Worker.MaxDisruption,
		},
	}

	return plan.CreateOrUpdate(ctx, log, cli, wpa)
}

func (p *ScaleRevision) applyKeda(ctx context.Context, cli client.Client, log logr.Logger, scaledMin int32, scaledMax int32) error {
	//If a trigger doesn't have an auth defined, fall back to the identity of the pod.
	for index, trigger := range p.KedaWorker.Triggers {
		if trigger.AuthenticationRef == nil {
			p.KedaWorker.Triggers[index].AuthenticationRef = &kedav1.ScaledObjectAuthRef{
				Name: p.Tag,
			}
			p.KedaWorker.Triggers[index].Metadata["identityOwner"] = "pod"
		}
	}
	keda := &kedav1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Tag,
			Namespace: p.Namespace,
			Labels:    p.Labels,
		},
		Spec: kedav1.ScaledObjectSpec{
			ScaleTargetRef: &kedav1.ScaleTarget{
				Name:       p.Tag,
				Kind:       "ReplicaSet",
				APIVersion: "apps/v1",
			},
			CooldownPeriod:   p.KedaWorker.CooldownPeriod,
			IdleReplicaCount: p.KedaWorker.IdleReplicaCount,
			MinReplicaCount:  &scaledMin,
			MaxReplicaCount:  &scaledMax,
			Advanced:         p.KedaWorker.Advanced,
			Triggers:         p.KedaWorker.Triggers,
			Fallback:         p.KedaWorker.Fallback,
		},
	}
	return plan.CreateOrUpdate(ctx, log, cli, keda)
}

func (p *ScaleRevision) applyKedaTriggerAuth(ctx context.Context, cli client.Client, log logr.Logger) error {
	triggerAuth := &kedav1.TriggerAuthentication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Tag,
			Namespace: p.Namespace,
			Labels:    p.Labels,
		},
		Spec: kedav1.TriggerAuthenticationSpec{
			PodIdentity: &kedav1.AuthPodIdentity{
				Provider: kedav1.PodIdentityProviderAwsKiam,
			},
		},
	}
	return plan.CreateOrUpdate(ctx, log, cli, triggerAuth)
}
