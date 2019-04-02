package releasemanager

import (
	"context"
	"math"
	"sort"
	"time"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/controller/utils"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type IncarnationStatus *picchuv1alpha1.ReleaseManagerRevisionStatus

type Incarnation struct {
	tag            string
	releaseManager *picchuv1alpha1.ReleaseManager
	revision       *picchuv1alpha1.Revision
	cluster        *picchuv1alpha1.Cluster
	client         client.Client
	configFetcher  *ConfigFetcher
	scheme         *runtime.Scheme
	log            logr.Logger
}

func (i *Incarnation) target() *picchuv1alpha1.RevisionTarget {
	for _, target := range i.revision.Spec.Targets {
		if target.Name == i.releaseManager.Spec.Target {
			return &target
		}
	}
	return nil
}

func (i *Incarnation) appName() string {
	return i.releaseManager.Spec.App
}

func (i *Incarnation) targetNamespace() string {
	return i.releaseManager.TargetNamespace()
}

func (i *Incarnation) image() string {
	return i.revision.Spec.App.Image
}

func (i *Incarnation) status() *picchuv1alpha1.ReleaseManagerRevisionStatus {
	return i.releaseManager.RevisionStatus(i.tag)
}

func (i *Incarnation) isDeployed() bool {
	return i.revision != nil && i.status().Deployed
}

func (i *Incarnation) wasEverDeployed() bool {
	return i.status().EverDeployed
}

func (i *Incarnation) isReleased() bool {
	return i.revision != nil && i.status().Released
}

func (i *Incarnation) isRetired() bool {
	return i.status().Retired
}

func (i *Incarnation) wasEverReleased() bool {
	return i.status().EverReleased
}

func (i *Incarnation) isReleaseEligible() bool {
	return i.revision != nil && i.target().Release.Eligible
}

func (i *Incarnation) listOptions() (*client.ListOptions, error) {
	selector, err := metav1.LabelSelectorAsSelector(i.target().ConfigSelector)
	if err != nil {
		return nil, err
	}
	return &client.ListOptions{
		LabelSelector: selector,
		Namespace:     i.releaseManager.Namespace,
	}, nil
}

func (i *Incarnation) defaultLabels() map[string]string {
	return map[string]string{
		picchuv1alpha1.LabelApp:    i.appName(),
		picchuv1alpha1.LabelTag:    i.tag,
		picchuv1alpha1.LabelTarget: i.target().Name,
	}
}

func (i *Incarnation) recordResourceStatus(resource runtime.Object, status string) {
	apiVersion := ""
	kind := ""

	kinds, _, _ := scheme.Scheme.ObjectKinds(resource)
	if len(kinds) == 1 {
		apiVersion, kind = kinds[0].ToAPIVersionAndKind()
	}

	metadata, _ := apimeta.Accessor(resource)
	namespace := metadata.GetNamespace()
	name := metadata.GetName()

	rs := i.status()
	rs.CreateOrUpdateResourceStatus(picchuv1alpha1.ReleaseManagerRevisionResourceStatus{
		ApiVersion: apiVersion,
		Kind:       kind,
		Metadata:   &types.NamespacedName{namespace, name},
		Status:     status,
	})
	i.releaseManager.UpdateRevisionStatus(rs)
}

func (i *Incarnation) recordDeleted() {
	rs := i.status()
	rs.Resources = []picchuv1alpha1.ReleaseManagerRevisionResourceStatus{}
	rs.CurrentPercent = 0
	rs.Retired = true
	rs.Deployed = false
	rs.Released = false
	rs.Scale.Current = 0
	rs.Scale.Desired = 0
	now := metav1.Now()
	rs.LastUpdate = &now
	i.releaseManager.UpdateRevisionStatus(rs)
}

func (i *Incarnation) recordExpired() {
	rs := i.status()
	rs.Expired = true
	now := metav1.Now()
	rs.LastUpdate = &now
	i.releaseManager.UpdateRevisionStatus(rs)
}

func (i *Incarnation) recordHealthStatus(replicaSet *appsv1.ReplicaSet) {
	rs := i.status()
	desired := replicaSet.Spec.Replicas
	current := replicaSet.Status.AvailableReplicas
	if current > 0 {
		rs.Deployed = true
	}
	if rs.Deployed {
		rs.EverDeployed = true
	}
	rs.Scale.Desired = *desired
	rs.Scale.Current = current
	if current > rs.Scale.Peak {
		rs.Scale.Peak = current
	}
	i.releaseManager.UpdateRevisionStatus(rs)
}

func (i *Incarnation) recordReleasedStatus(currentPercent uint32, isReleased bool, everReleased bool) {
	status := i.status()
	status.Released = isReleased
	status.EverReleased = status.EverReleased || everReleased
	if status.EverReleased && !isReleased {
		status.Retired = true
	}
	if status.CurrentPercent != currentPercent {
		now := metav1.Now()
		status.CurrentPercent = currentPercent
		status.LastUpdate = &now
	}
	if currentPercent > status.PeakPercent {
		status.PeakPercent = currentPercent
	}
	i.releaseManager.UpdateRevisionStatus(status)
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
		op, err := controllerutil.CreateOrUpdate(ctx, i.client, remote, func(runtime.Object) error {
			remote.Data = item.Data
			return nil
		})
		status := "created"
		if err != nil {
			status = "failed"
			i.log.Error(err, "Failed to sync secret")
			return envs, err
		}
		i.recordResourceStatus(remote, status)

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
		op, err := controllerutil.CreateOrUpdate(ctx, i.client, remote, func(runtime.Object) error {
			remote.Data = item.Data
			return nil
		})
		status := "created"
		if err != nil {
			status = "failed"
			i.log.Error(err, "Failed to sync configMap")
			return envs, err
		}
		i.recordResourceStatus(remote, status)

		i.log.Info("ConfigMap sync'd", "Op", op)
		envs = append(envs, corev1.EnvFromSource{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: remote.Name},
			},
		})
	}

	return envs, nil
}

func (i *Incarnation) replicaSet(envs []corev1.EnvFromSource) *appsv1.ReplicaSet {
	target := i.target()
	ports := []corev1.ContainerPort{}
	for _, port := range i.revision.Spec.Ports {
		ports = append(ports, corev1.ContainerPort{
			Name:          port.Name,
			Protocol:      port.Protocol,
			ContainerPort: port.ContainerPort,
		})
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
	i.log.Info("Creating ReplicaSet", "Replicas", target.Scale.Default)
	return &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      i.tag,
			Namespace: i.targetNamespace(),
			Labels:    i.defaultLabels(),
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: &target.Scale.Default,
			Selector: metav1.SetAsLabelSelector(podLabels),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        i.tag,
					Namespace:   i.targetNamespace(),
					Annotations: podAnnotations,
					Labels:      podLabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						EnvFrom:   envs,
						Image:     i.image(),
						Name:      i.appName(),
						Ports:     ports,
						Resources: target.Resources,
					}},
				},
			},
		},
	}
}

func (i *Incarnation) syncReplicaSet(ctx context.Context, envs []corev1.EnvFromSource) error {
	status := "created"
	rs := i.replicaSet(envs)
	op, err := controllerutil.CreateOrUpdate(ctx, i.client, rs, func(runtime.Object) error {
		return nil
	})
	i.log.Info("ReplicaSet sync'd", "Op", op)
	if err != nil {
		status = "failed"
		log.Error(err, "Failed to sync replicaSet")
		return err
	}
	i.recordResourceStatus(rs, status)
	i.recordHealthStatus(rs)
	return nil
}

func (i *Incarnation) deleteIfExists(ctx context.Context, obj runtime.Object) error {
	err := i.client.Delete(ctx, obj)
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
			MinReplicas:                    target.Scale.Min,
			MaxReplicas:                    target.Scale.Max,
			TargetCPUUtilizationPercentage: cpuTarget,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, i.client, hpa, func(runtime.Object) error {
		return nil
	})
	log.Info("HPA sync'd", "Op", op)
	if err != nil {
		i.recordResourceStatus(hpa, "failed")
		log.Error(err, "Failed to sync hpa")
		return err
	} else {
		i.recordResourceStatus(hpa, "created")
	}
	return nil
}

// Remotely sync the incarnation for it's current state
func (i *Incarnation) sync() error {
	if i.revision == nil {
		i.recordDeleted()
		return i.del()
	}
	ctx := context.TODO()

	configOpts, err := i.listOptions()
	if err != nil {
		return err
	}

	secrets, err := i.configFetcher.GetSecrets(ctx, configOpts)
	if err != nil {
		return err
	}
	configMaps, err := i.configFetcher.GetConfigMaps(ctx, configOpts)
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
		if err := i.client.List(ctx, opts, list.GetList()); err != nil {
			log.Error(err, "Failed to list resource")
			return err
		}
		for _, item := range list.GetItems() {
			kind := utils.MustGetKind(i.scheme, item)
			log.Info("Deleting remote resource",
				"Cluster.Name", i.releaseManager.Spec.Cluster,
				"Item.Kind", kind,
				"Item", item,
			)
			if err := i.client.Delete(ctx, item); err != nil {
				log.Error(err, "Failed to delete resource", "Tag", i.tag, "Resource", list)
				return err
			}
		}
	}
	return nil
}

func (i *Incarnation) currentPercentTarget(max uint32) uint32 {
	status := i.status()
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
	// This must be a rollback
	if i.wasEverReleased() {
		return max
	}
	deadline := time.Time{}
	if status.LastUpdate != nil {
		deadline = status.LastUpdate.Add(delay)
	}

	if deadline.After(time.Now()) {
		return current
	}
	current = current + increment
	if current > max {
		return max
	}
	return current
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
	itemSet        map[string]Incarnation
	releaseManager *picchuv1alpha1.ReleaseManager
	cluster        *picchuv1alpha1.Cluster
	configFetcher  *ConfigFetcher
	client         client.Client
	scheme         *runtime.Scheme
	log            logr.Logger
}

func newIncarnationCollection(
	rm *picchuv1alpha1.ReleaseManager,
	cluster *picchuv1alpha1.Cluster,
	client client.Client,
	configFetcher *ConfigFetcher,
	logger logr.Logger,
	scheme *runtime.Scheme,
) *IncarnationCollection {
	ic := &IncarnationCollection{
		itemSet:        make(map[string]Incarnation),
		releaseManager: rm,
		cluster:        cluster,
		client:         client,
		configFetcher:  configFetcher,
		log:            logger,
		scheme:         scheme,
	}
	// First seed all known revisions from status, since some might have been
	// deleted and don't have associated resources that are Add'd. If an
	// Incarnation has a nil Revision, it means it's been deleted.
	for _, r := range rm.Status.Revisions {
		l := logger.WithValues("Incarnation.Tag", r.Tag)
		ic.itemSet[r.Tag] = Incarnation{
			tag:            r.Tag,
			releaseManager: rm,
			cluster:        cluster,
			client:         client,
			configFetcher:  configFetcher,
			log:            l,
			scheme:         scheme,
		}
	}
	return ic
}

// Add adds a new Incarnation to the IncarnationManager
func (i *IncarnationCollection) add(revision *picchuv1alpha1.Revision) {
	i.itemSet[revision.Spec.App.Tag] = Incarnation{
		tag:            revision.Spec.App.Tag,
		releaseManager: i.releaseManager,
		revision:       revision,
		cluster:        i.cluster,
		client:         i.client,
		configFetcher:  i.configFetcher,
		log:            i.log,
		scheme:         i.scheme,
	}
}

// All returns every incarnation
func (i *IncarnationCollection) all() []Incarnation {
	r := []Incarnation{}
	for _, i := range i.itemSet {
		r = append(r, i)
	}
	return r
}

// Existing returns every incarnation that still has a live Revision
func (i *IncarnationCollection) existing() []Incarnation {
	r := []Incarnation{}
	for _, i := range i.itemSet {
		if i.revision != nil {
			r = append(r, i)
		}
	}
	return r
}

// Deployed returns deployed incarnation
func (i *IncarnationCollection) deployed() []Incarnation {
	r := []Incarnation{}
	for _, i := range i.itemSet {
		if i.isDeployed() {
			r = append(r, i)
		}
	}
	return r
}

// SortedReleases returns deployed release eligible incarnations in order from
// latest to oldest
func (i *IncarnationCollection) sortedReleases() []Incarnation {
	r := []Incarnation{}
	for _, i := range i.itemSet {
		if i.isDeployed() && i.isReleaseEligible() {
			r = append(r, i)
		}
	}
	sort.Slice(r, func(i, j int) bool {
		a := r[i].revision.GitTimestamp()
		b := r[j].revision.GitTimestamp()
		return a.After(b)
	})
	return r
}

// Revisioned returns all incarnations with revisions
func (i *IncarnationCollection) revisioned() []Incarnation {
	r := []Incarnation{}
	for _, i := range i.itemSet {
		if i.revision != nil {
			r = append(r, i)
		}
	}
	return r
}

// Revisioned returns all incarnations with revisions
func (i *IncarnationCollection) deleted() []Incarnation {
	r := []Incarnation{}
	for _, i := range i.itemSet {
		if i.revision == nil {
			r = append(r, i)
		}
	}
	return r
}

// retired returns all retired incarnations
func (i *IncarnationCollection) retired() []Incarnation {
	r := []Incarnation{}
	for _, i := range i.itemSet {
		if i.isRetired() {
			r = append(r, i)
		}
	}
	sort.Slice(r, func(i, j int) bool {
		a := r[i].revision.GitTimestamp()
		b := r[j].revision.GitTimestamp()
		return a.After(b)
	})
	return r
}
