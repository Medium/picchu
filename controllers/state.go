package controllers

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
)

var (
	handlers = map[string]StateHandler{
		string(created):          Created,
		string(deploying):        Deploying,
		string(deployed):         Deployed,
		string(pendingrelease):   PendingRelease,
		string(releasing):        Releasing,
		string(released):         Released,
		string(retiring):         Retiring,
		string(retired):          Retired,
		string(deleting):         Deleting,
		string(deleted):          Deleted,
		string(failing):          Failing,
		string(failed):           Failed,
		string(pendingtest):      PendingTest,
		string(testing):          Testing,
		string(tested):           Tested,
		string(canarying):        Canarying,
		string(canaried):         Canaried,
		string(canaryingdatadog): CanaryingDatadog,
		string(canarieddatadog):  CanariedDatadog,
		string(timingout):        Timingout,
	}
)

const (
	created          State = "created"
	deploying        State = "deploying"
	deployed         State = "deployed"
	pendingrelease   State = "pendingrelease"
	releasing        State = "releasing"
	released         State = "released"
	retiring         State = "retiring"
	retired          State = "retired"
	deleting         State = "deleting"
	deleted          State = "deleted"
	failing          State = "failing"
	failed           State = "failed"
	pendingtest      State = "pendingtest"
	testing          State = "testing"
	tested           State = "tested"
	canarying        State = "canarying"
	canaried         State = "canaried"
	canaryingdatadog State = "canaryingdatadog"
	canarieddatadog  State = "canarieddatadog"
	timingout        State = "timingout"

	DeployingTimeout = (time.Minute * 15)
	CreatedTimeout   = time.Hour
)

var AllStates []string

func init() {
	for name := range handlers {
		AllStates = append(AllStates, name)
	}
}

type State string
type StateHandler func(context.Context, Deployment, *time.Time) (State, error)

type Deployment interface {
	sync(context.Context) error
	retire(context.Context) error
	del(context.Context) error
	syncCanaryRules(context.Context) error
	deleteCanaryRules(context.Context) error
	hasRevision() bool
	schedulePermitsRelease() bool
	markedAsFailed() bool
	isReleaseEligible() bool
	getStatus() *picchuv1alpha1.ReleaseManagerRevisionStatus
	setState(target string)
	getLog() logr.Logger
	isDeployed() bool
	getExternalTestStatus() ExternalTestStatus
	currentPercent() uint32
	peakPercent() uint32
	isCanaryPending() bool
	datadogMonitoring() bool
	isTimingOut() bool
	isExpired() bool
}

// ExternalTestStatus summarizes a RevisionTarget's ExternalTest spec field.
type ExternalTestStatus int

const (
	ExternalTestUnknown ExternalTestStatus = iota
	ExternalTestDisabled
	ExternalTestPending
	ExternalTestStarted
	ExternalTestFailed
	ExternalTestSucceeded
)

func TargetExternalTestStatus(target *picchuv1alpha1.RevisionTarget) ExternalTestStatus {
	t := &target.ExternalTest

	switch {
	case t.Completed && t.Succeeded:
		return ExternalTestSucceeded
	case t.Completed:
		return ExternalTestFailed
	case t.Started:
		return ExternalTestStarted
	case t.Enabled:
		return ExternalTestPending
	default:
		return ExternalTestDisabled
	}
}

func (s ExternalTestStatus) Enabled() bool {
	switch s {
	case ExternalTestUnknown, ExternalTestDisabled:
		return false
	default:
		return true
	}
}

func (s ExternalTestStatus) Finished() bool {
	switch s {
	case ExternalTestSucceeded, ExternalTestFailed:
		return true
	default:
		return false
	}
}

func HasFailed(d Deployment) bool {
	return d.markedAsFailed() || d.getExternalTestStatus() == ExternalTestFailed
}

type DeploymentStateManager struct {
	deployment  Deployment
	lastUpdated *time.Time
}

func NewDeploymentStateManager(deployment Deployment, lastUpdated *time.Time) *DeploymentStateManager {
	return &DeploymentStateManager{
		deployment:  deployment,
		lastUpdated: lastUpdated,
	}
}

func (s *DeploymentStateManager) tick(ctx context.Context) error {
	currentState := s.deployment.getStatus().State.Current
	nextState, err := handlers[currentState](ctx, s.deployment, s.lastUpdated)
	if err != nil {
		return err
	}
	s.deployment.setState(string(nextState))

	return nil
}

func Created(ctx context.Context, deployment Deployment, lastUpdated *time.Time) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if lastUpdated != nil && lastUpdated.Add(CreatedTimeout).Before(time.Now()) {
		deployment.getLog().Error(nil, "State timed out", "state", "created")
		return timingout, nil
	}
	return deploying, nil
}

func Deploying(ctx context.Context, deployment Deployment, lastUpdated *time.Time) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if HasFailed(deployment) {
		return failing, nil
	}
	if err := deployment.sync(ctx); err != nil {
		return deploying, err
	}
	if deployment.isDeployed() {
		return deployed, nil
	}
	if lastUpdated != nil && lastUpdated.Add(DeployingTimeout).Before(time.Now()) {
		deployment.getLog().Error(nil, "State timed out", "state", "deploying")
		return timingout, nil
	}
	return deploying, nil
}

func Deployed(ctx context.Context, deployment Deployment, lastUpdated *time.Time) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if HasFailed(deployment) {
		return failing, nil
	}
	if err := deployment.sync(ctx); err != nil {
		return deployed, err
	}
	if !deployment.isDeployed() {
		return deploying, nil
	}
	switch deployment.getExternalTestStatus() {
	case ExternalTestSucceeded:
		return tested, nil
	case ExternalTestStarted:
		return testing, nil
	case ExternalTestPending:
		return pendingtest, nil
	}
	if deployment.isReleaseEligible() {
		if deployment.isCanaryPending() {
			// check if datadog or prometheus is enabled
			if deployment.datadogMonitoring() {
				return canaryingdatadog, nil
			} else {
				return canarying, nil
			}
		}
		return pendingrelease, nil
	}
	if deployment.isExpired() {
		return retiring, nil
	}
	return deployed, nil
}

func PendingTest(ctx context.Context, deployment Deployment, lastUpdated *time.Time) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if HasFailed(deployment) {
		return failing, nil
	}
	externalTestStatus := deployment.getExternalTestStatus()
	if externalTestStatus == ExternalTestSucceeded {
		return tested, nil
	}
	if deployment.isTimingOut() {
		return timingout, nil
	}
	if externalTestStatus == ExternalTestStarted {
		return testing, nil
	}
	if err := deployment.sync(ctx); err != nil {
		return pendingtest, err
	}
	if lastUpdated != nil && lastUpdated.Add(DeployingTimeout).Before(time.Now()) {
		deployment.getLog().Error(nil, "State timed out", "state", "pending test")
		return timingout, nil
	}
	return pendingtest, nil
}

func Testing(ctx context.Context, deployment Deployment, lastUpdated *time.Time) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	// Only transition to failure on external test failures, ignoring all other failure states.
	if deployment.getExternalTestStatus() == ExternalTestFailed {
		return failing, nil
	}
	externalTestStatus := deployment.getExternalTestStatus()
	if externalTestStatus == ExternalTestSucceeded {
		return tested, nil
	}
	if deployment.isTimingOut() {
		return timingout, nil
	}
	if externalTestStatus == ExternalTestPending {
		return pendingtest, nil
	}
	if !externalTestStatus.Enabled() {
		return deploying, nil
	}
	if err := deployment.sync(ctx); err != nil {
		return testing, err
	}
	if lastUpdated != nil && lastUpdated.Add(DeployingTimeout).Before(time.Now()) {
		deployment.getLog().Error(nil, "State timed out", "state", "testing")
		return timingout, nil
	}
	return testing, nil
}

func Tested(ctx context.Context, deployment Deployment, lastUpdated *time.Time) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if HasFailed(deployment) {
		return failing, nil
	}
	externalTestStatus := deployment.getExternalTestStatus()
	if externalTestStatus == ExternalTestPending {
		return pendingtest, nil
	}
	if !externalTestStatus.Finished() {
		return testing, nil
	}
	if deployment.isReleaseEligible() {
		if deployment.isCanaryPending() {
			// check if datadog or prometheus is enabled
			if deployment.datadogMonitoring() {
				return canaryingdatadog, nil
			} else {
				return canarying, nil
			}
		}
		return pendingrelease, nil
	}
	if lastUpdated != nil && lastUpdated.Add(DeployingTimeout).Before(time.Now()) {
		deployment.getLog().Error(nil, "State timed out", "state", "tested")
		return timingout, nil
	}
	return tested, nil
}

func PendingRelease(ctx context.Context, deployment Deployment, lastUpdated *time.Time) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if deployment.markedAsFailed() {
		return failing, nil
	}
	if !deployment.isReleaseEligible() {
		return retiring, nil
	}
	if deployment.schedulePermitsRelease() {
		return releasing, nil
	}
	if err := deployment.sync(ctx); err != nil {
		return pendingrelease, err
	}
	return pendingrelease, nil
}

func Releasing(ctx context.Context, deployment Deployment, lastUpdated *time.Time) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if deployment.markedAsFailed() {
		return failing, nil
	}
	if !deployment.isReleaseEligible() {
		return retiring, nil
	}
	if err := deployment.sync(ctx); err != nil {
		return releasing, err
	}
	if deployment.peakPercent() >= 100 {
		return released, nil
	}
	return releasing, nil
}

func Released(ctx context.Context, deployment Deployment, lastUpdated *time.Time) (State, error) {
	return Releasing(ctx, deployment, lastUpdated)
}

func Retiring(ctx context.Context, deployment Deployment, lastUpdated *time.Time) (State, error) {
	if deployment.currentPercent() > 0 {
		return retiring, nil
	}
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if deployment.markedAsFailed() {
		return failing, nil
	}
	if deployment.isReleaseEligible() {
		return deploying, nil
	}
	if deployment.currentPercent() <= 0 {
		return retired, deployment.retire(ctx)
	}
	return retiring, nil
}

func Retired(ctx context.Context, deployment Deployment, lastUpdated *time.Time) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if deployment.markedAsFailed() {
		return failing, nil
	}
	if deployment.isReleaseEligible() {
		return deploying, nil
	}
	return retired, deployment.retire(ctx)
}

func Deleting(ctx context.Context, deployment Deployment, lastUpdated *time.Time) (State, error) {
	if deployment.currentPercent() > 0 {
		return deleting, nil
	}
	if deployment.hasRevision() {
		return deploying, nil
	}

	if err := deployment.deleteCanaryRules(ctx); err != nil {
		return deleting, err
	}

	if deployment.currentPercent() <= 0 {
		return deleted, deployment.del(ctx)
	}
	if lastUpdated != nil && lastUpdated.Add(DeployingTimeout).Before(time.Now()) {
		deployment.getLog().Error(nil, "State timed out", "state", "deleting")
		return timingout, nil
	}
	return deleting, nil
}

func Deleted(ctx context.Context, deployment Deployment, lastUpdated *time.Time) (State, error) {
	if deployment.hasRevision() {
		return deploying, nil
	}
	return deleted, nil
}

func Failing(ctx context.Context, deployment Deployment, lastUpdated *time.Time) (State, error) {
	if deployment.currentPercent() > 0 {
		return failing, nil
	}
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if !HasFailed(deployment) {
		return deploying, nil
	}
	if err := deployment.deleteCanaryRules(ctx); err != nil {
		return failing, err
	}
	if deployment.currentPercent() <= 0 {
		return failed, deployment.retire(ctx)
	}
	return failing, nil
}

func Failed(ctx context.Context, deployment Deployment, lastUpdated *time.Time) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if !HasFailed(deployment) {
		return deploying, nil
	}
	return failed, deployment.retire(ctx)
}

func Canarying(ctx context.Context, deployment Deployment, lastUpdated *time.Time) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if deployment.markedAsFailed() {
		return failing, nil
	}
	if err := deployment.syncCanaryRules(ctx); err != nil {
		return canarying, err
	}

	if err := deployment.sync(ctx); err != nil {
		return canarying, err
	}
	if !deployment.isCanaryPending() {
		return canaried, nil
	}
	return canarying, nil
}

func CanaryingDatadog(ctx context.Context, deployment Deployment, lastUpdated *time.Time) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if deployment.markedAsFailed() {
		return failing, nil
	}
	if err := deployment.sync(ctx); err != nil {
		return canaryingdatadog, err
	}
	if !deployment.isCanaryPending() {
		return canarieddatadog, nil
	}
	return canaryingdatadog, nil
}

func Canaried(ctx context.Context, deployment Deployment, lastUpdated *time.Time) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if deployment.markedAsFailed() {
		return failing, nil
	}
	if err := deployment.deleteCanaryRules(ctx); err != nil {
		return canaried, err
	}
	if err := deployment.sync(ctx); err != nil {
		return canarying, err
	}
	if deployment.isReleaseEligible() {
		return pendingrelease, nil
	}
	return canaried, nil
}

func CanariedDatadog(ctx context.Context, deployment Deployment, lastUpdated *time.Time) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if deployment.markedAsFailed() {
		return failing, nil
	}
	if err := deployment.sync(ctx); err != nil {
		return canaryingdatadog, err
	}
	if deployment.isReleaseEligible() {
		return pendingrelease, nil
	}
	return canarieddatadog, nil
}

// Timingout state is responsible for cleaning up a timing out release
func Timingout(ctx context.Context, deployment Deployment, lastUpdated *time.Time) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if HasFailed(deployment) {
		return failing, nil
	}
	return timingout, nil
}
