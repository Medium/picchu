package utils

import "time"

type Config struct {
	HumaneReleasesEnabled     bool
	ManageRoute53             bool
	RequeueAfter              time.Duration
	PrometheusQueryAddress    string
	PrometheusQueryTTL        time.Duration
	SentryAuthToken           string
	SentryOrg                 string
	ServiceLevelsNamespace    string
	ServiceLevelsFleet        string
	ConcurrentRevisions       int
	ConcurrentReleaseManagers int
}
