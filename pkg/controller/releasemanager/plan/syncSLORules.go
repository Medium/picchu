package plan

import (
	"context"
	"fmt"
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
	App                         string
	Namespace                   string
	Labels                      map[string]string
	ServiceLevelObjectiveLabels picchuv1alpha1.ServiceLevelObjectiveLabels
	ServiceLevelObjectives      []*picchuv1alpha1.ServiceLevelObjective
}

type SLOConfig struct {
	SLO    *picchuv1alpha1.ServiceLevelObjective
	App    string
	Name   string
	Tag    string
	Labels picchuv1alpha1.ServiceLevelObjectiveLabels
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

// SLORules returns a PrometheusRuleList of recording rules to support tracking of the ServiceLevelObjectives
func (p *SyncSLORules) SLORules() ([]monitoringv1.PrometheusRule, error) {
	prs := []monitoringv1.PrometheusRule{}

	rule := p.prometheusRule()

	for _, slo := range p.ServiceLevelObjectives {
		config := SLOConfig{
			SLO:    slo,
			App:    p.App,
			Name:   sanitizeName(slo.Name),
			Labels: p.ServiceLevelObjectiveLabels,
		}
		recordingRules := config.recordingRules()
		for _, rg := range recordingRules {
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
			Labels:    p.Labels,
		},
	}
}

func (s *SLOConfig) recordingRules() []*monitoringv1.RuleGroup {
	ruleGroups := []*monitoringv1.RuleGroup{}

	labels := make(map[string]string)
	for k, v := range s.Labels.RuleLabels {
		labels[k] = v
	}
	for k, v := range s.SLO.ServiceLevelObjectiveLabels.RuleLabels {
		labels[k] = v
	}
	ruleGroup := &monitoringv1.RuleGroup{
		Name: s.recordingRuleName(),
		Rules: []monitoringv1.Rule{
			{
				Record: s.totalQuery(),
				Expr:   intstr.FromString(s.SLO.ServiceLevelIndicator.TotalQuery),
				Labels: labels,
			},
			{
				Record: s.errorQuery(),
				Expr:   intstr.FromString(s.SLO.ServiceLevelIndicator.ErrorQuery),
				Labels: labels,
			},
		},
	}
	ruleGroups = append(ruleGroups, ruleGroup)

	return ruleGroups
}

func (s *SLOConfig) recordingRuleName() string {
	return fmt.Sprintf("%s_record", s.Name)
}

func (s *SLOConfig) totalQuery() string {
	return fmt.Sprintf("%s:%s:%s", sanitizeName(s.App), sanitizeName(s.Name), "total")
}

func (s *SLOConfig) errorQuery() string {
	return fmt.Sprintf("%s:%s:%s", sanitizeName(s.App), sanitizeName(s.Name), "errors")
}

func (p *SyncSLORules) prometheusRuleName() string {
	return fmt.Sprintf("%s-slo", strings.ToLower(p.App))
}

//ToLower and non-numeric characters replaced with underscores
func sanitizeName(name string) string {
	return strings.ToLower(rgNameRegex.ReplaceAllString(name, "_"))
}
