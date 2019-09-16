package plan

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/go-logr/logr"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/plan"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DefaultAlertAfter           = "2m"
	DefaultTagKey               = "destination_workload"
	RuleNameTemplate            = "%s-%s-%s"
	Canary            AlertType = "canary"
	SLI               AlertType = "sli"
)

var (
	CanarySLIFailingQueryTemplate = template.Must(template.New("canaryTaggedAlerts").
					Parse(`100 * (1 - {{.ErrorQuery}}{ {{.TagKey}}="{{.TagValue}}" } / {{.TotalQuery}}{ {{.TagKey}}="{{.TagValue}}" }) + {{.CanaryAllowance}} < (100 * (1 - sum({{.ErrorQuery}}) / sum({{.TotalQuery}})))`))
	SLIFailingQueryTemplate = template.Must(template.New("sliTaggedAlerts").
				Parse(`100 * (1 - {{.ErrorQuery}}{ {{.TagKey}}="{{.TagValue}}" } / {{.TotalQuery}}{ {{.TagKey}}="{{.TagValue}}" }) < {{.AvailabilityObjective}}`))
)

type SLIQuery struct {
	ErrorQuery            string
	TotalQuery            string
	TagKey                string
	TagValue              string
	CanaryAllowance       float64
	AvailabilityObjective float64
}

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
	rules, err := p.rules()
	if err != nil {
		return err
	}
	if len(rules.Spec.Groups) > 0 {
		if err := plan.CreateOrUpdate(ctx, log, cli, rules); err != nil {
			return err
		}
	}

	return nil
}

func (p *SyncAlerts) rules() (*monitoringv1.PrometheusRule, error) {
	alertType := string(p.AlertType)

	rule := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf(RuleNameTemplate, p.App, p.Tag, alertType),
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
		if slo.Enabled && p.AlertType != Canary || slo.ServiceLevelIndicator.UseForCanary {
			alertAfter := DefaultAlertAfter
			if slo.ServiceLevelIndicator.AlertAfter != "" {
				alertAfter = slo.ServiceLevelIndicator.AlertAfter
			}
			tagKey := DefaultTagKey
			if slo.ServiceLevelIndicator.TagKey != "" {
				tagKey = slo.ServiceLevelIndicator.TagKey
			}

			labels := make(map[string]string)
			labels["app"] = p.App
			labels["alertType"] = alertType
			labels["tag"] = p.Tag

			var expr intstr.IntOrString
			var template template.Template
			q := bytes.NewBufferString("")
			query := SLIQuery{
				ErrorQuery:            slo.ServiceLevelIndicator.ErrorQuery,
				TotalQuery:            slo.ServiceLevelIndicator.TotalQuery,
				CanaryAllowance:       slo.ServiceLevelIndicator.CanaryAllowance,
				AvailabilityObjective: slo.AvailabilityObjectivePercent,
				TagKey:                tagKey,
				TagValue:              p.Tag,
			}

			switch p.AlertType {
			case Canary:
				template = *CanarySLIFailingQueryTemplate
			case SLI:
				template = *SLIFailingQueryTemplate
			}

			if err := template.Execute(q, query); err != nil {
				return nil, err
			}
			expr = intstr.FromString(q.String())

			rules = append(rules, monitoringv1.Rule{
				Labels: labels,
				Alert:  fmt.Sprintf(RuleNameTemplate, slo.Name, p.Tag, alertType),
				For:    alertAfter,
				Expr:   expr,
			})
		}
	}
	rule.Spec = monitoringv1.PrometheusRuleSpec{
		Groups: []monitoringv1.RuleGroup{{
			Name:  fmt.Sprintf(RuleNameTemplate, p.App, p.Tag, alertType),
			Rules: rules,
		}},
	}

	return rule, nil
}
