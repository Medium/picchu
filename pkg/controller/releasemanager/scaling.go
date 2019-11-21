package releasemanager

import (
	"time"

	"go.medium.engineering/picchu/pkg/controller/releasemanager/scaling"
)

type ScalableTargetAdapter struct {
	Incarnation
}

const ScalingFactor = 0.9

// IsReconciled returns true if the target is considered ready to be scaled to the next increment.
func (s *ScalableTargetAdapter) IsReconciled(desiredScale uint32) bool {
	target := s.Incarnation.target()
	status := s.Incarnation.status
	controller := s.Incarnation.controller
	log := s.Incarnation.getLog()

	if target == nil || target.Scale.Min == nil {
		return false
	}

	expectedReplicas := int32(float64(controller.expectedTotalReplicas(*target.Scale.Min, int32(desiredScale))) * ScalingFactor)

	log.Info(
		"Computed expectedReplicas",
		"desiredScale", desiredScale,
		"expectedReplicas", expectedReplicas,
		"currentReplicas", status.Scale.Current,
	)
	if status.Scale.Current < expectedReplicas {
		log.Info(
			"Deferring ramp-up; not all desired replicas are ready",
			"desiredScale", desiredScale,
			"expectedReplicas", expectedReplicas,
			"currentReplicas", status.Scale.Current,
		)
		return false
	}

	return true
}

func (s *ScalableTargetAdapter) CurrentPercent() uint32 {
	return s.Incarnation.status.CurrentPercent
}

func (s *ScalableTargetAdapter) PeakPercent() uint32 {
	return s.Incarnation.status.PeakPercent
}

func (s *ScalableTargetAdapter) Delay() time.Duration {
	return time.Duration(*s.Incarnation.target().Release.Rate.DelaySeconds) * time.Second
}

func (s *ScalableTargetAdapter) Increment() uint32 {
	return s.Incarnation.target().Release.Rate.Increment
}

func (s *ScalableTargetAdapter) Max() uint32 {
	return s.Incarnation.target().Release.Max
}

func (s *ScalableTargetAdapter) LastUpdated() time.Time {
	lu := s.Incarnation.status.LastUpdated
	if lu != nil {
		return lu.Time
	}
	return time.Time{}
}

func LinearScale(i Incarnation, max uint32, t time.Time) uint32 {
	sta := ScalableTargetAdapter{i}
	return scaling.LinearScale(&sta, max, t)
}
