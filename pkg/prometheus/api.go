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
	CanaryFiringTemplate = template.Must(template.
				New("canaryFiringAlerts").
				Parse(`sum by({{.TagLabel}},app)(ALERTS{ {{.TagLabel}}="{{.Tag}}",alertType="canary",alertstate="{{.AlertState}}"})`))
	SLOFiringTemplate = template.Must(template.
				New("sloFiringAlerts").
				Parse(`sum by({{.TagLabel}},app)(ALERTS{slo="true",alertstate="{{.AlertState}}"})`))
	log = logf.Log.WithName("prometheus_alerts")
)

type PromAPI interface {
	Query(context.Context, string, time.Time) (model.Value, error)
}

type AlertQuery struct {
	App        string
	AlertState string
	TagLabel   string
	Tag        string
}

func NewAlertQuery(app, tag string) AlertQuery {
	return AlertQuery{
		App:        app,
		AlertState: "firing",
		TagLabel:   "tag",
		Tag:        tag,
	}
}

type cachedValue struct {
	value       model.Value
	lastUpdated time.Time
}

type API struct {
	api   PromAPI
	cache map[string]cachedValue
	ttl   time.Duration
}

type noopAPI struct{}

func (a *noopAPI) Query(_ context.Context, _ string, _ time.Time) (model.Value, error) {
	return model.Vector{}, nil
}

func NewAPI(address string, ttl time.Duration) (*API, error) {
	log.Info("Creating API", "address", address)
	client, err := cli.NewClient(cli.Config{Address: address})
	if err != nil {
		return nil, err
	}
	if address == "" {
		log.Info("WARNING: No prometheus address defined, SLOs disabled")
		return &API{&noopAPI{}, map[string]cachedValue{}, ttl}, nil
	}
	return &API{api.NewAPI(client), map[string]cachedValue{}, ttl}, nil
}

func InjectAPI(a PromAPI, ttl time.Duration) *API {
	return &API{a, map[string]cachedValue{}, ttl}
}

func (a API) queryWithCache(ctx context.Context, query string, t time.Time) (model.Value, error) {
	if v, ok := a.cache[query]; ok {
		if v.lastUpdated.Add(a.ttl).After(time.Now()) {
			return v.value, nil
		}
	}
	val, err := a.api.Query(ctx, query, t)
	if err != nil {
		return nil, err
	}
	a.cache[query] = cachedValue{val, time.Now()}
	return val, nil
}

// TaggedAlerts returns a list of tags that are firing slo alerts for an app at
// a particular time.
func (a API) TaggedAlerts(ctx context.Context, query AlertQuery, t time.Time, canariesOnly bool) ([]string, error) {
	q := bytes.NewBufferString("")
	var template template.Template
	if canariesOnly {
		template = *CanaryFiringTemplate
	} else {
		template = *SLOFiringTemplate
	}
	if err := template.Execute(q, query); err != nil {
		return nil, err
	}
	val, err := a.queryWithCache(ctx, q.String(), t)
	if err != nil {
		return nil, err
	}
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
	for tag := range tagset {
		tags = append(tags, tag)
	}
	return tags, nil
}

// IsRevisionTriggered returns true if any slo alerts are currently triggered
// for the app/tag pair.
func (a API) IsRevisionTriggered(ctx context.Context, app, tag string, canariesOnly bool) (bool, error) {
	q := NewAlertQuery(app, tag)

	tags, err := a.TaggedAlerts(ctx, q, time.Now(), canariesOnly)
	if err != nil {
		return false, err
	}
	for _, t := range tags {
		if tag == t {
			return true, nil
		}
	}
	return false, nil
}
