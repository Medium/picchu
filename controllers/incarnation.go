package controllers

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"go.medium.engineering/picchu/controllers/schedule"

	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/controllers/observe"
	rmplan "go.medium.engineering/picchu/controllers/plan"
	"go.medium.engineering/picchu/controllers/utils"
	"go.medium.engineering/picchu/plan"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	istiov1alpha3 "istio.io/api/networking/v1alpha3"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Controller is incarnations interface into the outside scope
type Controller interface {
	expectedTotalReplicas(count int32, percent int32) int32
	applyPlan(context.Context, string, plan.Plan) error
	divideReplicas(count int32, percent int32) int32
	getReleaseManager() *picchuv1alpha1.ReleaseManager
	getLog() logr.Logger
	getConfigMaps(context.Context, *client.ListOptions) ([]runtime.Object, error)
	getSecrets(context.Context, *client.ListOptions) ([]runtime.Object, error)
	liveCount() int
}

// Incarnation respresents an applied revision
type Incarnation struct {
	deployed     bool
	controller   Controller
	tag          string
	revision     *picchuv1alpha1.Revision
	log          logr.Logger
	status       *picchuv1alpha1.ReleaseManagerRevisionStatus
	picchuConfig utils.Config
	isRamping    bool
}

// NewIncarnation creates a new Incarnation
func NewIncarnation(controller Controller, tag string, revision *picchuv1alpha1.Revision, log logr.Logger, di *observe.DeploymentInfo, config utils.Config) *Incarnation {
	status := controller.getReleaseManager().RevisionStatus(tag)

	if status.State.Current == "" {
		status.State.Current = "created"
	}

	if status.State.Current == "created" || status.State.Current == "deploying" {
		if revision != nil {
			status.GitTimestamp = &metav1.Time{Time: revision.GitTimestamp()}
			for _, target := range revision.Spec.Targets {
				if target.Name == controller.getReleaseManager().Spec.Target {
					status.ReleaseEligible = target.Release.Eligible
					status.TTL = target.Release.TTL
				}
			}
		} else {
			status.ReleaseEligible = false
		}
	}

	if status.GitTimestamp == nil {
		status.GitTimestamp = &metav1.Time{}
	}

	if status.RevisionTimestamp == nil {
		if revision == nil {
			now := metav1.Now()
			status.RevisionTimestamp = &now
		} else {
			ts := revision.GetCreationTimestamp()
			status.RevisionTimestamp = &ts
		}
	}

	var r *picchuv1alpha1.Revision
	if revision != nil {
		rev := *revision
		r = &rev
	}

	i := Incarnation{
		controller:   controller,
		tag:          tag,
		revision:     r,
		log:          log,
		status:       status,
		picchuConfig: config,
	}
	i.update(di)
	return &i
}

func (i *Incarnation) reportMetrics(log logr.Logger) {
	current := State(i.status.State.Current)

	if current == created && i.status.Metrics.GitCreateSecondsInt == nil {
		elapsed := time.Since(i.status.GitTimestamp.Time).Seconds()
		elapsedInt := int(elapsed)
		i.status.Metrics.GitCreateSecondsInt = &elapsedInt
		incarnationGitCreateLatency.With(prometheus.Labels{
			"app":    i.appName(),
			"target": i.targetName(),
		}).Observe(elapsed)
	}

	if current == deployed {
		if i.status.Metrics.GitDeploySecondsInt == nil {
			elapsed := time.Since(i.status.GitTimestamp.Time).Seconds()
			elapsedInt := int(elapsed)
			i.status.Metrics.GitDeploySecondsInt = &elapsedInt
			incarnationGitDeployLatency.With(prometheus.Labels{
				"app":    i.appName(),
				"target": i.targetName(),
			}).Observe(elapsed)
		}

		if i.status.Metrics.RevisionDeploySecondsInt == nil {
			elapsed := time.Since(i.status.RevisionTimestamp.Time).Seconds()
			elapsedInt := int(elapsed)
			i.status.Metrics.RevisionDeploySecondsInt = &elapsedInt
			incarnationRevisionDeployLatency.With(prometheus.Labels{
				"app":    i.appName(),
				"target": i.targetName(),
			}).Observe(elapsed)
		}

		if i.status.Metrics.DeploySecondsInt == nil {
			if i.status.DeployingStartTimestamp != nil {
				elapsed := time.Since(i.status.DeployingStartTimestamp.Time).Seconds()
				elapsedInt := int(elapsed)
				i.status.Metrics.DeploySecondsInt = &elapsedInt
				incarnationDeployLatency.With(prometheus.Labels{
					"app":    i.appName(),
					"target": i.targetName(),
				}).Observe(elapsed)
			}
		}
	}

	if current == canaried {
		if i.status.Metrics.GitCanarySecondsInt == nil {
			elapsed := time.Since(i.status.GitTimestamp.Time).Seconds()
			elapsedInt := int(elapsed)
			i.status.Metrics.GitCanarySecondsInt = &elapsedInt
			incarnationGitCanaryLatency.With(prometheus.Labels{
				"app":    i.appName(),
				"target": i.targetName(),
			}).Observe(elapsed)
		}

		if i.status.Metrics.RevisionCanarySecondsInt == nil {
			elapsed := time.Since(i.status.RevisionTimestamp.Time).Seconds()
			elapsedInt := int(elapsed)
			i.status.Metrics.RevisionCanarySecondsInt = &elapsedInt
			incarnationRevisionCanaryLatency.With(prometheus.Labels{
				"app":    i.appName(),
				"target": i.targetName(),
			}).Observe(elapsed)
		}

		if i.status.Metrics.CanarySecondsInt == nil {
			if i.status.CanaryStartTimestamp != nil {
				elapsed := time.Since(i.status.CanaryStartTimestamp.Time).Seconds()
				elapsedInt := int(elapsed)
				i.status.Metrics.CanarySecondsInt = &elapsedInt
				incarnationCanaryLatency.With(prometheus.Labels{
					"app":    i.appName(),
					"target": i.targetName(),
				}).Observe(elapsed)
			}
		}
	}

	if current == pendingrelease {
		if i.status.Metrics.GitPendingReleaseSecondsInt == nil {
			elapsed := time.Since(i.status.GitTimestamp.Time).Seconds()
			elapsedInt := int(elapsed)
			i.status.Metrics.GitPendingReleaseSecondsInt = &elapsedInt
			incarnationGitPendingReleaseLatency.With(prometheus.Labels{
				"app":    i.appName(),
				"target": i.targetName(),
			}).Observe(elapsed)
		}

		if i.status.Metrics.ReivisonPendingReleaseSecondsInt == nil {
			elapsed := time.Since(i.status.RevisionTimestamp.Time).Seconds()
			elapsedInt := int(elapsed)
			i.status.Metrics.ReivisonPendingReleaseSecondsInt = &elapsedInt
			incarnationRevisionPendingReleaseLatency.With(prometheus.Labels{
				"app":    i.appName(),
				"target": i.targetName(),
			}).Observe(elapsed)
		}
	}

	if current == released {
		if i.status.Metrics.GitReleaseSecondsInt == nil {
			elapsed := time.Since(i.status.GitTimestamp.Time).Seconds()
			elapsedInt := int(elapsed)
			i.status.Metrics.GitReleaseSecondsInt = &elapsedInt
			incarnationGitReleaseLatency.With(prometheus.Labels{
				"app":    i.appName(),
				"target": i.targetName(),
			}).Observe(elapsed)
		}

		if i.status.Metrics.RevisionReleaseSecondsInt == nil {
			elapsed := time.Since(i.status.RevisionTimestamp.Time).Seconds()
			elapsedInt := int(elapsed)
			i.status.Metrics.RevisionReleaseSecondsInt = &elapsedInt
			incarnationRevisionReleaseLatency.With(prometheus.Labels{
				"app":    i.appName(),
				"target": i.targetName(),
			}).Observe(elapsed)
		}

		if i.status.Metrics.ReleaseSecondsInt == nil {
			if i.status.ReleaseStartTimestamp != nil {
				elapsed := time.Since(i.status.ReleaseStartTimestamp.Time).Seconds()
				elapsedInt := int(elapsed)
				i.status.Metrics.ReleaseSecondsInt = &elapsedInt
				incarnationReleaseLatency.With(prometheus.Labels{
					"app":    i.appName(),
					"target": i.targetName(),
				}).Observe(elapsed)
			}
		}
	}

	if current == failed && i.status.Metrics.RevisionRollbackSecondsInt == nil {
		if i.revision != nil {
			elapsed := i.revision.SinceFailed().Seconds()
			elapsedInt := int(elapsed)
			i.status.Metrics.RevisionRollbackSecondsInt = &elapsedInt
			incarnationRevisionRollbackLatency.With(prometheus.Labels{
				"app":    i.appName(),
				"target": i.targetName(),
			}).Observe(elapsed)
		}
	}
}

// = Start Deployment interface
// Returns true if all target clusters are in deployed state
func (i *Incarnation) isDeployed() bool {
	if i.getStatus() == nil {
		return false
	}
	scale := i.getStatus().Scale
	return scale.Current >= scale.Desired && i.deployed
}

func (i *Incarnation) isCanaryPending() bool {
	target := i.target()
	if target == nil {
		return false
	}
	return target.IsCanaryPending(i.status.CanaryStartTimestamp)
}

func (i *Incarnation) ports() []picchuv1alpha1.PortInfo {
	target := i.target()
	if target == nil {
		return nil
	}
	return target.Ports
}

func (i *Incarnation) currentPercent() uint32 {
	if i.getStatus() == nil {
		return 0
	}
	return i.getStatus().CurrentPercent
}

func (i *Incarnation) peakPercent() uint32 {
	if i.getStatus() == nil {
		return 0
	}
	return i.getStatus().PeakPercent
}

func (i *Incarnation) getExternalTestStatus() ExternalTestStatus {
	target := i.target()
	if target == nil {
		return ExternalTestUnknown
	}
	return TargetExternalTestStatus(target)
}

func (i *Incarnation) isRoutable() bool {
	currentState := State(i.status.State.Current)
	return currentState != canaried && currentState != pendingrelease
}

func (i *Incarnation) isTimingOut() bool {
	target := i.target()
	if target == nil {
		return false
	}

	var lastUpdated time.Time
	var timeout time.Duration

	// TODO(mk) possiblity of generic timeout
	if i.status.State.LastUpdated != nil {
		lastUpdated = i.status.State.LastUpdated.Time
	}

	status := TargetExternalTestStatus(target)
	if status == ExternalTestStarted || status == ExternalTestPending {
		if target.ExternalTest.LastUpdated != nil {
			lastUpdated = target.ExternalTest.LastUpdated.Time
		}
		if target.ExternalTest.Timeout != nil {
			timeout = target.ExternalTest.Timeout.Duration
		}
	}
	if lastUpdated.IsZero() || timeout.Nanoseconds() == 0 {
		return false
	}

	i.log.Info("checking timeout",
		"lastUpdated", lastUpdated.String(),
		"timeout", timeout.String(),
	)
	return lastUpdated.Add(timeout).Before(time.Now())
}

// Remotely sync the incarnation for it's current state
func (i *Incarnation) sync(ctx context.Context) error {
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

	var configs []runtime.Object

	secrets, err := i.controller.getSecrets(ctx, configOpts)
	if err != nil {
		return err
	}
	configMaps, err := i.controller.getConfigMaps(ctx, configOpts)
	if err != nil {
		return err
	}

	var replicas int32
	if i.target().Scale.HasAutoscaler() {
		replicas = i.divideReplicas(i.target().Scale.Default)
	} else {
		replicas = i.divideReplicasNoAutoscale(i.target().Scale.Default)

	}
	syncPlan := &rmplan.SyncRevision{
		App:                i.appName(),
		Tag:                i.tag,
		Namespace:          i.targetNamespace(),
		Labels:             i.defaultLabels(),
		Configs:            append(append(configs, secrets...), configMaps...),
		Ports:              i.ports(),
		Replicas:           replicas,
		Image:              i.image(),
		Sidecars:           i.target().Sidecars,
		Resources:          i.target().Resources,
		IAMRole:            i.target().AWS.IAM.RoleARN,
		PodAnnotations:     i.target().PodAnnotations,
		ServiceAccountName: i.target().ServiceAccountName,
		ReadinessProbe:     i.target().ReadinessProbe,
		LivenessProbe:      i.target().LivenessProbe,
		MinReadySeconds:    i.target().Scale.MinReadySeconds,
		Worker:             i.target().Scale.Worker,
		Lifecycle:          i.target().Lifecycle,
		Affinity:           i.target().Affinity,
		PriorityClassName:  i.target().PriorityClassName,
		Tolerations:        i.target().Tolerations,
		EnvVars:            i.target().Env,
		Volumes:            i.target().Volumes,
		VolumeMounts:       i.target().VolumeMounts,
	}

	if !i.isRoutable() {
		syncPlan.Replicas = 0
	}

	return i.controller.applyPlan(ctx, "Sync and Scale Revision", plan.All(syncPlan, i.genScalePlan(ctx)))
}

func (i *Incarnation) syncCanaryRules(ctx context.Context) error {
	return i.controller.applyPlan(ctx, "Sync Canary Rules", &rmplan.SyncCanaryRules{
		App:                         i.appName(),
		Namespace:                   i.targetNamespace(),
		Tag:                         i.tag,
		Labels:                      i.defaultLabels(),
		ServiceLevelObjectiveLabels: i.target().ServiceLevelObjectiveLabels,
		ServiceLevelObjectives:      i.target().SlothServiceLevelObjectives,
	})
}

func (i *Incarnation) deleteCanaryRules(ctx context.Context) error {
	return i.controller.applyPlan(ctx, "Delete Canary Rules", &rmplan.DeleteCanaryRules{
		App:       i.appName(),
		Namespace: i.targetNamespace(),
		Tag:       i.tag,
	})
}

func (i *Incarnation) syncDeploymentRules(ctx context.Context) error {
	return i.controller.applyPlan(ctx, "Sync Deployment Rules", &rmplan.SyncDeploymentRules{
		App:                         i.appName(),
		Namespace:                   i.targetNamespace(),
		Tag:                         i.tag,
		Labels:                      i.defaultLabels(),
		ServiceLevelObjectiveLabels: i.target().ServiceLevelObjectiveLabels,
		ServiceLevelObjectives:      i.target().SlothServiceLevelObjectives,
	})
}

func (i *Incarnation) deleteDeploymentRules(ctx context.Context) error {
	return i.controller.applyPlan(ctx, "Delete Deployment Rules", &rmplan.DeleteDeploymentRules{
		App:       i.appName(),
		Namespace: i.targetNamespace(),
		Tag:       i.tag,
	})
}

func (i *Incarnation) syncTaggedServiceLevels(ctx context.Context) error {
	if i.picchuConfig.ServiceLevelsNamespace != "" {
		// Account for a fleet other than Delivery (old way of configuring SLOs) and Production (the only other place we ideally want SLOs to go)
		err := i.controller.applyPlan(
			ctx,
			"Ensure Service Levels Namespace",
			&rmplan.EnsureNamespace{Name: i.picchuConfig.ServiceLevelsNamespace},
		)
		if err != nil {
			return err
		}
		return i.controller.applyPlan(ctx, "Sync Tagged Service Levels", &rmplan.SyncTaggedServiceLevels{
			App:                         i.appName(),
			Target:                      i.targetName(),
			Namespace:                   i.picchuConfig.ServiceLevelsNamespace,
			Tag:                         i.tag,
			Labels:                      i.defaultLabels(),
			ServiceLevelObjectiveLabels: i.target().ServiceLevelObjectiveLabels,
			ServiceLevelObjectives:      i.target().SlothServiceLevelObjectives,
		})
	}
	i.log.Info("service-levels-fleet and service-levels-namespace not set, skipping SyncTaggedServiceLevels")
	return nil
}

func (i *Incarnation) deleteTaggedServiceLevels(ctx context.Context) error {
	if i.picchuConfig.ServiceLevelsNamespace != "" {
		return i.controller.applyPlan(
			ctx,
			"Delete Tagged Service Levels",
			&rmplan.DeleteTaggedServiceLevels{
				App:       i.appName(),
				Target:    i.targetName(),
				Namespace: i.picchuConfig.ServiceLevelsNamespace,
				Tag:       i.tag,
			},
		)
	}
	i.log.Info("service-levels-fleet and service-levels-namespace not set, skipping DeleteTaggedServiceLevels")
	return nil
}

func (i *Incarnation) genScalePlan(ctx context.Context) *rmplan.ScaleRevision {
	requestsRateTarget, err := i.target().Scale.TargetRequestsRateQuantity()
	if err != nil {
		i.log.Error(err, "Failed to parse targetRequestsRate", "TargetRequestsRate", i.target().Scale.TargetRequestsRate)
	}
	if requestsRateTarget != nil {
		requestsRateTarget = resource.NewMilliQuantity(int64(math.Floor(float64(requestsRateTarget.MilliValue())*i.targetScale())), requestsRateTarget.Format)
	}

	cpuTarget := i.target().Scale.TargetCPUUtilizationPercentage
	if cpuTarget != nil {
		*cpuTarget = int32(math.Floor(float64(*cpuTarget) * i.targetScale()))
	}

	memoryTarget := i.target().Scale.TargetMemoryUtilizationPercentage
	if memoryTarget != nil {
		*memoryTarget = int32(math.Floor(float64(*memoryTarget) * i.targetScale()))
	}

	min := i.divideReplicas(*i.target().Scale.Min)
	max := i.divideReplicas(i.target().Scale.Max)
	if i.target().Scale.Worker != nil && !i.isRoutable() {
		min = 0
		max = 0
	} else if i.status.CurrentPercent == 0 {
		min = max
	} else if i.target().Scale.Worker != nil && *i.target().Scale.Min == 0 {
		min = 0
	}

	return &rmplan.ScaleRevision{
		Tag:                i.tag,
		Namespace:          i.targetNamespace(),
		Min:                min,
		Max:                max,
		Labels:             i.defaultLabels(),
		CPUTarget:          cpuTarget,
		MemoryTarget:       memoryTarget,
		RequestsRateMetric: i.target().Scale.RequestsRateMetric,
		RequestsRateTarget: requestsRateTarget,
		Worker:             i.target().Scale.Worker,
	}
}

func (i *Incarnation) getLog() logr.Logger {
	return i.log
}

func (i *Incarnation) getStatus() *picchuv1alpha1.ReleaseManagerRevisionStatus {
	return i.status
}

// Tag is the revision tag
func (i *Incarnation) Tag() string {
	return i.tag
}

func (i *Incarnation) retire(ctx context.Context) error {
	return i.controller.applyPlan(ctx, "Retire Revision", &rmplan.RetireRevision{
		Tag:       i.tag,
		Namespace: i.targetNamespace(),
	})
}

func (i *Incarnation) schedulePermitsRelease() bool {
	if i.revision == nil {
		return false
	}

	// If previously released, allow outside of schedule
	if i.status.PeakPercent > 0 && i.status.PeakPercent > i.target().Canary.Percent {
		return true
	}

	// If engineers are out of office, don't start new releases
	if !i.picchuConfig.HumaneReleasesEnabled && i.target().Release.Schedule != "always" {
		return false
	}

	// if the revision was created during release schedule or we are currently
	// in the release schdeule
	times := []time.Time{
		i.revision.GetCreationTimestamp().Time,
		time.Now(),
	}
	for _, t := range times {
		if schedule.PermitsRelease(t, i.target().Release.Schedule) {
			return true
		}
	}
	return false
}

func (i *Incarnation) del(ctx context.Context) error {
	return i.controller.applyPlan(ctx, "Delete Revision", &rmplan.DeleteRevision{
		Labels:    i.defaultLabels(),
		Namespace: i.targetNamespace(),
	})
}

func (i *Incarnation) hasRevision() bool {
	return i.revision != nil
}

func (i *Incarnation) markedAsFailed() bool {
	if i.revision != nil {
		return i.revision.Spec.Failed
	}
	return false
}

func (i *Incarnation) setState(state string) {
	if state == "deploying" {
		if i.status.DeployingStartTimestamp == nil {
			t := metav1.Now()
			i.status.DeployingStartTimestamp = &t
		}
	}
	if state == "canarying" {
		if i.status.CanaryStartTimestamp == nil && i.status.CurrentPercent > 0 {
			t := metav1.Now()
			i.status.CanaryStartTimestamp = &t
		}
	}
	if state == "pendingRelease" {
		if i.status.PendingReleaseStartTimestamp == nil {
			t := metav1.Now()
			i.status.PendingReleaseStartTimestamp = &t
		}
	}
	if state == "releasing" {
		if i.status.ReleaseStartTimestamp == nil {
			t := metav1.Now()
			i.status.ReleaseStartTimestamp = &t
		}
	}
	if i.status.State.Current != state {
		timeNow := metav1.NewTime(time.Now())
		i.status.State.LastUpdated = &timeNow
	}
	i.status.State.Current = state
	i.status.State.Target = state
}

func (i *Incarnation) isReleaseEligible() bool {
	return i.status.ReleaseEligible
}

// isExpired returns true if the TTL of the revision is passed
func (i *Incarnation) isExpired() bool {
	return i.revision.CreationTimestamp.Add(i.TTL()).Before(time.Now())
}

// = End Deployment interface

func (i *Incarnation) target() *picchuv1alpha1.RevisionTarget {
	if i.revision == nil {
		return nil
	}
	for _, target := range i.revision.Spec.Targets {
		if target.Name == i.controller.getReleaseManager().Spec.Target {
			return &target
		}
	}
	return nil
}

func (i *Incarnation) appName() string {
	return i.controller.getReleaseManager().Spec.App
}

func (i *Incarnation) targetName() string {
	return i.controller.getReleaseManager().Spec.Target
}

func (i *Incarnation) targetNamespace() string {
	return i.controller.getReleaseManager().TargetNamespace()
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
		Namespace:     i.controller.getReleaseManager().Namespace,
	}, nil
}

func (i *Incarnation) defaultLabels() map[string]string {
	return map[string]string{
		picchuv1alpha1.LabelApp:          i.appName(),
		picchuv1alpha1.LabelTag:          i.tag,
		picchuv1alpha1.LabelTarget:       i.targetName(),
		picchuv1alpha1.LabelK8sName:      i.appName(),
		picchuv1alpha1.LabelK8sVersion:   i.tag,
		picchuv1alpha1.LabelIstioApp:     i.appName(),
		picchuv1alpha1.LabelIstioVersion: i.tag,
	}
}

func (i *Incarnation) setReleaseEligible(flag bool) {
	i.status.ReleaseEligible = flag
}

func (i *Incarnation) fastRelease() {
	// incarnations that have Peak == 100, scale to 100% immediately
	i.status.ReleaseEligible = true
	i.status.PeakPercent = 100
}

func (i *Incarnation) update(di *observe.DeploymentInfo) {
	if di == nil {
		i.deployed = false
		// Replicaset for revision is missing, make sure it's not released
		if i.status.CurrentPercent != 0 {
			now := metav1.Now()
			i.status.LastUpdated = &now
			i.status.CurrentPercent = 0
		}
		i.status.Scale.Current = 0
		i.status.Scale.Desired = 0
		i.status.Deleted = true
	} else {
		i.deployed = int(di.Current.Count) >= i.controller.liveCount() && di.Current.Min > 0
		i.status.Deleted = false
		i.status.Scale.Desired = di.Desired.Sum
		i.status.Scale.Current = di.Current.Sum
		if i.status.Scale.Current > i.status.Scale.Peak {
			i.status.Scale.Peak = i.status.Scale.Current
		}
	}
}

func (i *Incarnation) taggedRoutes(privateGateway string, serviceHost string) (http []*istiov1alpha3.HTTPRoute) {
	if i.revision == nil {
		return http
	}
	overrideLabel := fmt.Sprintf("pin/%s", i.appName())
	for _, port := range i.ports() {
		matches := []*istiov1alpha3.HTTPMatchRequest{{
			// mesh traffic from same tag'd service with and test tag
			SourceLabels: map[string]string{
				overrideLabel: i.tag,
			},
			Port:     uint32(port.Port),
			Gateways: []string{"mesh"},
		}}
		if privateGateway != "" {
			matches = append(matches,
				&istiov1alpha3.HTTPMatchRequest{
					// internal traffic with MEDIUM-TAG header
					Headers: map[string]*istiov1alpha3.StringMatch{
						"Medium-Tag": {MatchType: &istiov1alpha3.StringMatch_Exact{Exact: i.tag}},
					},
					Port:     uint32(port.IngressPort),
					Gateways: []string{privateGateway},
				},
			)
		}

		http = append(http, &istiov1alpha3.HTTPRoute{
			Match: matches,
			Route: []*istiov1alpha3.HTTPRouteDestination{
				{
					Destination: &istiov1alpha3.Destination{
						Host:   serviceHost,
						Port:   &istiov1alpha3.PortSelector{Number: uint32(port.Port)},
						Subset: i.tag,
					},
					Weight: 100,
				},
			},
		})
	}
	return
}

func (i *Incarnation) divideReplicas(count int32) int32 {
	status := i.getStatus()
	if status.State.Current == "deploying" || status.State.Current == "deployed" || status.State.Current == "pendingrelease" {
		return 1
	}

	var perc uint32 = 100
	if status.State.Current == "canarying" {
		perc = i.target().Canary.Percent
	} else {
		if i.isRamping {
			// since we sync before incrementing, we'll just err on the side of
			// caution and use the next increment percent.
			perc = NextIncrement(*i, i.target().Release.Max)
		} else {
			perc = i.currentPercent()
		}
	}

	return i.controller.divideReplicas(count, int32(perc))
}

// divideReplicaseNoAutoscale is used when there's no autoscaler. The initial ReplicaSet replica count will exist for
// the life of the revision, so it must be set to 100% no matter what state it's in
func (i *Incarnation) divideReplicasNoAutoscale(count int32) int32 {
	return i.controller.divideReplicas(count, 100)
}

// targetScale is used to scale HPA targets to prepare for next
// level of traffic. 10% -> 20% would be 1/2, 20% -> 30% would be 2/3. We
// prepare the revision for the next level by scaling it's target so it's
// not under-provisioned when it receives more traffic.
func (i *Incarnation) targetScale() float64 {
	next := float64(NextIncrement(*i, 100))
	current := i.currentPercent()

	if !i.isRamping || current <= 0 || current >= 100 {
		return 1.0
	}

	return float64(current) / next
}

func (i *Incarnation) currentPercentTarget(max uint32) uint32 {
	status := i.getStatus()

	if i.revision == nil || status == nil {
		return 0
	}

	if State(status.State.Current) == canarying {
		if max > i.target().Canary.Percent {
			max = i.target().Canary.Percent
		}
	}

	return Scale(*i, max, time.Now())
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
	expected := ExpectedReleaseLatency(*i, i.target().Release.Max)
	latency := time.Since(start.Time) - expected
	return latency.Seconds()
}

// IncarnationCollection helps us collect and select appropriate incarnations
type IncarnationCollection struct {
	// Incarnations key'd on revision.spec.app.tag
	itemSet    map[string]*Incarnation
	controller Controller
}

func newIncarnationCollection(controller Controller, revisionList *picchuv1alpha1.RevisionList, observation *observe.Observation, config utils.Config) *IncarnationCollection {
	ic := &IncarnationCollection{
		controller: controller,
		itemSet:    make(map[string]*Incarnation),
	}
	add := func(tag string, revision *picchuv1alpha1.Revision) {
		if _, ok := ic.itemSet[tag]; revision == nil && ok {
			return
		}

		l := controller.getLog().WithValues("Tag", tag)
		di := observation.InfoForTag(tag)
		var live *observe.DeploymentInfo
		if di != nil {
			live = di.Live
		}
		ic.itemSet[tag] = NewIncarnation(controller, tag, revision, l, live, config)
	}

	for i := range revisionList.Items {
		r := &revisionList.Items[i]
		add(r.Spec.App.Tag, r)
	}

	// add any deleted revisions that still have status
	rm := controller.getReleaseManager()
	for _, r := range rm.Status.Revisions {
		add(r.Tag, nil)
	}

	// mark ramping incarnation
	sorted := ic.releasable()
	for i := range sorted {
		inc := sorted[i]
		state := State(inc.status.State.Current)
		if state != canaried && state != canarying {
			inc.isRamping = true
			break
		}
	}

	return ic
}

// deployed returns deployed incarnation
func (i *IncarnationCollection) deployed() (r []*Incarnation) {
	r = []*Incarnation{}
	for _, i := range i.sorted() {
		switch i.status.State.Current {
		case "deployed", "pendingtest", "testing", "tested", "pendingrelease", "releasing", "released", "canarying", "canaried":
			r = append(r, i)
		}
	}
	return
}

func (i *IncarnationCollection) willRelease() (r []*Incarnation) {
	r = []*Incarnation{}
	for _, i := range i.sorted() {
		switch i.status.State.Current {
		case "deploying", "deployed", "precanary", "canarying", "canaried", "pendingrelease":
			if i.isReleaseEligible() {
				r = append(r, i)
			}
		}
	}
	return
}

// releasable returns deployed release eligible and released incarnations in
// order from releaseEligible to !releaseEligible, then latest to oldest
func (i *IncarnationCollection) releasable() (r []*Incarnation) {
	r = []*Incarnation{}
	for _, i := range i.sorted() {
		switch i.status.State.Current {
		case "canarying":
			r = append(r, i)
		}
	}
	for _, i := range i.sorted() {
		switch i.status.State.Current {
		case "releasing", "released":
			r = append(r, i)
		}
	}
	// Add incarnations transitioned out of released/releasing
	for _, i := range i.sorted() {
		if i.currentPercent() > 0 {
			switch i.getStatus().State.Current {
			case "retiring", "deleting", "failing":
				r = append(r, i)
			}
		}
	}
	return
}

func (i *IncarnationCollection) unreleasable() (unreleasable []*Incarnation) {
	releasableTags := map[string]bool{}
	for _, incarnation := range i.releasable() {
		releasableTags[incarnation.tag] = true
	}

	unreleasable = []*Incarnation{}
	for _, incarnation := range i.sorted() {
		if _, ok := releasableTags[incarnation.tag]; ok {
			continue
		}
		unreleasable = append(unreleasable, incarnation)
	}
	return
}

// alertable returns the incarnations which could currently
// have alerting enabled
// TODO(micah): deprecate this when RevisionTarget.AlertRules is deprecated.
func (i *IncarnationCollection) alertable() (r []*Incarnation) {
	r = []*Incarnation{}
	for _, i := range i.revisioned() {
		switch i.status.State.Current {
		case "canarying", "canaried", "pendingrelease":
			r = append(r, i)
		}
	}
	for _, i := range i.revisioned() {
		switch i.status.State.Current {
		case "releasing", "released":
			r = append(r, i)
		}
	}
	return
}

func (i *IncarnationCollection) unretirable() (r []*Incarnation) {
	r = []*Incarnation{}
	for _, i := range i.sorted() {
		cur := i.status.State.Current
		target := i.target()
		elig := false
		if target != nil {
			elig = target.Release.Eligible
		}
		triggered := i.markedAsFailed()
		if cur == "retired" || ((cur == "released" || cur == "releasing") && !elig && !triggered) {
			r = append(r, i)
		}
	}
	return
}

func (i *IncarnationCollection) revisioned() (r []*Incarnation) {
	r = []*Incarnation{}
	for _, i := range i.sorted() {
		if i.revision != nil {
			r = append(r, i)
		}
	}
	return
}

func (i *IncarnationCollection) sorted() (r []*Incarnation) {
	r = []*Incarnation{}
	for _, item := range i.itemSet {
		r = append(r, item)
	}

	sort.Slice(r, func(x, y int) bool {
		a := time.Time{}
		b := time.Time{}
		if r[x].revision != nil {
			a = r[x].revision.GitTimestamp()
		}
		if r[y].revision != nil {
			b = r[y].revision.GitTimestamp()
		}
		if a.Equal(b) {
			if r[x].revision != nil {
				a = r[x].revision.CreationTimestamp.Time
			}
			if r[y].revision != nil {
				b = r[y].revision.CreationTimestamp.Time
			}

		}
		return a.After(b)
	})
	return
}

func (i *IncarnationCollection) update(observation *observe.Observation) {
	for _, item := range i.itemSet {
		info := observation.InfoForTag(item.Tag())
		var live *observe.DeploymentInfo
		if info != nil {
			live = info.Live
		}
		item.update(live)
	}
}
