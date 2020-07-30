package scaling

import (
	"time"
)

// LinearScale scales a ScalableTarget linearly
func LinearScale(st ScalableTarget, max uint32, t time.Time) uint32 {
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
	deadline := st.LastUpdated().Add(time.Duration(*release.Rate.DelaySeconds) * time.Second)
	if deadline.After(t) {
		return current
	}

	increment := release.Rate.Increment
	desired := current + increment

	if desired > max {
		desired = max
	}

	if st.IsReconciled(desired) {
		return desired
	}

	return current
}
