package plan

import (
	"context"

	slov1alpha1 "github.com/Medium/service-level-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/go-logr/logr"
	slov1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/plan"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeleteServiceLevels struct {
	App       string
	Target    string
	Namespace string
}

func (p *DeleteServiceLevels) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	// delete both sloth and legact slos
	sllist := &slov1alpha1.ServiceLevelList{}
	slothsllist := &slov1.PrometheusServiceLevelList{}

	opts := &client.ListOptions{
		Namespace: p.Namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{
			picchuv1alpha1.LabelApp:    p.App,
			picchuv1alpha1.LabelTarget: p.Target,
		}),
	}

	if err := cli.List(ctx, sllist, opts); err != nil {
		log.Error(err, "Failed to delete Service Levels")
		return err
	}

	for _, sl := range sllist.Items {
		err := cli.Delete(ctx, &sl)
		if err != nil && !errors.IsNotFound(err) {
			plan.LogSync(log, "deleted", err, &sl)
			return err
		}
		if err == nil {
			plan.LogSync(log, "deleted", err, &sl)
		}
	}

	if err := cli.List(ctx, slothsllist, opts); err != nil {
		log.Error(err, "Failed to delete Service Levels")
		return err
	}

	for _, sl := range slothsllist.Items {
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
