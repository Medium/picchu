package plan

import (
	"fmt"
	"strconv"

	"github.com/go-logr/logr"
	slov1alpha1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	prometheus "go.medium.engineering/picchu/prometheus"
)

// Receiver pointer functions for SLOConfig can be found next to relevant syncer
type SLOConfig struct {
	SLO    *picchuv1alpha1.SlothServiceLevelObjective
	App    string
	Name   string
	Tag    string
	Labels picchuv1alpha1.ServiceLevelObjectiveLabels
}

func (s *SLOConfig) totalQuery() string {
	return fmt.Sprintf("%s:%s:%s", sanitizeName(s.App), sanitizeName(s.Name), "total")
}

func (s *SLOConfig) errorQuery() string {
	return fmt.Sprintf("%s:%s:%s", sanitizeName(s.App), sanitizeName(s.Name), "errors")
}

func (s *SLOConfig) serviceLevelObjective(log logr.Logger) *slov1alpha1.SLO {
	labels := make(map[string]string)
	for k, v := range s.Labels.ServiceLevelLabels {
		labels[k] = v
	}

	for k, v := range s.SLO.ServiceLevelObjectiveLabels.ServiceLevelLabels {
		labels[k] = v
	}

	if s.Tag != "" {
		labels[prometheus.TagLabel] = s.Tag
	}

	var objectivePercent float64
	if s.SLO.Objective != "" {
		f, err := strconv.ParseFloat(s.SLO.Objective, 64)
		if err != nil {
			log.Error(err, "Could not parse %v to float", s.SLO.Objective)
		} else {
			objectivePercent = f
		}
	}
	slo := &slov1alpha1.SLO{
		Name:        s.Name,
		Objective:   objectivePercent,
		Description: s.SLO.Description,
		Labels:      labels,
	}
	if s.SLO.Alerting.Name != "" {
		alerting := slov1alpha1.Alerting{
			Name:        s.SLO.Alerting.Name,
			Labels:      s.SLO.Alerting.Labels,
			Annotations: s.SLO.Alerting.Annotations,
			PageAlert: slov1alpha1.Alert{
				Disable:     s.SLO.Alerting.PageAlert.Disable,
				Labels:      s.SLO.Alerting.PageAlert.Labels,
				Annotations: s.SLO.Alerting.PageAlert.Annotations,
			},
			TicketAlert: slov1alpha1.Alert{
				Disable:     s.SLO.Alerting.TicketAlert.Disable,
				Labels:      s.SLO.Alerting.TicketAlert.Labels,
				Annotations: s.SLO.Alerting.TicketAlert.Annotations,
			},
		}
		slo.Alerting = alerting
	}
	return slo
}

// tagKey returns the label that carries the revision tag on this SLO's source
// metrics. kbfd defaults serviceLevelIndicator.tagKey to "tag"; services may
// override it when their metrics expose the revision under another label
// (e.g. destination_workload).
func (s *SLOConfig) tagKey() string {
	if k := s.SLO.ServiceLevelIndicator.TagKey; k != "" {
		return k
	}
	return prometheus.TagLabel
}

// sliRate wraps a recording rule in rate() and, when the SLO uses a custom
// tagKey, re-presents that label under the canonical "tag" label so Sloth
// recording rules and burn-rate alerts keep the label the rollback pipeline
// (prometheus/api.go) matches on.
func (s *SLOConfig) sliRate(query string) string {
	rate := fmt.Sprintf("rate(%s[{{.window}}])", query)
	if k := s.tagKey(); k != prometheus.TagLabel {
		rate = fmt.Sprintf("label_replace(%s, %q, \"$1\", %q, \"(.*)\")", rate, prometheus.TagLabel, k)
	}
	return rate
}

// sliSource returns SLI queries that preserve the tag dimension via `sum by (tag)`.
// Sloth wraps these as (error)/(total) with no additional aggregation, so the
// resulting recording rule retains `tag`. Sloth's `max(...) without (sloth_window)`
// burn-rate alert template preserves `tag` through to the alert series, allowing
// picchu's IsRevisionTriggered to match by sample.Metric["tag"].
func (s *SLOConfig) sliSource() *slov1alpha1.SLIEvents {
	return &slov1alpha1.SLIEvents{
		ErrorQuery: fmt.Sprintf("sum by (%s) (%s)", prometheus.TagLabel, s.sliRate(s.errorQuery())),
		TotalQuery: fmt.Sprintf("sum by (%s) (%s)", prometheus.TagLabel, s.sliRate(s.totalQuery())),
	}
}

func (s *SLOConfig) sliSourceGRPC() *slov1alpha1.SLIEvents {
	return &slov1alpha1.SLIEvents{
		ErrorQuery: fmt.Sprintf("sum by (%s, grpc_method) (%s)", prometheus.TagLabel, s.sliRate(s.errorQuery())),
		TotalQuery: fmt.Sprintf("sum by (%s, grpc_method) (%s)", prometheus.TagLabel, s.sliRate(s.totalQuery())),
	}
}
