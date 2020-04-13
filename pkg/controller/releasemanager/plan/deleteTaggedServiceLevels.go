package plan

import (
	"context"

	slov1alpha1 "github.com/Medium/service-level-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/go-logr/logr"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/plan"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeleteTaggedServiceLevels struct {
	App       string
	Target    string
	Namespace string
	Tag       string
}

func (p *DeleteTaggedServiceLevels) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	sllist := &slov1alpha1.ServiceLevelList{}

	opts := &client.ListOptions{
		Namespace: p.Namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{
			picchuv1alpha1.LabelApp:    p.App,
			picchuv1alpha1.LabelTarget: p.Target,
			picchuv1alpha1.LabelTag:    p.Tag,
		}),
	}

	if err := cli.List(ctx, sllist, opts); err != nil {
		log.Error(err, "Failed to delete tagged Service Levels")
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

	return nil
}
