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

type SyncCanaryRules struct {
	App                    string
	Namespace              string
	Tag                    string
	ServiceLevelObjectives []*picchuv1alpha1.ServiceLevelObjective
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

	for _, slo := range p.ServiceLevelObjectives {
		if slo.ServiceLevelIndicator.Canary.Enabled {
			config := SLOConfig{
				SLO:  slo,
				App:  p.App,
				Name: sanitizeName(slo.Name),
				Tag:  p.Tag,
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
	return &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.canaryRuleName(),
			Namespace: p.Namespace,
			Labels: map[string]string{
				picchuv1alpha1.LabelApp: p.App,
				picchuv1alpha1.LabelTag: p.Tag,
				"prometheus":            "slo",
			},
		},
	}
}

func (s *SLOConfig) canaryRules() []*monitoringv1.RuleGroup {
	ruleGroups := []*monitoringv1.RuleGroup{}

	labels := map[string]string{
		"app":    s.App,
		"tag":    s.Tag,
		"canary": "true",
		"slo":    "true",
	}

	for k, v := range s.SLO.Labels {
		labels[k] = v
	}

	annotations := map[string]string{}
	for k, v := range s.SLO.Annotations {
		annotations[k] = v
	}

	ruleGroup := &monitoringv1.RuleGroup{
		Name: s.canaryAlertName(),
		Rules: []monitoringv1.Rule{
			{
				Alert:       s.canaryAlertName(),
				For:         s.SLO.ServiceLevelIndicator.Canary.FailAfter,
				Expr:        intstr.FromString(s.canaryQuery()),
				Labels:      labels,
				Annotations: annotations,
			},
		},
	}
	ruleGroups = append(ruleGroups, ruleGroup)

	return ruleGroups
}

func (p *SyncCanaryRules) canaryRuleName() string {
	return fmt.Sprintf("%s-canary-%s", p.App, p.Tag)
}

func (s *SLOConfig) canaryAlertName() string {
	return fmt.Sprintf("%s_canary", s.Name)
}

func (s *SLOConfig) canaryQuery() string {
	return fmt.Sprintf("%s{%s=\"%s\"} / %s{%s=\"%s\"} + %v < ignoring(%s) sum(%s) / sum(%s)",
		s.errorQuery(), s.SLO.ServiceLevelIndicator.TagKey, s.Tag,
		s.totalQuery(), s.SLO.ServiceLevelIndicator.TagKey, s.Tag,
		s.SLO.ServiceLevelIndicator.Canary.AllowancePercent, s.SLO.ServiceLevelIndicator.TagKey, s.errorQuery(), s.totalQuery())
}
