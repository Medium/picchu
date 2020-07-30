package scaling

import (
	"github.com/go-logr/logr"
	"time"
)

// LinearScale scales a ScalableTarget linearly
func LinearScale(st ScalableTarget, max uint32, t time.Time, log logr.Logger) uint32 {
	desired := LinearNextIncrement(st, max, t, log)

	if desired <= st.CurrentPercent() {
		return desired
	}

	if st.CanRampTo(desired) {
		return desired
	}

	return st.CurrentPercent()
}

func LinearNextIncrement(st ScalableTarget, max uint32, t time.Time, log logr.Logger) uint32 {
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
	next := current + increment

	if next > max {
		next = max
	}

	return next
}
