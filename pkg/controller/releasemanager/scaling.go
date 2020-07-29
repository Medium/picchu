package releasemanager

import (
	picchu "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"math"
	"time"

	"go.medium.engineering/picchu/pkg/controller/releasemanager/scaling"
)

type ScalableTargetAdapter struct {
	Incarnation
}

const Threshold = 0.95

// IsReconciled returns true if the target is considered ready to be scaled to the next increment.
func (s *ScalableTargetAdapter) IsReconciled(desiredScale uint32) bool {
	target := s.Incarnation.target()
	status := s.Incarnation.status
	controller := s.Incarnation.controller
	log := s.Incarnation.getLog()

	if target == nil || target.Scale.Min == nil {
		return false
	}

	totalReplicas := float64(controller.expectedTotalReplicas(*target.Scale.Min, int32(desiredScale)))
	expectedReplicas := int32(math.Ceil(totalReplicas * Threshold))

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

func (s *ScalableTargetAdapter) ReleaseInfo() picchu.ReleaseInfo {
	return s.Incarnation.target().Release
}

func (s *ScalableTargetAdapter) LastUpdated() time.Time {
	lu := s.Incarnation.status.LastUpdated
	if lu != nil {
		return lu.Time
	}
	return time.Time{}
}

func Scale(i Incarnation, max uint32, t time.Time) uint32 {
	sta := ScalableTargetAdapter{i}
	switch sta.ReleaseInfo().ScalingStrategy {
	case picchu.ScalingStrategyLinear:
		return scaling.LinearScale(&sta, max, t)
	case picchu.ScalingStrategyGeometric:
		return scaling.GeometricScale(&sta, max, t)
	}
	return sta.CurrentPercent()
}
