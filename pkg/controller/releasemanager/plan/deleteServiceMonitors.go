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

type DeleteServiceMonitors struct {
	App       string
	Namespace string
}

func (p *DeleteServiceMonitors) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	smlist := &monitoringv1.ServiceMonitorList{}

	opts := &client.ListOptions{
		Namespace: p.Namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{
			picchuv1alpha1.LabelApp: p.App,
		}),
	}

	if err := cli.List(ctx, smlist, opts); err != nil {
		log.Error(err, "Failed to delete ServiceMonitors")
		return err
	}

	for _, sm := range smlist.Items {
		err := cli.Delete(ctx, sm)
		if err != nil && !errors.IsNotFound(err) {
			plan.LogSync(log, "deleted", err, sm)
			return err
		}
		if err == nil {
			plan.LogSync(log, "deleted", err, sm)
		}
	}

	return nil
}
