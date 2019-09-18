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
	// Only scale if previous target is met.
	if !st.IsReconciled() {
		if current > max {
			return max
		}
		return current
	}
	increment := st.Increment()
	// We can skip scale up for revisions that already scaled
	/* TODO(bob): Is this still needed? causes scaling problems.
	if current+st.Increment() < st.PeakPercent() {
		increment = st.PeakPercent()
	}
	*/
	if st.Max() < max {
		max = st.Max()
	}
	deadline := st.LastUpdated().Add(st.Delay())
	if deadline.After(t) {
		if current > max {
			return max
		}
		return current
	}

	current = current + increment
	if current > max {
		return max
	}
	return current
}
