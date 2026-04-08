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

const Threshold = 0.90

// rampReplicaComputation holds the fleet-wide replica requirement for a traffic percentage step.
type rampReplicaComputation struct {
	expectedReplicas    int32
	totalPods           int32
	totalTrafficPercent uint32
	hasUnscaledRevision bool
	baseCapacity        int32
}

// computeFleetReplicasRequiredForRamp returns the fleet-wide replica count Picchu requires before
// allowing traffic to reach desiredPercent (same formula as CanRampTo). Used to raise HPA/KEDA min
// during ramp so load-based autoscalers can create enough pods for the next step.
func (s *ScalableTargetAdapter) computeFleetReplicasRequiredForRamp(desiredPercent uint32) (*rampReplicaComputation, bool) {
	target := s.Incarnation.target()
	controller := s.Incarnation.controller

	if target == nil || target.Scale.Min == nil {
		return nil, false
	}

	currentTag := s.Incarnation.tag
	rm := controller.getReleaseManager()

	var totalPods int32
	var totalTrafficPercent uint32
	var hasUnscaledRevision bool
	if rm != nil {
		for _, rev := range rm.Status.Revisions {
			if rev.Tag == currentTag {
				continue
			}
			if rev.CurrentPercent > 0 && rev.Scale.Current > 0 {
				totalPods += int32(rev.Scale.Current)
				totalTrafficPercent += rev.CurrentPercent
				if rev.PeakPercent == 100 && rev.CurrentPercent < 100 {
					hasUnscaledRevision = true
				}
			}
		}
	}

	var baseCapacity int32
	if totalPods == 0 || totalTrafficPercent == 0 {
		baseCapacity = int32(*target.Scale.Min)
	} else if hasUnscaledRevision {
		if !target.Scale.HasAutoscaler() {
			baseCapacity = int32(*target.Scale.Min)
			if baseCapacity == 0 {
				baseCapacity = target.Scale.Max
			}
		} else {
			baseCapacity = totalPods
		}
	} else {
		normalizedCapacity := float64(totalPods) / (float64(totalTrafficPercent) / 100.0)
		baseCapacity = int32(math.Ceil(normalizedCapacity))
	}

	totalReplicas := float64(controller.expectedTotalReplicas(baseCapacity, int32(desiredPercent)))
	expectedReplicas := int32(math.Ceil(totalReplicas * Threshold))
	// Never require more fleet replicas than Scale.Max can produce (normalized baseCapacity can exceed Max).
	if target.Scale.Max > 0 {
		maxFleet := controller.expectedTotalReplicas(target.Scale.Max, 100)
		if expectedReplicas > maxFleet {
			expectedReplicas = maxFleet
		}
	}

	return &rampReplicaComputation{
		expectedReplicas:    expectedReplicas,
		totalPods:           totalPods,
		totalTrafficPercent: totalTrafficPercent,
		hasUnscaledRevision: hasUnscaledRevision,
		baseCapacity:        baseCapacity,
	}, true
}

// CanRampTo returns true if the target is considered ready to be scaled to the next increment.
func (s *ScalableTargetAdapter) CanRampTo(desiredPercent uint32) bool {
	status := s.Incarnation.status
	log := s.Incarnation.getLog()

	c, ok := s.computeFleetReplicasRequiredForRamp(desiredPercent)
	if !ok {
		return false
	}

	log.Info(
		"Computed expectedReplicas",
		"desiredPercent", desiredPercent,
		"totalPods", c.totalPods,
		"totalTrafficPercent", c.totalTrafficPercent,
		"hasUnscaledRevision", c.hasUnscaledRevision,
		"baseCapacity", c.baseCapacity,
		"expectedReplicas", c.expectedReplicas,
		"currentReplicas", status.Scale.Current,
	)
	if status.Scale.Current < c.expectedReplicas {
		log.Info(
			"Deferring ramp-up; not all desired replicas are ready",
			"desiredPercent", desiredPercent,
			"expectedReplicas", c.expectedReplicas,
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
