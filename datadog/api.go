package datadog

import (
	"context"
	"sync"

	_nethttp "net/http"

	datadog "github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	datadogV1 "github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	log = logf.Log.WithName("datadog_alerts")
)

// type noopDDOGAPI struct{}

type DatadogAPI interface {
	ListSLOs(ctx context.Context, o ...datadogV1.ListSLOsOptionalParameters) (datadogV1.SLOListResponse, *_nethttp.Response, error)
}

func ListSLOs(ctx context.Context, o ...datadogV1.ListSLOsOptionalParameters) (datadogV1.SLOListResponse, *_nethttp.Response, error) {
	return datadogV1.SLOListResponse{}, nil, nil
}

type DDOGAPI struct {
	api DatadogAPI
	// cache map[string]cachedValue
	lock *sync.RWMutex
}

func NewAPI() (*DDOGAPI, error) {
	// ctx := datadog.NewDefaultContext(context.Background())

	configuration := datadog.NewConfiguration()
	apiClient := datadog.NewAPIClient(configuration)

	log.Info("Creating datadog SLO API")
	return &DDOGAPI{datadogV1.NewServiceLevelObjectivesApi(apiClient), &sync.RWMutex{}}, nil
}

// IsRevisionTriggered returns the offending alerts if any SLO alerts are currently triggered for the app/tag pair.
func (a DDOGAPI) IsRevisionTriggered(ctx context.Context, app, tag string, canariesOnly bool) (bool, []string, error) {
	// q := NewAlertQuery(app, tag)

	// tags, err := a.TaggedAlerts(ctx, q, time.Now(), canariesOnly)
	// if err != nil {
	// 	return false, nil, err
	// }

	// if alerts, ok := tags[tag]; ok && len(alerts) > 0 {
	// 	return true, alerts, nil
	// }

	return false, nil, nil
}
