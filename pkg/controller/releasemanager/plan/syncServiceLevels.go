package plan

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	slov1alpha1 "github.com/Medium/service-level-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/go-logr/logr"
	slov1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/plan"
	prometheus "go.medium.engineering/picchu/pkg/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SyncServiceLevels struct {
	App                         string
	Target                      string
	Namespace                   string
	Labels                      map[string]string
	ServiceLevelObjectiveLabels picchuv1alpha1.ServiceLevelObjectiveLabels
	ServiceLevelObjectives      []*picchuv1alpha1.ServiceLevelObjective
	SlothServiceLevelObjectives []*picchuv1alpha1.SlothServiceLevelObjective
}

func (p *SyncServiceLevels) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	serviceLevels, slothServiceLevels, err := p.serviceLevels(log)
	if err != nil {
		return err
	}
	// if legacy or sloth
	if len(serviceLevels) > 0 {
		for i := range serviceLevels {
			if err := plan.CreateOrUpdate(ctx, log, cli, serviceLevels[i]); err != nil {
				return err
			}
		}
	} else if len(slothServiceLevels) > 0 {
		for i := range slothServiceLevels {
			if err := plan.CreateOrUpdate(ctx, log, cli, slothServiceLevels[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *SyncServiceLevels) serviceLevels(log logr.Logger) ([]*slov1alpha1.ServiceLevel, []*slov1.PrometheusServiceLevel, error) {
	var sl []*slov1alpha1.ServiceLevel
	var slos []slov1alpha1.SLO
	var slothsl []*slov1.PrometheusServiceLevel
	var slothslos []slov1.SLO

	if p.ServiceLevelObjectives != nil {
		for i := range p.ServiceLevelObjectives {
			if p.ServiceLevelObjectives[i].Enabled {
				config := SLOConfig{
					SLO:    p.ServiceLevelObjectives[i],
					App:    p.App,
					Name:   sanitizeName(p.ServiceLevelObjectives[i].Name),
					Labels: p.ServiceLevelObjectiveLabels,
				}
				serviceLevelObjective := config.serviceLevelObjective(log)
				serviceLevelObjective.ServiceLevelIndicator.SLISource.Prometheus = config.sliSource()
				slos = append(slos, *serviceLevelObjective)
			}
		}

		serviceLevel := &slov1alpha1.ServiceLevel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      p.serviceLevelName(),
				Namespace: p.Namespace,
				Labels:    p.Labels,
			},
			Spec: slov1alpha1.ServiceLevelSpec{
				ServiceLevelName:       p.App,
				ServiceLevelObjectives: slos,
			},
		}
		sl = append(sl, serviceLevel)
	} else if p.SlothServiceLevelObjectives != nil {
		for i := range p.SlothServiceLevelObjectives {
			if p.SlothServiceLevelObjectives[i].Enabled {
				config := SlothSLOConfig{
					SLO:    p.SlothServiceLevelObjectives[i],
					App:    p.App,
					Name:   sanitizeName(p.SlothServiceLevelObjectives[i].Name),
					Labels: p.ServiceLevelObjectiveLabels,
				}
				serviceLevelObjective := config.serviceLevelObjective(log)
				serviceLevelObjective.SLI.Events = config.sliSource()
				slothslos = append(slothslos, *serviceLevelObjective)
			}
		}

		serviceLevel := &slov1.PrometheusServiceLevel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      p.serviceLevelName(),
				Namespace: p.Namespace,
				Labels:    p.Labels,
			},
			Spec: slov1.PrometheusServiceLevelSpec{
				Service: p.App,
				SLOs:    slothslos,
			},
		}
		slothsl = append(slothsl, serviceLevel)
	}

	return sl, slothsl, nil
}

func (s *SLOConfig) sliSource() *slov1alpha1.PrometheusSLISource {
	source := &slov1alpha1.PrometheusSLISource{
		ErrorQuery: s.serviceLevelErrorQuery(),
		TotalQuery: s.serviceLevelTotalQuery(),
	}
	return source
}

func (s *SlothSLOConfig) sliSource() *slov1.SLIEvents {
	source := &slov1.SLIEvents{
		ErrorQuery: s.serviceLevelErrorQuery(),
		TotalQuery: s.serviceLevelTotalQuery(),
	}
	return source
}

func (s *SlothSLOConfig) serviceLevelObjective(log logr.Logger) *slov1.SLO {
	labels := make(map[string]string)
	for k, v := range s.Labels.ServiceLevelLabels {
		labels[k] = v
	}

	for k, v := range s.SLO.ServiceLevelObjectiveLabels.ServiceLevelLabels {
		labels[k] = v
	}

	if s.Tag != "" {
		labels[prometheus.TagLabel] = s.Tag
	}

	var objectivePercent float64
	if s.SLO.Objective != "" {
		f, err := strconv.ParseFloat(s.SLO.Objective, 64)
		if err != nil {
			log.Error(err, "Could not parse %v to float", s.SLO.Objective)
		} else {
			objectivePercent = f
		}
	}
	slo := &slov1.SLO{
		Name:        s.Name,
		Objective:   objectivePercent,
		Description: s.SLO.Description,
		Labels:      labels,
	}
	return slo
}

func (s *SLOConfig) serviceLevelObjective(log logr.Logger) *slov1alpha1.SLO {
	labels := make(map[string]string)
	for k, v := range s.Labels.ServiceLevelLabels {
		labels[k] = v
	}

	for k, v := range s.SLO.ServiceLevelObjectiveLabels.ServiceLevelLabels {
		labels[k] = v
	}

	if s.Tag != "" {
		labels[prometheus.TagLabel] = s.Tag
	}

	var objectivePercent float64
	if s.SLO.ObjectivePercentString != "" {
		f, err := strconv.ParseFloat(s.SLO.ObjectivePercentString, 64)
		if err != nil {
			log.Error(err, "Could not parse %v to float", s.SLO.ObjectivePercentString)
		} else {
			objectivePercent = f
		}
	}
	slo := &slov1alpha1.SLO{
		Name:                         s.Name,
		AvailabilityObjectivePercent: objectivePercent,
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

func (p *SyncServiceLevels) serviceLevelName() string {
	return fmt.Sprintf("%s-%s-servicelevels", strings.ToLower(p.App), strings.ToLower(p.Target))
}

func (s *SLOConfig) serviceLevelTotalQuery() string {
	return fmt.Sprintf("sum(%s)", s.totalQuery())
}

func (s *SLOConfig) serviceLevelErrorQuery() string {
	return fmt.Sprintf("sum(%s)", s.errorQuery())
}

func (s *SlothSLOConfig) serviceLevelTotalQuery() string {
	return fmt.Sprintf("sum(%s)", s.totalQuery())
}

func (s *SlothSLOConfig) serviceLevelErrorQuery() string {
	return fmt.Sprintf("sum(%s)", s.errorQuery())
}
