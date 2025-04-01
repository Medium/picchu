package datadog

import (
	"context"
	"strings"
	"sync"
	"time"

	"net/http"

	datadog "github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	datadogV1 "github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	log = logf.Log.WithName("datadog_alerts")
)

type DatadogMonitorAPI interface {
	SearchMonitors(ctx context.Context, o ...datadogV1.SearchMonitorsOptionalParameters) (datadogV1.MonitorSearchResponse, *http.Response, error)
	ListMonitors(ctx context.Context, o ...datadogV1.ListMonitorsOptionalParameters) ([]datadogV1.Monitor, *http.Response, error)
}

type DDOGMONITORAPI struct {
	api  DatadogMonitorAPI
	ttl  time.Duration
	lock *sync.RWMutex
}

func NewMonitorAPI(ttl time.Duration) (*DDOGMONITORAPI, error) {
	log.Info("Creating Datadog Monitor API")

	configuration := datadog.NewConfiguration()
	apiClient := datadog.NewAPIClient(configuration)

	return &DDOGMONITORAPI{datadogV1.NewMonitorsApi(apiClient), ttl, &sync.RWMutex{}}, nil
}

func InjectAPI(a DatadogMonitorAPI, ttl time.Duration) *DDOGMONITORAPI {
	return &DDOGMONITORAPI{a, ttl, &sync.RWMutex{}}
}

func (a DDOGMONITORAPI) TaggedCanaryMonitors(app string, tag string) (map[string][]string, error) {
	app = "\"" + app + "\""
	search_params := datadogV1.SearchMonitorsOptionalParameters{
		Query: &app,
	}

	datadog_ctx := datadog.NewDefaultContext(context.Background())
	resp, r, err := a.api.SearchMonitors(datadog_ctx, *datadogV1.NewSearchMonitorsOptionalParameters().WithQuery(*search_params.Query))

	if err != nil {
		log.Error(err, "Error when calling `MonitorsApi.SearchMonitors`\n", "error", err, "response", r)
		return nil, err
	}

	monitors := resp.GetMonitors()

	if len(monitors) == 0 {
		log.Info("No monitors found when calling `MonitorsApi.SearchMonitors` for service", "App", app)
		return nil, nil
	}

	canary_monitors := map[string][]string{}
	for r := range monitors {
		m := monitors[r]
		if m.Name == nil {
			continue
		}
		if strings.Contains(*m.Name, "canary") && strings.Contains(*m.Name, tag) {
			if m.Status == nil {
				continue
			}
			log.Info("Echo canary SLO found", "canary monitor", *m.Name, "status", m.Status)
			if *m.Status == datadogV1.MONITOROVERALLSTATES_ALERT {
				if canary_monitors[tag] == nil {
					canary_monitors[tag] = []string{}
				}
				canary_monitors[tag] = append(canary_monitors[tag], *m.Name)
			}
		}
	}

	return canary_monitors, nil
}

// IsRevisionTriggered returns the offending alerts if any SLO alerts are currently triggered for the app/tag pair.
func (a DDOGMONITORAPI) IsRevisionTriggered(ctx context.Context, app string, tag string) (bool, []string, error) {
	canary_monitors, err := a.TaggedCanaryMonitors(app, tag)
	if err != nil {
		return false, nil, err
	}

	if monitors, ok := canary_monitors[tag]; ok && len(monitors) > 0 {
		return true, monitors, nil
	}
	return false, nil, nil
}
