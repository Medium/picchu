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

func (s *SLOConfig) taggedSLISource() *slov1alpha1.SLIEvents {
	source := &slov1alpha1.SLIEvents{
		ErrorQuery: s.serviceLevelTaggedErrorQuery(),
		TotalQuery: s.serviceLevelTaggedTotalQuery(),
	}
	return source
}

func (s *SLOConfig) serviceLevelTaggedTotalQuery() string {
	return fmt.Sprintf("sum(rate(%s{%s=\"%s\"}[{{.window}}]))", s.totalQuery(), s.SLO.ServiceLevelIndicator.TagKey, s.Tag)
}

func (s *SLOConfig) serviceLevelTaggedErrorQuery() string {
	return fmt.Sprintf("sum(rate(%s{%s=\"%s\"}[{{.window}}]))", s.errorQuery(), s.SLO.ServiceLevelIndicator.TagKey, s.Tag)
}

func (s *SLOConfig) taggedSLISourceGRPC() *slov1alpha1.SLIEvents {
	source := &slov1alpha1.SLIEvents{
		ErrorQuery: s.serviceLevelTaggedErrorQueryGRPC(),
		TotalQuery: s.serviceLevelTaggedTotalQueryGRPC(),
	}
	return source
}

func (s *SLOConfig) serviceLevelTaggedTotalQueryGRPC() string {
	return fmt.Sprintf("sum by (grpc_method) (rate(%s{%s=\"%s\"}[{{.window}}]))", s.totalQuery(), s.SLO.ServiceLevelIndicator.TagKey, s.Tag)
}

func (s *SLOConfig) serviceLevelTaggedErrorQueryGRPC() string {
	return fmt.Sprintf("sum by (grpc_method) (rate(%s{%s=\"%s\"}[{{.window}}]))", s.errorQuery(), s.SLO.ServiceLevelIndicator.TagKey, s.Tag)
}
