package plan

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	slov1alpha1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/plan"
	prometheus "go.medium.engineering/picchu/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
	if len(serviceLevels) > 0 {
		for i := range serviceLevels {
			if err := plan.CreateOrUpdate(ctx, log, cli, serviceLevels[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *SyncServiceLevels) serviceLevels(log logr.Logger) ([]*slov1alpha1.PrometheusServiceLevel, error) {
	var sl []*slov1alpha1.PrometheusServiceLevel
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
			serviceLevelObjective.SLI.Events = config.sliSource()
			slos = append(slos, *serviceLevelObjective)
		}
	}

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
	sl = append(sl, serviceLevel)

	return sl, nil
}

func (s *SLOConfig) sliSource() *slov1alpha1.SLIEvents {
	source := &slov1alpha1.SLIEvents{
		ErrorQuery: s.serviceLevelErrorQuery(),
		TotalQuery: s.serviceLevelTotalQuery(),
	}
	return source
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
	if s.SLO.Objective != "" {
		f, err := strconv.ParseFloat(s.SLO.Objective, 64)
		if err != nil {
			log.Error(err, "Could not parse %v to float", s.SLO.Objective)
		} else {
			objectivePercent = f
		}
	}
	slo := &slov1alpha1.SLO{
		Name:        s.Name,
		Objective:   objectivePercent,
		Description: s.SLO.Description,
		Labels:      labels,
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
