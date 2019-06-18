package releasemanager

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/controller/releasemanager/plan"
	"go.medium.engineering/picchu/pkg/controller/utils"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/go-logr/logr"
	istiocommonv1alpha1 "github.com/knative/pkg/apis/istio/common/v1alpha1"
	istiov1alpha3 "github.com/knative/pkg/apis/istio/v1alpha3"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Controller interface {
	scheme() *runtime.Scheme
	client() client.Client
	releaseManager() *picchuv1alpha1.ReleaseManager
	log() logr.Logger
	getConfigMaps(context.Context, *client.ListOptions) ([]runtime.Object, error)
	getSecrets(context.Context, *client.ListOptions) ([]runtime.Object, error)
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
	if status.State.Target == "" || status.State.Target == "created" || status.State.Target == "deployed" {
		if revision != nil {
			status.UseNewTagStyle = revision.Spec.UseNewTagStyle
			status.GitTimestamp = &metav1.Time{revision.GitTimestamp()}
			for _, target := range revision.Spec.Targets {
				if target.Name == controller.releaseManager().Spec.Target {
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
	return Incarnation{
		controller: controller,
		tag:        tag,
		revision:   r,
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

	configs := []runtime.Object{}

	secrets, err := i.controller.getSecrets(ctx, configOpts)
	if err != nil {
		return err
	}
	configMaps, err := i.controller.getConfigMaps(ctx, configOpts)
	if err != nil {
		return err
	}

	return i.applyPlan(plan.All(
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
			UseNewTagStyle:     i.status.UseNewTagStyle,
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

func (i *Incarnation) scale() error {
	return i.applyPlan(&plan.ScaleRevision{
		Tag:       i.tag,
		Namespace: i.targetNamespace(),
		Min:       i.divideReplicas(*i.target().Scale.Min),
		Max:       i.divideReplicas(i.target().Scale.Max),
		Labels:    i.defaultLabels(),
		CPUTarget: i.target().Scale.TargetCPUUtilizationPercentage,
	})
}

func (i *Incarnation) applyPlan(p plan.Plan) error {
	ctx := context.TODO()
	err := p.Apply(ctx, i.controller.client(), i.log)
	if err != nil {
		return err
	}
	return i.updateHealthStatus(ctx)
}

func (i *Incarnation) getLog() logr.Logger {
	return i.log
}

func (i *Incarnation) getStatus() *picchuv1alpha1.ReleaseManagerRevisionStatus {
	return i.status
}

func (i *Incarnation) retire() error {
	return i.applyPlan(&plan.RetireRevision{
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

func (i *Incarnation) del() error {
	return i.applyPlan(&plan.DeleteRevision{
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
		if target.Name == i.controller.releaseManager().Spec.Target {
			return &target
		}
	}
	return nil
}

func (i *Incarnation) appName() string {
	return i.controller.releaseManager().Spec.App
}

func (i *Incarnation) targetName() string {
	return i.controller.releaseManager().Spec.Target
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
		picchuv1alpha1.LabelApp:            i.appName(),
		picchuv1alpha1.LabelTag:            i.tag,
		picchuv1alpha1.LabelTarget:         i.targetName(),
		"tag.picchu.medium.engineering":    i.tag,
		"target.picchu.medium.engineering": i.targetName(),
		"app.picchu.medium.engineering":    i.appName(),
	}
}

func (i *Incarnation) setReleaseEligible(flag bool) {
	i.status.ReleaseEligible = flag
}

func (i *Incarnation) updateHealthStatus(ctx context.Context) error {
	rs := &appsv1.ReplicaSet{}
	key := client.ObjectKey{
		Name:      i.tag,
		Namespace: i.targetNamespace(),
	}
	if err := i.controller.client().Get(ctx, key, rs); err != nil {
		if errors.IsNotFound(err) {
			if i.status.CurrentPercent != 0 {
				now := metav1.Now()
				i.status.LastUpdated = &now
				i.status.CurrentPercent = 0
			}
			i.status.Scale.Current = 0
			i.status.Scale.Desired = 0
			i.status.Deleted = true
			return nil
		}
		return err
	}

	i.status.Deleted = false
	i.status.Scale.Desired = *rs.Spec.Replicas
	i.status.Scale.Current = rs.Status.AvailableReplicas
	if i.status.Scale.Current > i.status.Scale.Peak {
		i.status.Scale.Peak = i.status.Scale.Current
	}
	return nil
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
		plan.LogSync(i.log, op, err, rule)
		if err != nil {
			return err
		}
	} else {
		if err := i.controller.client().Delete(ctx, rule); err != nil && !errors.IsNotFound(err) {
			i.log.Info("Failed to delete PrometheusRule")
			return err
		}
		i.log.Info("Deleted PrometheusRule")
	}
	return nil
}

func (i *Incarnation) divideReplicas(count int32) int32 {
	r := utils.Max(count/i.controller.fleetSize(), 1)
	release := i.target().Release
	if release.Eligible {
		// since we sync before incrementing, we'll just err on the side of
		// caution.
		i.log.Info("Compute count", "CurrentPercent", i.status.CurrentPercent, "Increment", release.Rate.Increment, "Count", r)
		c := i.status.CurrentPercent + release.Rate.Increment
		if c > 100 {
			c = 100
		}
		r = int32(float64(r) * float64(c) / float64(100))
		r = utils.Max(r, 1)
		i.log.Info("Resulting count", "Result", r)
	}
	return r
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
		if i.status.State.Target == "released" {
			r = append(r, i)
		}
	}
	if len(r) == 0 {
		i.controller.log().Info("there are no releases, looking for retired release to unretire")
		candidates := i.unretirable()
		unretiredCount := 0
		// We scale retired back up to `peakPercent`. We want to unretire enough to make 100% as
		// fast as possible for cases where we are replacing failed revisions.
		var percRemaining uint32 = 100
		for _, candidate := range candidates {
			if percRemaining <= 0 {
				break
			}
			i.controller.log().Info("Unretiring", "tag", candidates[0].tag)
			candidates[0].setReleaseEligible(true)
			unretiredCount++
			percRemaining -= candidate.getStatus().PeakPercent
		}

		if unretiredCount <= 0 {
			i.controller.log().Info("No available releases retired")
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
