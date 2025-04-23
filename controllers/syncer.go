package controllers

import (
	"context"
	"strings"
	"time"

	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/controllers/observe"
	rmplan "go.medium.engineering/picchu/controllers/plan"
	"go.medium.engineering/picchu/controllers/utils"
	"go.medium.engineering/picchu/plan"

	datadogV1 "github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
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
	deliveryClient  client.Client
	deliveryApplier plan.Applier
	planApplier     plan.Applier
	observer        observe.Observer
	instance        *picchuv1alpha1.ReleaseManager
	incarnations    Incarnations
	reconciler      *ReleaseManagerReconciler
	log             logr.Logger
	picchuConfig    utils.Config
	faults          []picchuv1alpha1.HTTPPortFault
	DatadogSLOAPI   DatadogSLOAPI
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
	if err = r.syncDatadogCanaryMonitors(ctx); err != nil {
		return
	}
	if err = r.syncDatadogSLOsMonitors(ctx); err != nil {
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

func (r *ResourceSyncer) applyDeliveryPlan(ctx context.Context, name string, p plan.Plan) error {
	return r.deliveryApplier.Apply(ctx, p)
}

func (r *ResourceSyncer) applyPlan(ctx context.Context, name string, p plan.Plan) error {
	return r.planApplier.Apply(ctx, p)
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

func (r *ResourceSyncer) syncDatadogCanaryMonitors(ctx context.Context) error {
	ddog_slos := r.prepareDatadogSLOs()

	if r.instance.Spec.Fleet == "production" {
		if len(ddog_slos) > 0 {
			if r.picchuConfig.DatadogSLOsFleet != "" && r.picchuConfig.DatadogSLONamespace != "" {
				// only applied to the delivery cluster
				err := r.applyDeliveryPlan(ctx, "Ensure Datadog Namespace", &rmplan.EnsureNamespace{
					Name: r.picchuConfig.DatadogSLONamespace,
				})

				if err != nil {
					return err
				}

				if err := r.applyDeliveryPlan(ctx, "Sync App DatadogCanaryMonitors", &rmplan.SyncDatadogCanaryMonitors{
					// only applied to the datadog namespace
					App:         r.instance.Spec.App,
					Namespace:   r.picchuConfig.DatadogSLONamespace,
					Labels:      r.defaultLabels(),
					DatadogSLOs: ddog_slos,
				}); err != nil {
					return err
				}
			} else {
				r.log.Info("datadog-slo-fleet and datadog-slo-namespace not set, skipping syncDatadogCanaryMonitors", "DatadogSLOsFleet")
			}
		} else {
			if r.picchuConfig.DatadogSLOsFleet != "" && r.picchuConfig.DatadogSLONamespace != "" {
				if err := r.applyDeliveryPlan(ctx, "Delete App DatadogCanaryMonitors", &rmplan.DeleteDatadogCanaryMonitors{
					App:       r.instance.Spec.App,
					Namespace: r.picchuConfig.DatadogSLONamespace,
				}); err != nil {
					return err
				}
			} else {
				r.log.Info("datadog-slo-fleet and datadog-slo-namespace not set, skipping deleteDatadogCanaryMonitors")
			}
		}
	}

	return nil
}

func (r *ResourceSyncer) syncDatadogSLOsMonitors(ctx context.Context) error {
	ddog_slos := r.prepareDatadogSLOs()
	current_ddog_slos := r.currentDatadogSLOs(r.instance.Spec.App)

	if r.instance.Spec.Fleet == "production" {
		if len(ddog_slos) > 0 {
			if r.picchuConfig.DatadogSLOsFleet != "" && r.picchuConfig.DatadogSLONamespace != "" {
				// only applied to the delivery cluster
				err := r.applyDeliveryPlan(ctx, "Ensure Datadog Namespace", &rmplan.EnsureNamespace{
					Name: r.picchuConfig.DatadogSLONamespace,
				})

				if err != nil {
					return err
				}

				// find slos to delete if any
				to_remove := r.datadogSLOsToDelete(ddog_slos, current_ddog_slos)
				if len(to_remove) > 0 {
					// delete
					if err_ddog := r.applyDeliveryPlan(ctx, "Delete Datadog Monitors", &rmplan.DeleteSpecificDatadogMonitors{
						App:       r.instance.Spec.App,
						Namespace: r.picchuConfig.DatadogSLONamespace,
						ToRemove:  to_remove,
					}); err_ddog != nil {
						return err_ddog
					}

				}

				err_ddog := r.applyDeliveryPlan(ctx, "Sync Datadog SLOs", &rmplan.SyncDatadogSLOs{
					App: r.instance.Spec.App,
					// only applied to the datadog namespace
					Namespace:   r.picchuConfig.DatadogSLONamespace,
					DatadogSLOs: ddog_slos,
					Labels:      r.defaultLabels(),
				})

				if err_ddog != nil {
					return err_ddog
				}

				if err_ddog := r.applyDeliveryPlan(ctx, "Sync Datadog Monitors", &rmplan.SyncDatadogMonitors{
					App: r.instance.Spec.App,
					// only applied to the datadog namespace
					Namespace:     r.picchuConfig.DatadogSLONamespace,
					DatadogSLOs:   ddog_slos,
					Labels:        r.defaultLabels(),
					DatadogSLOAPI: r.DatadogSLOAPI,
				}); err_ddog != nil {
					return err_ddog
				}
			} else {
				r.log.Info("datadog-slo-fleet and datadog-slo-namespace not set, skipping syncDatadogSLOS and syncDatadogMonitors")
			}
		} else {
			if r.picchuConfig.DatadogSLOsFleet != "" && r.picchuConfig.DatadogSLONamespace != "" {
				err := r.applyDeliveryPlan(ctx, "Delete Datadog SLOs", &rmplan.DeleteDatadogSLOs{
					App:       r.instance.Spec.App,
					Namespace: r.picchuConfig.DatadogSLONamespace,
				})

				if err != nil {
					return err
				}
				if err_ddog := r.applyDeliveryPlan(ctx, "Delete Datadog Monitors", &rmplan.DeleteDatadogMonitors{
					App:       r.instance.Spec.App,
					Namespace: r.picchuConfig.DatadogSLONamespace,
				}); err_ddog != nil {
					return err_ddog
				}

			} else {
				r.log.Info("datadog-slo-fleet and datadog-slo-namespace not set, skipping deleteDatadogSLOs and deleteDatadogMonitors")
			}
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

// returns the datadogSLOs from the latest released revision
func (r *ResourceSyncer) prepareDatadogSLOs() []*picchuv1alpha1.DatadogSLO {
	var ddog_slos []*picchuv1alpha1.DatadogSLO

	if len(r.incarnations.deployed()) > 0 {
		releasable := r.incarnations.releasable()
		for _, i := range releasable {
			if i.target() != nil {
				if strings.Contains(i.target().Name, "production") {
					return i.target().DatadogSLOs
				}
			}
		}
	}

	return ddog_slos
}

func (r *ResourceSyncer) currentDatadogSLOs(app string) []datadogV1.SearchServiceLevelObjective {
	// query ddog api
	current_ddog_slos, err := r.DatadogSLOAPI.GetCurrentDatadogSLOs(app)
	if err != nil {
		r.log.Error(err, "Error when calling `GetCurrentDatadogSLOs`", "error", err)
		return nil
	}

	return current_ddog_slos
}

func (r *ResourceSyncer) datadogSLOsToDelete(incoming []*picchuv1alpha1.DatadogSLO, current []datadogV1.SearchServiceLevelObjective) []datadogV1.SearchServiceLevelObjective {
	// incoming source of truth

	incoming_set := make(map[string]struct{})

	for _, d := range incoming {
		incoming_set[d.Name] = struct{}{}
	}

	to_remove := []datadogV1.SearchServiceLevelObjective{}
	for _, c := range current {
		name := *c.Data.Attributes.Name
		if _, ok := incoming_set[name]; ok {
			continue
		} else {
			to_remove = append(to_remove, c)
		}
	}

	return to_remove
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
