package utils

import "time"

type Config struct {
	ManageRoute53          bool
	RequeueAfter           time.Duration
	PrometheusQueryAddress string
	PrometheusQueryTTL     time.Duration
	SentryAuthToken        string
	SentryOrg              string
}
