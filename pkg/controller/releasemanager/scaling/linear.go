package scaling

import (
	"time"
)

// LinearScale scales a ScalableTarget linearly
func LinearScale(st ScalableTarget, max uint32, t time.Time) uint32 {
	current := st.CurrentPercent()

	if max <= 0 {
		return 0
	}

	if st.Max() < max {
		max = st.Max()
	}

	// Scaling down
	if current > max {
		return max
	}

	// Wait longer to scale
	deadline := st.LastUpdated().Add(st.Delay())
	if deadline.After(t) {
		return current
	}

	increment := st.Increment()
	desired := current + increment

	if desired > max {
		desired = max
	}

	if desired == current || st.IsReconciled(desired) {
		return desired
	}

	return current
}
