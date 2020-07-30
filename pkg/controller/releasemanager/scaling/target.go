package scaling

import (
	picchu "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"time"
)

// ScalableTarget is an interface to a revision target that is scalable.
type ScalableTarget interface {
	CanRampTo(uint32) bool
	CurrentPercent() uint32
	PeakPercent() uint32
	LastUpdated() time.Time
	ReleaseInfo() picchu.ReleaseInfo
}
