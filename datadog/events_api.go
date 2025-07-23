package datadog

import (
	"context"
	"sync"
	"time"

	"net/http"

	datadog "github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	datadogV2 "github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
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

	now := time.Now()
	// 5 minutes before current time
	fiveMinutesAgo := now.Add(-5 * time.Minute)

	body := datadogV2.EventsListRequest{
		Filter: &datadogV2.EventsQueryFilter{
			// Query: datadog.PtrString("<service>-composite-canary, destination_workload:main-20250711-124839-8ed97ca881"),
			Query: datadog.PtrString(query),
			From:  datadog.PtrString(fiveMinutesAgo.Format(time.RFC3339)),
			To:    datadog.PtrString(now.Format(time.RFC3339)),
		},
		Sort: datadogV2.EVENTSSORT_TIMESTAMP_ASCENDING.Ptr(),
		Page: &datadogV2.EventsRequestPage{
			Limit: datadog.PtrInt32(5),
		},
	}

	search_params := datadogV2.SearchEventsOptionalParameters{
		Body: &body,
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

// IsRevisionTriggered returns the offending alerts if any SLO alerts are currently triggered for the app/tag pair.
func (a DDOGEVENTSAPI) IsRevisionTriggered(ctx context.Context, app string, tag string) (bool, error) {
	// Query: datadog.PtrString("<service>-composite-canary, destination_workload:main-20250711-124839-8ed97ca881"),
	canary_monitor := app + "-composite-canary, destination_workload:" + tag
	val, err := a.queryWithCache(ctx, canary_monitor)
	if err != nil {
		monitor_log.Error(err, "Error when calling `queryWithCach`\n", "error", err, "response", val)
		return false, err
	}
	e := datadogV2.EVENTSTATUSTYPE_ERROR
	if val.Data[0].Attributes.Attributes.Status == &e {
		return true, nil
	}

	return false, nil
}
