package datadog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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
func (a DDOGAPI) IsRevisionTriggered(ctx context.Context, app string, tag string) (bool, []string, error) {
	canary_monitors, err := a.TaggedCanaryMonitors(app, tag)
	if err != nil {
		return false, nil, err
	}

	if monitors, ok := canary_monitors[tag]; ok && len(monitors) > 0 {
		return true, monitors, nil
	}
	return false, nil, nil
}

// compare the two series
// how time
// how to know in the canary phase
func (a DDOGAPI) EvalMetrics(app string, tag string, datadogSLOs []*picchuv1alpha1.DatadogSLO) (bool, error) {
	// iterate through datadog slos - use bad events / total events + inject tag and acceptance percentage to evaluate the
	// canary slo over a 5min period of time?
	// how do we know the time of canary?
	// the target will be production or production similar

	canary_queries := []string{}
	// create the queries to evaluate
	for slo := range datadogSLOs {
		badEvenets := a.injectTag(datadogSLOs[slo].Query.BadEvents, tag)
		totalEvents := a.injectTag(datadogSLOs[slo].Query.TotalEvents, tag)
		// allowancePercent := "0.01"
		query_first := "(" + badEvenets + " / " + totalEvents + ") - 0.01"
		query_second := "(" + datadogSLOs[slo].Query.BadEvents + " / " + datadogSLOs[slo].Query.TotalEvents + ")"
		canary_queries = append(canary_queries, query_first)
		canary_queries = append(canary_queries, query_second)
		// canary_queries = append(canary_queries, datadogSLOs[slo].Query.BadEvents)
		// canary_queries = append(canary_queries, datadogSLOs[slo].Query.TotalEvents)
	}

	// api
	ctx := datadog.NewDefaultContext(context.Background())
	configuration := datadog.NewConfiguration()
	apiClient := datadog.NewAPIClient(configuration)
	api := datadogV1.NewMetricsApi(apiClient)

	now := time.Now().Unix()
	fifteenmin_before := time.Now().AddDate(0, 0, 0).Unix() - 300000

	first, _, _ := api.QueryMetrics(ctx, fifteenmin_before, now, canary_queries[0])
	second, _, _ := api.QueryMetrics(ctx, fifteenmin_before, now, canary_queries[1])

	responseContent, _ := json.MarshalIndent(first, "", "  ")
	fmt.Fprintf(os.Stdout, "Response from `MetricsApi.QueryMetrics`:\n%s\n", responseContent)

	responseContent_second, _ := json.MarshalIndent(second, "", "  ")
	fmt.Fprintf(os.Stdout, "Response from `MetricsApi.QueryMetrics`:\n%s\n", responseContent_second)

	first_points := first.Series[0].Pointlist
	second_points := second.Series[0].Pointlist

	if len(first_points) > len(second_points) {
		fmt.Println("first is greater")
		for i := range second_points {
			f := first_points[i][1]
			s := second_points[i][1]

			if f == nil || s == nil {
				continue
			}
			// now what
			// compare the two series

			if *f > *s {
				f_str := strconv.FormatFloat(*f, 'f', -1, 64)
				s_str := strconv.FormatFloat(*s, 'f', -1, 64)
				str := "canary triggered" + f_str + ">" + s_str
				fmt.Println(str)
				fmt.Println("canary triggered")
			}

		}
	} else {
		fmt.Println("second is greater")
		for i := range first_points {
			f := first_points[i][1]
			s := second_points[i][1]

			if f == nil || s == nil {
				continue
			}
			// now what
			// compare the two series

			fmt.Println(*f)
			fmt.Println(*s)
			if *f > *s {
				f_str := strconv.FormatFloat(*f, 'f', -1, 64)
				s_str := strconv.FormatFloat(*s, 'f', -1, 64)
				str := "canary triggered" + f_str + ">" + s_str
				fmt.Println(str)
				fmt.Println("canary triggered")
			}

		}
	}
	// for _, c := range canary_queries {
	// 	fmt.Println(c)
	// 	resp, r, err := api.QueryMetrics(ctx, fifteenmin_before, now, c)

	// 	if err != nil {
	// 		fmt.Fprintf(os.Stderr, "Error when calling `MetricsApi.QueryMetrics`: %v\n", err)
	// 		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	// 	}
	// 	for _, s := range resp.Series {
	// 		if len(s.Pointlist) > 0 {
	// 			fmt.Println(c)
	// 			fmt.Println(len(s.Pointlist))
	// 		}
	// 	}

	// 	responseContent, _ := json.MarshalIndent(resp, "", "  ")
	// 	fmt.Fprintf(os.Stdout, "Response from `MetricsApi.QueryMetrics`:\n%s\n", responseContent)
	// }

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
			// fmt.Errorf("Could not parse to float %v", err)
			fmt.Println("err")
		} else {
			allowancePercent = f
		}
	}
	r := allowancePercent / 100
	return fmt.Sprintf("%.10g", r)
}
