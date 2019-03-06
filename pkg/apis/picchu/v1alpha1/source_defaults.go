package v1alpha1

import (
	"time"
)

const (
	defaultReleaseMax           = 100
	defaultReleaseSchedule      = HumaneSchedule
	defaultReleaseRateIncrement = 5
	defaultReleaseRateDelay     = time.Duration(10) * time.Second
)

func SetDefaults_RevisionSpec(spec *RevisionSpec) {
	for _, target := range spec.Targets {
		SetReleaseDefaults(&target.Release)
	}
}

func SetDefaults_IncarnationSpec(spec *IncarnationSpec) {
	SetReleaseDefaults(&spec.Release)
}

func SetReleaseDefaults(release *ReleaseInfo) {
	if release.Max == 0 {
		release.Max = defaultReleaseMax
	}
	if release.Schedule == "" {
		release.Schedule = HumaneSchedule
	}
	if release.Rate.Increment == 0 {
		release.Rate.Increment = defaultReleaseRateIncrement
	}
	if release.Rate.Delay == 0 {
		release.Rate.Increment = defaultReleaseRateIncrement
	}
}
