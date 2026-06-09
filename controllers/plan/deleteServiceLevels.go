package plan

import (
	"context"

	"github.com/go-logr/logr"
	slov1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeleteServiceLevels struct {
	App       string
	Target    string
	Namespace string
}

func (p *DeleteServiceLevels) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	log = log.WithValues("Applier", "DeleteServiceLevels")

	// Delete by deterministic name. A label-selector list would risk matching
	// legacy per-tag PSLs that share the same {app, target} labels.
	name := sharedServiceLevelName(p.App, p.Target)
	sl := &slov1.PrometheusServiceLevel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: p.Namespace,
		},
	}
	if err := cli.Delete(ctx, sl); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		log.Error(err, "Failed to delete shared service level", "name", name, "namespace", p.Namespace)
		return err
	}
	log.Info("Deleted shared service level", "name", name, "namespace", p.Namespace)
	return nil
}
