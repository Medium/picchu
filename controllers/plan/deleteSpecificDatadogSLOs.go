package plan

import (
	"context"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	ddog "github.com/DataDog/datadog-operator/api/datadoghq/v1alpha1"
	"github.com/go-logr/logr"
	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/plan"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeleteSpecificDatadogSLOs struct {
	App       string
	Namespace string
	ToRemove  []datadogV1.SearchServiceLevelObjective
}

func (p *DeleteSpecificDatadogSLOs) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	ddogslolist := &ddog.DatadogSLOList{}

	// for each slo to remove, remove it
	for _, slo := range p.ToRemove {
		opts := &client.ListOptions{
			Namespace: p.Namespace,
			LabelSelector: labels.SelectorFromSet(map[string]string{
				picchuv1alpha1.LabelApp:     p.App,
				picchuv1alpha1.LabelSLOName: *slo.Data.Attributes.Name,
			}),
		}

		if err := cli.List(ctx, ddogslolist, opts); err != nil {
			log.Error(err, "Failed to delete datadog slo")
			return err
		}

		for _, sl := range ddogslolist.Items {
			if p.App == "echo" {
				log.Info("Deleting datadog slo", "name", sl.Name, "slo name", sl.Labels[picchuv1alpha1.LabelSLOName])
			}
			err := cli.Delete(ctx, &sl)
			if err != nil && !errors.IsNotFound(err) {
				plan.LogSync(log, "deleted", err, &sl)
				return err
			}
			if err == nil {
				plan.LogSync(log, "deleted", err, &sl)
			}
		}
	}

	return nil
}
