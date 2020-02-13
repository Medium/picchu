package plan

import (
	"context"
	"fmt"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/go-logr/logr"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/plan"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CanaryAppLabel = "app"
	CanaryTagLabel = "tag"
	CanarySLOLabel = "slo"
	CanaryLabel    = "canary"
)

type SyncCanaryRules struct {
	App                         string
	Namespace                   string
	Tag                         string
	Labels                      map[string]string
	ServiceLevelObjectiveLabels picchuv1alpha1.ServiceLevelObjectiveLabels
	ServiceLevelObjectives      []*picchuv1alpha1.ServiceLevelObjective
}

func (p *SyncCanaryRules) Apply(ctx context.Context, cli client.Client, scalingFactor float64, log logr.Logger) error {
	prometheusRules, err := p.prometheusRules()
	if err != nil {
		return err
	}

	if len(prometheusRules.Items) > 0 {
		for _, pr := range prometheusRules.Items {
			if err := plan.CreateOrUpdate(ctx, log, cli, pr); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *SyncCanaryRules) prometheusRules() (*monitoringv1.PrometheusRuleList, error) {
	prl := &monitoringv1.PrometheusRuleList{}
	prs := []*monitoringv1.PrometheusRule{}

	rule := p.prometheusRule()

	for i := range p.ServiceLevelObjectives {
		if p.ServiceLevelObjectives[i].ServiceLevelIndicator.Canary.Enabled {
			config := SLOConfig{
				SLO:    p.ServiceLevelObjectives[i],
				App:    p.App,
				Name:   sanitizeName(p.ServiceLevelObjectives[i].Name),
				Tag:    p.Tag,
				Labels: p.ServiceLevelObjectiveLabels,
			}
			canaryRules := config.canaryRules()
			for _, rg := range canaryRules {
				rule.Spec.Groups = append(rule.Spec.Groups, *rg)
			}
		}
	}

	prs = append(prs, rule)
	prl.Items = prs
	return prl, nil
}

func (p *SyncCanaryRules) prometheusRule() *monitoringv1.PrometheusRule {
	labels := make(map[string]string)

	for k, v := range p.Labels {
		labels[k] = v
	}
	for k, v := range p.ServiceLevelObjectiveLabels.RuleLabels {
		labels[k] = v
	}

	return &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.canaryRuleName(),
			Namespace: p.Namespace,
			Labels:    labels,
		},
	}
}

func (s *SLOConfig) canaryRules() []*monitoringv1.RuleGroup {
	ruleGroups := []*monitoringv1.RuleGroup{}

	labels := s.canaryRuleLabels()
	for k, v := range s.Labels.AlertLabels {
		labels[k] = v
	}
	for k, v := range s.SLO.ServiceLevelObjectiveLabels.AlertLabels {
		labels[k] = v
	}

	ruleGroup := &monitoringv1.RuleGroup{
		Name: s.canaryAlertName(),
		Rules: []monitoringv1.Rule{
			{
				Alert:  s.canaryAlertName(),
				For:    s.SLO.ServiceLevelIndicator.Canary.FailAfter,
				Expr:   intstr.FromString(s.canaryQuery()),
				Labels: labels,
			},
		},
	}
	ruleGroups = append(ruleGroups, ruleGroup)

	return ruleGroups
}

func (s *SLOConfig) canaryRuleLabels() map[string]string {
	return map[string]string{
		CanaryAppLabel: s.App,
		CanaryTagLabel: s.Tag,
		CanaryLabel:    "true",
		CanarySLOLabel: "true",
	}
}

func (p *SyncCanaryRules) canaryRuleName() string {
	return fmt.Sprintf("%s-canary-%s", p.App, p.Tag)
}

func (s *SLOConfig) canaryAlertName() string {
	return fmt.Sprintf("%s_canary", s.Name)
}

func (s *SLOConfig) canaryQuery() string {
	return fmt.Sprintf("%s{%s=\"%s\"} / %s{%s=\"%s\"} - %v > ignoring(%s) sum(%s) / sum(%s)",
		s.errorQuery(), s.SLO.ServiceLevelIndicator.TagKey, s.Tag,
		s.totalQuery(), s.SLO.ServiceLevelIndicator.TagKey, s.Tag,
		s.formatAllowancePercent(), s.SLO.ServiceLevelIndicator.TagKey, s.errorQuery(), s.totalQuery())
}

func (s *SLOConfig) formatAllowancePercent() string {
	r := s.SLO.ServiceLevelIndicator.Canary.AllowancePercent / 100
	return fmt.Sprintf("%.10g", r)
}
