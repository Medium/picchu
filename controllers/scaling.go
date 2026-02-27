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

	// Use actual total current capacity across all OTHER revisions (excluding this one) instead of Scale.Min
	// This ensures we check against real traffic capacity, not just a static minimum
	// Excluding the current revision prevents circular dependency where we check if we have enough pods
	// to handle traffic that includes our own pods.
	currentTag := s.Incarnation.tag
	rm := controller.getReleaseManager()

	// Calculate normalized 100% capacity based on other revisions' pods and traffic percentages
	// If old revision has 4 pods at 60% traffic, then 100% capacity = 4 / 0.6 = 6.67 pods
	// However, if the old revision was previously at 100% (PeakPercent == 100) and is now serving less traffic,
	// it either hasn't scaled down yet OR has scaled down to Scale.Min and can't go lower.
	// In both cases, the pods are sized for more traffic than they're currently serving, so use pod count as-is.
	var totalPods int32 = 0
	var totalTrafficPercent uint32 = 0
	var hasUnscaledRevision bool = false
	for _, rev := range rm.Status.Revisions {
		if rev.Tag == currentTag {
			continue
		}
		if rev.CurrentPercent > 0 && rev.Scale.Current > 0 {
			totalPods += int32(rev.Scale.Current)
			totalTrafficPercent += rev.CurrentPercent
			// If a revision was at 100% and is now serving less traffic, it either:
			// 1. Hasn't scaled down yet (pods still sized for 100%)
			// 2. Has scaled down to Scale.Min and can't go lower
			// In both cases, use pod count as baseline for 100% capacity
			if rev.PeakPercent == 100 && rev.CurrentPercent < 100 {
				hasUnscaledRevision = true
			}
		}
	}

	// Fallback to Scale.Min if no current capacity (e.g., first deployment)
	var baseCapacity int32
	if totalPods == 0 || totalTrafficPercent == 0 {
		baseCapacity = int32(*target.Scale.Min)
	} else if hasUnscaledRevision {
		// Old revision was at 100% and is now serving less traffic but hasn't fully scaled down
		// (either still scaling down or at Scale.Min). Use pod count as baseline for 100% capacity.
		// Exception: fixed-scale services (no HPA, min=max) cap each revision at Scale.Min pods.
		// The "100% capacity" for the service is Scale.Min, not sum of pods across revisions.
		if !target.Scale.HasAutoscaler() {
			baseCapacity = int32(*target.Scale.Min)
			if baseCapacity == 0 {
				baseCapacity = target.Scale.Max
			}
		} else {
			baseCapacity = totalPods
		}
	} else {
		// Old revisions have scaled down to match their traffic percentage.
		// Normalize: if we have X pods serving Y% traffic, then 100% capacity = X / (Y/100)
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
