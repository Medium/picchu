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

	// Use actual capacity from other revisions; exclude this tag to avoid circular dependency.
	currentTag := s.Incarnation.tag
	rm := controller.getReleaseManager()

	// Calculate 100% capacity based on other revisions' pods and traffic percentages.
	var totalPods int32 = 0
	var totalTrafficPercent uint32 = 0
	var hasUnscaledRevision bool = false
	for _, rev := range rm.Status.Revisions {
		if rev.Tag == currentTag {
			continue
		}
		if rev.CurrentPercent == 0 || rev.Scale.Current == 0 {
			continue
		}

		effectivePods := int32(rev.Scale.Current)
		totalTrafficPercent += rev.CurrentPercent
		// If a revision was at 100% and is now serving less, prefer its peak pods as baseline.
		if rev.PeakPercent == 100 && rev.CurrentPercent < 100 {
			hasUnscaledRevision = true
			if rev.Scale.Peak > rev.Scale.Current {
				effectivePods = int32(rev.Scale.Peak)
			}
		}
		totalPods += effectivePods
	}

	// Fallback to Scale.Min if no current capacity (e.g., first deployment).
	var baseCapacity int32
	if totalPods == 0 || totalTrafficPercent == 0 {
		baseCapacity = int32(*target.Scale.Min)
	} else {
		// Normalize: X pods serving Y% traffic => 100% capacity = X / (Y/100).
		normalizedCapacity := float64(totalPods) / (float64(totalTrafficPercent) / 100.0)
		baseCapacity = int32(math.Ceil(normalizedCapacity))
	}

	totalReplicas := float64(controller.expectedTotalReplicas(baseCapacity, int32(desiredPercent)))
	expectedReplicas := int32(math.Ceil(totalReplicas * Threshold))

	log.Info(
		"Computed expectedReplicas",
		"desiredPercent", desiredPercent,
		"totalPods", totalPods,
		"totalTrafficPercent", totalTrafficPercent,
		"hasUnscaledRevision", hasUnscaledRevision,
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
