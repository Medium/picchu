package plan

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"

	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/plan"

	es "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const annotationDatadogTolerateUnready = "ad.datadoghq.com/tolerate-unready"

var (
	defaultLivenessProbe  *corev1.Probe
	defaultReadinessProbe *corev1.Probe
)

// TODO(bob): Move to Revision spec
func init() {
	defaultLivenessProbe = &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
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
		ProbeHandler: corev1.ProbeHandler{
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
	App                 string
	Tag                 string
	Namespace           string
	Labels              map[string]string // Labels applied to all resources
	Configs             []runtime.Object  // Secret and ConfigMap objects supported and mapped to environment
	Ports               []picchuv1alpha1.PortInfo
	Replicas            int32
	Image               string
	Resources           corev1.ResourceRequirements
	IAMRole             string            // AWS iam role
	PodAnnotations      map[string]string // metadata.annotations in the Pod template
	ServiceAccountName  string            // k8s ServiceAccount
	LivenessProbe       *corev1.Probe
	ReadinessProbe      *corev1.Probe
	MinReadySeconds     int32
	Worker              *picchuv1alpha1.WorkerScaleInfo
	KedaWorker          *picchuv1alpha1.KedaScaleInfo
	Lifecycle           *corev1.Lifecycle
	PriorityClassName   string
	Affinity            *corev1.Affinity
	Tolerations         []corev1.Toleration
	EnvVars             []corev1.EnvVar
	Sidecars            []corev1.Container // Additional sidecar containers.
	VolumeMounts        []corev1.VolumeMount
	Volumes             []corev1.Volume
	PodDisruptionBudget *policyv1.PodDisruptionBudget
	ExternalSecrets     []es.ExternalSecret
	EventDriven         bool
}

func (p *SyncRevision) Printable() interface{} {
	return struct {
		App                string
		Tag                string
		Namespace          string
		Labels             map[string]string
		Ports              []picchuv1alpha1.PortInfo
		Replicas           int32
		Image              string
		Resources          corev1.ResourceRequirements
		IAMRole            string            // AWS iam role
		PodAnnotations     map[string]string // metadata.annotations in the Pod template
		ServiceAccountName string            // k8s ServiceAccount
		LivenessProbe      *corev1.Probe
		ReadinessProbe     *corev1.Probe
		MinReadySeconds    int32
		Lifecycle          *corev1.Lifecycle
		Affinity           *corev1.Affinity
		PriorityClassName  string
		EventDriven        bool
	}{

		App:                p.App,
		Tag:                p.Tag,
		Namespace:          p.Namespace,
		Labels:             p.Labels,
		Ports:              p.Ports,
		Replicas:           p.Replicas,
		Image:              p.Image,
		Resources:          p.Resources,
		IAMRole:            p.IAMRole,
		PodAnnotations:     p.PodAnnotations,
		ServiceAccountName: p.ServiceAccountName,
		LivenessProbe:      p.LivenessProbe,
		ReadinessProbe:     p.ReadinessProbe,
		MinReadySeconds:    p.MinReadySeconds,
		Lifecycle:          p.Lifecycle,
		Affinity:           p.Affinity,
		PriorityClassName:  p.PriorityClassName,
		EventDriven:        p.EventDriven,
	}
}

func (p *SyncRevision) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {

	// Clone passed in objects to prevent concurrency issues
	var configs []runtime.Object
	var envs []corev1.EnvFromSource
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
			e := errors.New("unsupported config")
			log.Error(e, "Unsupported config", "Config", config)
			return e
		}
	}

	if err := p.patchServiceAccount(ctx, cli, log); err != nil {
		return err
	}

	if p.PodDisruptionBudget == nil {
		maxUnavailable := intstr.FromString(picchuv1alpha1.MaxUnavailable)
		p.PodDisruptionBudget = &policyv1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{
				Name:      p.App,
				Namespace: p.Namespace,
				Labels:    p.Labels,
			},
			Spec: policyv1.PodDisruptionBudgetSpec{
				MaxUnavailable: &maxUnavailable,
			},
		}
	}
	if err := p.syncPodDisruptionBudget(ctx, cli, log); err != nil {
		return err
	}

	for i := range configs {
		config := configs[i]
		if err := plan.CreateOrUpdate(ctx, log, cli, config); err != nil {
			return err
		}
	}

	if p.ExternalSecrets != nil {
		var externalSecrets []runtime.Object
		for i := range p.ExternalSecrets {
			externalSecrets = append(externalSecrets, p.ExternalSecrets[i].DeepCopyObject())
		}

		for i := range externalSecrets {
			externalSecret := externalSecrets[i]
			resource := externalSecret.(*es.ExternalSecret)
			resource.ObjectMeta = metav1.ObjectMeta{
				Name:      resource.Name,
				Namespace: p.Namespace,
				Labels:    p.Labels,
			}
			envs = append(envs, corev1.EnvFromSource{
				SecretRef: &corev1.SecretEnvSource{
					// reference target secret that the ExternalSecret will generate
					LocalObjectReference: corev1.LocalObjectReference{Name: resource.Spec.Target.Name},
				},
			})

			if err := plan.CreateOrUpdate(ctx, log, cli, externalSecret); err != nil {
				return err
			}
		}
	}

	// This is required for DestinationRule mapping, which doesn't allow slashes in label names.
	labels := map[string]string{
		"tag.picchu.medium.engineering": p.Tag,
	}
	for k, v := range p.Labels {
		labels[k] = v
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
		e := fmt.Errorf("cluster scalingFactor is nil")
		log.Error(e, "Failed to sync revision")
		return e
	}

	if cluster.Spec.DisableEventDriven && p.EventDriven {
		log.Info(fmt.Sprintf("Event driven scaling is disabled for this cluster:%s, ignoring event driven scaling for %s, Namespace: %s", cluster.Name, p.Tag, p.Namespace))
		f := 0.0
		scalingFactor = &f
	}

	return p.syncReplicaSet(ctx, cli, *scalingFactor, labels, envs, log)
}

func (p *SyncRevision) syncReplicaSet(
	ctx context.Context,
	cli client.Client,
	scalingFactor float64,
	podLabels map[string]string,
	envs []corev1.EnvFromSource,
	log logr.Logger,
) error {
	var livenessProbe *corev1.Probe
	var readinessProbe *corev1.Probe
	var lifecycle *corev1.Lifecycle
	if p.LivenessProbe != nil {
		probe := *p.LivenessProbe
		livenessProbe = &probe
	}
	if p.ReadinessProbe != nil {
		probe := *p.ReadinessProbe
		readinessProbe = &probe
	}
	if p.Lifecycle != nil {
		lc := *p.Lifecycle
		lifecycle = &lc
	}

	var ports []corev1.ContainerPort
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

	sort.Slice(ports, func(i, j int) bool {
		return ports[i].ContainerPort < ports[j].ContainerPort
	})

	appContainer := corev1.Container{
		EnvFrom:        envs,
		Env:            p.EnvVars,
		Image:          p.Image,
		Name:           p.App,
		Ports:          ports,
		Resources:      p.Resources,
		LivenessProbe:  livenessProbe,
		ReadinessProbe: readinessProbe,
		Lifecycle:      lifecycle,
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

	var containers []corev1.Container
	containers = append(containers, appContainer)
	containers = append(containers, p.Sidecars...)

	for i := range containers {
		containers[i].EnvFrom = envs
		containers[i].Env = p.EnvVars
		containers[i].VolumeMounts = p.VolumeMounts
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
			Containers:         containers,
			DNSConfig:          DefaultDNSConfig(),
			Affinity:           p.Affinity,
			PriorityClassName:  p.PriorityClassName,
			Tolerations:        p.Tolerations,
			Volumes:            p.Volumes,
		},
	}

	if termGraceStr := p.PodAnnotations["go.medium.engineering/PodTerminationGracePeriodSeconds"]; termGraceStr != "" {
		terminationGracePeriodSeconds, err := strconv.Atoi(termGraceStr)
		if err == nil {
			termGraceInt64 := int64(terminationGracePeriodSeconds)
			template.Spec.TerminationGracePeriodSeconds = &termGraceInt64
		}
	}

	for ann, value := range p.PodAnnotations {
		template.Annotations[ann] = value
	}

	autoScaler := picchuv1alpha1.AutoscalerTypeHPA
	if p.Worker != nil {
		autoScaler = picchuv1alpha1.AutoscalerTypeWPA
	} else if p.KedaWorker != nil {
		autoScaler = picchuv1alpha1.AutoscalerTypeKEDA
	}

	rsAnnotations := map[string]string{
		picchuv1alpha1.AnnotationAutoscaler: autoScaler,
	}

	if p.IAMRole != "" {
		template.Annotations[picchuv1alpha1.AnnotationIAMRole] = p.IAMRole
		rsAnnotations[picchuv1alpha1.AnnotationIAMRole] = p.IAMRole
	}

	template.Annotations[annotationDatadogTolerateUnready] = "true"

	scaledReplicas := int32(math.Ceil(float64(p.Replicas) * scalingFactor))

	// Safe to remove once no existing ReplicaSets are missing the new picchuv1alpha1.LabelIstioApp label
	tempSelector := make(map[string]string)
	for k, v := range podLabels {
		if k != picchuv1alpha1.LabelIstioApp && k != picchuv1alpha1.LabelIstioVersion {
			tempSelector[k] = v
		}
	}
	// End badness

	replicaSet := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   p.Namespace,
			Name:        p.Tag,
			Labels:      p.Labels,
			Annotations: rsAnnotations,
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas:        &scaledReplicas,
			Selector:        metav1.SetAsLabelSelector(tempSelector),
			Template:        template,
			MinReadySeconds: p.MinReadySeconds,
		},
	}

	return plan.CreateOrUpdate(ctx, log, cli, replicaSet)
}

func (p *SyncRevision) syncPodDisruptionBudget(ctx context.Context, cli client.Client, log logr.Logger) error {
	p.PodDisruptionBudget.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			picchuv1alpha1.LabelApp: p.App,
		},
	}
	return plan.CreateOrUpdate(ctx, log, cli, p.PodDisruptionBudget)
}

func (p *SyncRevision) patchServiceAccount(ctx context.Context, cli client.Client, log logr.Logger) error {
	sa := &corev1.ServiceAccount{}
	if err := cli.Get(ctx, client.ObjectKey{Namespace: p.Namespace, Name: p.App}, sa); err != nil {
		return err
	}

	if sa.Annotations == nil {
		sa.Annotations = map[string]string{}
	}

	if p.IAMRole == "" {
		//Skipping ServiceAccount patch, IAMRole is empty
		return nil
	}

	if sa.Annotations[picchuv1alpha1.AnnotationKedaServiceAccount] == p.IAMRole {
		// ServiceAccount already has IAMRole annotation
		return nil
	}

	log.Info("Patching ServiceAccount with annotation", "key", picchuv1alpha1.AnnotationKedaServiceAccount, "value", p.IAMRole)
	sa.Annotations[picchuv1alpha1.AnnotationKedaServiceAccount] = p.IAMRole
	return cli.Update(ctx, sa)
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
