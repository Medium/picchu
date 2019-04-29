package utils

import "time"

type Config struct {
	ManageRoute53          bool
	RequeueAfter           time.Duration
	TaggedRoutesEnabled    bool
	PrometheusQueryAddress string
	PrometheusQueryTTL     time.Duration
}
