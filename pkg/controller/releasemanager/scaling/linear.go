package scaling

import (
	"github.com/go-logr/logr"
	"math"
	"time"
)

// LinearScale scales a ScalableTarget linearly
func LinearScale(st ScalableTarget, max uint32, t time.Time, log logr.Logger) uint32 {
	desired := LinearNextIncrement(st, max, log)
	current := st.CurrentPercent()

	if desired <= st.CurrentPercent() {
		return desired
	}

	release := st.ReleaseInfo()
	duration := release.LinearScaling.Delay.Duration
	// TODO(bob): remove
	if release.Rate.DelaySeconds != nil {
		duration = time.Duration(*release.Rate.DelaySeconds) * time.Second
	}
	deadline := st.LastUpdated().Add(duration)
	// Wait longer to scale
	if deadline.After(t) {
		return current
	}

	if st.CanRampTo(desired) {
		return desired
	}

	return st.CurrentPercent()
}

func LinearNextIncrement(st ScalableTarget, max uint32, log logr.Logger) uint32 {
	current := st.CurrentPercent()
	release := st.ReleaseInfo()

	if max <= 0 {
		return 0
	}

	if release.Max < max {
		max = release.Max
	}

	// Scaling down
	if current > max {
		return max
	}

	increment := release.LinearScaling.Increment
	// TODO(bob): remove
	if release.Rate.Increment > 0 {
		increment = release.Rate.Increment
	}
	next := current + increment

	if next > max {
		next = max
	}

	return next
}

func LinearExpectedReleaseLatency(st ScalableTarget, max uint32, log logr.Logger) time.Duration {
	release := st.ReleaseInfo()
	increment := release.LinearScaling.Increment
	// TODO(bob): remove
	if release.Rate.Increment > 0 {
		increment = release.Rate.Increment
	}
	duration := release.LinearScaling.Delay.Duration
	// TODO(bob): remove
	if release.Rate.DelaySeconds != nil {
		duration = time.Duration(*release.Rate.DelaySeconds) * time.Second
	}
	iterations := math.Ceil(float64(max) / float64(increment))
	return duration * time.Duration(iterations)
}
