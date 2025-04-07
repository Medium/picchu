package datadog

import (
	"context"
	"fmt"
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

type DatadogMonitorAPI interface {
	SearchMonitors(ctx context.Context, o ...datadogV1.SearchMonitorsOptionalParameters) (datadogV1.MonitorSearchResponse, *http.Response, error)
	ListMonitors(ctx context.Context, o ...datadogV1.ListMonitorsOptionalParameters) ([]datadogV1.Monitor, *http.Response, error)
	SearchMonitorGroups(ctx context.Context, o ...datadogV1.SearchMonitorGroupsOptionalParameters) (datadogV1.MonitorGroupSearchResponse, *http.Response, error)
}

type DDOGMONITORAPI struct {
	api   DatadogMonitorAPI
	cache map[string]cachedValue
	ttl   time.Duration
	lock  *sync.RWMutex
}

type cachedValue struct {
	value       datadogV1.MonitorGroupSearchResponse
	lastUpdated time.Time
}

func NewMonitorAPI(ttl time.Duration) (*DDOGMONITORAPI, error) {
	log.Info("Creating Datadog Monitor API")

	configuration := datadog.NewConfiguration()
	apiClient := datadog.NewAPIClient(configuration)

	return &DDOGMONITORAPI{datadogV1.NewMonitorsApi(apiClient), map[string]cachedValue{}, ttl, &sync.RWMutex{}}, nil
}

func InjectAPI(a DatadogMonitorAPI, ttl time.Duration) *DDOGMONITORAPI {
	return &DDOGMONITORAPI{a, map[string]cachedValue{}, ttl, &sync.RWMutex{}}
}

func (a DDOGMONITORAPI) checkCache(ctx context.Context, query string) (datadogV1.MonitorGroupSearchResponse, bool) {
	a.lock.RLock()
	defer a.lock.RUnlock()
	if v, ok := a.cache[query]; ok {
		if v.lastUpdated.Add(a.ttl).After(time.Now()) {
			return v.value, true
		}
	}
	return datadogV1.MonitorGroupSearchResponse{}, false
}

func (a DDOGMONITORAPI) queryWithCache(ctx context.Context, query string) (datadogV1.MonitorGroupSearchResponse, error) {
	if v, ok := a.checkCache(ctx, query); ok {
		return v, nil
	}

	search_params := datadogV1.SearchMonitorGroupsOptionalParameters{
		Query: &query,
	}

	datadog_ctx := datadog.NewDefaultContext(context.Background())
	val, r, err := a.api.SearchMonitorGroups(datadog_ctx, search_params)
	if err != nil {
		log.Error(err, "Error when calling `MonitorsApi.SearchMonitors`\n", "error", err, "response", r)
		return datadogV1.MonitorGroupSearchResponse{}, err
	}

	a.lock.Lock()
	defer a.lock.Unlock()
	a.cache[query] = cachedValue{val, time.Now()}
	fmt.Println("val", val)
	return val, nil
}

func (a DDOGMONITORAPI) TaggedCanaryMonitors(ctx context.Context, app string, tag string, datadogSLOs []*picchuv1alpha1.DatadogSLO) (map[string][]string, error) {

	var canary_monitor string
	for _, d := range datadogSLOs {
		// query: echo-http-availability-canary group:(env:production AND version:main-20250403-211750-e4cbf655ea)
		canary_monitor = app + "-" + d.Name + "-canary group:(env:production AND version:" + tag + ") triggered:15"
	}

	val, err := a.queryWithCache(ctx, canary_monitor)
	if err != nil {
		return nil, err
	}

	monitors := val.GetGroups()

	canary_monitors := map[string][]string{}
	for _, m := range monitors {
		if m.MonitorName == nil {
			log.Info("Nil name for echo canary monitor", "status", m.Status)
			continue
		}
		if m.Status == nil {
			log.Info("Nil status for echo canary monitor", "status", m.Status)
			continue
		}
		log.Info("Echo canary SLO found", "canary monitor", *m.MonitorName, "status", m.Status)
		if *m.Status == datadogV1.MONITOROVERALLSTATES_ALERT {
			if canary_monitors[tag] == nil {
				canary_monitors[tag] = []string{}
			}
			canary_monitors[tag] = append(canary_monitors[tag], *m.MonitorName)
		}
	}

	return canary_monitors, nil
}

// IsRevisionTriggered returns the offending alerts if any SLO alerts are currently triggered for the app/tag pair.
func (a DDOGMONITORAPI) IsRevisionTriggered(ctx context.Context, app string, tag string, datadogSLOs []*picchuv1alpha1.DatadogSLO) (bool, []string, error) {
	canary_monitors, err := a.TaggedCanaryMonitors(ctx, app, tag, datadogSLOs)
	log.Info("echo found canary_monitors", "canary_monitors", canary_monitors)
	if err != nil {
		log.Info("error != nil IsRevisionTriggered")
		return false, nil, err
	}

	if monitors, ok := canary_monitors[tag]; ok && len(monitors) > 0 {
		log.Info("triggered monitors found for echo tag, length greater than 0", "monitors", monitors)
		return true, monitors, nil
	}
	return false, nil, nil
}
