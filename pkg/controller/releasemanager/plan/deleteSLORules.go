package plan

import (
	"context"

	"github.com/go-logr/logr"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/plan"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeleteSLORules struct {
	App       string
	Namespace string
}

func (p *DeleteSLORules) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	prlist := &monitoringv1.PrometheusRuleList{}

	opts := &client.ListOptions{
		Namespace: p.Namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{
			picchuv1alpha1.LabelApp:      p.App,
			picchuv1alpha1.LabelRuleType: RuleTypeSLO,
		}),
	}

	if err := cli.List(ctx, prlist, opts); err != nil {
		log.Error(err, "Failed to delete SLO Rules")
		return err
	}

	for _, prometheusRule := range prlist.Items {
		err := cli.Delete(ctx, prometheusRule)
		if err != nil && !errors.IsNotFound(err) {
			plan.LogSync(log, "deleted", err, prometheusRule)
			return err
		}
		if err == nil {
			plan.LogSync(log, "deleted", err, prometheusRule)
		}
	}

	return nil
}
