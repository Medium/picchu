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
	DefaultAlertAfter                       = "2m"
	CanarySLIFailingQueryTemplate           = "100 * (1 - %s / %s) + %f < %f"
	SLIFailingQueryTemplate                 = "100 * (1 - %s / %s) < %f"
	DefaultTagExpression                    = "{{ $labels.destination_workload }}"
	RuleNameTemplate                        = "picchu.%s.rules"
	Canary                        AlertType = "canary"
	SLI                           AlertType = "sli"
)

type SyncAlerts struct {
	App                    string
	Namespace              string
	Tag                    string
	Target                 string
	AlertType              AlertType
	ServiceLevelObjectives []picchuv1alpha1.ServiceLevelObjective
}

type AlertType string

func (p *SyncAlerts) Apply(ctx context.Context, cli client.Client, log logr.Logger) error {
	rules := p.rules()

	if len(rules.Spec.Groups) > 0 {
		if err := plan.CreateOrUpdate(ctx, log, cli, rules); err != nil {
			return err
		}
	}

	return nil
}

func (p *SyncAlerts) rules() *monitoringv1.PrometheusRule {
	alertType := string(p.AlertType)

	rule := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-%s", p.App, p.Tag, alertType),
			Namespace: p.Namespace,
			Labels: map[string]string{
				picchuv1alpha1.LabelApp:      p.App,
				picchuv1alpha1.LabelTag:      p.Tag,
				picchuv1alpha1.LabelTarget:   p.Target,
				picchuv1alpha1.LabelRuleType: alertType,
			},
		},
	}
	rules := []monitoringv1.Rule{}

	for _, slo := range p.ServiceLevelObjectives {
		if p.AlertType != Canary || slo.ServiceLevelIndicator.UseForCanary {
			alertAfter := DefaultAlertAfter
			if slo.ServiceLevelIndicator.AlertAfter != "" {
				alertAfter = slo.ServiceLevelIndicator.AlertAfter
			}
			tagExpression := DefaultTagExpression
			if slo.ServiceLevelIndicator.TagExpression != "" {
				tagExpression = slo.ServiceLevelIndicator.TagExpression
			}

			labels := make(map[string]string)
			labels["app"] = p.App
			labels["alertType"] = alertType
			labels["tag"] = tagExpression
			labels["slo"] = "true"

			var expr intstr.IntOrString
			switch p.AlertType {
			case Canary:
				expr = intstr.FromString(fmt.Sprintf(CanarySLIFailingQueryTemplate,
					slo.ServiceLevelIndicator.ErrorQuery, slo.ServiceLevelIndicator.TotalQuery,
					slo.ServiceLevelIndicator.CanaryAllowance, slo.AvailabilityObjectivePercent))
			case SLI:
				expr = intstr.FromString(fmt.Sprintf(SLIFailingQueryTemplate,
					slo.ServiceLevelIndicator.ErrorQuery, slo.ServiceLevelIndicator.TotalQuery, slo.AvailabilityObjectivePercent))
			}

			rules = append(rules, monitoringv1.Rule{
				Labels: labels,
				Alert:  slo.Name,
				For:    alertAfter,
				Expr:   expr,
			})
		}
	}
	rule.Spec = monitoringv1.PrometheusRuleSpec{
		Groups: []monitoringv1.RuleGroup{{
			Name:  fmt.Sprintf(RuleNameTemplate, alertType),
			Rules: rules,
		}},
	}

	return rule
}
