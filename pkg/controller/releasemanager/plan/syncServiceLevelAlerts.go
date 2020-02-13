package plan

import (
	"context"
	"fmt"
	"strings"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/go-logr/logr"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/plan"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SyncServiceLevelAlerts struct {
	App                         string
	Target                      string
	Namespace                   string
	Labels                      map[string]string
	ServiceLevelObjectiveLabels picchuv1alpha1.ServiceLevelObjectiveLabels
	ServiceLevelObjectives      []*picchuv1alpha1.ServiceLevelObjective
}

func (p *SyncServiceLevelAlerts) Apply(ctx context.Context, cli client.Client, scalingFactor float64, log logr.Logger) error {
	serviceLevelAlerts, err := p.ServiceLevelAlerts()
	if err != nil {
		return err
	}

	if len(serviceLevelAlerts) > 0 {
		for _, pr := range serviceLevelAlerts {
			if err := plan.CreateOrUpdate(ctx, log, cli, &pr); err != nil {
				return err
			}
		}
	}

	return nil
}

// ServiceLevelAlerts returns a PrometheusRuleList of alerts for ServiceLevels
func (p *SyncServiceLevelAlerts) ServiceLevelAlerts() ([]monitoringv1.PrometheusRule, error) {
	prs := []monitoringv1.PrometheusRule{}

	rule := p.alertRule()

	for i := range p.ServiceLevelObjectives {
		config := SLOConfig{
			SLO:    p.ServiceLevelObjectives[i],
			App:    p.App,
			Name:   sanitizeName(p.ServiceLevelObjectives[i].Name),
			Labels: p.ServiceLevelObjectiveLabels,
		}
		recordingRules := config.alertRules()
		for _, rg := range recordingRules {
			rule.Spec.Groups = append(rule.Spec.Groups, *rg)
		}
	}

	prs = append(prs, *rule)
	return prs, nil
}

func (p *SyncServiceLevelAlerts) alertRule() *monitoringv1.PrometheusRule {
	return &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.alertRuleName(),
			Namespace: p.Namespace,
			Labels:    p.Labels,
		},
	}
}

func (s *SLOConfig) alertRules() []*monitoringv1.RuleGroup {
	ruleGroups := []*monitoringv1.RuleGroup{}

	labels := make(map[string]string)
	for k, v := range s.Labels.AlertLabels {
		labels[k] = v
	}
	for k, v := range s.SLO.ServiceLevelObjectiveLabels.AlertLabels {
		labels[k] = v
	}

	ruleGroup := &monitoringv1.RuleGroup{
		Name: s.alertRuleName(),
		Rules: []monitoringv1.Rule{
			{
				Alert:  "SLOErrorRateTooFast1h",
				For:    s.SLO.ServiceLevelIndicator.AlertAfter,
				Expr:   intstr.FromString(s.burnRateAlertQuery1h()),
				Labels: labels,
			},
			{
				Alert:  "SLOErrorRateTooFast6h",
				For:    s.SLO.ServiceLevelIndicator.AlertAfter,
				Expr:   intstr.FromString(s.burnRateAlertQuery6h()),
				Labels: labels,
			},
		},
	}
	ruleGroups = append(ruleGroups, ruleGroup)

	return ruleGroups
}

func (p *SyncServiceLevelAlerts) alertRuleName() string {
	return fmt.Sprintf("%s-%s-slo-alerts", strings.ToLower(p.App), strings.ToLower(p.Target))
}

func (s *SLOConfig) alertRuleName() string {
	return fmt.Sprintf("%s_alert", s.Name)
}

func (s *SLOConfig) burnRateAlertQuery1h() string {
	sloName := sanitizeName(s.Name)
	labels := fmt.Sprintf("service_level=\"%s\", slo=\"%s\"", s.App, sloName)

	return fmt.Sprintf("(increase(service_level_sli_result_error_ratio_total{%[1]s}[1h]) "+
		"/ increase(service_level_sli_result_count_total{%[1]s}[1h])) "+
		"> (1 - service_level_slo_objective_ratio{%[1]s}) * 14.6", labels)
}

func (s *SLOConfig) burnRateAlertQuery6h() string {
	sloName := sanitizeName(s.Name)
	labels := fmt.Sprintf("service_level=\"%s\", slo=\"%s\"", s.App, sloName)

	return fmt.Sprintf("(increase(service_level_sli_result_error_ratio_total{%[1]s}[6h]) "+
		"/ increase(service_level_sli_result_count_total{%[1]s}[6h])) "+
		"> (1 - service_level_slo_objective_ratio{%[1]s}) * 6", labels)
}
