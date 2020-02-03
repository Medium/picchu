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
	DefaultCanaryFailAfter = "1m"
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
			canaryRules := p.canaryRules(slo)
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
			Name:      canaryRuleName(p.App, p.Tag),
			Namespace: p.Namespace,
			Labels: map[string]string{
				picchuv1alpha1.LabelApp: p.App,
				picchuv1alpha1.LabelTag: p.Tag,
				"prometheus":            "slo",
			},
		},
	}
}

func (p *SyncCanaryRules) canaryRules(slo *picchuv1alpha1.ServiceLevelObjective) []*monitoringv1.RuleGroup {
	ruleGroups := []*monitoringv1.RuleGroup{}

	name := sanitizeName(slo.Name)

	labels := map[string]string{
		"app":    p.App,
		"tag":    p.Tag,
		"canary": "true",
		"slo":    "true",
	}

	for k, v := range slo.Labels {
		labels[k] = v
	}

	annotations := map[string]string{}
	for k, v := range slo.Annotations {
		annotations[k] = v
	}

	canaryFailAfter := DefaultCanaryFailAfter
	if slo.ServiceLevelIndicator.Canary.FailAfter != "" {
		canaryFailAfter = slo.ServiceLevelIndicator.Canary.FailAfter
	}

	ruleGroup := &monitoringv1.RuleGroup{
		Name: canaryAlertName(name),
		Rules: []monitoringv1.Rule{
			{
				Alert:       canaryAlertName(name),
				For:         canaryFailAfter,
				Expr:        intstr.FromString(canaryQuery(slo, p.App, p.Tag)),
				Labels:      labels,
				Annotations: annotations,
			},
		},
	}
	ruleGroups = append(ruleGroups, ruleGroup)

	return ruleGroups
}

func canaryRuleName(name string, tag string) string {
	return fmt.Sprintf("%s-canary-%s", name, tag)
}

func canaryAlertName(name string) string {
	return fmt.Sprintf("%s_canary", name)
}

func canaryQuery(slo *picchuv1alpha1.ServiceLevelObjective, app string, tag string) string {
	name := sanitizeName(slo.Name)
	errorQueryName := errorQueryName(slo, app, name)
	totalQueryName := totalQueryName(slo, app, name)

	return fmt.Sprintf("%s{%s=\"%s\"} / %s{%s=\"%s\"} + %v < ignoring(%s) sum(%s) / sum(%s)",
		errorQueryName, slo.ServiceLevelIndicator.TagKey, tag,
		totalQueryName, slo.ServiceLevelIndicator.TagKey, tag,
		slo.ServiceLevelIndicator.Canary.AllowancePercent, slo.ServiceLevelIndicator.TagKey, errorQueryName, totalQueryName)
}
