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
)

type DatadogMetricAPI interface {
	QueryMetrics(ctx context.Context, from int64, to int64, query string) (datadogV1.MetricsQueryResponse, *http.Response, error)
}

type DDOGMETRICAPI struct {
	api  DatadogMetricAPI
	ttl  time.Duration
	lock *sync.RWMutex
}

func NewMetricAPI(ttl time.Duration) (*DDOGMETRICAPI, error) {
	log.Info("Creating Datadog Metric API")

	configuration := datadog.NewConfiguration()
	apiClient := datadog.NewAPIClient(configuration)

	return &DDOGMETRICAPI{datadogV1.NewMetricsApi(apiClient), ttl, &sync.RWMutex{}}, nil
}

func (a DDOGMETRICAPI) EvalMetrics(app string, tag string, datadogSLOs []*picchuv1alpha1.DatadogSLO) (bool, error) {
	// iterate through datadog slos - use bad events / total events + inject tag and acceptance percentage to evaluate the
	// canary slo over a 5min period of time? - how do we know the time of canary?
	// the target will be production or production similar

	var query_first string
	var query_second string
	// create the queries to evaluate
	for slo := range datadogSLOs {
		badEvenets := a.injectTag(datadogSLOs[slo].Query.BadEvents, tag)
		totalEvents := a.injectTag(datadogSLOs[slo].Query.TotalEvents, tag)

		// default allowance percent to 0.01 for now
		query_first = "(" + badEvenets + " / " + totalEvents + ") - 0.01"
		query_second = "(" + datadogSLOs[slo].Query.BadEvents + " / " + datadogSLOs[slo].Query.TotalEvents + ")"
	}

	// grab time now and fifteen minutes before
	now := time.Now().Unix()
	fifteenmin_before := time.Now().AddDate(0, 0, 0).Unix() - 300000

	// query datadog api
	datadog_ctx := datadog.NewDefaultContext(context.Background())
	first, _, _ := a.api.QueryMetrics(datadog_ctx, fifteenmin_before, now, query_first)
	second, _, _ := a.api.QueryMetrics(datadog_ctx, fifteenmin_before, now, query_second)

	responseContent, _ := json.MarshalIndent(first, "", "  ")
	fmt.Fprintf(os.Stdout, "Response from `MetricsApi.QueryMetrics`:\n%s\n", responseContent)

	responseContent_second, _ := json.MarshalIndent(second, "", "  ")
	fmt.Fprintf(os.Stdout, "Response from `MetricsApi.QueryMetrics`:\n%s\n", responseContent_second)

	// get the point list from both queries
	first_points := first.Series[0].Pointlist
	second_points := second.Series[0].Pointlist

	number_of_points := len(first_points)
	if len(first_points) > len(second_points) {
		number_of_points = len(second_points)
	}

	for i := range number_of_points {
		f := first_points[i][1]
		s := second_points[i][1]

		if f == nil || s == nil {
			continue
		}
		// compare the two series?
		if *f > *s {
			f_str := strconv.FormatFloat(*f, 'f', -1, 64)
			s_str := strconv.FormatFloat(*s, 'f', -1, 64)
			str := "canary triggered" + f_str + ">" + s_str
			fmt.Println(str)
			fmt.Println("canary triggered")
		}

	}
	return false, nil
}

func (a DDOGMETRICAPI) injectTag(query string, tag string) string {
	bracket_index := strings.Index(query, "{")
	tag_string := "destination_version:" + tag + " AND "
	return query[:bracket_index+1] + tag_string + query[bracket_index+1:]
}

// func (a DDOGMETRICAPI) formatAllowancePercent(datadogslo *picchuv1alpha1.DatadogSLO) string {
// 	allowancePercent := datadogslo.Canary.AllowancePercent
// 	if datadogslo.Canary.AllowancePercentString != "" {
// 		f, err := strconv.ParseFloat(datadogslo.Canary.AllowancePercentString, 64)
// 		if err != nil {
// 			// fmt.Errorf("Could not parse to float %v", err)
// 			fmt.Println("err")
// 		} else {
// 			allowancePercent = f
// 		}
// 	}
// 	r := allowancePercent / 100
// 	return fmt.Sprintf("%.10g", r)
// }
