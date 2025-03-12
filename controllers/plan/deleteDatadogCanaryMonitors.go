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

type DeleteDatadogCanaryMonitors struct {
	App       string
	Target    string
	Namespace string
	Tag       string
}

func (p *DeleteDatadogCanaryMonitors) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	ddogMonitorlist := &ddog.DatadogMonitorList{}

	opts := &client.ListOptions{
		Namespace: p.Namespace,
		// hm? labels?
		LabelSelector: labels.SelectorFromSet(map[string]string{
			picchuv1alpha1.LabelApp:    p.App,
			picchuv1alpha1.LabelTarget: p.Target,
			picchuv1alpha1.LabelTag:    p.Tag,
		}),
	}

	if err := cli.List(ctx, ddogMonitorlist, opts); err != nil {
		log.Error(err, "Failed to delete datadog slo")
		return err
	}

	for _, sl := range ddogMonitorlist.Items {
		err := cli.Delete(ctx, &sl)
		if err != nil && !errors.IsNotFound(err) {
			plan.LogSync(log, "deleted", err, &sl)
			return err
		}
		if err == nil {
			plan.LogSync(log, "deleted", err, &sl)
		}
	}

	return nil
}
