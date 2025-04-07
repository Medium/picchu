package datadog

import (
	"context"
	"testing"
	"time"

	datadogV1 "github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	"github.com/stretchr/testify/assert"
	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/prometheus/mocks"
	"go.uber.org/mock/gomock"
)

func TestDatadogCache(t *testing.T) {
	testAlertCache(t)
}

func TestDatadogAlerts(t *testing.T) {
	testAlert(t)
}

func testAlertCache(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	m := mocks.NewMockDatadogMonitorAPI(ctrl)
	api := InjectAPI(m, time.Duration(25)*time.Millisecond)

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

	m.
		EXPECT().
		SearchMonitorGroups(gomock.Any(), gomock.Any()).
		Return(datadogV1.MonitorGroupSearchResponse{}, nil, nil).
		Times(2)

	for i := 0; i < 5; i++ {
		r, err := api.TaggedCanaryMonitors(context.TODO(), "echo", "main-123", datadogSLOs)
		assert.Nil(t, err, "Should succeed in querying alerts")
		assert.Equal(t, map[string][]string{}, r, "Should get no firing alerts")
	}
	time.Sleep(time.Duration(25) * time.Millisecond)
	r, err := api.TaggedCanaryMonitors(context.TODO(), "echo", "main-123", datadogSLOs)
	assert.Nil(t, err, "Should succeed in querying alerts")
	assert.Equal(t, map[string][]string{}, r, "Should get no firing alerts")
}

func testAlert(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	m := mocks.NewMockDatadogMonitorAPI(ctrl)
	api := InjectAPI(m, time.Duration(25)*time.Millisecond)

	datadogSLOs := []*picchuv1alpha1.DatadogSLO{
		{
			Name:        "istio-http-success",
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

	monitor_name_echo_iha := "echo-iha-canary"
	monitor_name_echo_iha2 := "echo-iha2-canary"
	alert := datadogV1.MONITOROVERALLSTATES_ALERT
	not_alert := datadogV1.MONITOROVERALLSTATES_OK
	response := datadogV1.MonitorGroupSearchResponse{
		Groups: []datadogV1.MonitorGroupSearchResult{
			{
				MonitorName: &monitor_name_echo_iha,
				Status:      &not_alert,
			},
			{
				MonitorName: &monitor_name_echo_iha2,
				Status:      &alert,
			},
		},
	}

	query := "echo-istio-http-success-canary group:(env:production AND version:main-123) triggered:15"

	search_params := datadogV1.SearchMonitorGroupsOptionalParameters{
		Query: &query,
	}

	m.
		EXPECT().
		SearchMonitorGroups(gomock.Any(), gomock.Eq(search_params)).
		Return(response, nil, nil).
		Times(1)

	r_echo, err := api.TaggedCanaryMonitors(context.TODO(), "echo", "main-123", datadogSLOs)
	assert.Nil(t, err, "Should succeed in querying alerts")
	assert.Equal(t, map[string][]string{"main-123": {"echo-iha2-canary"}}, r_echo, "Should get 2 firing alerts (v1)")

}
