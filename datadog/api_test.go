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
