package releasemanager

import (
	"go.medium.engineering/picchu/pkg/controller/releasemanager/scaling"
	"time"
)

type ScalableTargetAdapter struct {
	Incarnation
}

func (s *ScalableTargetAdapter) CurrentPercent() uint32 {
	return s.Incarnation.status.CurrentPercent
}

func (s *ScalableTargetAdapter) PeakPercent() uint32 {
	return s.Incarnation.status.PeakPercent
}

func (s *ScalableTargetAdapter) Delay() time.Duration {
	return time.Duration(*s.Incarnation.target().Release.Rate.DelaySeconds) * time.Second
}

func (s *ScalableTargetAdapter) Increment() uint32 {
	return s.Incarnation.target().Release.Rate.Increment
}

func (s *ScalableTargetAdapter) Max() uint32 {
	return s.Incarnation.target().Release.Max
}

func (s *ScalableTargetAdapter) LastUpdated() time.Time {
	lu := s.Incarnation.status.LastUpdated
	if lu != nil {
		return lu.Time
	}
	return time.Time{}
}

func LinearScale(i Incarnation, max uint32, t time.Time) uint32 {
	sta := ScalableTargetAdapter{i}
	return scaling.LinearScale(&sta, max, t)
}
