package scaling

import (
	"time"
)

// ScalableTarget is an interface to a revision target that is scalable.
type ScalableTarget interface {
	CurrentPercent() uint32
	PeakPercent() uint32
	Delay() time.Duration
	Increment() uint32
	Max() uint32
	LastUpdated() time.Time
}
