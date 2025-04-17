package datadog

import (
	"fmt"
	"testing"
	"time"

	datadogV1 "github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	"github.com/stretchr/testify/assert"
	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/datadog/mocks"
	"go.uber.org/mock/gomock"
)

func TestDatadogSLOCache(t *testing.T) {
	testSLOAlertCache(t)
}

func testSLOAlertCache(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	m := mocks.NewMockDatadogSLOAPI(ctrl)
	api := InjectSLOAPI(m, time.Duration(25)*time.Millisecond)

	datadogSLOs := &picchuv1alpha1.DatadogSLO{
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
	}

	response := datadogV1.SearchSLOResponse{
		Data: &datadogV1.SearchSLOResponseData{
			Attributes: &datadogV1.SearchSLOResponseDataAttributes{
				Slos: []datadogV1.SearchServiceLevelObjective{},
			},
		},
	}
	id_1 := "1234"
	data_1 := datadogV1.SearchServiceLevelObjectiveData{
		Id: &id_1,
	}
	slo_1 := datadogV1.SearchServiceLevelObjective{
		Data: &data_1,
	}
	response.Data.Attributes.Slos = append(response.Data.Attributes.Slos, slo_1)

	m.
		EXPECT().
		SearchSLO(gomock.Any(), gomock.Any()).
		Return(response, nil, nil).
		Times(2)

	for i := 0; i < 5; i++ {
		fmt.Println(i)
		r, err := api.GetDatadogSLOID("echo", datadogSLOs)
		assert.Nil(t, err, "Should succeed in querying alerts")
		assert.Equal(t, "1234", r, "Should get one id")
	}
	time.Sleep(time.Duration(25) * time.Millisecond)
	r, err := api.GetDatadogSLOID("echo", datadogSLOs)
	assert.Nil(t, err, "Should succeed in querying alerts")
	assert.Equal(t, "1234", r, "Should get no firing alerts")
}
