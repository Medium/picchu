package datadog

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"net/http"

	datadog "github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	datadogV1 "github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	log = logf.Log.WithName("datadog_alerts")
)

// type noopDDOGAPI struct{}

type DatadogAPI interface {
	SearchMonitors(ctx context.Context, o ...datadogV1.SearchMonitorsOptionalParameters) (datadogV1.MonitorSearchResponse, *http.Response, error)
}

func SearchMonitors(ctx context.Context, o ...datadogV1.SearchMonitorsOptionalParameters) (datadogV1.MonitorSearchResponse, *http.Response, error) {
	return datadogV1.MonitorSearchResponse{}, nil, nil
}

type cachedValue struct {
	value       datadogV1.MonitorSearchResponse
	lastUpdated time.Time
}

type DDOGAPI struct {
	api   DatadogAPI
	cache map[datadogV1.SearchMonitorsOptionalParameters]cachedValue
	ttl   time.Duration
	lock  *sync.RWMutex
}

func NewAPI(ttl time.Duration) (*DDOGAPI, error) {
	log.Info("Creating Datadog API")

	configuration := datadog.NewConfiguration()
	apiClient := datadog.NewAPIClient(configuration)

	return &DDOGAPI{datadogV1.NewMonitorsApi(apiClient), map[datadogV1.SearchMonitorsOptionalParameters]cachedValue{}, ttl, &sync.RWMutex{}}, nil
}

func (a DDOGAPI) checkCache(ctx context.Context, query datadogV1.SearchMonitorsOptionalParameters) (datadogV1.MonitorSearchResponse, bool) {
	a.lock.RLock()
	defer a.lock.RUnlock()
	if v, ok := a.cache[query]; ok {
		if v.lastUpdated.Add(a.ttl).After(time.Now()) {
			return v.value, true
		}
	}
	return datadogV1.MonitorSearchResponse{}, false
}

func (a DDOGAPI) queryWithCache(ctx context.Context, query datadogV1.SearchMonitorsOptionalParameters) (datadogV1.MonitorSearchResponse, *http.Response, error) {
	if v, ok := a.checkCache(ctx, query); ok {
		return v, nil, nil
	}
	val, r, err := a.api.SearchMonitors(ctx, query)

	if err != nil {
		log.Error(err, "Error when calling `MonitorsApi.SearchMonitors`\n", "error", err, "response", r)
		return datadogV1.MonitorSearchResponse{}, r, err
	}

	a.lock.Lock()
	defer a.lock.Unlock()
	a.cache[query] = cachedValue{val, time.Now()}
	return val, r, nil
}

func (a DDOGAPI) FiringAlerts(ctx context.Context, app string, tag string, t time.Time, canariesOnly bool) (bool, error) {
	serach_params := datadogV1.SearchMonitorsOptionalParameters{
		Query: &app,
	}
	resp, r, err := a.queryWithCache(ctx, serach_params)
	if err != nil {
		log.Error(err, "Error when calling `MonitorsApi.SearchMonitors`\n", "error", err, "response", r)
		return false, err
	}

	monitors := resp.GetMonitors()
	var canary_monitors []datadogV1.MonitorSearchResult
	for r := range monitors {
		m := monitors[r]
		if strings.Contains(*m.Name, "canary") && strings.Contains(*m.Name, tag) {
			if *m.Status == datadogV1.MONITOROVERALLSTATES_ALERT {
				canary_monitors = append(canary_monitors, m)
			}

		}
	}

	if len(canary_monitors) == 0 {
		return true, nil
	}

	return false, nil

}

// IsRevisionTriggered returns the offending alerts if any SLO alerts are currently triggered for the app/tag pair.
func (a DDOGAPI) IsRevisionTriggered(ctx context.Context, app string, tag string, canariesOnly bool) (bool, error) {
	firing, err := a.FiringAlerts(ctx, app, tag, time.Now(), canariesOnly)
	if err != nil {
		return false, err
	}

	return firing, nil
}

func (a DDOGAPI) EvalMetrics(ctx context.Context, app string, tag string, t time.Time, datadogSLOs []*picchuv1alpha1.DatadogSLO) (bool, error) {
	// iterate through datadog slos - use bad events / total events + inject tag and acceptance percentage to evaluate the
	// canary slo over a 5min period of time?
	// how do we know the time of canary?
	// the target will be production or production similar

	canary_queries := []string{}
	// create the queries to evaluate
	for slo := range datadogSLOs {
		badEvenets := a.injectTag(datadogSLOs[slo].Query.BadEvents, tag)
		totalEvents := a.injectTag(datadogSLOs[slo].Query.TotalEvents, tag)
		allowancePercent := a.formatAllowancePercent(datadogSLOs[slo])
		query_first := "((" + badEvenets + " / " + totalEvents + ") - " + allowancePercent + ") - "
		query_second := "(" + datadogSLOs[slo].Query.BadEvents + " / " + datadogSLOs[slo].Query.TotalEvents + ") >= 0"
		query := "sum(last_2m):" + query_first + query_second
		canary_queries = append(canary_queries, query)
	}

	return false, nil
}

func (a DDOGAPI) injectTag(query string, tag string) string {
	bracket_index := strings.Index(query, "{")
	tag_string := "destination_version:" + tag + " AND "
	return query[:bracket_index+1] + tag_string + query[bracket_index+1:]
}

func (a DDOGAPI) formatAllowancePercent(datadogslo *picchuv1alpha1.DatadogSLO) string {
	allowancePercent := datadogslo.Canary.AllowancePercent
	if datadogslo.Canary.AllowancePercentString != "" {
		f, err := strconv.ParseFloat(datadogslo.Canary.AllowancePercentString, 64)
		if err != nil {
			fmt.Errorf("Could not parse to float %v", err)
		} else {
			allowancePercent = f
		}
	}
	r := allowancePercent / 100
	return fmt.Sprintf("%.10g", r)
}
