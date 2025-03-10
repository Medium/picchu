package plan

import (
	"context"

	ddog "github.com/DataDog/datadog-operator/api/datadoghq/v1alpha1"
	"github.com/go-logr/logr"

	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/plan"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeleteDatadogMetricsCanary struct {
	App       string
	Namespace string
	Tag       string
}

func (p *DeleteDatadogMetricsCanary) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	ddogmlist := &ddog.DatadogMetricList{}

	opts := &client.ListOptions{
		Namespace: p.Namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{
			picchuv1alpha1.LabelApp:      p.App,
			picchuv1alpha1.LabelTag:      p.Tag,
			picchuv1alpha1.LabelRuleType: RuleTypeCanary,
		}),
	}

	if err := cli.List(ctx, ddogmlist, opts); err != nil {
		log.Error(err, "Failed to delete DatadogMetricsCanary")
		return err
	}

	for _, datadogMetric := range ddogmlist.Items {
		err := cli.Delete(ctx, &datadogMetric)
		if err != nil && !errors.IsNotFound(err) {
			plan.LogSync(log, "deleted", err, &datadogMetric)
			return err
		}
		if err == nil {
			plan.LogSync(log, "deleted", err, &datadogMetric)
		}
	}

	return nil
}
