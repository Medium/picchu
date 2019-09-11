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
	DefaultCanaryLabel      = "canary"
	DefaultCanaryRulesName  = "picchu.canary.rules"
	DefaultCanaryAlertAfter = "5m"
	canarySLIFailingQuery   = "100 * (1 - %s / %s) + %f < %f"
)

func init() {

}

type SyncCanary struct {
	App                    string
	Namespace              string
	Tag                    string
	Target                 string
	ServiceLevelObjectives []picchuv1alpha1.ServiceLevelObjective
}

func (p *SyncCanary) Apply(ctx context.Context, cli client.Client, log logr.Logger) error {
	canaryRule := p.canaryRule()

	if len(canaryRule.Spec.Groups) > 0 {
		if err := plan.CreateOrUpdate(ctx, log, cli, canaryRule); err != nil {
			return err
		}
	}

	return nil
}

func (p *SyncCanary) canaryRule() *monitoringv1.PrometheusRule {
	canaryRule := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-%s", p.App, p.Tag, DefaultCanaryLabel),
			Namespace: p.Namespace,
			Labels: map[string]string{
				picchuv1alpha1.LabelApp:      p.App,
				picchuv1alpha1.LabelTag:      p.Tag,
				picchuv1alpha1.LabelTarget:   p.Target,
				picchuv1alpha1.LabelRuleType: DefaultCanaryLabel,
			},
		},
	}
	canaryRules := []monitoringv1.Rule{}

	for _, slo := range p.ServiceLevelObjectives {
		if slo.ServiceLevelIndicator.UseForCanary {
			alertAfter := DefaultCanaryAlertAfter
			if slo.ServiceLevelIndicator.AlertAfter != "" {
				alertAfter = slo.ServiceLevelIndicator.AlertAfter
			}

			expr := intstr.FromString(fmt.Sprintf(canarySLIFailingQuery,
				slo.ServiceLevelIndicator.ErrorQuery, slo.ServiceLevelIndicator.TotalQuery,
				slo.ServiceLevelIndicator.CanaryAllowance, slo.AvailabilityObjectivePercent))

			labels := make(map[string]string)
			labels["app"] = p.App
			labels["alertType"] = DefaultCanaryLabel

			//TODO(micah): deprecate this approach to labeling
			labels["canary"] = "true"

			canaryRules = append(canaryRules, monitoringv1.Rule{
				Labels: labels,
				Alert:  slo.Name,
				For:    alertAfter,
				Expr:   expr,
			})
		}
	}
	canaryRule.Spec = monitoringv1.PrometheusRuleSpec{
		Groups: []monitoringv1.RuleGroup{{
			Name:  DefaultCanaryRulesName,
			Rules: canaryRules,
		}},
	}

	return canaryRule
}
