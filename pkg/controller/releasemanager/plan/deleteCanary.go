package plan

import (
	"context"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/go-logr/logr"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/plan"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeleteCanary struct {
	App       string
	Namespace string
	Tag       string
	Target    string
}

func (p *DeleteCanary) Apply(ctx context.Context, cli client.Client, log logr.Logger) error {
	canaryRules := &monitoringv1.PrometheusRuleList{}

	opts := client.
		MatchingLabels(map[string]string{
			picchuv1alpha1.LabelApp:      p.App,
			picchuv1alpha1.LabelTag:      p.Tag,
			picchuv1alpha1.LabelTarget:   p.Target,
			picchuv1alpha1.LabelRuleType: DefaultCanaryLabel,
		}).
		InNamespace(p.Namespace)

	if err := cli.List(ctx, opts, canaryRules); err != nil {
		log.Error(err, "Failed to fetch canary rules")
		return err
	}

	for _, canaryRule := range canaryRules.Items {
		err := cli.Delete(ctx, canaryRule)
		if err != nil && !errors.IsNotFound(err) {
			plan.LogSync(log, "deleted", err, canaryRule)
			return nil
		}
		if err == nil {
			plan.LogSync(log, "deleted", err, canaryRule)
		}
	}

	return nil
}
