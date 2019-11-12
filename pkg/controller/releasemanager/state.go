package releasemanager

import (
	"context"

	"github.com/go-logr/logr"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
)

var (
	handlers = map[string]StateHandler{
		string(created):        Created,
		string(deploying):      Deploying,
		string(deployed):       Deployed,
		string(pendingrelease): PendingRelease,
		string(releasing):      Releasing,
		string(released):       Released,
		string(retiring):       Retiring,
		string(retired):        Retired,
		string(deleting):       Deleting,
		string(deleted):        Deleted,
		string(failing):        Failing,
		string(failed):         Failed,
		string(pendingtest):    PendingTest,
		string(testing):        Testing,
		string(tested):         Tested,
		string(canarying):      Canarying,
		string(canaried):       Canaried,
	}
)

const (
	created        State = "created"
	deploying      State = "deploying"
	deployed       State = "deployed"
	pendingrelease State = "pendingrelease"
	releasing      State = "releasing"
	released       State = "released"
	retiring       State = "retiring"
	retired        State = "retired"
	deleting       State = "deleting"
	deleted        State = "deleted"
	failing        State = "failing"
	failed         State = "failed"
	pendingtest    State = "pendingtest"
	testing        State = "testing"
	tested         State = "tested"
	canarying      State = "canarying"
	canaried       State = "canaried"
)

type State string
type StateHandler func(context.Context, Deployment) (State, error)

type Deployment interface {
	sync(context.Context) error
	retire(context.Context) error
	del(context.Context) error
	syncCanaryRules(context.Context) error
	deleteCanaryRules(context.Context) error
	syncSLIRules(context.Context) error
	deleteSLIRules(context.Context) error
	hasRevision() bool
	schedulePermitsRelease() bool
	isAlarmTriggered() bool
	isReleaseEligible() bool
	getStatus() *picchuv1alpha1.ReleaseManagerRevisionStatus
	setState(target string)
	getLog() logr.Logger
	isDeployed() bool
	isTestPending() bool
	isTestStarted() bool
	didTestSucceed() bool
	currentPercent() uint32
	peakPercent() uint32
	isCanaryPending() bool
}

type DeploymentStateManager struct {
	deployment Deployment
}

func NewDeploymentStateManager(deployment Deployment) *DeploymentStateManager {
	return &DeploymentStateManager{deployment}
}

func (s *DeploymentStateManager) tick(ctx context.Context) error {
	current := s.deployment.getStatus().State.Current
	s.deployment.getLog().Info("Advancing state", "tag", s.deployment.getStatus().Tag, "current", current)
	state, err := handlers[current](ctx, s.deployment)
	if err != nil {
		return err
	}
	s.deployment.setState(string(state))
	s.deployment.getLog().Info("Advanced state", "tag", s.deployment.getStatus().Tag, "current", string(state))
	return nil
}

func Created(ctx context.Context, deployment Deployment) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	return deploying, nil
}

func Deploying(ctx context.Context, deployment Deployment) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if deployment.isAlarmTriggered() {
		return failing, nil
	}
	if err := deployment.sync(ctx); err != nil {
		return deploying, err
	}
	if deployment.isDeployed() {
		return deployed, nil
	}
	return deploying, nil
}

func Deployed(ctx context.Context, deployment Deployment) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if deployment.isAlarmTriggered() {
		return failing, nil
	}
	if err := deployment.sync(ctx); err != nil {
		return deployed, err
	}
	if !deployment.isDeployed() {
		return deploying, nil
	}
	if deployment.isTestPending() {
		return pendingtest, nil
	}
	if deployment.isCanaryPending() {
		return canarying, nil
	}
	if deployment.isReleaseEligible() {
		return pendingrelease, nil
	}
	return deployed, nil
}

func PendingTest(ctx context.Context, deployment Deployment) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if deployment.isAlarmTriggered() {
		return failing, nil
	}
	if deployment.isTestStarted() {
		return testing, nil
	}
	return pendingtest, nil
}

func Testing(ctx context.Context, deployment Deployment) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if deployment.isAlarmTriggered() {
		return failing, nil
	}
	if !deployment.isTestPending() {
		if deployment.didTestSucceed() {
			return tested, nil
		}
		return failing, nil
	}
	return testing, nil
}

func Tested(ctx context.Context, deployment Deployment) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if deployment.isAlarmTriggered() {
		return failing, nil
	}
	if deployment.isCanaryPending() {
		return canarying, nil
	}
	if deployment.isReleaseEligible() {
		return pendingrelease, nil
	}
	return tested, nil
}

func PendingRelease(ctx context.Context, deployment Deployment) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if deployment.isAlarmTriggered() {
		return failing, nil
	}
	if !deployment.isReleaseEligible() {
		return retiring, nil
	}
	if deployment.schedulePermitsRelease() {
		return releasing, nil
	}
	return pendingrelease, nil
}

func Releasing(ctx context.Context, deployment Deployment) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if deployment.isAlarmTriggered() {
		return failing, nil
	}
	if !deployment.isReleaseEligible() {
		return retiring, nil
	}
	if err := deployment.sync(ctx); err != nil {
		return releasing, err
	}
	if err := deployment.syncSLIRules(ctx); err != nil {
		return releasing, err
	}
	if deployment.peakPercent() >= 100 {
		return released, nil
	}
	return releasing, nil
}

func Released(ctx context.Context, deployment Deployment) (State, error) {
	return Releasing(ctx, deployment)
}

func Retiring(ctx context.Context, deployment Deployment) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if deployment.isAlarmTriggered() {
		return failing, nil
	}
	if deployment.isReleaseEligible() {
		return deploying, nil
	}
	if err := deployment.deleteSLIRules(ctx); err != nil {
		return retiring, err
	}
	if deployment.currentPercent() <= 0 {
		return retired, deployment.retire(ctx)
	}
	return retiring, nil
}

func Retired(ctx context.Context, deployment Deployment) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if deployment.isAlarmTriggered() {
		return failing, nil
	}
	if deployment.isReleaseEligible() {
		return deploying, nil
	}
	return retired, deployment.retire(ctx)
}

func Deleting(ctx context.Context, deployment Deployment) (State, error) {
	if deployment.hasRevision() {
		return deploying, nil
	}

	if err := deployment.deleteCanaryRules(ctx); err != nil {
		return deleting, err
	}

	if err := deployment.deleteSLIRules(ctx); err != nil {
		return deleting, err
	}

	if deployment.currentPercent() <= 0 {
		return deleted, deployment.del(ctx)
	}

	return deleting, nil
}

func Deleted(ctx context.Context, deployment Deployment) (State, error) {
	if deployment.hasRevision() {
		return deploying, nil
	}
	return deleted, nil
}

func Failing(ctx context.Context, deployment Deployment) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if !deployment.isAlarmTriggered() {
		return deploying, nil
	}
	if err := deployment.deleteCanaryRules(ctx); err != nil {
		return failing, err
	}
	if err := deployment.deleteSLIRules(ctx); err != nil {
		return failing, err
	}
	if deployment.currentPercent() <= 0 {
		return failed, deployment.retire(ctx)
	}
	return failing, nil
}

func Failed(ctx context.Context, deployment Deployment) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if !deployment.isAlarmTriggered() {
		return deploying, nil
	}
	return failed, deployment.retire(ctx)
}

func Canarying(ctx context.Context, deployment Deployment) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if deployment.isAlarmTriggered() {
		return failing, nil
	}
	if err := deployment.syncCanaryRules(ctx); err != nil {
		return canarying, err
	}
	if !deployment.isCanaryPending() {
		return canaried, nil
	}
	return canarying, nil
}

func Canaried(ctx context.Context, deployment Deployment) (State, error) {
	if !deployment.hasRevision() {
		return deleting, nil
	}
	if deployment.isAlarmTriggered() {
		return failing, nil
	}
	if err := deployment.deleteCanaryRules(ctx); err != nil {
		return canaried, err
	}
	if deployment.isReleaseEligible() {
		return pendingrelease, nil
	}
	return canaried, nil
}
