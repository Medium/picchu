package plan

import (
	"context"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/go-logr/logr"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/plan"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeleteAlerts struct {
	App       string
	Namespace string
	Tag       string
	AlertType AlertType
}

func (p *DeleteAlerts) Apply(ctx context.Context, cli client.Client, scalingFactor float64, log logr.Logger) error {
	rules := &monitoringv1.PrometheusRuleList{}
	alertType := string(p.AlertType)

	opts := &client.ListOptions{
		Namespace: p.Namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{
			picchuv1alpha1.LabelApp:      p.App,
			picchuv1alpha1.LabelTag:      p.Tag,
			picchuv1alpha1.LabelRuleType: alertType,
		}),
	}

	if err := cli.List(ctx, rules, opts); err != nil {
		log.Error(err, "Failed to fetch rules")
		return err
	}

	if rules.Items != nil {
		for _, rule := range rules.Items {
			err := cli.Delete(ctx, rule)
			if err != nil && !errors.IsNotFound(err) {
				plan.LogSync(log, "deleted", err, rule)
				return err
			}
			if err == nil {
				plan.LogSync(log, "deleted", err, rule)
			}
		}
	}
	return nil
}
