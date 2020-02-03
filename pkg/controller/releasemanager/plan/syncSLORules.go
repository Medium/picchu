package plan

import (
	"context"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/go-logr/logr"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/plan"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	rgNameRegex = regexp.MustCompile("[^a-zA-Z0-9]+")
)

type SyncSLORules struct {
	App                    string
	Namespace              string
	ServiceLevelObjectives []*picchuv1alpha1.ServiceLevelObjective
}

func (p *SyncSLORules) Apply(ctx context.Context, cli client.Client, scalingFactor float64, log logr.Logger) error {
	sloRules, err := p.SLORules()
	if err != nil {
		return err
	}

	if len(sloRules) > 0 {
		for _, pr := range sloRules {
			if err := plan.CreateOrUpdate(ctx, log, cli, &pr); err != nil {
				return err
			}
		}
	}

	return nil
}

// PrometheusRules creates a PrometheusRuleList of recording and alert rules to support tracking of the ServiceLevelObjectives
func (p *SyncSLORules) SLORules() ([]monitoringv1.PrometheusRule, error) {
	prs := []monitoringv1.PrometheusRule{}

	rule := p.prometheusRule()

	for _, slo := range p.ServiceLevelObjectives {
		recordingRules := p.recordingRules(slo)
		for _, rg := range recordingRules {
			rule.Spec.Groups = append(rule.Spec.Groups, *rg)
		}
		alertRules := p.alertRules(slo)
		for _, rg := range alertRules {
			rule.Spec.Groups = append(rule.Spec.Groups, *rg)
		}
	}

	prs = append(prs, *rule)
	return prs, nil
}

func (p *SyncSLORules) prometheusRule() *monitoringv1.PrometheusRule {
	return &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prometheusRuleName(p.App),
			Namespace: p.Namespace,
			Labels: map[string]string{
				picchuv1alpha1.LabelApp: p.App,
				"prometheus":            "slo",
			},
		},
	}
}

func (p *SyncSLORules) recordingRules(slo *picchuv1alpha1.ServiceLevelObjective) []*monitoringv1.RuleGroup {
	ruleGroups := []*monitoringv1.RuleGroup{}

	name := sanitizeName(slo.Name)

	ruleGroup := &monitoringv1.RuleGroup{
		Name: recordingRuleName(name),
		Rules: []monitoringv1.Rule{
			{
				Record: totalQueryName(slo, p.App, name),
				Expr:   intstr.FromString(slo.ServiceLevelIndicator.TotalQuery),
			},
			{
				Record: errorQueryName(slo, p.App, name),
				Expr:   intstr.FromString(slo.ServiceLevelIndicator.ErrorQuery),
			},
		},
	}
	ruleGroups = append(ruleGroups, ruleGroup)

	return ruleGroups
}

func (p *SyncSLORules) alertRules(slo *picchuv1alpha1.ServiceLevelObjective) []*monitoringv1.RuleGroup {
	ruleGroups := []*monitoringv1.RuleGroup{}

	name := sanitizeName(slo.Name)

	labels := map[string]string{
		"app": p.App,
	}

	for k, v := range slo.Labels {
		labels[k] = v
	}

	annotations := map[string]string{}
	for k, v := range slo.Annotations {
		annotations[k] = v
	}

	ruleGroup := &monitoringv1.RuleGroup{
		Name: alertRuleName(name),
		Rules: []monitoringv1.Rule{
			{
				Alert:       "SLOErrorRateTooFast1h",
				For:         slo.ServiceLevelIndicator.AlertAfter,
				Expr:        intstr.FromString(burnRateAlertQuery1h(p.App, name)),
				Labels:      labels,
				Annotations: annotations,
			},
			{
				Alert:       "SLOErrorRateTooFast6h",
				For:         slo.ServiceLevelIndicator.AlertAfter,
				Expr:        intstr.FromString(burnRateAlertQuery6h(p.App, name)),
				Labels:      labels,
				Annotations: annotations,
			},
		},
	}
	ruleGroups = append(ruleGroups, ruleGroup)

	return ruleGroups
}

func prometheusRuleName(app string) string {
	return fmt.Sprintf("%s-slo", strings.ToLower(app))
}

func recordingRuleName(name string) string {
	return fmt.Sprintf("%s_record", name)
}

func alertRuleName(name string) string {
	return fmt.Sprintf("%s_alert", name)
}

func burnRateAlertQuery1h(app, name string) string {
	sloName := sanitizeName(name)
	labels := fmt.Sprintf("service_level=\"%s\", slo=\"%s\"", app, sloName)

	return fmt.Sprintf("(increase(service_level_sli_result_error_ratio_total{%[1]s}[1h]) "+
		"/ increase(service_level_sli_result_count_total{%[1]s}[1h])) "+
		"> (1 - service_level_slo_objective_ratio{%[1]s}) * 14.6", labels)
}

func burnRateAlertQuery6h(app, name string) string {
	sloName := sanitizeName(name)
	labels := fmt.Sprintf("service_level=\"%s\", slo=\"%s\"", app, sloName)

	return fmt.Sprintf("(increase(service_level_sli_result_error_ratio_total{%[1]s}[6h]) "+
		"/ increase(service_level_sli_result_count_total{%[1]s}[6h])) "+
		"> (1 - service_level_slo_objective_ratio{%[1]s}) * 6", labels)
}

func totalQueryName(slo *picchuv1alpha1.ServiceLevelObjective, app, name string) string {
	return fmt.Sprintf("%s:%s:%s", sanitizeName(app), sanitizeName(name), "total")
}

func errorQueryName(slo *picchuv1alpha1.ServiceLevelObjective, app, name string) string {
	return fmt.Sprintf("%s:%s:%s", sanitizeName(app), sanitizeName(name), "errors")
}

//ToLower and non-numeric characters replaced with underscores
func sanitizeName(name string) string {
	return strings.ToLower(rgNameRegex.ReplaceAllString(name, "_"))
}

func formatObjectivePercent(slo *picchuv1alpha1.ServiceLevelObjective) string {
	var x, y, objPct big.Float
	x.SetFloat64(slo.ObjectivePercent)
	y.SetInt64(100)
	objPct.Quo(&x, &y)
	return fmt.Sprintf("%.10g", &objPct)
}
