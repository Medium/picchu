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

type DeleteDatadogSLOs struct {
	App string
	// we need the target to know what the query should be?
	Target    string
	Namespace string
	// do we need the tag??
	Tag string
}

func (p *DeleteDatadogSLOs) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	ddogslolist := &ddog.DatadogSLOList{}

	opts := &client.ListOptions{
		Namespace: p.Namespace,
		// hm? labels?
		LabelSelector: labels.SelectorFromSet(map[string]string{
			picchuv1alpha1.LabelApp:    p.App,
			picchuv1alpha1.LabelTarget: p.Target,
			picchuv1alpha1.LabelTag:    p.Tag,
		}),
	}

	if err := cli.List(ctx, ddogslolist, opts); err != nil {
		log.Error(err, "Failed to delete datadog slo")
		return err
	}

	for _, sl := range ddogslolist.Items {
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
