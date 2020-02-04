package plan

import (
	"context"
	"fmt"

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

func (s *SyncServiceLevels) Apply(ctx context.Context, cli client.Client, scalingFactor float64, log logr.Logger) error {
	serviceLevels, err := s.serviceLevels()
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

func (s *SyncServiceLevels) serviceLevels() (*slov1alpha1.ServiceLevelList, error) {
	sll := &slov1alpha1.ServiceLevelList{}
	sl := []slov1alpha1.ServiceLevel{}
	slos := []slov1alpha1.SLO{}

	for _, slo := range s.ServiceLevelObjectives {
		if slo.Enabled {
			config := SLOConfig{
				SLO:  slo,
				App:  s.App,
				Name: sanitizeName(slo.Name),
			}
			serviceLevelObjective := config.serviceLevelObjective()
			serviceLevelObjective.ServiceLevelIndicator.SLISource.Prometheus = config.sliSource()
			slos = append(slos, *serviceLevelObjective)
		}
	}

	serviceLevel := &slov1alpha1.ServiceLevel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.App,
			Namespace: s.Namespace,
			Labels:    s.serviceLevelLabels(),
		},
		Spec: slov1alpha1.ServiceLevelSpec{
			ServiceLevelName:       s.App,
			ServiceLevelObjectives: slos,
		},
	}
	sl = append(sl, *serviceLevel)

	sll.Items = sl
	return sll, nil
}

func (s *SLOConfig) sliSource() *slov1alpha1.PrometheusSLISource {
	source := &slov1alpha1.PrometheusSLISource{
		ErrorQuery: s.serviceLevelErrorQuery(),
		TotalQuery: s.serviceLevelTotalQuery(),
	}
	return source
}

func (s *SLOConfig) serviceLevelObjective() *slov1alpha1.SLO {
	labels := make(map[string]string)
	for k, v := range s.SLO.Labels {
		labels[k] = v
	}

	if s.Tag != "" {
		labels["tag"] = s.Tag
	}

	slo := &slov1alpha1.SLO{
		Name:                         s.Name,
		AvailabilityObjectivePercent: s.SLO.ObjectivePercent,
		Description:                  s.SLO.Description,
		Disable:                      false,
		Output: slov1alpha1.Output{
			Prometheus: &slov1alpha1.PrometheusOutputSource{
				Labels: labels,
			},
		},
	}
	return slo
}

func (s *SyncServiceLevels) serviceLevelLabels() map[string]string {
	labels := map[string]string{
		picchuv1alpha1.LabelApp: s.App,
	}
	return labels
}

func (s *SLOConfig) serviceLevelTotalQuery() string {
	return fmt.Sprintf("sum(%s)", s.totalQuery())
}

func (s *SLOConfig) serviceLevelErrorQuery() string {
	return fmt.Sprintf("sum(%s)", s.errorQuery())
}
