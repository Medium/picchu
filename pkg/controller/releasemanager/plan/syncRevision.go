package plan

import (
	"context"
	"errors"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	livenessProbe  *corev1.Probe
	readinessProbe *corev1.Probe
)

// TODO(bob): Move to Revision spec
func init() {
	livenessProbe = &corev1.Probe{
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
		FailureThreshold:    7, // FIXME(lyra)
	}

	readinessProbe = &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/running", // FIXME(lyra)
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
}

func (p *SyncRevision) Apply(ctx context.Context, cli client.Client, log logr.Logger) error {
	envs := []corev1.EnvFromSource{}

	for _, i := range p.Configs {
		switch resource := i.(type) {
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
			log.Error(e, "config", i)
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
		EnvFrom:   envs,
		Image:     p.Image,
		Name:      p.App,
		Ports:     ports,
		Resources: p.Resources,
	}

	if hasStatusPort {
		appContainer.LivenessProbe = livenessProbe
		appContainer.ReadinessProbe = readinessProbe
	}

	podLabels := map[string]string{
		picchuv1alpha1.LabelTag: p.Tag,
		picchuv1alpha1.LabelApp: p.App,
	}

	template := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name:        p.Tag,
			Namespace:   p.Namespace,
			Labels:      podLabels,
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

	copyReplicas := p.Replicas
	replicaSet := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: p.Namespace,
			Name:      p.Tag,
			Labels:    p.Labels,
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: &copyReplicas,
			Selector: metav1.SetAsLabelSelector(podLabels),
			Template: template,
		},
	}

	for _, i := range append(p.Configs, replicaSet) {
		orig := i.DeepCopyObject()
		op, err := controllerutil.CreateOrUpdate(ctx, cli, i, func(runtime.Object) error {
			switch obj := i.(type) {
			case *corev1.Secret:
				obj.Data = orig.(*corev1.Secret).Data
				obj.Labels = p.Labels
			case *corev1.ConfigMap:
				obj.Data = orig.(*corev1.ConfigMap).Data
				obj.Labels = p.Labels
			case *appsv1.ReplicaSet:
				obj.Spec = orig.(*appsv1.ReplicaSet).Spec
				obj.Labels = p.Labels
			default:
				e := errors.New("Unknown type")
				log.Error(e, "item", i)
				return e
			}
			return nil
		})
		LogSync(log, op, err, i)
		if err != nil {
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
