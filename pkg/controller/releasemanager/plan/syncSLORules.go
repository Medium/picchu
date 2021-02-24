package plan

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/go-logr/logr"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/plan"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// RuleTypeSLO is the value of the LabelRuleType label for SLO rules.
	RuleTypeSLO = "slo"
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

func (p *SyncSLORules) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
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
	var prs []monitoringv1.PrometheusRule

	rule := p.prometheusRule()

	for i := range p.ServiceLevelObjectives {
		config := SLOConfig{
			SLO:    p.ServiceLevelObjectives[i],
			App:    p.App,
			Name:   sanitizeName(p.ServiceLevelObjectives[i].Name),
			Labels: p.ServiceLevelObjectiveLabels,
		}
		recordingRules := config.recordingRules()
		for j := range recordingRules {
			rule.Spec.Groups = append(rule.Spec.Groups, *recordingRules[j])
		}
	}

	prs = append(prs, *rule)
	return prs, nil
}

func (p *SyncSLORules) prometheusRule() *monitoringv1.PrometheusRule {
	labels := make(map[string]string)
	for k, v := range p.Labels {
		labels[k] = v
	}
	for k, v := range p.ServiceLevelObjectiveLabels.RuleLabels {
		labels[k] = v
	}

	labels[picchuv1alpha1.LabelRuleType] = RuleTypeSLO

	return &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.prometheusRuleName(),
			Namespace: p.Namespace,
			Labels:    labels,
		},
	}
}

func (s *SLOConfig) recordingRules() []*monitoringv1.RuleGroup {
	var ruleGroups []*monitoringv1.RuleGroup

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

func (s *SLOConfig) recordingRuleName() string {
	return fmt.Sprintf("%s_record", s.Name)
}

func (p *SyncSLORules) prometheusRuleName() string {
	return fmt.Sprintf("%s-slo", strings.ToLower(p.App))
}

//ToLower and non-numeric characters replaced with underscores
func sanitizeName(name string) string {
	return strings.ToLower(rgNameRegex.ReplaceAllString(name, "_"))
}
