package prometheus

import (
	"bytes"
	"context"
	"fmt"
	"text/template"
	"time"

	cli "github.com/prometheus/client_golang/api"
	api "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	alertTemplate = template.Must(template.
			New("taggedAlerts").
			Parse(`sum by({{.TagLabel}},app)(ALERTS{slo="true",alertstate="{{.AlertState}}"})`))
	log = logf.Log.WithName("prometheus_alerts")
)

type AlertQuery struct {
	App        string
	AlertState string
	TagLabel   string
}

func NewAlertQuery(app string) AlertQuery {
	return AlertQuery{
		App:        app,
		AlertState: "firing",
		TagLabel:   "tag",
	}
}

type cachedValue struct {
	value       model.Value
	lastUpdated time.Time
}

type API struct {
	api.API
	cache map[string]cachedValue
	ttl   time.Duration
}

func NewAPI(address string, ttl time.Duration) (*API, error) {
	log.Info("Creating API", "address", address)
	client, err := cli.NewClient(cli.Config{Address: address})
	if err != nil {
		return nil, err
	}
	return &API{api.NewAPI(client), map[string]cachedValue{}, ttl}, nil
}

func (a API) queryWithCache(ctx context.Context, query string, t time.Time) (model.Value, error) {
	if v, ok := a.cache[query]; ok {
		if v.lastUpdated.Add(a.ttl).After(time.Now()) {
			fmt.Println("Using cache")
			return v.value, nil
		}
	}
	val, err := a.API.Query(ctx, query, t)
	if err != nil {
		return nil, err
	}
	a.cache[query] = cachedValue{val, time.Now()}
	return val, nil
}

func (a API) TaggedAlerts(ctx context.Context, query AlertQuery, t time.Time) ([]string, error) {
	log.Info("Butts")
	q := bytes.NewBufferString("")
	if err := alertTemplate.Execute(q, query); err != nil {
		return nil, err
	}
	val, err := a.queryWithCache(ctx, q.String(), t)
	if err != nil {
		return nil, err
	}
	log.Info("Query result", "Query", q, "Result", val)
	tagset := map[string]bool{}
	switch v := val.(type) {
	case model.Vector:
		for _, sample := range v {
			appMatch := false
			tag := ""
			for name, value := range sample.Metric {
				if string(name) == "app" && string(value) == query.App {
					appMatch = true
				}
				if string(name) == "tag" {
					tag = string(value)
				}
			}
			if appMatch && tag != "" {
				tagset[tag] = true
			}
		}
	default:
		return nil, fmt.Errorf("Unexpected prom response: %+v", v)
	}

	tags := []string{}
	for tag, _ := range tagset {
		tags = append(tags, tag)
	}
	return tags, nil
}
