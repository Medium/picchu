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
	var sl []slov1alpha1.PrometheusServiceLevel
	var slos []slov1alpha1.SLO

	for i := range p.ServiceLevelObjectives {
		if p.ServiceLevelObjectives[i].Enabled {
			config := SLOConfig{
				SLO:    p.ServiceLevelObjectives[i],
				App:    p.App,
				Name:   sanitizeName(p.ServiceLevelObjectives[i].Name),
				Labels: p.ServiceLevelObjectiveLabels,
			}
			serviceLevelObjective := config.serviceLevelObjective(log)

			// if a grpc slo
			if _, ok := p.ServiceLevelObjectives[i].ServiceLevelObjectiveLabels.ServiceLevelLabels["is_grpc"]; ok {
				serviceLevelObjective.SLI.Events = config.sliSourceGRPC()
			} else {
				serviceLevelObjective.SLI.Events = config.sliSource()
			}

			slos = append(slos, *serviceLevelObjective)
		}
	}

	if len(slos) > 0 {
		serviceLevel := &slov1alpha1.PrometheusServiceLevel{
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
		sl = append(sl, *serviceLevel)
	}

	sll.Items = sl
	return sll, nil
}

func (p *SyncServiceLevels) serviceLevelName() string {
	return fmt.Sprintf("%s-%s-servicelevels", p.App, p.Target)
}
