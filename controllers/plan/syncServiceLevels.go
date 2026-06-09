package plan

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	slov1alpha1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/plan"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SyncServiceLevels creates a single tag-agnostic PrometheusServiceLevel per app/target.
// SLI queries aggregate by tag so Sloth recording rules and burn-rate alerts preserve
// per-revision granularity without per-deploy PSL churn.
type SyncServiceLevels struct {
	App                         string
	Target                      string
	Namespace                   string
	Labels                      map[string]string
	ServiceLevelObjectiveLabels picchuv1alpha1.ServiceLevelObjectiveLabels
	ServiceLevelObjectives      []*picchuv1alpha1.SlothServiceLevelObjective
}

func (p *SyncServiceLevels) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	log = log.WithValues("Applier", "SyncServiceLevels")
	if len(p.ServiceLevelObjectives) == 0 {
		return nil
	}
	return plan.CreateOrUpdate(ctx, log, cli, p.serviceLevel(log))
}

// Enabled filtering happens at the source (ResourceSyncer.prepareServiceLevelObjectives);
// every SLO passed in is rendered.
func (p *SyncServiceLevels) serviceLevel(log logr.Logger) *slov1alpha1.PrometheusServiceLevel {
	var slos []slov1alpha1.SLO
	for i := range p.ServiceLevelObjectives {
		config := SLOConfig{
			SLO:    p.ServiceLevelObjectives[i],
			App:    p.App,
			Name:   sanitizeName(p.ServiceLevelObjectives[i].Name),
			Labels: p.ServiceLevelObjectiveLabels,
		}
		slo := config.serviceLevelObjective(log)
		if _, ok := p.ServiceLevelObjectives[i].ServiceLevelObjectiveLabels.ServiceLevelLabels["is_grpc"]; ok {
			slo.SLI.Events = config.sliSourceGRPC()
		} else {
			slo.SLI.Events = config.sliSource()
		}
		slos = append(slos, *slo)
	}
	return &slov1alpha1.PrometheusServiceLevel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sharedServiceLevelName(p.App, p.Target),
			Namespace: p.Namespace,
			Labels:    p.Labels,
		},
		Spec: slov1alpha1.PrometheusServiceLevelSpec{
			Service: p.App,
			SLOs:    slos,
		},
	}
}

// sharedServiceLevelName returns the deterministic name of the app/target-shared
// PrometheusServiceLevel. SyncServiceLevels and DeleteServiceLevels must agree on
// this name — a mismatch turns the delete path into a silent no-op.
func sharedServiceLevelName(app, target string) string {
	return fmt.Sprintf("%s-%s-servicelevels", app, target)
}
