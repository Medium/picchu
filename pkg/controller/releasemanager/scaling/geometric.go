package scaling

import "time"

func GeometricScale(st ScalableTarget, max uint32, t time.Time) uint32 {
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

	desired := current * release.GeometricScaling.Factor
	if current < release.GeometricScaling.Start {
		desired = release.GeometricScaling.Start
	}

	if desired > max {
		desired = max
	}

	if st.CanRampTo(desired) {
		return desired
	}

	return current
}
