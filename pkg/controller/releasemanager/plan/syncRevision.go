package plan

import (
	"context"
	"errors"
	"math"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/plan"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	defaultLivenessProbe  *corev1.Probe
	defaultReadinessProbe *corev1.Probe
)

// TODO(bob): Move to Revision spec
func init() {
	defaultLivenessProbe = &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/running",
				Port: intstr.FromString("status"),
			},
		},
		InitialDelaySeconds: 10,
		PeriodSeconds:       10,
		TimeoutSeconds:      1,
		SuccessThreshold:    1,
		FailureThreshold:    7,
	}

	defaultReadinessProbe = &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/running",
				Port: intstr.FromString("status"),
			},
		},
		InitialDelaySeconds: 10,
		PeriodSeconds:       10,
		TimeoutSeconds:      1,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}
}

type SyncRevision struct {
	App                string
	Tag                string
	Namespace          string
	Labels             map[string]string // Labels applied to all resources
	Configs            []runtime.Object  // Secret and ConfigMap objects supported and mapped to environment
	Ports              []picchuv1alpha1.PortInfo
	Replicas           int32
	Image              string
	Resources          corev1.ResourceRequirements
	IAMRole            string // AWS iam role
	ServiceAccountName string // k8s ServiceAccount
	LivenessProbe      *corev1.Probe
	ReadinessProbe     *corev1.Probe
	MinReadySeconds    int32
}

func (p *SyncRevision) Apply(ctx context.Context, cli client.Client, scalingFactor float64, log logr.Logger) error {
	var livenessProbe *corev1.Probe
	var readinessProbe *corev1.Probe
	if p.LivenessProbe != nil {
		probe := *p.LivenessProbe
		livenessProbe = &probe
	}
	if p.ReadinessProbe != nil {
		probe := *p.ReadinessProbe
		readinessProbe = &probe
	}
	envs := []corev1.EnvFromSource{}

	// Clone passed in objects to prevent concurrency issues
	configs := []runtime.Object{}
	for i := range p.Configs {
		configs = append(configs, p.Configs[i].DeepCopyObject())
	}

	for i := range configs {
		config := configs[i]
		switch resource := config.(type) {
		case *corev1.Secret:
			resource.ObjectMeta = metav1.ObjectMeta{
				Name:      resource.Name,
				Namespace: p.Namespace,
				Labels:    p.Labels,
			}
			envs = append(envs, corev1.EnvFromSource{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: resource.Name},
				},
			})
		case *corev1.ConfigMap:
			resource.ObjectMeta = metav1.ObjectMeta{
				Name:      resource.Name,
				Namespace: p.Namespace,
				Labels:    p.Labels,
			}
			envs = append(envs, corev1.EnvFromSource{
				ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: resource.Name},
				},
			})
		default:
			e := errors.New("Unsupported config")
			log.Error(e, "Unsupported config", "Config", config)
			return e
		}
	}

	ports := []corev1.ContainerPort{}
	hasStatusPort := false
	for _, port := range p.Ports {
		ports = append(ports, corev1.ContainerPort{
			Name:          port.Name,
			Protocol:      port.Protocol,
			ContainerPort: port.ContainerPort,
		})
		if port.Name == "status" {
			hasStatusPort = true
		}
	}

	appContainer := corev1.Container{
		EnvFrom:        envs,
		Image:          p.Image,
		Name:           p.App,
		Ports:          ports,
		Resources:      p.Resources,
		LivenessProbe:  livenessProbe,
		ReadinessProbe: readinessProbe,
	}

	if hasStatusPort {
		if p.LivenessProbe == nil {
			probe := *defaultLivenessProbe
			appContainer.LivenessProbe = &probe
		}
		if p.ReadinessProbe == nil {
			probe := *defaultReadinessProbe
			appContainer.ReadinessProbe = &probe
		}
	}

	template := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name:        p.Tag,
			Namespace:   p.Namespace,
			Labels:      p.Labels,
			Annotations: map[string]string{},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: p.ServiceAccountName,
			Containers:         []corev1.Container{appContainer},
			DNSConfig:          DefaultDNSConfig(),
		},
	}

	if p.IAMRole != "" {
		template.Annotations[picchuv1alpha1.AnnotationIAMRole] = p.IAMRole
	}

	scaledReplicas := int32(math.Ceil(float64(p.Replicas) * scalingFactor))
	replicaSet := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: p.Namespace,
			Name:      p.Tag,
			Labels:    p.Labels,
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas:        &scaledReplicas,
			Selector:        metav1.SetAsLabelSelector(p.Labels),
			Template:        template,
			MinReadySeconds: p.MinReadySeconds,
		},
	}

	for _, i := range append(configs, replicaSet) {
		if err := plan.CreateOrUpdate(ctx, log, cli, i); err != nil {
			return err
		}
	}
	return nil
}

func DefaultDNSConfig() *corev1.PodDNSConfig {
	oneStr := "1"
	return &corev1.PodDNSConfig{
		Options: []corev1.PodDNSConfigOption{{
			Name:  "ndots",
			Value: &oneStr,
		}},
	}
}
