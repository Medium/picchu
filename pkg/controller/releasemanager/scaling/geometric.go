package scaling

import (
	"time"
)

func GeometricNextIncrement(st ScalableTarget, max uint32, t time.Time) uint32 {
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

	// Wait longer to scale
	deadline := st.LastUpdated().Add(release.GeometricScaling.Delay.Duration)
	if deadline.After(t) {
		return current
	}

	next := current * release.GeometricScaling.Factor
	if current < release.GeometricScaling.Start {
		next = release.GeometricScaling.Start
	}

	if next > max {
		next = max
	}

	return next
}

func GeometricScale(st ScalableTarget, max uint32, t time.Time) uint32 {
	desired := GeometricNextIncrement(st, max, t)

	if desired <= st.CurrentPercent() {
		return desired
	}

	if st.CanRampTo(desired) {
		return desired
	}

	return st.CurrentPercent()
}
