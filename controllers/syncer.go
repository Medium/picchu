package controllers

import (
	"context"
	"time"

	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/controllers/observe"
	rmplan "go.medium.engineering/picchu/controllers/plan"
	"go.medium.engineering/picchu/controllers/utils"
	"go.medium.engineering/picchu/plan"

	"github.com/go-logr/logr"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/prometheus/client_golang/prometheus"
	istio "istio.io/api/networking/v1alpha3"
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
	// deliveryApplier plan.Applier
	planApplier  plan.Applier
	observer     observe.Observer
	instance     *picchuv1alpha1.ReleaseManager
	incarnations Incarnations
	reconciler   *ReleaseManagerReconciler
	log          logr.Logger
	picchuConfig utils.Config
	faults       []picchuv1alpha1.HTTPPortFault
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
	if err = r.syncServiceAccount(ctx); err != nil {
		return
	}
	if err = r.syncRBAC(ctx); err != nil {
		return
	}
	if err = r.reportMetrics(); err != nil {
		r.log.Error(err, "Failed to report metrics")
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

// func (r *ResourceSyncer) applyDeliveryPlan(ctx context.Context, name string, p plan.Plan) error {
// 	return r.deliveryApplier.Apply(ctx, p)
// }

func (r *ResourceSyncer) applyPlan(ctx context.Context, name string, p plan.Plan) error {
	return r.planApplier.Apply(ctx, p)
}

func (r *ResourceSyncer) syncNamespace(ctx context.Context) error {
	ambientMesh := r.effectiveAmbientMesh()
	ns := r.instance.TargetNamespace()
	if err := r.applyPlan(ctx, "Ensure Namespace", &rmplan.EnsureNamespace{
		Name:        ns,
		OwnerName:   r.instance.Name,
		OwnerType:   picchuv1alpha1.OwnerReleaseManager,
		AmbientMesh: ambientMesh,
	}); err != nil {
		return err
	}
	if ambientMesh {
		if err := r.applyPlan(ctx, "Ensure Waypoint", &rmplan.EnsureWaypoint{Namespace: ns}); err != nil {
			return err
		}
		if hpa := r.effectiveWaypointHPA(); hpa != nil {
			if err := r.applyPlan(ctx, "Ensure Waypoint HPA", &rmplan.EnsureWaypointHPA{Namespace: ns, HPA: hpa}); err != nil {
				return err
			}
		}
		return nil
	}
	if err := r.applyPlan(ctx, "Delete Waypoint", &rmplan.DeleteWaypoint{Namespace: ns}); err != nil {
		return err
	}
	return r.applyPlan(ctx, "Delete Waypoint HPA", &rmplan.DeleteWaypointHPA{Namespace: ns})
}

// effectiveWaypointHPA returns the waypoint HPA spec from the latest incarnation when set; otherwise nil (no HPA).
// When non-nil, defaults targetCPUUtilizationPercentage to 70 if unset.
func (r *ResourceSyncer) effectiveWaypointHPA() *picchuv1alpha1.WaypointHPASpec {
	sorted := r.incarnations.sorted()
	if len(sorted) == 0 {
		return nil
	}
	t := sorted[0].target()
	if t == nil || t.WaypointHPA == nil || t.WaypointHPA.MaxReplicas < 1 {
		return nil
	}
	spec := *t.WaypointHPA
	if spec.TargetCPUUtilizationPercentage < 1 {
		spec.TargetCPUUtilizationPercentage = 70
	}
	return &spec
}

// effectiveAmbientMesh returns true if the latest (newest by GitTimestamp) incarnation's target has AmbientMesh enabled.
// sorted() returns newest first, so we use the first element.
func (r *ResourceSyncer) effectiveAmbientMesh() bool {
	sorted := r.incarnations.sorted()
	if len(sorted) == 0 {
		return false
	}
	t := sorted[0].target()
	return t != nil && t.AmbientMesh
}

func (r *ResourceSyncer) syncServiceAccount(ctx context.Context) error {
	return r.applyPlan(ctx, "Ensure ServiceAccount", &rmplan.EnsureServiceAccount{
		Name:      r.instance.Spec.App,
		OwnerName: r.instance.Name,
		OwnerType: picchuv1alpha1.OwnerReleaseManager,
		Namespace: r.instance.TargetNamespace(),
	})
}

func (r *ResourceSyncer) syncRBAC(ctx context.Context) error {
	return r.applyPlan(ctx, "Ensure RBAC", &rmplan.EnsureRBAC{
		Name:               r.instance.Spec.App,
		OwnerName:          r.instance.Name,
		OwnerType:          picchuv1alpha1.OwnerReleaseManager,
		Namespace:          r.instance.TargetNamespace(),
		ServiceAccountName: r.instance.Spec.App,
	})
}

func (r *ResourceSyncer) reportMetrics() error {
	incarnationsInState := map[string]int{}
	oldestIncarnationsInState := map[string]float64{}
	oldestIncarnations := map[string]string{}

	// initialize oldestIncarnationInState with zeros to zero out any non-existant states
	for _, state := range AllStates {
		oldestIncarnationsInState[state] = 0
		incarnationsInState[state] = 0
		oldestIncarnations[state] = ""
	}

	for _, incarnation := range r.incarnations.sorted() {
		current := incarnation.status.State.Current
		incarnationsInState[current]++

		var lastUpdated *time.Time
		var age float64
		if incarnation.status.State.LastUpdated != nil {
			lastUpdated = &incarnation.status.State.LastUpdated.Time
			age = time.Since(*lastUpdated).Seconds()
		}

		if oldest, ok := oldestIncarnationsInState[current]; !ok || oldest < age {
			oldestIncarnationsInState[current] = age
			oldestIncarnations[current] = incarnation.tag
		}

		incarnation.reportMetrics(r.log)
	}
	for state, numIncarnations := range incarnationsInState {
		incarnationReleaseStateGauge.With(prometheus.Labels{
			"app":    r.instance.Spec.App,
			"target": r.instance.Spec.Target,
			"state":  state,
		}).Set(float64(numIncarnations))
		if age, ok := oldestIncarnationsInState[state]; ok {
			incarnationRevisionOldestStateGauge.With(prometheus.Labels{
				"app":      r.instance.Spec.App,
				"target":   r.instance.Spec.Target,
				"state":    state,
				"revision": oldestIncarnations[state],
			}).Set(age)
		}
	}
	return nil
}

func (r *ResourceSyncer) tickIncarnations(ctx context.Context) error {
	for _, incarnation := range r.incarnations.sorted() {
		var lastUpdated *time.Time
		if incarnation.status.State.LastUpdated != nil {
			lastUpdated = &incarnation.status.State.LastUpdated.Time
		}
		sm := NewDeploymentStateManager(incarnation, lastUpdated)

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

	if len(incarnations) < 1 {
		return nil
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

	latestTarget := incarnations[len(incarnations)-1].target()
	defaultIngressPorts := latestTarget.DefaultIngressPorts
	var istioSidecarConfig *picchuv1alpha1.IstioSidecar
	if latestTarget.Istio != nil {
		istioSidecarConfig = latestTarget.Istio.Sidecar
	}

	err := r.applyPlan(ctx, "Sync Application", &rmplan.SyncApp{
		App:                  r.instance.Spec.App,
		Target:               r.instance.Spec.Target,
		Fleet:                r.instance.Spec.Fleet,
		Namespace:            r.instance.TargetNamespace(),
		Labels:               r.defaultLabels(),
		DeployedRevisions:    revisions,
		AlertRules:           alertRules,
		Ports:                ports,
		HTTPPortFaults:       r.faults,
		IstioSidecarConfig:   istioSidecarConfig,
		DefaultIngressPorts:  defaultIngressPorts,
		DevRoutesServiceHost: r.picchuConfig.DevRoutesServiceHost,
		DevRoutesServicePort: r.picchuConfig.DevRoutesServicePort,
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
func (r *ResourceSyncer) prepareServiceLevelObjectives() ([]*picchuv1alpha1.SlothServiceLevelObjective, picchuv1alpha1.ServiceLevelObjectiveLabels) {
	var slos []*picchuv1alpha1.SlothServiceLevelObjective

	if len(r.incarnations.deployed()) > 0 {
		releasable := r.incarnations.releasable()
		for _, i := range releasable {
			if i.target() != nil {
				return i.target().SlothServiceLevelObjectives, i.target().ServiceLevelObjectiveLabels
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

	// The idea here is we will work through releases from newest to oldest
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

		if firstNonCanary == -1 && incarnation.isRamping {
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
				"percRemaining", percRemaining)
			panic("Assertion failed")
		}
		incarnation.updateCurrentPercent(currentPercent)

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
