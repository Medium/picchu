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
// Unlike the old per-tag approach, the SLI queries aggregate across all deployment revisions
// so burn rate windows are continuous across deploys.
type SyncServiceLevels struct {
	App                         string
	Target                      string
	Namespace                   string
	Labels                      map[string]string
	ServiceLevelObjectiveLabels picchuv1alpha1.ServiceLevelObjectiveLabels
	ServiceLevelObjectives      []*picchuv1alpha1.SlothServiceLevelObjective
}

func (p *SyncServiceLevels) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	serviceLevels, err := p.serviceLevels(log)
	if err != nil {
		return err
	}
	if len(serviceLevels.Items) > 0 {
		for i := range serviceLevels.Items {
			if err := plan.CreateOrUpdate(ctx, log, cli, &serviceLevels.Items[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *SyncServiceLevels) serviceLevels(log logr.Logger) (*slov1alpha1.PrometheusServiceLevelList, error) {
	sll := &slov1alpha1.PrometheusServiceLevelList{}
	var slos []slov1alpha1.SLO

	for i := range p.ServiceLevelObjectives {
		if p.ServiceLevelObjectives[i].Enabled {
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
	}

	if len(slos) > 0 {
		sl := slov1alpha1.PrometheusServiceLevel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      p.serviceLevelName(),
				Namespace: p.Namespace,
				Labels:    p.Labels,
			},
			Spec: slov1alpha1.PrometheusServiceLevelSpec{
				Service: p.App,
				SLOs:    slos,
			},
		}
		sll.Items = append(sll.Items, sl)
	}

	return sll, nil
}

func (p *SyncServiceLevels) serviceLevelName() string {
	return fmt.Sprintf("%s-%s-servicelevels", p.App, p.Target)
}
