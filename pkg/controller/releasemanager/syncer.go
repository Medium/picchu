package releasemanager

import (
	"context"
	"time"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/controller/releasemanager/observe"
	rmplan "go.medium.engineering/picchu/pkg/controller/releasemanager/plan"
	"go.medium.engineering/picchu/pkg/controller/utils"
	"go.medium.engineering/picchu/pkg/plan"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/go-logr/logr"
	istiov1alpha3 "github.com/knative/pkg/apis/istio/v1alpha3"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Incarnations interface {
	deployed() (r []*Incarnation)
	willRelease() (r []*Incarnation)
	releasable() (r []*Incarnation)
	unreleasable() (r []*Incarnation)
	alertable() (r []*Incarnation)
	unretirable() (r []*Incarnation)
	revisioned() (r []*Incarnation)
	sorted() (r []*Incarnation)
	update(observation *observe.Observation)
}

type ResourceSyncer struct {
	deliveryClient client.Client
	planApplier    plan.Applier
	observer       observe.Observer
	instance       *picchuv1alpha1.ReleaseManager
	incarnations   Incarnations
	reconciler     *ReconcileReleaseManager
	log            logr.Logger
	clusterConfig  ClusterConfig
}

func (r *ResourceSyncer) getSecrets(ctx context.Context, opts *client.ListOptions) ([]runtime.Object, error) {
	secrets := &corev1.SecretList{}
	err := r.deliveryClient.List(ctx, opts, secrets)
	if errors.IsNotFound(err) {
		return []runtime.Object{}, nil
	}
	return utils.MustExtractList(secrets), err
}

func (r *ResourceSyncer) getConfigMaps(ctx context.Context, opts *client.ListOptions) ([]runtime.Object, error) {
	configMaps := &corev1.ConfigMapList{}
	err := r.deliveryClient.List(ctx, opts, configMaps)
	if errors.IsNotFound(err) {
		return []runtime.Object{}, nil
	}
	return utils.MustExtractList(configMaps), err
}

func (r *ResourceSyncer) sync(ctx context.Context) ([]picchuv1alpha1.ReleaseManagerRevisionStatus, error) {
	rs := []picchuv1alpha1.ReleaseManagerRevisionStatus{}

	// No more incarnations, delete myself
	if len(r.incarnations.revisioned()) == 0 {
		r.log.Info("No revisions found for releasemanager, deleting")
		return rs, r.deliveryClient.Delete(ctx, r.instance)
	}

	if err := r.syncNamespace(ctx); err != nil {
		return rs, err
	}
	if err := r.tickIncarnations(ctx); err != nil {
		return rs, err
	}
	if err := r.observe(ctx); err != nil {
		return rs, err
	}
	if err := r.syncApp(ctx); err != nil {
		return rs, err
	}

	sorted := r.incarnations.sorted()
	for i := len(sorted) - 1; i >= 0; i-- {
		status := sorted[i].status
		if !(status.State.Current == "deleted" && sorted[i].revision == nil) {
			rs = append(rs, *status)
		}
	}
	return rs, nil
}

func (r *ResourceSyncer) del(ctx context.Context) error {
	return r.applyPlan(ctx, "Delete App", &rmplan.DeleteApp{
		Namespace: r.instance.TargetNamespace(),
	})
}

func (r *ResourceSyncer) applyPlan(ctx context.Context, name string, p plan.Plan) error {
	r.log.Info("Applying plan", "Name", name, "Plan", p)
	return r.planApplier.Apply(ctx, p)
}

func (r *ResourceSyncer) syncNamespace(ctx context.Context) error {
	return r.applyPlan(ctx, "Ensure Namespace", &rmplan.EnsureNamespace{
		Name:      r.instance.TargetNamespace(),
		OwnerName: r.instance.Name,
		OwnerType: picchuv1alpha1.OwnerReleaseManager,
	})
}

func (r *ResourceSyncer) tickIncarnations(ctx context.Context) error {
	r.log.Info("Incarnation count", "count", len(r.incarnations.sorted()))
	for _, incarnation := range r.incarnations.sorted() {
		sm := NewDeploymentStateManager(incarnation)
		if err := sm.tick(ctx); err != nil {
			return err
		}
		current := incarnation.status.State.Current
		if (current == "deployed" || current == "released") && incarnation.status.Metrics.GitDeploySeconds == nil {
			gitElapsed := time.Since(incarnation.status.GitTimestamp.Time).Seconds()
			incarnation.status.Metrics.GitDeploySeconds = &gitElapsed
			incarnationGitDeployLatency.With(prometheus.Labels{
				"app":    r.instance.Spec.App,
				"target": r.instance.Spec.Target,
			}).Observe(gitElapsed)
			revElapsed := time.Since(incarnation.status.RevisionTimestamp.Time).Seconds()
			incarnation.status.Metrics.RevisionDeploySeconds = &revElapsed
			incarnationRevisionDeployLatency.With(prometheus.Labels{
				"app":    r.instance.Spec.App,
				"target": r.instance.Spec.Target,
			}).Observe(revElapsed)
		}
		if current == "released" && incarnation.status.Metrics.GitReleaseSeconds == nil {
			elapsed := time.Since(incarnation.status.GitTimestamp.Time).Seconds()
			incarnation.status.Metrics.GitReleaseSeconds = &elapsed
			incarnationGitReleaseLatency.With(prometheus.Labels{
				"app":    r.instance.Spec.App,
				"target": r.instance.Spec.Target,
			}).Observe(elapsed)
		}
		if current == "failed" && incarnation.status.Metrics.RevisionRollbackSeconds == nil {
			if incarnation.revision != nil {
				r.log.Info("observing rollback elapsed time")
				elapsed := incarnation.revision.SinceFailed().Seconds()
				incarnation.status.Metrics.RevisionRollbackSeconds = &elapsed
				incarnationRevisionRollbackLatency.With(prometheus.Labels{
					"app":    r.instance.Spec.App,
					"target": r.instance.Spec.Target,
				}).Observe(elapsed)
			} else {
				r.log.Info("no revision rollback revision found")
			}
		}
	}
	return nil
}

func (r *ResourceSyncer) observe(ctx context.Context) error {
	observation, err := r.observer.Observe(ctx, r.instance.TargetNamespace())
	if err != nil {
		return err
	}
	r.incarnations.update(observation)
	return nil
}

func (r *ResourceSyncer) syncApp(ctx context.Context) error {
	portMap := map[string]picchuv1alpha1.PortInfo{}
	incarnations := r.incarnations.deployed()

	// Include ports from apps in "deploying" state
	for _, incarnation := range r.incarnations.sorted() {
		if incarnation.status.State.Current == "deploying" {
			incarnations = append(incarnations, incarnation)
		}
	}

	for _, incarnation := range incarnations {
		for _, port := range incarnation.ports() {
			_, ok := portMap[port.Name]
			if !ok {
				portMap[port.Name] = port
			}
		}
	}
	ports := make([]picchuv1alpha1.PortInfo, 0, len(portMap))
	for _, port := range portMap {
		ports = append(ports, port)
	}

	// Used to label Service and selector
	labels := map[string]string{
		picchuv1alpha1.LabelApp:       r.instance.Spec.App,
		picchuv1alpha1.LabelOwnerType: picchuv1alpha1.OwnerReleaseManager,
		picchuv1alpha1.LabelOwnerName: r.instance.Name,
	}

	revisions, alertRules := r.prepareRevisionsAndRules()

	// TODO(bob): figure out defaultDomain and gateway names
	err := r.applyPlan(ctx, "Sync Application", &rmplan.SyncApp{
		App:               r.instance.Spec.App,
		Namespace:         r.instance.TargetNamespace(),
		Labels:            labels,
		DefaultDomain:     r.clusterConfig.DefaultDomain,
		PublicGateway:     r.clusterConfig.PublicIngressGateway,
		PrivateGateway:    r.clusterConfig.PrivateIngressGateway,
		DeployedRevisions: revisions,
		AlertRules:        alertRules,
		Ports:             ports,
		TrafficPolicy:     r.currentTrafficPolicy(),
	})
	if err != nil {
		return err
	}

	for _, revision := range revisions {
		revisionReleaseWeightGauge.
			With(prometheus.Labels{
				"app":    r.instance.Spec.App,
				"target": r.instance.Spec.Target,
			}).
			Set(float64(revision.Weight))
	}
	return nil
}

// currentTrafficPolicy gets the latest release's traffic policy, or if there
// are no releases, then the latest revisions traffic policy.
func (r *ResourceSyncer) currentTrafficPolicy() *istiov1alpha3.TrafficPolicy {
	for _, incarnation := range r.incarnations.releasable() {
		if incarnation.revision != nil {
			return incarnation.revision.Spec.TrafficPolicy
		}
	}
	for _, incarnation := range r.incarnations.sorted() {
		if incarnation.revision != nil {
			return incarnation.revision.Spec.TrafficPolicy
		}
	}
	return nil
}

func (r *ResourceSyncer) prepareRevisionsAndRules() ([]rmplan.Revision, []monitoringv1.Rule) {
	alertRules := []monitoringv1.Rule{}

	if len(r.incarnations.deployed()) == 0 {
		return []rmplan.Revision{}, alertRules
	}

	revisionsMap := map[string]*rmplan.Revision{}
	for _, i := range r.incarnations.deployed() {
		tagRoutingHeader := ""
		if i.revision != nil {
			tagRoutingHeader = i.revision.Spec.TagRoutingHeader
		}
		revisionsMap[i.tag] = &rmplan.Revision{
			Tag:              i.tag,
			Weight:           0,
			TagRoutingHeader: tagRoutingHeader,
		}
	}

	// This ensures that incarnations that are no longer releasable get their
	// currentPercent status set to 0%.
	for _, incarnation := range r.incarnations.unreleasable() {
		incarnation.updateCurrentPercent(0)
	}

	// The idea here is we will work through releases from newest to oldest,
	// incrementing their weight if enough time has passed since their last
	// update, and stopping when we reach 100%. This will cause newer releases
	// to take from oldest fnord release.
	var percRemaining uint32 = 100
	// Tracking one route per port number
	incarnations := r.incarnations.releasable()
	count := len(incarnations)

	// setup alerts from latest release that has revision
	alertable := r.incarnations.alertable()
	for _, i := range alertable {
		alertRules = i.target().AlertRules
		break
	}

	firstNonCanary := -1
	for i, incarnation := range incarnations {
		status := incarnation.status
		oldCurrent := status.CurrentPercent
		currentState := State(status.State.Current)

		if firstNonCanary == -1 && currentState != canaried && currentState != canarying {
			r.log.Info("Found first ramping release", "firstNonCanary", i)
			firstNonCanary = i
		}

		// what this means in practice is that only the latest "releasing" revision will be incremented,
		// the remaining will either stay the same or be decremented.
		max := percRemaining
		if firstNonCanary != -1 && i > firstNonCanary {
			max = uint32(utils.Min(int32(status.CurrentPercent), int32(percRemaining)))
		}
		current := incarnation.currentPercentTarget(max)

		if current > percRemaining {
			r.log.Info(
				"Percent target greater than percRemaining",
				"current", current,
				"percRemaining", percRemaining,
				"increment", incarnation.target().Release.Rate.Increment)
			panic("Assertion failed")
		}
		incarnation.updateCurrentPercent(current)
		r.log.Info(
			"CurrentPercentage Update",
			"Tag", incarnation.tag,
			"Old", oldCurrent,
			"Current", current,
			"desiredReplicas", status.Scale.Desired,
			"currentReplicas", status.Scale.Current,
		)
		if current <= 0 {
			incarnation.setReleaseEligible(false)
		}
		percRemaining -= current
		tagRoutingHeader := ""
		if incarnation.revision != nil {
			tagRoutingHeader = incarnation.revision.Spec.TagRoutingHeader
		}
		revisionsMap[incarnation.tag] = &rmplan.Revision{
			Tag:              incarnation.tag,
			Weight:           current,
			TagRoutingHeader: tagRoutingHeader,
		}
		if i == count-1 && percRemaining > 0 && firstNonCanary != -1 {
			revisionsMap[incarnations[firstNonCanary].tag].Weight += percRemaining
			incarnations[firstNonCanary].updateCurrentPercent(incarnations[firstNonCanary].currentPercent() + percRemaining)
		}
	}

	revisions := make([]rmplan.Revision, 0, len(revisionsMap))
	for _, revision := range revisionsMap {
		revisions = append(revisions, *revision)
	}
	return revisions, alertRules
}
