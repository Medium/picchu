package releasemanager

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/controller/releasemanager/observe"
	"go.medium.engineering/picchu/pkg/controller/releasemanager/plan"

	"github.com/go-logr/logr"
	istiocommonv1alpha1 "github.com/knative/pkg/apis/istio/common/v1alpha1"
	istiov1alpha3 "github.com/knative/pkg/apis/istio/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Controller interface {
	applyPlan(context.Context, string, plan.Plan) error
	divideReplicas(int32, int32) int32
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
}

func NewIncarnation(controller Controller, tag string, revision *picchuv1alpha1.Revision, log logr.Logger, di *observe.DeploymentInfo) Incarnation {
	status := controller.getReleaseManager().RevisionStatus(tag)
	if status.State.Target == "" || status.State.Target == "created" || status.State.Target == "deployed" {
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
		if status.State.Target == "" {
			status.State.Target = "created"
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
		controller: controller,
		tag:        tag,
		revision:   r,
		log:        log,
		status:     status,
	}
	i.update(di)
	return i
}

// = Start Deployment interface
// Returns true if all target clusters are in deployed state
func (i *Incarnation) isDeployed() bool {
	return i.deployed
}

func (i *Incarnation) currentPercent() uint32 {
	if i.getStatus() == nil {
		return 0
	}
	return i.getStatus().CurrentPercent
}

// Returns true if testing is disabled or testing has passed
func (i *Incarnation) isTestPending() bool {
	if !i.hasRevision() {
		return false
	}
	if i.revision.Spec.Failed {
		return false
	}
	return i.target().IsExternalTestPending()
}

// Returns true if testing is started
func (i *Incarnation) isTestStarted() bool {
	if !i.hasRevision() {
		return false
	}
	return i.target().ExternalTest.Started
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

	return i.controller.applyPlan(ctx, "Sync and Scale Revision", plan.All(
		&plan.SyncRevision{
			App:                i.appName(),
			Tag:                i.tag,
			Namespace:          i.targetNamespace(),
			Labels:             i.defaultLabels(),
			Configs:            append(append(configs, secrets...), configMaps...),
			Ports:              i.revision.Spec.Ports,
			Replicas:           i.divideReplicas(i.target().Scale.Default),
			Image:              i.image(),
			Resources:          i.target().Resources,
			IAMRole:            i.target().AWS.IAM.RoleARN,
			ServiceAccountName: i.target().ServiceAccountName,
			ReadinessProbe:     i.target().ReadinessProbe,
			LivenessProbe:      i.target().LivenessProbe,
		},
		&plan.ScaleRevision{
			Tag:       i.tag,
			Namespace: i.targetNamespace(),
			Min:       i.divideReplicas(*i.target().Scale.Min),
			Max:       i.divideReplicas(i.target().Scale.Max),
			Labels:    i.defaultLabels(),
			CPUTarget: i.target().Scale.TargetCPUUtilizationPercentage,
		},
	))
}

func (i *Incarnation) scale(ctx context.Context) error {
	return i.controller.applyPlan(ctx, "Scale Revision", &plan.ScaleRevision{
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

func (i *Incarnation) retire(ctx context.Context) error {
	return i.controller.applyPlan(ctx, "Retire Revision", &plan.RetireRevision{
		Tag:       i.tag,
		Namespace: i.targetNamespace(),
	})
}

func (i *Incarnation) schedulePermitsRelease() bool {
	if i.revision == nil {
		return false
	}

	// If previously released, allow outside of schedule
	if i.status.PeakPercent > 0 {
		return true
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
	return i.controller.applyPlan(ctx, "Delete Revision", &plan.DeleteRevision{
		Labels:    i.defaultLabels(),
		Namespace: i.targetNamespace(),
	})
}

func (i *Incarnation) hasRevision() bool {
	return i.revision != nil
}

func (i *Incarnation) isAlarmTriggered() bool {
	if i.revision != nil {
		return i.revision.Spec.Failed
	}
	return false
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
		picchuv1alpha1.LabelApp:    i.appName(),
		picchuv1alpha1.LabelTag:    i.tag,
		picchuv1alpha1.LabelTarget: i.targetName(),
	}
}

func (i *Incarnation) setReleaseEligible(flag bool) {
	i.status.ReleaseEligible = flag
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
	if i.revision == nil {
		return 0
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
	itemSet    map[string]Incarnation
	controller Controller
}

func newIncarnationCollection(controller Controller, revisionList *picchuv1alpha1.RevisionList, observation *observe.Observation) *IncarnationCollection {
	ic := &IncarnationCollection{
		controller: controller,
		itemSet:    make(map[string]Incarnation),
	}
	rm := controller.getReleaseManager()
	for _, r := range revisionList.Items {
		tag := r.Spec.App.Tag
		l := controller.getLog().WithValues("Tag", tag)
		di := observation.ForTag(tag)
		ic.itemSet[tag] = NewIncarnation(controller, tag, &r, l, di)
	}
	// add any deleted revisions that still have status
	for _, r := range rm.Status.Revisions {
		l := controller.getLog().WithValues("Tag", r.Tag)
		if _, ok := ic.itemSet[r.Tag]; ok {
			continue
		}
		di := observation.ForTag(r.Tag)
		ic.itemSet[r.Tag] = NewIncarnation(controller, r.Tag, nil, l, di)
	}

	return ic
}

// deployed returns deployed incarnation
func (i *IncarnationCollection) deployed() []Incarnation {
	r := []Incarnation{}
	for _, i := range i.sorted() {
		state := i.status.State.Target
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
		if i.status.State.Target == "released" {
			r = append(r, i)
		}
	}
	if len(r) == 0 {
		i.controller.getLog().Info("there are no releases, looking for retired release to unretire")
		candidates := i.unretirable()
		unretiredCount := 0
		// We scale retired back up to `peakPercent`. We want to unretire enough to make 100% as
		// fast as possible for cases where we are replacing failed revisions.
		var percRemaining uint32 = 100
		for _, candidate := range candidates {
			if percRemaining <= 0 {
				break
			}
			i.controller.getLog().Info("Unretiring", "tag", candidate.tag)
			candidate.setReleaseEligible(true)
			unretiredCount++
			percRemaining -= candidate.getStatus().PeakPercent
		}

		if unretiredCount <= 0 {
			i.controller.getLog().Info("No available releases retired")
		}
	}
	for _, i := range i.sorted() {
		if i.status.State.Current == "released" && i.status.State.Target != "released" {
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
	return r
}

func (i *IncarnationCollection) update(observation *observe.Observation) {
	for _, item := range i.itemSet {
		item.update(observation.ForTag(item.tag))
	}
}
