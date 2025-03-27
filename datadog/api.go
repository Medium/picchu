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

type DatadogAPI interface {
	SearchMonitors(ctx context.Context, o ...datadogV1.SearchMonitorsOptionalParameters) (datadogV1.MonitorSearchResponse, *http.Response, error)
	ListMonitors(ctx context.Context, o ...datadogV1.ListMonitorsOptionalParameters) ([]datadogV1.Monitor, *http.Response, error)
}

type DDOGAPI struct {
	api  DatadogAPI
	ttl  time.Duration
	lock *sync.RWMutex
}

func NewAPI(ttl time.Duration) (*DDOGAPI, error) {
	log.Info("Creating Datadog API")

	configuration := datadog.NewConfiguration()
	apiClient := datadog.NewAPIClient(configuration)

	return &DDOGAPI{datadogV1.NewMonitorsApi(apiClient), ttl, &sync.RWMutex{}}, nil
}

func InjectAPI(a DatadogAPI, ttl time.Duration) *DDOGAPI {
	return &DDOGAPI{a, ttl, &sync.RWMutex{}}
}

func (a DDOGAPI) TaggedCanaryMonitors(app string, tag string) (map[string][]string, error) {
	app = "\"" + app + "\""
	serach_params := datadogV1.SearchMonitorsOptionalParameters{
		Query: &app,
	}

	datadog_ctx := datadog.NewDefaultContext(context.Background())
	resp, r, err := a.api.SearchMonitors(datadog_ctx, *datadogV1.NewSearchMonitorsOptionalParameters().WithQuery(*serach_params.Query))

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
func (a DDOGAPI) IsRevisionTriggered(ctx context.Context, app string, tag string) (bool, []string, error) {
	canary_monitors, err := a.TaggedCanaryMonitors(app, tag)
	if err != nil {
		return false, nil, err
	}

	if len(canary_monitors) == 0 {
		return false, nil, nil
	}
	if monitors, ok := canary_monitors[tag]; ok && len(monitors) > 0 {
		return true, monitors, nil
	}
	return false, nil, nil
}

// func (a DDOGAPI) EvalMetrics(ctx context.Context, app string, tag string, t time.Time, datadogSLOs []*picchuv1alpha1.DatadogSLO) (bool, error) {
// 	// iterate through datadog slos - use bad events / total events + inject tag and acceptance percentage to evaluate the
// 	// canary slo over a 5min period of time?
// 	// how do we know the time of canary?
// 	// the target will be production or production similar

// 	canary_queries := []string{}
// 	// create the queries to evaluate
// 	for slo := range datadogSLOs {
// 		badEvenets := a.injectTag(datadogSLOs[slo].Query.BadEvents, tag)
// 		totalEvents := a.injectTag(datadogSLOs[slo].Query.TotalEvents, tag)
// 		allowancePercent := a.formatAllowancePercent(datadogSLOs[slo])
// 		query_first := "((" + badEvenets + " / " + totalEvents + ") - " + allowancePercent + ") - "
// 		query_second := "(" + datadogSLOs[slo].Query.BadEvents + " / " + datadogSLOs[slo].Query.TotalEvents + ") >= 0"
// 		query := "sum(last_2m):" + query_first + query_second
// 		canary_queries = append(canary_queries, query)
// 	}

// 	return false, nil
// }

// func (a DDOGAPI) injectTag(query string, tag string) string {
// 	bracket_index := strings.Index(query, "{")
// 	tag_string := "destination_version:" + tag + " AND "
// 	return query[:bracket_index+1] + tag_string + query[bracket_index+1:]
// }

// func (a DDOGAPI) formatAllowancePercent(datadogslo *picchuv1alpha1.DatadogSLO) string {
// 	allowancePercent := datadogslo.Canary.AllowancePercent
// 	if datadogslo.Canary.AllowancePercentString != "" {
// 		f, err := strconv.ParseFloat(datadogslo.Canary.AllowancePercentString, 64)
// 		if err != nil {
// 			fmt.Errorf("Could not parse to float %v", err)
// 		} else {
// 			allowancePercent = f
// 		}
// 	}
// 	r := allowancePercent / 100
// 	return fmt.Sprintf("%.10g", r)
// }
