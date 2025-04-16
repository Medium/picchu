package datadog

import (
	"context"
	"net/http"
	"sync"
	"time"

	datadog "github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	datadogV1 "github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
)

// var (
// 	log = logf.Log.WithName("datadog_alerts")
// )

type DatadogSLOAPI interface {
	SearchSLO(ctx context.Context, o ...datadogV1.SearchSLOOptionalParameters) (datadogV1.SearchSLOResponse, *http.Response, error)
}

type DDOGSLOAPI struct {
	api   DatadogSLOAPI
	cache map[string]cachedValueSLO
	ttl   time.Duration
	lock  *sync.RWMutex
}

type cachedValueSLO struct {
	value       datadogV1.SearchSLOResponse
	lastUpdated time.Time
}

func NewSLOAPI(ttl time.Duration) (*DDOGSLOAPI, error) {
	log.Info("Creating Datadog Monitor API")

	configuration := datadog.NewConfiguration()
	apiClient := datadog.NewAPIClient(configuration)

	return &DDOGSLOAPI{datadogV1.NewServiceLevelObjectivesApi(apiClient), map[string]cachedValueSLO{}, ttl, &sync.RWMutex{}}, nil
}

func InjectSLOAPI(a DatadogSLOAPI, ttl time.Duration) *DDOGSLOAPI {
	return &DDOGSLOAPI{a, map[string]cachedValueSLO{}, ttl, &sync.RWMutex{}}
}

func (a DDOGSLOAPI) checkCache(query string) (datadogV1.SearchSLOResponse, bool) {
	a.lock.RLock()
	defer a.lock.RUnlock()
	if v, ok := a.cache[query]; ok {
		if v.lastUpdated.Add(a.ttl).After(time.Now()) {
			return v.value, true
		}
	}
	return datadogV1.SearchSLOResponse{}, false
}

func (a DDOGSLOAPI) queryWithCache(query string) (datadogV1.SearchSLOResponse, error) {
	if v, ok := a.checkCache(query); ok {
		log.Info("echo queryWithCache checkCache", "val", v, "slos", v.Data.Attributes.Slos)
		return v, nil
	}

	search_params := datadogV1.SearchSLOOptionalParameters{
		Query: &query,
	}

	datadog_ctx := datadog.NewDefaultContext(context.Background())
	resp, r, err := a.api.SearchSLO(datadog_ctx, search_params)

	if err != nil {
		log.Error(err, "Error when calling `MonitorsApi.SearchSLO`\n", "error", err, "response", r)
		return datadogV1.SearchSLOResponse{}, err
	}

	if resp.Data == nil || len(resp.Data.Attributes.Slos) == 0 {
		log.Info("No SLOs found when calling `ServiceLevelObjectivesApi.NewSearchSLOOptionalParameters` for service")
		return datadogV1.SearchSLOResponse{}, nil
	}

	a.lock.Lock()
	defer a.lock.Unlock()
	a.cache[query] = cachedValueSLO{resp, time.Now()}
	return resp, nil
}

func (a DDOGSLOAPI) GetDatadogSLOID(app string, datadogSLO *picchuv1alpha1.DatadogSLO) (string, error) {
	// get the SLO ID from the datadog API
	if app == "echo" {
		ddogslo_name := "\"" + app + "-" + datadogSLO.Name + "-slo" + "\""

		val, err := a.queryWithCache(ddogslo_name)
		if err != nil {
			log.Info("echo queryWithCache GetDatadogSLOIDs error", "err", err)
			return "", err
		}

		if val.Data == nil {
			log.Info("echo GetDatadogSLOIDs no SLOs found", "err", err, "response", val)
			return "", err
		}

		if len(val.Data.Attributes.Slos) == 1 {
			return *val.Data.Attributes.Slos[0].Data.Id, nil
		} else {
			log.Info("echo GetDatadogSLOIDs response was more than one slo object", "err", err, "response", val)
			return "", err
		}
	}
	// if app is not echo, return empty string for now
	return "", nil
}
