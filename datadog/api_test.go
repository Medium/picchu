package datadog

import (
	"testing"
	"time"

	datadogV1 "github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	"github.com/stretchr/testify/assert"
	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/prometheus/mocks"
	"go.uber.org/mock/gomock"
)

func TestDatadogAlerts(t *testing.T) {
	testAlert(t)
}

func TestDatadogEvalMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	m := mocks.NewMockDatadogMetricAPI(ctrl)
	api := InjectMetricAPI(m, time.Duration(25)*time.Millisecond)

	datadogSLOs := []*picchuv1alpha1.DatadogSLO{
		{
			Name:        "istio-request-success",
			Enabled:     true,
			Description: "test create example datadogSLO one",
			Query: picchuv1alpha1.DatadogSLOQuery{
				GoodEvents:  "sum:istio.mesh.request.count.total{(response_code:2* OR response_code:3* OR response_code:4*) AND destination_service:echo.echo-production.svc.cluster.local AND reporter:destination}.as_count()",
				TotalEvents: "sum:istio.mesh.request.count.total{destination_service:echo.echo-production.svc.cluster.local AND reporter:destination}.as_count()",
				BadEvents:   "sum:istio.mesh.request.count.total{destination_service:echo.echo-production.svc.cluster.local AND reporter:destination AND response_code:5*}.as_count()",
			},
			Tags: []string{
				"service:example",
				"env:prod",
			},
			TargetThreshold: "99.9",
			Timeframe:       "7d",
			Type:            "metric",
		},
	}

	api.EvalMetrics("echo", "main-20250325-171435-74f8c7230e", datadogSLOs)
	assert.Equal(t, map[string][]string{}, map[string][]string{"f": {"m"}}, "Should get no firing alerts (v1)")
}

func testAlert(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	m := mocks.NewMockDatadogMonitorAPI(ctrl)
	api := InjectAPI(m, time.Duration(25)*time.Millisecond)

	monitor_name_echo := "echo-main-123"
	monitor_name_gotest := "gotest-main-345"
	monitor_name_echo_ha := "echo-ha-main-123-canary"
	monitor_name_echo_iha := "echo-iha-main-123-canary"
	alert := datadogV1.MONITOROVERALLSTATES_ALERT
	not_alert := datadogV1.MONITOROVERALLSTATES_OK
	response := datadogV1.MonitorSearchResponse{
		Monitors: []datadogV1.MonitorSearchResult{
			{
				Name:   &monitor_name_echo,
				Status: &not_alert,
			},
			{
				Name:   &monitor_name_echo_ha,
				Status: &alert,
			},
			{
				Name:   &monitor_name_echo_iha,
				Status: &alert,
			},
			{
				Name:   &monitor_name_gotest,
				Status: &not_alert,
			},
		},
	}

	m.
		EXPECT().
		SearchMonitors(gomock.Any(), gomock.Any()).
		Return(response, nil, nil).
		Times(2)

	r_echo, err := api.TaggedCanaryMonitors("echo", "main-123")
	assert.Nil(t, err, "Should succeed in querying alerts")
	assert.Equal(t, map[string][]string{"main-123": {"echo-ha-main-123-canary", "echo-iha-main-123-canary"}}, r_echo, "Should get 1 firing alerts (v1)")

	r_gotest, err := api.TaggedCanaryMonitors("gotest", "main-345")
	assert.Nil(t, err, "Should succeed in querying alerts")
	assert.Equal(t, map[string][]string{}, r_gotest, "Should get no firing alerts (v1)")

}
