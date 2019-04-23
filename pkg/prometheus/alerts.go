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
			Parse(`sum by({{.TagLabel}})(ALERTS{slo="true",app="{{.App}}",alertstate="{{.AlertState}}"})`))
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

type API struct {
	api.API
}

func NewAPI(address string) (*API, error) {
	log.Info("Creating API", "address", address)
	client, err := cli.NewClient(cli.Config{Address: address})
	if err != nil {
		return nil, err
	}
	return &API{api.NewAPI(client)}, nil
}

func (a API) TaggedAlerts(ctx context.Context, query AlertQuery, t time.Time) ([]string, error) {
	log.Info("Butts")
	q := bytes.NewBufferString("")
	if err := alertTemplate.Execute(q, query); err != nil {
		return nil, err
	}
	val, err := a.API.Query(ctx, q.String(), t)
	if err != nil {
		return nil, err
	}
	log.Info("Query result", "Query", q, "Result", val)
	tagset := map[string]bool{}
	switch v := val.(type) {
	case model.Vector:
		for _, sample := range v {
			for _, tag := range sample.Metric {
				if tag != "" {
					tagset[string(tag)] = true
				}
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
