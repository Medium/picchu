package datadog

import (
	"context"
	"sync"
	"time"

	"net/http"

	datadog "github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	datadogV2 "github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	events_log = logf.Log.WithName("datadog_alerts")
)

type DatadogEventsAPI interface {
	SearchEvents(ctx context.Context, o ...datadogV2.SearchEventsOptionalParameters) (datadogV2.EventsListResponse, *http.Response, error)
}

type DDOGEVENTSAPI struct {
	api   DatadogEventsAPI
	cache map[string]cachedEventsValue
	ttl   time.Duration
	lock  *sync.RWMutex
}

type cachedEventsValue struct {
	value       datadogV2.EventsListResponse
	lastUpdated time.Time
}

func NewEventsAPI(ttl time.Duration) (*DDOGEVENTSAPI, error) {
	monitor_log.Info("Creating Datadog Monitor API")

	configuration := datadog.NewConfiguration()
	apiClient := datadog.NewAPIClient(configuration)

	return &DDOGEVENTSAPI{datadogV2.NewEventsApi(apiClient), map[string]cachedEventsValue{}, ttl, &sync.RWMutex{}}, nil
}

func InjectEventsAPI(a DatadogEventsAPI, ttl time.Duration) *DDOGEVENTSAPI {
	return &DDOGEVENTSAPI{a, map[string]cachedEventsValue{}, ttl, &sync.RWMutex{}}
}

func (a DDOGEVENTSAPI) checkCache(ctx context.Context, query string) (datadogV2.EventsListResponse, bool) {
	a.lock.RLock()
	defer a.lock.RUnlock()
	if v, ok := a.cache[query]; ok {
		if v.lastUpdated.Add(a.ttl).After(time.Now()) {
			return v.value, true
		}
	}
	return datadogV2.EventsListResponse{}, false
}

func (a DDOGEVENTSAPI) queryWithCache(ctx context.Context, query string) (datadogV2.EventsListResponse, error) {
	if v, ok := a.checkCache(ctx, query); ok {
		return v, nil
	}

	// fix this
	search_params := datadogV2.SearchEventsOptionalParameters{
		Body: &datadogV2.EventsListRequest{},
	}

	datadog_ctx := datadog.NewDefaultContext(context.Background())
	val, r, err := a.api.SearchEvents(datadog_ctx, search_params)
	if err != nil {
		monitor_log.Error(err, "Error when calling `EventsApi.SearchEvents`\n", "error", err, "response", r)
		return datadogV2.EventsListResponse{}, err
	}

	a.lock.Lock()
	defer a.lock.Unlock()
	a.cache[query] = cachedEventsValue{val, time.Now()}
	return val, nil
}

// func (a DDOGEVENTSAPI) TaggedCanaryMonitors(ctx context.Context, app string, tag string, datadogSLOs []*picchuv1alpha1.DatadogSLO) (map[string][]string, error) {
// 	var canary_monitor string
// 	canary_monitor = "("
// 	for i := range datadogSLOs {
// 		if datadogSLOs[i].Canary.Enabled {
// 			if i == 0 {
// 				canary_monitor = canary_monitor + app + "-" + datadogSLOs[i].Name + "-canary"
// 				continue
// 			}
// 			// query: (<slo-1> OR <slo-2>) group:(env:production AND version:<tag>) triggered:15
// 			canary_monitor = canary_monitor + " OR " + app + "-" + datadogSLOs[i].Name + "-canary"
// 		}
// 	}
// 	canary_monitor = canary_monitor + ") group:(env:production AND version:" + tag + ") triggered:15"

// 	val, err := a.queryWithCache(ctx, canary_monitor)
// 	if err != nil {
// 		monitor_log.Error(err, "Error when calling `queryWithCach`\n", "error", err, "response", val)
// 		return nil, err
// 	}

// 	monitors := val.GetGroups()

// 	canary_monitors := map[string][]string{}
// 	for _, m := range monitors {
// 		if m.MonitorName == nil {
// 			monitor_log.Info("Nil name for canary monitor", "status", m.Status, "monitor", m, "app", app, "tag", tag)
// 			continue
// 		}
// 		if m.Status == nil {
// 			monitor_log.Info("Nil status for canary monitor", "status", m.Status, "monitor", m, "app", app, "tag", tag)
// 			continue
// 		}
// 		if *m.Status == datadogV1.MONITOROVERALLSTATES_ALERT {
// 			if canary_monitors[tag] == nil {
// 				canary_monitors[tag] = []string{}
// 			}
// 			canary_monitors[tag] = append(canary_monitors[tag], *m.MonitorName)
// 		}
// 	}

// 	return canary_monitors, nil
// }

// IsRevisionTriggered returns the offending alerts if any SLO alerts are currently triggered for the app/tag pair.
func (a DDOGEVENTSAPI) IsRevisionTriggered(ctx context.Context, app string, tag string, datadogSLOs []*picchuv1alpha1.DatadogSLO) (bool, []string, error) {
	// canary_monitors, err := a.TaggedCanaryMonitors(ctx, app, tag, datadogSLOs)
	// if err != nil {
	// 	monitor_log.Error(err, "Error when calling `IsRevisionTriggered`\n", "error", err)
	// 	return false, nil, err
	// }

	// if monitors, ok := canary_monitors[tag]; ok && len(monitors) > 0 {
	// 	return true, monitors, nil
	// }
	return false, nil, nil
}
