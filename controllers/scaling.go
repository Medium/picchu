package controllers

import (
	"math"
	"time"

	picchu "go.medium.engineering/picchu/api/v1alpha1"

	"go.medium.engineering/picchu/controllers/scaling"
)

type ScalableTargetAdapter struct {
	Incarnation
}

const Threshold = 0.95

// CanRampTo returns true if the target is considered ready to be scaled to the next increment.
func (s *ScalableTargetAdapter) CanRampTo(desiredPercent uint32) bool {
	target := s.Incarnation.target()
	status := s.Incarnation.status
	controller := s.Incarnation.controller
	log := s.Incarnation.getLog()

	if target == nil || target.Scale.Min == nil {
		return false
	}

	// Use actual total current capacity across all revisions instead of Scale.Min
	// This ensures we check against real traffic capacity, not just a static minimum
	totalCurrentCapacity := controller.getTotalCurrentCapacity()

	// Fallback to Scale.Min if no current capacity (e.g., first deployment)
	baseCapacity := totalCurrentCapacity
	if baseCapacity == 0 {
		baseCapacity = int32(*target.Scale.Min)
	}

	totalReplicas := float64(controller.expectedTotalReplicas(baseCapacity, int32(desiredPercent)))
	expectedReplicas := int32(math.Ceil(totalReplicas * Threshold))

	log.Info(
		"Computed expectedReplicas",
		"desiredPercent", desiredPercent,
		"totalCurrentCapacity", totalCurrentCapacity,
		"baseCapacity", baseCapacity,
		"expectedReplicas", expectedReplicas,
		"currentReplicas", status.Scale.Current,
	)
	if status.Scale.Current < expectedReplicas {
		log.Info(
			"Deferring ramp-up; not all desired replicas are ready",
			"desiredPercent", desiredPercent,
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
		return scaling.LinearScale(&sta, max, t, i.log)
	case picchu.ScalingStrategyGeometric:
		return scaling.GeometricScale(&sta, max, t, i.log)
	}
	i.log.Info("Not Scaling, no scalingStrategySelected")
	return sta.CurrentPercent()
}

func NextIncrement(i Incarnation, max uint32) uint32 {
	sta := ScalableTargetAdapter{i}
	switch sta.ReleaseInfo().ScalingStrategy {
	case picchu.ScalingStrategyLinear:
		return scaling.LinearNextIncrement(&sta, max, i.log)
	case picchu.ScalingStrategyGeometric:
		return scaling.GeometricNextIncrement(&sta, max, i.log)
	}
	i.log.Info("Not Scaling, no scalingStrategySelected")
	return sta.CurrentPercent()
}

func ExpectedReleaseLatency(i Incarnation, max uint32) time.Duration {
	sta := ScalableTargetAdapter{i}
	switch sta.ReleaseInfo().ScalingStrategy {
	case picchu.ScalingStrategyLinear:
		return scaling.LinearExpectedReleaseLatency(&sta, max, i.log)
	case picchu.ScalingStrategyGeometric:
		return scaling.GeometricExpectedReleaseLatency(&sta, max, i.log)
	}
	i.log.Info("Not Scaling, no scalingStrategySelected")
	return time.Duration(0)
}
