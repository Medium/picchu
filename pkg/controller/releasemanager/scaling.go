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

// CanRampTo returns true if the target is considered ready to be scaled to the next increment.
func (s *ScalableTargetAdapter) CanRampTo(desiredScale uint32) bool {
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
		return scaling.LinearScale(&sta, max, t, i.log)
	case picchu.ScalingStrategyGeometric:
		return scaling.GeometricScale(&sta, max, t, i.log)
	}
	i.log.Info("Not Scaling, no scalingStrategySelected")
	return sta.CurrentPercent()
}

func NextIncrement(i Incarnation, max uint32, t time.Time) uint32 {
	sta := ScalableTargetAdapter{i}
	switch sta.ReleaseInfo().ScalingStrategy {
	case picchu.ScalingStrategyLinear:
		return scaling.LinearNextIncrement(&sta, max, t, i.log)
	case picchu.ScalingStrategyGeometric:
		return scaling.GeometricNextIncrement(&sta, max, t, i.log)
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
