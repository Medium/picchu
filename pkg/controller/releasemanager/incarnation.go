package releasemanager

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/controller/utils"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/go-logr/logr"
	istiocommonv1alpha1 "github.com/knative/pkg/apis/istio/common/v1alpha1"
	istiov1alpha3 "github.com/knative/pkg/apis/istio/v1alpha3"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Controller interface {
	scheme() *runtime.Scheme
	client() client.Client
	releaseManager() *picchuv1alpha1.ReleaseManager
	log() logr.Logger
	getConfigMaps(context.Context, *client.ListOptions) (*corev1.ConfigMapList, error)
	getSecrets(context.Context, *client.ListOptions) (*corev1.SecretList, error)
	fleetSize() int32
}

type Incarnation struct {
	controller Controller
	tag        string
	revision   *picchuv1alpha1.Revision
	log        logr.Logger
	status     *picchuv1alpha1.ReleaseManagerRevisionStatus
}

func NewIncarnation(controller Controller, tag string, revision *picchuv1alpha1.Revision, log logr.Logger) Incarnation {
	status := controller.releaseManager().RevisionStatus(tag)
	if status.State.Target == "" || status.State.Target == "created" {
		if revision != nil {
			status.GitTimestamp = &metav1.Time{revision.GitTimestamp()}
			for _, target := range revision.Spec.Targets {
				if target.Name == controller.releaseManager().Spec.Target {
					status.ReleaseEligible = target.Release.Eligible
					status.TTL = target.Release.TTL
				}
			}
		} else {
			status.GitTimestamp = &metav1.Time{}
			status.ReleaseEligible = false
		}
		status.State.Target = "created"
	}

	var r picchuv1alpha1.Revision
	if revision != nil {
		r = *revision
	}
	return Incarnation{
		controller: controller,
		tag:        tag,
		revision:   &r,
		log:        log,
		status:     status,
	}
}

// = Start Deployment interface
// Remotely sync the incarnation for it's current state
func (i *Incarnation) sync() error {
	ctx := context.TODO()

	// Revision deleted
	if !i.hasRevision() {
		return nil
	}

	if i.target() == nil {
		return nil
	}

	configOpts, err := i.listOptions()
	if err != nil {
		return err
	}

	secrets, err := i.controller.getSecrets(ctx, configOpts)
	if err != nil {
		return err
	}
	configMaps, err := i.controller.getConfigMaps(ctx, configOpts)
	if err != nil {
		return err
	}

	sEnvs, err := i.syncSecrets(ctx, secrets)
	if err != nil {
		return err
	}
	cmEnvs, err := i.syncConfigMaps(ctx, configMaps)
	if err != nil {
		return err
	}

	if err = i.syncReplicaSet(ctx, append(sEnvs, cmEnvs...)); err != nil {
		return err
	}

	if err = i.syncHPA(ctx); err != nil {
		return err
	}
	return nil
}

func (i *Incarnation) getLog() logr.Logger {
	return i.log
}

func (i *Incarnation) getStatus() *picchuv1alpha1.ReleaseManagerRevisionStatus {
	return i.status
}

func (i *Incarnation) retire() error {
	ctx := context.TODO()
	rs := &appsv1.ReplicaSet{}
	s := types.NamespacedName{i.tag, i.targetNamespace()}
	err := i.controller.client().Get(ctx, s, rs)
	if err != nil {
		if errors.IsNotFound(err) {
			i.status.Scale.Current = 0
			i.status.Scale.Desired = 0
			return nil
		}
		i.log.Error(err, "Failed to update replicaset to 0 replicas")
		return err
	}
	var r int32 = 0
	rs.Spec.Replicas = &r
	err = i.controller.client().Update(ctx, rs)
	if err != nil {
		return err
	}
	i.recordHealthStatus(rs)
	return nil
}

func (i *Incarnation) del() error {
	ownerLabels := map[string]string{
		picchuv1alpha1.LabelTag: i.tag,
	}
	opts := client.
		MatchingLabels(ownerLabels).
		InNamespace(i.targetNamespace())
	ctx := context.TODO()

	lists := []List{
		NewSecretList(),
		NewConfigMapList(),
		NewReplicaSetList(),
		NewHorizontalPodAutoscalerList(),
	}
	for _, list := range lists {
		if err := i.controller.client().List(ctx, opts, list.GetList()); err != nil {
			log.Error(err, "Failed to list resource")
			return err
		}
		for _, item := range list.GetItems() {
			kind := utils.MustGetKind(i.controller.scheme(), item)
			log.Info("Deleting remote resource",
				"Cluster.Name", i.controller.releaseManager().Spec.Cluster,
				"Item.Kind", kind,
				"Item", item,
			)
			if err := i.controller.client().Delete(ctx, item); err != nil {
				log.Error(err, "Failed to delete resource", "Tag", i.tag, "Resource", list)
				return err
			}
		}
	}
	i.status.Resources = []picchuv1alpha1.ReleaseManagerRevisionResourceStatus{}
	i.status.CurrentPercent = 0
	i.status.Scale.Current = 0
	i.status.Scale.Desired = 0
	now := metav1.Now()
	i.status.LastUpdated = &now
	return nil
}

func (i *Incarnation) hasRevision() bool {
	return i.revision != nil
}

func (i *Incarnation) isAlarmTriggered() bool {
	return len(i.status.TriggeredAlarms) > 0
}

func (i *Incarnation) setState(target string, reached bool) {
	i.status.State.Target = target
	if reached {
		i.status.State.Current = target
	}
}

func (i *Incarnation) isReleaseEligible() bool {
	return i.status.ReleaseEligible
}

// = End Deployment interface

func (i *Incarnation) target() *picchuv1alpha1.RevisionTarget {
	if i.revision == nil {
		return nil
	}
	for _, target := range i.revision.Spec.Targets {
		if target.Name == i.controller.releaseManager().Spec.Target {
			return &target
		}
	}
	return nil
}

func (i *Incarnation) appName() string {
	return i.controller.releaseManager().Spec.App
}

func (i *Incarnation) targetNamespace() string {
	return i.controller.releaseManager().TargetNamespace()
}

func (i *Incarnation) image() string {
	return i.revision.Spec.App.Image
}

func (i *Incarnation) listOptions() (*client.ListOptions, error) {
	selector, err := metav1.LabelSelectorAsSelector(i.target().ConfigSelector)
	if err != nil {
		return nil, err
	}
	return &client.ListOptions{
		LabelSelector: selector,
		Namespace:     i.controller.releaseManager().Namespace,
	}, nil
}

func (i *Incarnation) defaultLabels() map[string]string {
	return map[string]string{
		picchuv1alpha1.LabelApp:    i.appName(),
		picchuv1alpha1.LabelTag:    i.tag,
		picchuv1alpha1.LabelTarget: i.target().Name,
	}
}

func (i *Incarnation) updateResourceStatus(resource runtime.Object) {
	apiVersion := ""
	kind := ""

	kinds, _, _ := scheme.Scheme.ObjectKinds(resource)
	if len(kinds) == 1 {
		apiVersion, kind = kinds[0].ToAPIVersionAndKind()
	}

	metadata, _ := apimeta.Accessor(resource)
	namespace := metadata.GetNamespace()
	name := metadata.GetName()

	i.status.CreateOrUpdateResourceStatus(picchuv1alpha1.ReleaseManagerRevisionResourceStatus{
		ApiVersion: apiVersion,
		Kind:       kind,
		Metadata:   &types.NamespacedName{namespace, name},
	})
}

func (i *Incarnation) removeResourceStatus(resource runtime.Object) {
	apiVersion := ""
	kind := ""

	kinds, _, _ := scheme.Scheme.ObjectKinds(resource)
	if len(kinds) == 1 {
		apiVersion, kind = kinds[0].ToAPIVersionAndKind()
	}

	metadata, _ := apimeta.Accessor(resource)
	namespace := metadata.GetNamespace()
	name := metadata.GetName()

	i.status.RemoveResourceStatus(picchuv1alpha1.ReleaseManagerRevisionResourceStatus{
		ApiVersion: apiVersion,
		Kind:       kind,
		Metadata:   &types.NamespacedName{namespace, name},
	})
}

func (i *Incarnation) setReleaseEligible(flag bool) {
	i.status.ReleaseEligible = flag
}

func (i *Incarnation) recordHealthStatus(replicaSet *appsv1.ReplicaSet) {
	desired := replicaSet.Spec.Replicas
	current := replicaSet.Status.AvailableReplicas
	i.status.Scale.Desired = *desired
	i.status.Scale.Current = current
	if current > i.status.Scale.Peak {
		i.status.Scale.Peak = current
	}
}

func (i *Incarnation) taggedRoutes(privateGateway string, serviceHost string) []istiov1alpha3.HTTPRoute {
	http := []istiov1alpha3.HTTPRoute{}
	if i.revision == nil {
		return http
	}
	overrideLabel := fmt.Sprintf("pin/%s", i.appName())
	for _, port := range i.revision.Spec.Ports {
		matches := []istiov1alpha3.HTTPMatchRequest{{
			// mesh traffic from same tag'd service with and test tag
			SourceLabels: map[string]string{
				overrideLabel: i.tag,
			},
			Port:     uint32(port.Port),
			Gateways: []string{"mesh"},
		}}
		if privateGateway != "" {
			matches = append(matches,
				istiov1alpha3.HTTPMatchRequest{
					// internal traffic with MEDIUM-TAG header
					Headers: map[string]istiocommonv1alpha1.StringMatch{
						"Medium-Tag": {Exact: i.tag},
					},
					Port:     uint32(port.IngressPort),
					Gateways: []string{privateGateway},
				},
			)
		}

		http = append(http, istiov1alpha3.HTTPRoute{
			Match: matches,
			Route: []istiov1alpha3.DestinationWeight{
				{
					Destination: istiov1alpha3.Destination{
						Host:   serviceHost,
						Port:   istiov1alpha3.PortSelector{Number: uint32(port.Port)},
						Subset: i.tag,
					},
					Weight: 100,
				},
			},
		})
	}
	return http
}

func (i *Incarnation) syncSecrets(ctx context.Context, secrets *corev1.SecretList) ([]corev1.EnvFromSource, error) {
	envs := []corev1.EnvFromSource{}

	for _, item := range secrets.Items {
		remote := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      item.Name,
				Namespace: i.targetNamespace(),
				Labels:    i.defaultLabels(),
			},
		}
		op, err := controllerutil.CreateOrUpdate(ctx, i.controller.client(), remote, func(runtime.Object) error {
			remote.Data = item.Data
			return nil
		})
		if err != nil {
			i.log.Error(err, "Failed to sync secret")
			return envs, err
		}
		i.updateResourceStatus(remote)

		i.log.Info("Secret sync'd", "Op", op)
		envs = append(envs, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: remote.Name},
			},
		})
	}

	return envs, nil
}

func (i *Incarnation) syncConfigMaps(ctx context.Context, configMaps *corev1.ConfigMapList) ([]corev1.EnvFromSource, error) {
	envs := []corev1.EnvFromSource{}

	for _, item := range configMaps.Items {
		remote := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      item.Name,
				Namespace: i.targetNamespace(),
				Labels:    i.defaultLabels(),
			},
		}
		op, err := controllerutil.CreateOrUpdate(ctx, i.controller.client(), remote, func(runtime.Object) error {
			remote.Data = item.Data
			return nil
		})
		if err != nil {
			i.log.Error(err, "Failed to sync configMap")
			return envs, err
		}
		i.updateResourceStatus(remote)

		i.log.Info("ConfigMap sync'd", "Op", op)
		envs = append(envs, corev1.EnvFromSource{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: remote.Name},
			},
		})
	}

	return envs, nil
}

func (i *Incarnation) syncPrometheusRules(ctx context.Context) error {
	rule := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-rule",
			Namespace: i.targetNamespace(),
		},
	}
	if i.target() != nil && len(i.target().AlertRules) > 0 {
		op, err := controllerutil.CreateOrUpdate(ctx, i.controller.client(), rule, func(runtime.Object) error {
			rule.Labels = map[string]string{picchuv1alpha1.LabelTag: i.tag}
			rule.Spec.Groups = []monitoringv1.RuleGroup{{
				Name:  "picchu.rules",
				Rules: i.target().AlertRules,
			}}
			return nil
		})
		if err != nil {
			i.log.Info("Failed to sync'd PrometheusRule")
			return err
		}
		i.log.Info("Sync'd PrometheusRule", "Op", op)
		i.updateResourceStatus(rule)
	} else {
		if err := i.controller.client().Delete(ctx, rule); err != nil && !errors.IsNotFound(err) {
			i.log.Info("Failed to delete PrometheusRule")
			return err
		}
		i.log.Info("Deleted PrometheusRule")
		i.removeResourceStatus(rule)
	}
	return nil
}

func (i *Incarnation) replicaSet(envs []corev1.EnvFromSource) *appsv1.ReplicaSet {
	target := i.target()
	ports := []corev1.ContainerPort{}
	hasStatusPort := false
	for _, port := range i.revision.Spec.Ports {
		ports = append(ports, corev1.ContainerPort{
			Name:          port.Name,
			Protocol:      port.Protocol,
			ContainerPort: port.ContainerPort,
		})
		if port.Name == "status" {
			hasStatusPort = true
		}
	}
	podAnnotations := make(map[string]string, 1)
	if role := target.AWS.IAM.RoleARN; role != "" {
		podAnnotations[picchuv1alpha1.AnnotationIAMRole] = role
	}
	// Used for Container label and Container Selector
	podLabels := map[string]string{
		picchuv1alpha1.LabelApp: i.appName(),
		picchuv1alpha1.LabelTag: i.tag,
	}
	replicaCount := *i.divideReplicas(target.Scale.Default)

	appContainer := corev1.Container{
		EnvFrom:   envs,
		Image:     i.image(),
		Name:      i.appName(),
		Ports:     ports,
		Resources: target.Resources,
	}
	if hasStatusPort {
		appContainer.LivenessProbe = &corev1.Probe{
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
		appContainer.ReadinessProbe = &corev1.Probe{
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

	i.log.Info("Creating ReplicaSet", "Replicas", target.Scale.Default)
	return &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      i.tag,
			Namespace: i.targetNamespace(),
			Labels:    i.defaultLabels(),
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: &replicaCount,
			Selector: metav1.SetAsLabelSelector(podLabels),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        i.tag,
					Namespace:   i.targetNamespace(),
					Annotations: podAnnotations,
					Labels:      podLabels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: target.ServiceAccountName,
					Containers:         []corev1.Container{appContainer},
				},
			},
		},
	}
}

func (i *Incarnation) syncReplicaSet(ctx context.Context, envs []corev1.EnvFromSource) error {
	rs := i.replicaSet(envs)
	op, err := controllerutil.CreateOrUpdate(ctx, i.controller.client(), rs, func(runtime.Object) error {
		if *rs.Spec.Replicas == 0 {
			var one int32 = 1
			rs.Spec.Replicas = &one
		}
		return nil
	})
	i.log.Info("ReplicaSet sync'd", "Op", op)
	if err != nil {
		log.Error(err, "Failed to sync replicaSet")
		return err
	}
	i.updateResourceStatus(rs)
	i.recordHealthStatus(rs)
	return nil
}

func (i *Incarnation) deleteIfExists(ctx context.Context, obj runtime.Object) error {
	err := i.controller.client().Delete(ctx, obj)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

func (i *Incarnation) syncHPA(ctx context.Context) error {
	target := i.target()
	cpuTarget := target.Scale.TargetCPUUtilizationPercentage
	if cpuTarget != nil && *cpuTarget == 0 {
		hpa := &autoscalingv1.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      i.tag,
				Namespace: i.targetNamespace(),
			},
		}
		if err := i.deleteIfExists(ctx, hpa); err != nil {
			return err
		}
		return nil
	}

	// We don't want to share the value between the target and the hpa, so
	// we have to make a copy
	var min *int32
	if target.Scale.Min != nil {
		m := *target.Scale.Min
		min = &m
	}
	hpa := &autoscalingv1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      i.tag,
			Namespace: i.targetNamespace(),
			Labels:    i.defaultLabels(),
		},
		Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
				Kind:       "ReplicaSet",
				Name:       i.tag,
				APIVersion: "apps/v1",
			},
			MinReplicas:                    i.divideReplicas(*min),
			MaxReplicas:                    *i.divideReplicas(target.Scale.Max),
			TargetCPUUtilizationPercentage: cpuTarget,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, i.controller.client(), hpa, func(runtime.Object) error {
		return nil
	})
	log.Info("HPA sync'd", "Op", op)
	if err != nil {
		log.Error(err, "Failed to sync hpa")
		return err
	}
	i.updateResourceStatus(hpa)
	return nil
}

func (i *Incarnation) divideReplicas(count int32) *int32 {
	var current int32 = 100
	if i.status.ReleaseEligible {
		current = int32(i.status.CurrentPercent)
	}
	r := utils.Max((count*current)/100*i.controller.fleetSize(), 1)
	return &r
}

func (i *Incarnation) currentPercentTarget(max uint32) uint32 {
	if i.revision == nil {
		return 0
	}
	status := i.status
	target := i.target()
	current := status.CurrentPercent
	// TODO(bob): gate increment on current time for "humane" scheduled
	// deployments that are currently at 0 percent of traffic
	if max <= 0 {
		return 0
	}
	delay := time.Duration(*target.Release.Rate.DelaySeconds) * time.Second
	increment := target.Release.Rate.Increment
	if target.Release.Max < max {
		max = target.Release.Max
	}
	deadline := time.Time{}
	if status.LastUpdated != nil {
		deadline = status.LastUpdated.Add(delay)
	}
	if deadline.After(time.Now()) {
		return current
	}

	if !schedulePermitsRelease(time.Now(), target.Release.Schedule) {
		return current
	}

	current = current + increment
	if current > max {
		return max
	}
	return current
}

func (i *Incarnation) updateCurrentPercent(current uint32) {
	if i.status.CurrentPercent != current {
		now := metav1.Now()
		i.status.CurrentPercent = current
		i.status.LastUpdated = &now
	}
	if current > i.status.PeakPercent {
		i.status.PeakPercent = current
	}
}

func (i *Incarnation) secondsSinceRevision() float64 {
	start := i.revision.CreationTimestamp
	rate := i.target().Release.Rate
	delay := *rate.DelaySeconds
	increment := rate.Increment
	expected := time.Second * time.Duration(delay*int64(math.Ceil(100.0/float64(increment))))
	latency := time.Since(start.Time) - expected
	return latency.Seconds()
}

// IncarnationCollection helps us collect and select appropriate incarnations
type IncarnationCollection struct {
	// Incarnations key'd on revision.spec.app.tag
	itemSet    map[string]Incarnation
	controller Controller
}

func newIncarnationCollection(
	controller Controller,
) *IncarnationCollection {
	ic := &IncarnationCollection{
		controller: controller,
		itemSet:    make(map[string]Incarnation),
	}
	// First seed all known revisions from status, since some might have been
	// deleted and don't have associated resources that are Add'd. If an
	// Incarnation has a nil Revision, it means it's been deleted.
	rm := controller.releaseManager()
	for _, r := range rm.Status.Revisions {
		l := controller.log().WithValues("Incarnation.Tag", r.Tag)
		ic.itemSet[r.Tag] = NewIncarnation(controller, r.Tag, nil, l)
	}
	return ic
}

// Add adds a new Incarnation to the IncarnationManager
func (i *IncarnationCollection) add(revision *picchuv1alpha1.Revision) {
	l := i.controller.log().WithValues("Incarnation.Tag", revision.Spec.App.Tag)
	i.itemSet[revision.Spec.App.Tag] = NewIncarnation(i.controller, revision.Spec.App.Tag, revision, l)
}

func (i *IncarnationCollection) ensureValidRelease() {
	r := []Incarnation{}
	for _, i := range i.sorted() {
		state := i.status.State.Current
		if (state == "deployed" || state == "released") && i.isReleaseEligible() {
			r = append(r, i)
		}
	}

	if len(r) == 0 {
		i.controller.log().Info("there are no releases, looking for retired release to unretire")
		candidates := i.unretirable()
		if len(candidates) > 0 {
			i.controller.log().Info("Unretiring", "tag", candidates[0].tag)
			candidates[0].setReleaseEligible(true)
		} else {
			i.controller.log().Info("No available releases retired")
		}
	}
}

// deployed returns deployed incarnation
func (i *IncarnationCollection) deployed() []Incarnation {
	r := []Incarnation{}
	for _, i := range i.sorted() {
		state := i.status.State.Current
		if state == "deployed" || state == "released" {
			r = append(r, i)
		}
	}
	return r
}

// releasable returns deployed release eligible incarnations in order from
// releaseEligible to !releaseEligible, then latest to oldest
func (i *IncarnationCollection) releasable() []Incarnation {
	r := []Incarnation{}
	for _, i := range i.sorted() {
		if i.status.State.Target == "released" && i.isReleaseEligible() {
			r = append(r, i)
		}
	}
	for _, i := range i.sorted() {
		if i.status.State.Target == "released" && !i.isReleaseEligible() {
			r = append(r, i)
		}
	}
	return r
}

func (i *IncarnationCollection) unretirable() []Incarnation {
	r := []Incarnation{}
	for _, i := range i.sorted() {
		cur := i.status.State.Current
		elig := i.isReleaseEligible()
		triggered := i.isAlarmTriggered()
		if cur == "retired" || (cur == "released" && !elig && !triggered) {
			r = append(r, i)
		}
	}
	return r
}

func (i *IncarnationCollection) revisioned() []Incarnation {
	r := []Incarnation{}
	for _, i := range i.sorted() {
		if i.revision != nil {
			r = append(r, i)
		}
	}
	return r
}

func (i *IncarnationCollection) sorted() []Incarnation {
	r := []Incarnation{}
	for _, i := range i.itemSet {
		r = append(r, i)
	}

	sort.Slice(r, func(i, j int) bool {
		a := time.Time{}
		b := time.Time{}
		if r[i].revision != nil {
			a = r[i].revision.GitTimestamp()
		}
		if r[j].revision != nil {
			b = r[j].revision.GitTimestamp()
		}
		return a.After(b)
	})
	return r
}
