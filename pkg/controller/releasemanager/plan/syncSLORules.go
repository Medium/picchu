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

type SLOConfig struct {
	SLO  *picchuv1alpha1.ServiceLevelObjective
	App  string
	Name string
	Tag  string
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
		config := SLOConfig{
			SLO:  slo,
			App:  p.App,
			Name: sanitizeName(slo.Name),
		}
		recordingRules := config.recordingRules()
		for _, rg := range recordingRules {
			rule.Spec.Groups = append(rule.Spec.Groups, *rg)
		}
		alertRules := config.alertRules()
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
			Name:      p.prometheusRuleName(),
			Namespace: p.Namespace,
			Labels: map[string]string{
				picchuv1alpha1.LabelApp: p.App,
				"prometheus":            "slo",
			},
		},
	}
}

func (s *SLOConfig) recordingRules() []*monitoringv1.RuleGroup {
	ruleGroups := []*monitoringv1.RuleGroup{}

	ruleGroup := &monitoringv1.RuleGroup{
		Name: s.recordingRuleName(),
		Rules: []monitoringv1.Rule{
			{
				Record: s.totalQuery(),
				Expr:   intstr.FromString(s.SLO.ServiceLevelIndicator.TotalQuery),
			},
			{
				Record: s.errorQuery(),
				Expr:   intstr.FromString(s.SLO.ServiceLevelIndicator.ErrorQuery),
			},
		},
	}
	ruleGroups = append(ruleGroups, ruleGroup)

	return ruleGroups
}

func (s *SLOConfig) alertRules() []*monitoringv1.RuleGroup {
	ruleGroups := []*monitoringv1.RuleGroup{}

	labels := map[string]string{
		"app": s.App,
	}

	for k, v := range s.SLO.Labels {
		labels[k] = v
	}

	annotations := map[string]string{}
	for k, v := range s.SLO.Annotations {
		annotations[k] = v
	}

	ruleGroup := &monitoringv1.RuleGroup{
		Name: s.alertRuleName(),
		Rules: []monitoringv1.Rule{
			{
				Alert:       "SLOErrorRateTooFast1h",
				For:         s.SLO.ServiceLevelIndicator.AlertAfter,
				Expr:        intstr.FromString(s.burnRateAlertQuery1h()),
				Labels:      labels,
				Annotations: annotations,
			},
			{
				Alert:       "SLOErrorRateTooFast6h",
				For:         s.SLO.ServiceLevelIndicator.AlertAfter,
				Expr:        intstr.FromString(s.burnRateAlertQuery6h()),
				Labels:      labels,
				Annotations: annotations,
			},
		},
	}
	ruleGroups = append(ruleGroups, ruleGroup)

	return ruleGroups
}

func (s *SLOConfig) recordingRuleName() string {
	return fmt.Sprintf("%s_record", s.Name)
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

func (s *SLOConfig) totalQuery() string {
	return fmt.Sprintf("%s:%s:%s", sanitizeName(s.App), sanitizeName(s.Name), "total")
}

func (s *SLOConfig) errorQuery() string {
	return fmt.Sprintf("%s:%s:%s", sanitizeName(s.App), sanitizeName(s.Name), "errors")
}

func (s *SLOConfig) formatObjectivePercent() string {
	var x, y, objPct big.Float
	x.SetFloat64(s.SLO.ObjectivePercent)
	y.SetInt64(100)
	objPct.Quo(&x, &y)
	return fmt.Sprintf("%.10g", &objPct)
}

func (p *SyncSLORules) prometheusRuleName() string {
	return fmt.Sprintf("%s-slo", strings.ToLower(p.App))
}

//ToLower and non-numeric characters replaced with underscores
func sanitizeName(name string) string {
	return strings.ToLower(rgNameRegex.ReplaceAllString(name, "_"))
}
