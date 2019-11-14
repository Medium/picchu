package releasemanager

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/controller/releasemanager/observe"
	rmplan "go.medium.engineering/picchu/pkg/controller/releasemanager/plan"
	"go.medium.engineering/picchu/pkg/plan"

	"github.com/go-logr/logr"
	istiocommonv1alpha1 "github.com/knative/pkg/apis/istio/common/v1alpha1"
	istiov1alpha3 "github.com/knative/pkg/apis/istio/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Controller interface {
	expectedTotalReplicas(count int32, percent int32) int32
	applyPlan(context.Context, string, plan.Plan) error
	divideReplicas(count int32, percent int32) int32
	getReleaseManager() *picchuv1alpha1.ReleaseManager
	getLog() logr.Logger
	getConfigMaps(context.Context, *client.ListOptions) ([]runtime.Object, error)
	getSecrets(context.Context, *client.ListOptions) ([]runtime.Object, error)
}

type Incarnation struct {
	deployed   bool
	controller Controller
	tag        string
	revision   *picchuv1alpha1.Revision
	log        logr.Logger
	status     *picchuv1alpha1.ReleaseManagerRevisionStatus

	// TODO(lyra): should this be a parameter of schedulePermitsRelease()?
	humaneReleasesEnabled bool
}

func NewIncarnation(controller Controller, tag string, revision *picchuv1alpha1.Revision, log logr.Logger, di *observe.DeploymentInfo, humaneReleasesEnabled bool) *Incarnation {
	status := controller.getReleaseManager().RevisionStatus(tag)

	if status.State.Current == "" {
		status.State.Current = "created"
	}

	if status.State.Current == "created" || status.State.Current == "deploying" {
		if revision != nil {
			status.GitTimestamp = &metav1.Time{revision.GitTimestamp()}
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
		controller:            controller,
		tag:                   tag,
		revision:              r,
		log:                   log,
		status:                status,
		humaneReleasesEnabled: humaneReleasesEnabled,
	}
	i.update(di)
	return &i
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
	ports := i.revision.Spec.Ports
	target := i.target()

	if target != nil {
		if len(target.Ports) > 0 {
			ports = target.Ports
		} else {
			i.log.Info("revision.spec.ports is deprecated")
		}
	}

	return ports
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
	if target := i.target(); target != nil {
		return TargetExternalTestStatus(target)
	}

	return ExternalTestUnknown
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

	configs := []runtime.Object{}

	secrets, err := i.controller.getSecrets(ctx, configOpts)
	if err != nil {
		return err
	}
	configMaps, err := i.controller.getConfigMaps(ctx, configOpts)
	if err != nil {
		return err
	}

	syncPlan := &rmplan.SyncRevision{
		App:                i.appName(),
		Tag:                i.tag,
		Namespace:          i.targetNamespace(),
		Labels:             i.defaultLabels(),
		Configs:            append(append(configs, secrets...), configMaps...),
		Ports:              i.ports(),
		Replicas:           i.divideReplicas(i.target().Scale.Default),
		Image:              i.image(),
		Resources:          i.target().Resources,
		IAMRole:            i.target().AWS.IAM.RoleARN,
		ServiceAccountName: i.target().ServiceAccountName,
		ReadinessProbe:     i.target().ReadinessProbe,
		LivenessProbe:      i.target().LivenessProbe,
		MinReadySeconds:    i.target().Scale.MinReadySeconds,
	}
	scalePlan := &rmplan.ScaleRevision{
		Tag:       i.tag,
		Namespace: i.targetNamespace(),
		Min:       i.divideReplicas(*i.target().Scale.Min),
		Max:       i.divideReplicas(i.target().Scale.Max),
		Labels:    i.defaultLabels(),
		CPUTarget: i.target().Scale.TargetCPUUtilizationPercentage,
	}

	currentState := State(i.status.State.Current)
	if currentState == canaried || currentState == pendingrelease {
		syncPlan.Replicas = 0
	}

	return i.controller.applyPlan(ctx, "Sync and Scale Revision", plan.All(syncPlan, scalePlan))
}

func (i *Incarnation) syncCanaryRules(ctx context.Context) error {
	return i.controller.applyPlan(ctx, "Sync Canary Rules", &rmplan.SyncAlerts{
		App:                    i.appName(),
		Namespace:              i.targetNamespace(),
		Tag:                    i.tag,
		Target:                 i.target().Name,
		ServiceLevelObjectives: i.target().ServiceLevelObjectives,
		AlertType:              rmplan.Canary,
	})
}

func (i *Incarnation) deleteCanaryRules(ctx context.Context) error {
	return i.controller.applyPlan(ctx, "Delete Canary Rules", &rmplan.DeleteAlerts{
		App:       i.appName(),
		Namespace: i.targetNamespace(),
		Tag:       i.tag,
		AlertType: rmplan.Canary,
	})
}

func (i *Incarnation) syncSLIRules(ctx context.Context) error {
	return i.controller.applyPlan(ctx, "Sync SLI Rules", &rmplan.SyncAlerts{
		App:                    i.appName(),
		Namespace:              i.targetNamespace(),
		Tag:                    i.tag,
		Target:                 i.target().Name,
		ServiceLevelObjectives: i.target().ServiceLevelObjectives,
		AlertType:              rmplan.SLI,
	})
}

func (i *Incarnation) deleteSLIRules(ctx context.Context) error {
	return i.controller.applyPlan(ctx, "Delete SLI Rules", &rmplan.DeleteAlerts{
		App:       i.appName(),
		Namespace: i.targetNamespace(),
		Tag:       i.tag,
		AlertType: rmplan.SLI,
	})
}

func (i *Incarnation) scale(ctx context.Context) error {
	return i.controller.applyPlan(ctx, "Scale Revision", &rmplan.ScaleRevision{
		Tag:       i.tag,
		Namespace: i.targetNamespace(),
		Min:       i.divideReplicas(*i.target().Scale.Min),
		Max:       i.divideReplicas(i.target().Scale.Max),
		Labels:    i.defaultLabels(),
		CPUTarget: i.target().Scale.TargetCPUUtilizationPercentage,
	})
}

func (i *Incarnation) getLog() logr.Logger {
	return i.log
}

func (i *Incarnation) getStatus() *picchuv1alpha1.ReleaseManagerRevisionStatus {
	return i.status
}

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
	if !i.humaneReleasesEnabled && i.target().Release.Schedule != "always" {
		return false
	}

	// if the revision was created during release schedule or we are currently
	// in the release schdeule
	times := []time.Time{
		i.revision.GetCreationTimestamp().Time,
		time.Now(),
	}
	for _, t := range times {
		if schedulePermitsRelease(t, i.target().Release.Schedule) {
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
	if state == "canarying" {
		if i.status.CanaryStartTimestamp == nil {
			t := metav1.Now()
			i.status.CanaryStartTimestamp = &t
		}
	}
	i.status.State.Current = state
	i.status.State.Target = state
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
		picchuv1alpha1.LabelApp:        i.appName(),
		picchuv1alpha1.LabelTag:        i.tag,
		picchuv1alpha1.LabelTarget:     i.targetName(),
		picchuv1alpha1.LabelK8sName:    i.appName(),
		picchuv1alpha1.LabelK8sVersion: i.tag,
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
		i.deployed = di.Deployed
		i.status.Deleted = false
		i.status.Scale.Desired = di.Desired
		i.status.Scale.Current = di.Current
		if i.status.Scale.Current > i.status.Scale.Peak {
			i.status.Scale.Peak = i.status.Scale.Current
		}
	}
}

func (i *Incarnation) taggedRoutes(privateGateway string, serviceHost string) []istiov1alpha3.HTTPRoute {
	http := []istiov1alpha3.HTTPRoute{}
	if i.revision == nil {
		return http
	}
	overrideLabel := fmt.Sprintf("pin/%s", i.appName())
	for _, port := range i.ports() {
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

func (i *Incarnation) divideReplicas(count int32) int32 {
	release := i.target().Release
	perc := int32(100)
	if release.Eligible {
		// since we sync before incrementing, we'll just err on the side of
		// caution and use the next increment percent.
		perc = int32(i.status.CurrentPercent + release.Rate.Increment)
	}
	return i.controller.divideReplicas(count, perc)
}

func (i *Incarnation) currentPercentTarget(max uint32) uint32 {
	status := i.getStatus()
	target := i.target()

	if i.revision == nil || status == nil {
		return 0
	}

	currentScale := status.CurrentPercent
	desiredScale := i.currentDesiredScale(max)

	expectedReplicas := status.Scale.Desired
	targetExists := false
	if target != nil && target.Scale.Min != nil {
		targetExists = true
		expectedReplicas = i.controller.expectedTotalReplicas(*target.Scale.Min, int32(desiredScale))
	}
	i.getLog().Info(
		"TODO(bob): remove me :: debug scaling",
		"currentScale", currentScale,
		"desiredScale", desiredScale,
		"desiredReplicas", status.Scale.Desired,
		"expectedReplicas", expectedReplicas,
		"currentReplicas", status.Scale.Current,
		"targetExists", targetExists,
	)
	if desiredScale > currentScale && status.Scale.Current < expectedReplicas {
		i.getLog().Info(
			"Deferring ramp-up; not all desired replicas are ready",
			"currentScale", currentScale,
			"desiredScale", desiredScale,
			"desiredReplicas", status.Scale.Desired,
			"currentReplicas", status.Scale.Current,
		)
		return currentScale
	}

	return desiredScale
}

func (i *Incarnation) currentDesiredScale(max uint32) uint32 {
	status := i.getStatus()

	if i.revision == nil || status == nil {
		return 0
	}

	if status.State.Current == "canarying" {
		if max < i.target().Canary.Percent {
			return max
		}
		return i.target().Canary.Percent
	}

	return LinearScale(*i, max, time.Now())
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
	itemSet    map[string]*Incarnation
	controller Controller
}

func newIncarnationCollection(controller Controller, revisionList *picchuv1alpha1.RevisionList, observation *observe.Observation, humaneReleasesEnabled bool) *IncarnationCollection {
	ic := &IncarnationCollection{
		controller: controller,
		itemSet:    make(map[string]*Incarnation),
	}
	add := func(tag string, revision *picchuv1alpha1.Revision) {
		if _, ok := ic.itemSet[tag]; revision == nil && ok {
			return
		}

		l := controller.getLog().WithValues("Tag", tag)
		di := observation.ForTag(tag)
		ic.itemSet[tag] = NewIncarnation(controller, tag, revision, l, di, humaneReleasesEnabled)
	}

	for _, r := range revisionList.Items {
		add(r.Spec.App.Tag, &r)
	}

	// add any deleted revisions that still have status
	rm := controller.getReleaseManager()
	for _, r := range rm.Status.Revisions {
		add(r.Tag, nil)
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
	if len(r) == 0 && len(i.willRelease()) == 0 {
		i.controller.getLog().Info("there are no releases, looking for retired release to unretire")
		candidates := i.unretirable()
		if len(candidates) > 0 {
			candidates[0].fastRelease()
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
		elig := i.isReleaseEligible()
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
		return a.After(b)
	})
	return
}

func (i *IncarnationCollection) update(observation *observe.Observation) {
	for _, item := range i.itemSet {
		item.update(observation.ForTag(item.Tag()))
	}
}
