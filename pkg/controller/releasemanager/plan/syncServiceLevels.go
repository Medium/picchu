package plan

import (
	"context"

	slov1alpha1 "github.com/Medium/service-level-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/go-logr/logr"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/plan"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SyncServiceLevels struct {
	App                    string
	Namespace              string
	ServiceLevelObjectives []*picchuv1alpha1.ServiceLevelObjective
}

func (p *SyncServiceLevels) Apply(ctx context.Context, cli client.Client, scalingFactor float64, log logr.Logger) error {
	serviceLevels, err := p.serviceLevels()
	if err != nil {
		return err
	}
	if len(serviceLevels.Items) > 0 {
		for _, sl := range serviceLevels.Items {
			if err := plan.CreateOrUpdate(ctx, log, cli, &sl); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *SyncServiceLevels) serviceLevels() (*slov1alpha1.ServiceLevelList, error) {
	sll := &slov1alpha1.ServiceLevelList{}
	sl := []slov1alpha1.ServiceLevel{}
	slos := []slov1alpha1.SLO{}

	for _, s := range p.ServiceLevelObjectives {
		if s.Enabled {
			name := sanitizeName(s.Name)
			labels := map[string]string{}

			for k, v := range s.Labels {
				labels[k] = v
			}

			errorQuery := errorQueryName(s, p.App, name)
			totalQuery := totalQueryName(s, p.App, name)
			slo := serviceLevelObjective(s, name, errorQuery, totalQuery, labels)
			slos = append(slos, *slo)
		}
	}

	serviceLevel := &slov1alpha1.ServiceLevel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.App,
			Namespace: p.Namespace,
			Labels:    p.serviceLevelLabels(),
		},
		Spec: slov1alpha1.ServiceLevelSpec{
			ServiceLevelName:       p.App,
			ServiceLevelObjectives: slos,
		},
	}
	sl = append(sl, *serviceLevel)

	sll.Items = sl
	return sll, nil
}

func serviceLevelObjective(slo *picchuv1alpha1.ServiceLevelObjective, name, errorQuery, totalQuery string, labels map[string]string) *slov1alpha1.SLO {
	sliSource := &slov1alpha1.PrometheusSLISource{
		ErrorQuery: errorQuery,
		TotalQuery: totalQuery,
	}
	s := &slov1alpha1.SLO{
		Name:                         sanitizeName(name),
		AvailabilityObjectivePercent: slo.ObjectivePercent,
		Description:                  slo.Description,
		Disable:                      false,
		Output: slov1alpha1.Output{
			Prometheus: &slov1alpha1.PrometheusOutputSource{
				Labels: labels,
			},
		},
		ServiceLevelIndicator: slov1alpha1.SLI{
			SLISource: slov1alpha1.SLISource{
				Prometheus: sliSource,
			},
		},
	}
	return s
}

func (p *SyncServiceLevels) serviceLevelLabels() map[string]string {
	labels := map[string]string{
		picchuv1alpha1.LabelApp: p.App,
	}
	return labels
}
