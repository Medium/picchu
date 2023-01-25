package scaling

import (
	"time"

	picchu "go.medium.engineering/picchu/api/v1alpha1"
)

// ScalableTarget is an interface to a revision target that is scalable.
type ScalableTarget interface {
	CanRampTo(uint32) bool
	CurrentPercent() uint32
	PeakPercent() uint32
	LastUpdated() time.Time
	ReleaseInfo() picchu.ReleaseInfo
}
