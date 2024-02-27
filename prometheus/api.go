package prometheus

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"sync"
	"text/template"
	"time"

	cli "github.com/prometheus/client_golang/api"
	api "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// i think this queries thanos
var (
	CanaryFiringTemplate = template.Must(template.
				New("canaryFiringAlerts").
				Parse(`sum by({{.TagLabel}},app,alertname)(ALERTS{ {{.TagLabel}}="{{.Tag}}",canary="true",alertstate="{{.AlertState}}"})`))
	SLOFiringTemplate = template.Must(template.
				New("sloFiringAlerts").
				Parse(`sum by({{.TagLabel}},app,alertname)(ALERTS{slo="true",alertstate="{{.AlertState}}"})`))
	DeploymentFiringTemplate = template.Must(template.
					New("deploymentFiringAlerts").
					Parse(`sum by (container, namespace, pod) (kube_pod_container_status_waiting_reason{reason="CrashLoopBackOff",pod=~"{{.Tag}}-.+",container="{{.App}}"}) > 0`))
	log = logf.Log.WithName("prometheus_alerts")
)

const (
	TagLabel = "tag"
)

type PromAPI interface {
	Query(ctx context.Context, query string, ts time.Time) (model.Value, api.Warnings, error)
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
		TagLabel:   TagLabel,
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
	lock  *sync.RWMutex
}

type noopAPI struct{}

func (a *noopAPI) Query(_ context.Context, _ string, _ time.Time) (model.Value, api.Warnings, error) {
	return model.Vector{}, nil, nil
}

func NewAPI(address string, ttl time.Duration) (*API, error) {
	log.Info("Creating API", "address", address)
	client, err := cli.NewClient(cli.Config{Address: address})
	if err != nil {
		return nil, err
	}
	if address == "" {
		log.Info("WARNING: No prometheus address defined, SLOs disabled")
		return &API{&noopAPI{}, map[string]cachedValue{}, ttl, &sync.RWMutex{}}, nil
	}
	return &API{api.NewAPI(client), map[string]cachedValue{}, ttl, &sync.RWMutex{}}, nil
}

func InjectAPI(a PromAPI, ttl time.Duration) *API {
	return &API{a, map[string]cachedValue{}, ttl, &sync.RWMutex{}}
}

func (a API) checkCache(ctx context.Context, query string) (model.Value, bool) {
	a.lock.RLock()
	defer a.lock.RUnlock()
	if v, ok := a.cache[query]; ok {
		if v.lastUpdated.Add(a.ttl).After(time.Now()) {
			return v.value, true
		}
	}
	return nil, false
}

func (a API) queryWithCache(ctx context.Context, query string, t time.Time) (model.Value, error) {
	if v, ok := a.checkCache(ctx, query); ok {
		return v, nil
	}
	val, _, err := a.api.Query(ctx, query, t)
	if err != nil {
		return nil, err
	}

	a.lock.Lock()
	defer a.lock.Unlock()
	a.cache[query] = cachedValue{val, time.Now()}
	return val, nil
}

// TaggedAlerts returns a set of tags that are firing SLO alerts for an app at a given time.
func (a API) TaggedAlerts(ctx context.Context, query AlertQuery, t time.Time, canariesOnly bool) (map[string][]string, error) {
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

	tagset := map[string][]string{}

	switch v := val.(type) {
	case model.Vector:
		for _, sample := range v {
			metricTag := string(sample.Metric["tag"])
			if string(sample.Metric["app"]) == query.App && metricTag != "" {
				if tagset[metricTag] == nil {
					tagset[metricTag] = []string{}
				}
				tagset[metricTag] = append(tagset[metricTag], string(sample.Metric["alertname"]))
			}
		}
	default:
		return nil, fmt.Errorf("Unexpected prom response: %+v", v)
	}

	return tagset, nil
}

// TaggedAlerts returns a set of tags that are firing SLO alerts for an app at a given time.
func (a API) TaggedDeploymentQueries(ctx context.Context, query AlertQuery, t time.Time, canariesOnly bool) (map[string][]string, error) {
	q := bytes.NewBufferString("")

	// check for deployment query
	var deployment_template = *DeploymentFiringTemplate

	if err := deployment_template.Execute(q, query); err != nil {
		return nil, err
	}
	val_deployment, err_deployment := a.queryWithCache(ctx, q.String(), t)
	if err_deployment != nil {
		return nil, err_deployment
	}

	tagset := map[string][]string{}

	// deploy
	// (container, namespace, pod)
	switch v := val_deployment.(type) {
	case model.Vector:
		for _, sample := range v {
			// get the tag from the pod name
			metricTag := strings.TrimRight(string(sample.Metric["pod"]), "-")

			// i think the containter is usually the app name
			if string(sample.Metric["container"]) == query.App && metricTag != "" {
				if tagset[metricTag] == nil {
					tagset[metricTag] = []string{}
				}
				tagset[metricTag] = append(tagset[metricTag], string(sample.Metric["namespace"]))
			}
		}
	default:
		return nil, fmt.Errorf("Unexpected prom response: %+v", v)
	}

	return tagset, nil
}

// IsRevisionTriggered returns the offending alerts if any SLO alerts are currently triggered for the app/tag pair.
func (a API) IsRevisionTriggered(ctx context.Context, app, tag string, canariesOnly bool) (bool, []string, error) {
	q := NewAlertQuery(app, tag)

	tags, err := a.TaggedAlerts(ctx, q, time.Now(), canariesOnly)
	if err != nil {
		return false, nil, err
	}

	// does the current tag exist in the list of triggered canary or generic slos
	if alerts, ok := tags[tag]; ok && len(alerts) > 0 {
		return true, alerts, nil
	}

	return false, nil, nil
}

func (a API) IsDeploymentTriggered(ctx context.Context, app, tag string) (bool, []string, error) {
	q := NewAlertQuery(app, tag)

	tags, err := a.TaggedDeploymentQueries(ctx, q, time.Now())
	if err != nil {
		return false, nil, err
	}

	// does the current tag exist in the list of triggered canary or generic slos
	if alerts, ok := tags[tag]; ok && len(alerts) > 0 {
		return true, alerts, nil
	}

	return false, nil, nil
}
