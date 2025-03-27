package datadog

import (
	"testing"
	"time"

	datadogV1 "github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	"github.com/stretchr/testify/assert"
	"go.medium.engineering/picchu/prometheus/mocks"
	"go.uber.org/mock/gomock"
)

func TestDatadogAlerts(t *testing.T) {
	testAlert(t)
}

func testAlert(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	m := mocks.NewMockDatadogAPI(ctrl)
	api := InjectAPI(m, time.Duration(25)*time.Millisecond)

	monitor_name := "echo-main-123"
	monitor_name_two := "echo-main-123-canary"
	alert := datadogV1.MONITOROVERALLSTATES_ALERT
	not_alert := datadogV1.MONITOROVERALLSTATES_OK
	response := datadogV1.MonitorSearchResponse{
		Monitors: []datadogV1.MonitorSearchResult{
			{
				Name:   &monitor_name,
				Status: &not_alert,
			},
			{
				Name:   &monitor_name_two,
				Status: &alert,
			},
		},
	}

	m.
		EXPECT().
		SearchMonitors(gomock.Any(), gomock.Any()).
		Return(response, nil, nil).
		Times(1)

	r, err := api.TaggedCanaryMonitors("echo", "main-123")
	assert.Nil(t, err, "Should succeed in querying alerts")
	assert.Equal(t, map[string][]string{"main-123": {"echo-main-123-canary"}}, r, "Should get 1 firing alerts (v1)")
}
