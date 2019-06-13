package releasemanager

import (
	"github.com/go-logr/logr"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
)

var (
	handlers = map[string]StateHandler{
		string(created):  &Created{},
		string(deployed): &Deployed{},
		string(released): &Released{},
		string(retired):  &Retired{},
		string(deleted):  &Deleted{},
		string(failed):   &Failed{},
	}
)

const (
	created  State = "created"
	deployed State = "deployed"
	released State = "released"
	retired  State = "retired"
	deleted  State = "deleted"
	failed   State = "failed"
)

type State string
type InvalidState error

type StateHandler interface {
	tick(Deployment) (State, error)
	reached(Deployment) bool
}

type Deployment interface {
	sync() error
	retire() error
	del() error
	scale() error
	hasRevision() bool
	schedulePermitsRelease() bool
	isAlarmTriggered() bool
	isReleaseEligible() bool
	getStatus() *picchuv1alpha1.ReleaseManagerRevisionStatus
	setState(target string, reached bool)
	getLog() logr.Logger
}

type DeploymentStateManager struct {
	deployment Deployment
}

func NewDeploymentStateManager(deployment Deployment) *DeploymentStateManager {
	return &DeploymentStateManager{deployment}
}

func (s *DeploymentStateManager) tick() error {
	target := s.deployment.getStatus().State.Target
	current := s.deployment.getStatus().State.Current
	s.deployment.getLog().Info("Advancing state", "tag", s.deployment.getStatus().Tag, "current", current, "target", target)
	state, err := handlers[target].tick(s.deployment)
	if err != nil {
		return err
	}
	reached := handlers[target].reached(s.deployment)
	s.deployment.setState(target, reached)
	target = string(state)
	reached = handlers[target].reached(s.deployment)
	s.deployment.setState(target, reached)
	target = s.deployment.getStatus().State.Target
	current = s.deployment.getStatus().State.Current
	s.deployment.getLog().Info("Advanced state", "tag", s.deployment.getStatus().Tag, "current", current, "target", target)
	return nil
}

func (s *DeploymentStateManager) reached() bool {
	target := s.deployment.getStatus().State.Target
	return handlers[target].reached(s.deployment)
}

type Created struct{}

func (s *Created) tick(deployment Deployment) (State, error) {
	if !deployment.hasRevision() {
		return deleted, nil
	}
	return deployed, nil
}

// probably never called since sync'ing triggers transition to deployed
func (s *Created) reached(deployment Deployment) bool {
	return deployment.hasRevision()
}

type Deployed struct{}

func (s *Deployed) tick(deployment Deployment) (State, error) {
	if !deployment.hasRevision() {
		return deleted, nil
	}
	if err := deployment.sync(); err != nil {
		return deployed, err
	}
	if deployment.isAlarmTriggered() {
		return failed, nil
	}
	if deployment.isReleaseEligible() && s.reached(deployment) && deployment.schedulePermitsRelease() {
		return released, nil
	}
	return deployed, nil
}

func (s *Deployed) reached(deployment Deployment) bool {
	scale := deployment.getStatus().Scale
	return scale.Current >= scale.Desired && scale.Current >= 1
}

type Released struct{}

func (s *Released) tick(deployment Deployment) (State, error) {
	if !deployment.hasRevision() {
		return deleted, nil
	}
	if deployment.isAlarmTriggered() {
		return failed, nil
	}
	if !deployment.isReleaseEligible() {
		return retired, nil
	}
	return released, deployment.scale()
}

func (s *Released) reached(deployment Deployment) bool {
	return deployment.getStatus().CurrentPercent > 0
}

type Retired struct{}

func (s *Retired) tick(deployment Deployment) (State, error) {
	if !deployment.hasRevision() {
		return deleted, nil
	}
	if deployment.isReleaseEligible() {
		return deployed, nil
	}
	if deployment.getStatus().CurrentPercent <= 0 {
		return retired, deployment.retire()
	}
	return retired, nil
}

func (s *Retired) reached(deployment Deployment) bool {
	return deployment.getStatus().Scale.Current+deployment.getStatus().Scale.Desired == 0
}

type Deleted struct{}

func (s *Deleted) tick(deployment Deployment) (State, error) {
	err := deployment.del()
	if deployment.hasRevision() {
		return deployed, nil
	}
	return deleted, err
}

func (s *Deleted) reached(deployment Deployment) bool {
	status := deployment.getStatus()
	return status == nil || status.Deleted
}

type Failed struct{}

func (s *Failed) tick(deployment Deployment) (State, error) {
	if !deployment.hasRevision() {
		return deleted, nil
	}
	if !deployment.isAlarmTriggered() {
		return deployed, nil
	}
	if deployment.getStatus().CurrentPercent <= 0 {
		return failed, deployment.retire()
	}
	return failed, nil
}

func (s *Failed) reached(deployment Deployment) bool {
	return deployment.getStatus().Scale.Current == 0
}
