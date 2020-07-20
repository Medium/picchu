package releasemanager

import (
	"context"
	istio "istio.io/api/networking/v1alpha3"
	"time"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/controller/releasemanager/observe"
	rmplan "go.medium.engineering/picchu/pkg/controller/releasemanager/plan"
	"go.medium.engineering/picchu/pkg/controller/utils"
	"go.medium.engineering/picchu/pkg/plan"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"

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
	deliveryClient  client.Client
	deliveryApplier plan.Applier
	planApplier     plan.Applier
	observer        observe.Observer
	instance        *picchuv1alpha1.ReleaseManager
	incarnations    Incarnations
	reconciler      *ReconcileReleaseManager
	log             logr.Logger
	picchuConfig    utils.Config
	faults          []picchuv1alpha1.HTTPPortFault
}

func (r *ResourceSyncer) sync(ctx context.Context) (rs []picchuv1alpha1.ReleaseManagerRevisionStatus, err error) {
	// No more incarnations, delete myself
	if len(r.incarnations.revisioned()) == 0 {
		r.log.Info("No revisions found for releasemanager, deleting")
		return rs, r.deliveryClient.Delete(ctx, r.instance)
	}

	if err = r.syncNamespace(ctx); err != nil {
		return
	}
	if err = r.reportMetrics(); err != nil {
		return
	}
	if err = r.tickIncarnations(ctx); err != nil {
		return
	}
	if err = r.observe(ctx); err != nil {
		return
	}
	if err = r.syncApp(ctx); err != nil {
		return
	}
	if err = r.syncServiceMonitors(ctx); err != nil {
		return
	}
	if err = r.syncServiceLevels(ctx); err != nil {
		return
	}
	if err = r.syncSLORules(ctx); err != nil {
		return
	}
	if err = r.garbageCollection(ctx); err != nil {
		return
	}

	var foundGoodRelease bool
	latestRetiredStatus := -1
	sorted := r.incarnations.sorted()       // sorted by latest to oldest
	for i := len(sorted) - 1; i >= 0; i-- { // reverse sort
		status := sorted[i].status
		revision := sorted[i].revision
		state := State(status.State.Current)

		switch state {
		case deleted:
			if revision == nil {
				continue
			}
		case retired, retiring:
			latestRetiredStatus = len(rs)
		case deploying, deployed, pendingrelease, releasing, released:
			if !revision.Failed() && status.PeakPercent > 0 {
				foundGoodRelease = true
			}
		}

		rs = append(rs, *status)
	}
	if !foundGoodRelease && rs != nil && latestRetiredStatus > -1 && latestRetiredStatus < len(rs) {
		r.log.Info("recommissioning retired release", "tag", rs[latestRetiredStatus].Tag)
		rs[latestRetiredStatus].ReleaseEligible = true
		rs[latestRetiredStatus].PeakPercent = 100 // fast release
	}

	return
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

func (r *ResourceSyncer) applyDeliveryPlan(ctx context.Context, name string, p plan.Plan) error {
	r.log.Info("Applying delivery plan", "Name", name, "Plan", p)
	return r.deliveryApplier.Apply(ctx, p)
}

func (r *ResourceSyncer) syncNamespace(ctx context.Context) error {
	return r.applyPlan(ctx, "Ensure Namespace", &rmplan.EnsureNamespace{
		Name:      r.instance.TargetNamespace(),
		OwnerName: r.instance.Name,
		OwnerType: picchuv1alpha1.OwnerReleaseManager,
	})
}

func (r *ResourceSyncer) reportMetrics() error {
	incarnationsInState := map[string]int{}
	oldestIncarnationsInState := map[string]float64{}

	for _, incarnation := range r.incarnations.sorted() {
		current := incarnation.status.State.Current
		incarnationsInState[current]++
		gitElapsed := time.Since(incarnation.status.GitTimestamp.Time).Seconds()
		if age, ok := oldestIncarnationsInState[current]; !ok || age < gitElapsed {
			oldestIncarnationsInState[current] = gitElapsed
		}

		incarnation.reportMetrics()
	}
	for state, numIncarnations := range incarnationsInState {
		incarnationReleaseStateGauge.With(prometheus.Labels{
			"app":    r.instance.Spec.App,
			"target": r.instance.Spec.Target,
			"state":  state,
		}).Set(float64(numIncarnations))
		if age, ok := oldestIncarnationsInState[state]; ok {
			incarnationRevisionOldestStateGauge.With(prometheus.Labels{
				"app":    r.instance.Spec.App,
				"target": r.instance.Spec.Target,
				"state":  state,
			}).Set(age)
		}
	}
	return nil
}

func (r *ResourceSyncer) tickIncarnations(ctx context.Context) error {
	r.log.Info("Incarnation count", "count", len(r.incarnations.sorted()))

	for _, incarnation := range r.incarnations.sorted() {
		sm := NewDeploymentStateManager(incarnation)

		if err := sm.tick(ctx); err != nil {
			return err
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

	revisions := r.prepareRevisions()
	alertRules := r.prepareAlertRules()

	err := r.applyPlan(ctx, "Sync Application", &rmplan.SyncApp{
		App:               r.instance.Spec.App,
		Target:            r.instance.Spec.Target,
		Fleet:             r.instance.Spec.Fleet,
		Namespace:         r.instance.TargetNamespace(),
		Labels:            r.defaultLabels(),
		DeployedRevisions: revisions,
		AlertRules:        alertRules,
		Ports:             ports,
		HTTPPortFaults:    r.faults,
		DefaultVariant:    utils.VariantEnabled(r.instance, picchuv1alpha1.VariantPortDefault),
		IngressesVariant:  utils.VariantEnabled(r.instance, picchuv1alpha1.VariantIngresses),
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

// Used to label Service and selector
func (r *ResourceSyncer) defaultLabels() map[string]string {
	return map[string]string{
		picchuv1alpha1.LabelApp:       r.instance.Spec.App,
		picchuv1alpha1.LabelK8sName:   r.instance.Spec.App,
		picchuv1alpha1.LabelIstioApp:  r.instance.Spec.App,
		picchuv1alpha1.LabelOwnerType: picchuv1alpha1.OwnerReleaseManager,
		picchuv1alpha1.LabelOwnerName: r.instance.Name,
	}
}

func (r *ResourceSyncer) syncServiceMonitors(ctx context.Context) error {
	serviceMonitors := r.prepareServiceMonitors()
	slos, _ := r.prepareServiceLevelObjectives()

	if len(serviceMonitors) == 0 {
		if err := r.applyPlan(ctx, "Delete Service Monitors", &rmplan.DeleteServiceMonitors{
			App:       r.instance.Spec.App,
			Namespace: r.instance.TargetNamespace(),
		}); err != nil {
			return err
		}
	} else {
		if err := r.applyPlan(ctx, "Sync Service Monitors", &rmplan.SyncServiceMonitors{
			App:                    r.instance.Spec.App,
			Namespace:              r.instance.TargetNamespace(),
			Labels:                 r.defaultLabels(),
			ServiceMonitors:        serviceMonitors,
			ServiceLevelObjectives: slos,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (r *ResourceSyncer) delServiceLevels(ctx context.Context) error {
	return r.applyDeliveryPlan(ctx, "Delete App ServiceLevels", &rmplan.DeleteServiceLevels{
		App:       r.instance.Spec.App,
		Target:    r.instance.Spec.Target,
		Namespace: r.picchuConfig.ServiceLevelsNamespace,
	})
}

func (r *ResourceSyncer) syncServiceLevels(ctx context.Context) error {
	if r.picchuConfig.ServiceLevelsFleet != "" && r.picchuConfig.ServiceLevelsNamespace != "" {
		slos, sloLabels := r.prepareServiceLevelObjectives()
		if len(slos) > 0 {
			if err := r.applyDeliveryPlan(ctx, "Ensure Service Levels Namespace", &rmplan.EnsureNamespace{
				Name: r.picchuConfig.ServiceLevelsNamespace,
			}); err != nil {
				return err
			}

			labels := r.defaultLabels()
			labels[picchuv1alpha1.LabelTarget] = r.instance.Spec.Target

			if err := r.applyDeliveryPlan(ctx, "Sync App ServiceLevels", &rmplan.SyncServiceLevels{
				App:                         r.instance.Spec.App,
				Target:                      r.instance.Spec.Target,
				Namespace:                   r.picchuConfig.ServiceLevelsNamespace,
				Labels:                      labels,
				ServiceLevelObjectiveLabels: sloLabels,
				ServiceLevelObjectives:      slos,
			}); err != nil {
				return err
			}
		} else {
			return r.delServiceLevels(ctx)
		}
	} else {
		r.log.Info("service-levels-fleet and service-levels-namespace not set, skipping SyncServiceLevels")
	}
	return nil
}

func (r *ResourceSyncer) syncSLORules(ctx context.Context) error {
	slos, labels := r.prepareServiceLevelObjectives()
	if len(slos) > 0 {
		if err := r.applyPlan(ctx, "Sync App SLO Rules", &rmplan.SyncSLORules{
			App:                         r.instance.Spec.App,
			Namespace:                   r.instance.TargetNamespace(),
			Labels:                      r.defaultLabels(),
			ServiceLevelObjectiveLabels: labels,
			ServiceLevelObjectives:      slos,
		}); err != nil {
			return err
		}
	} else {
		if err := r.applyPlan(ctx, "Delete App SLO Rules", &rmplan.DeleteSLORules{
			App:       r.instance.Spec.App,
			Namespace: r.instance.TargetNamespace(),
		}); err != nil {
			return err
		}
	}

	return nil
}

func (r *ResourceSyncer) garbageCollection(ctx context.Context) error {
	return markGarbage(ctx, r.log, r.deliveryClient, r.incarnations.sorted())
}

// returns the AlertRules from the latest deployed revision
func (r *ResourceSyncer) prepareAlertRules() []monitoringv1.Rule {
	var alertRules []monitoringv1.Rule

	if len(r.incarnations.deployed()) > 0 {
		alertable := r.incarnations.alertable()
		for _, i := range alertable {
			if i.target() != nil {
				alertRules = i.target().AlertRules
				break
			}
		}
	}

	return alertRules
}

// returns the ServiceMonitors from the latest deployed revision
func (r *ResourceSyncer) prepareServiceMonitors() []*picchuv1alpha1.ServiceMonitor {
	var sm []*picchuv1alpha1.ServiceMonitor

	if len(r.incarnations.deployed()) > 0 {
		alertable := r.incarnations.alertable()
		for _, i := range alertable {
			if i.target() != nil {
				sm = i.target().ServiceMonitors
				break
			}
		}
	}

	return sm
}

// returns the PrometheusRules to support SLOs from the latest released revision
func (r *ResourceSyncer) prepareServiceLevelObjectives() ([]*picchuv1alpha1.ServiceLevelObjective, picchuv1alpha1.ServiceLevelObjectiveLabels) {
	var slos []*picchuv1alpha1.ServiceLevelObjective

	if len(r.incarnations.deployed()) > 0 {
		releasable := r.incarnations.releasable()
		for _, i := range releasable {
			if i.target() != nil {
				return i.target().ServiceLevelObjectives, i.target().ServiceLevelObjectiveLabels
			}
		}
	}

	return slos, picchuv1alpha1.ServiceLevelObjectiveLabels{}
}

func (r *ResourceSyncer) prepareRevisions() []rmplan.Revision {
	if len(r.incarnations.deployed()) == 0 {
		return []rmplan.Revision{}
	}

	revisionsMap := map[string]*rmplan.Revision{}
	for _, i := range r.incarnations.deployed() {
		var trafficPolicy *istio.TrafficPolicy
		if i.target() != nil && i.target().Istio != nil {
			trafficPolicy = i.target().Istio.TrafficPolicy
		}
		tagRoutingHeader := ""
		if i.revision != nil && i.isRoutable() {
			tagRoutingHeader = i.revision.Spec.TagRoutingHeader
		}
		revisionsMap[i.tag] = &rmplan.Revision{
			Tag:              i.tag,
			Weight:           0,
			TagRoutingHeader: tagRoutingHeader,
			TrafficPolicy:    trafficPolicy,
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

	firstNonCanary := -1
	firstNonCanaryTag := ""
	for i, incarnation := range incarnations {
		status := incarnation.status
		oldCurrentPercent := status.CurrentPercent
		currentState := State(status.State.Current)

		if firstNonCanary == -1 && currentState != canaried && currentState != canarying {
			r.log.Info(
				"Found first ramping release",
				"firstNonCanary", i,
				"Tag", incarnation.tag,
				"currentState", status.State.Current,
			)
			firstNonCanary = i
			firstNonCanaryTag = incarnation.tag
		}

		// what this means in practice is that only the latest "releasing" revision will be incremented,
		// the remaining will either stay the same or be decremented.
		max := percRemaining
		if firstNonCanary != -1 && i > firstNonCanary {
			max = uint32(utils.Min(int32(status.CurrentPercent), int32(percRemaining)))
		}
		currentPercent := incarnation.currentPercentTarget(max)

		if currentPercent > percRemaining {
			r.log.Info(
				"Percent target greater than percRemaining",
				"current", currentPercent,
				"percRemaining", percRemaining,
				"increment", incarnation.target().Release.Rate.Increment)
			panic("Assertion failed")
		}
		incarnation.updateCurrentPercent(currentPercent)
		r.log.Info(
			"Updated incarnation CurrentPercent",
			"Tag", incarnation.tag,
			"oldCurrentPercent", oldCurrentPercent,
			"currentPercent", currentPercent,
			"weightChanged", currentPercent != oldCurrentPercent,
			"desiredReplicas", status.Scale.Desired,
			"currentReplicas", status.Scale.Current,
			"currentState", status.State.Current,
		)

		percRemaining -= currentPercent

		if currentPercent <= 0 && firstNonCanary != -1 && i > firstNonCanary {
			r.log.Info(
				"Setting incarnation release-eligibility to false; will trigger retirement",
				"Tag", incarnation.tag,
				"rampingTag", firstNonCanaryTag,
				"oldCurrentPercent", oldCurrentPercent,
				"currentPercent", currentPercent,
				"currentState", status.State.Current,
			)
			incarnation.setReleaseEligible(false)
		}

		tagRoutingHeader := ""
		if incarnation.revision != nil && incarnation.isRoutable() {
			tagRoutingHeader = incarnation.revision.Spec.TagRoutingHeader
		}
		var trafficPolicy *istio.TrafficPolicy
		if incarnation.target() != nil && incarnation.target().Istio != nil {
			trafficPolicy = incarnation.target().Istio.TrafficPolicy
		}
		revisionsMap[incarnation.tag] = &rmplan.Revision{
			Tag:              incarnation.tag,
			Weight:           currentPercent,
			TagRoutingHeader: tagRoutingHeader,
			TrafficPolicy:    trafficPolicy,
		}
		if i == count-1 && percRemaining > 0 && firstNonCanary != -1 {
			revisionsMap[incarnations[firstNonCanary].tag].Weight += percRemaining
			incarnations[firstNonCanary].updateCurrentPercent(incarnations[firstNonCanary].currentPercent() + percRemaining)
			percRemaining = 0
		}
	}

	if percRemaining > 0 && len(incarnations) > 0 {
		incarnation := incarnations[0]
		perc := incarnation.currentPercent() + percRemaining
		// make sure percRemaining is expended
		revisionsMap[incarnation.tag].Weight = perc
		incarnation.updateCurrentPercent(perc)
		r.log.Info("Assigning leftover percentage", "incarnation", incarnation.tag)
	}

	revisions := make([]rmplan.Revision, 0, len(revisionsMap))
	for _, revision := range revisionsMap {
		revisions = append(revisions, *revision)
	}
	return revisions
}
