package plan

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-logr/logr"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/plan"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CanaryAppLabel = "app"
	CanaryTagLabel = "tag"
	CanarySLOLabel = "slo"
	CanaryLabel    = "canary"

	CanarySummaryAnnotation = "summary"
	CanaryMessageAnnotation = "message"

	// RuleTypeCanary is the value of the LabelRuleType label for canary rules.
	RuleTypeCanary = "canary"
)

type SyncCanaryRules struct {
	App                         string
	Namespace                   string
	Tag                         string
	Labels                      map[string]string
	ServiceLevelObjectiveLabels picchuv1alpha1.ServiceLevelObjectiveLabels
	ServiceLevelObjectives      []*picchuv1alpha1.SlothServiceLevelObjective
}

func (p *SyncCanaryRules) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	prometheusRules, err := p.prometheusRules(log)
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

func (p *SyncCanaryRules) prometheusRules(log logr.Logger) (*monitoringv1.PrometheusRuleList, error) {
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
			canaryRules := config.canaryRules(log)

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

	labels[picchuv1alpha1.LabelRuleType] = RuleTypeCanary

	return &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.canaryRuleName(),
			Namespace: p.Namespace,
			Labels:    labels,
		},
	}
}

func (s *SLOConfig) canaryRules(log logr.Logger) []*monitoringv1.RuleGroup {
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
				Alert:       s.canaryAlertName(),
				For:         monitoringv1.Duration(s.SLO.ServiceLevelIndicator.Canary.FailAfter),
				Expr:        intstr.FromString(s.canaryQuery(log)),
				Labels:      labels,
				Annotations: s.canaryRuleAnnotations(log),
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

func (s *SLOConfig) canaryRuleAnnotations(log logr.Logger) map[string]string {
	return map[string]string{
		CanarySummaryAnnotation: "Canary is failing SLO",
		CanaryMessageAnnotation: s.serviceLevelObjective(log).Description,
	}
}

func (p *SyncCanaryRules) canaryRuleName() string {
	return fmt.Sprintf("%s-canary-%s", p.App, p.Tag)
}

func (s *SLOConfig) canaryAlertName() string {
	return fmt.Sprintf("%s_canary", s.Name)
}

func (s *SLOConfig) canaryQuery(log logr.Logger) string {
	return fmt.Sprintf("%s{%s=\"%s\"} / %s{%s=\"%s\"} - %v > sum(%s) / ignoring(%s) sum(%s)",
		s.errorQuery(), s.SLO.ServiceLevelIndicator.TagKey, s.Tag,
		s.totalQuery(), s.SLO.ServiceLevelIndicator.TagKey, s.Tag,
		s.formatAllowancePercent(log), s.errorQuery(), s.SLO.ServiceLevelIndicator.TagKey, s.totalQuery(),
	)
}

func (s *SLOConfig) formatAllowancePercent(log logr.Logger) string {
	allowancePercent := s.SLO.ServiceLevelIndicator.Canary.AllowancePercent
	if s.SLO.ServiceLevelIndicator.Canary.AllowancePercentString != "" {
		f, err := strconv.ParseFloat(s.SLO.ServiceLevelIndicator.Canary.AllowancePercentString, 64)
		if err != nil {
			log.Error(err, "Could not parse %v to float", s.SLO.ServiceLevelIndicator.Canary.AllowancePercentString)
		} else {
			allowancePercent = f
		}
	}
	r := allowancePercent / 100
	return fmt.Sprintf("%.10g", r)
}
