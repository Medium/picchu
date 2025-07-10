package utils

import "time"

type Config struct {
	HumaneReleasesEnabled     bool
	ManageRoute53             bool
	RequeueAfter              time.Duration
	PrometheusQueryAddress    string
	PrometheusQueryTTL        time.Duration
	DatadogQueryTTL           time.Duration
	ServiceLevelsNamespace    string
	ServiceLevelsFleet        string
	DatadogSLONamespace       string
	DatadogSLOsFleet          string
	ConcurrentRevisions       int
	ConcurrentReleaseManagers int
	DevRoutesServiceHost      string
	DevRoutesServicePort      int
	useKedaClusterAuth        bool
}
