package plan

import (
	"context"
	"fmt"

	slov1alpha1 "github.com/Medium/service-level-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/go-logr/logr"
	slov1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/plan"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SyncTaggedServiceLevels struct {
	App                         string
	Target                      string
	Namespace                   string
	Tag                         string
	Labels                      map[string]string
	ServiceLevelObjectiveLabels picchuv1alpha1.ServiceLevelObjectiveLabels
	ServiceLevelObjectives      []*picchuv1alpha1.ServiceLevelObjective
}

type SyncTaggedServiceLevelsSloth struct {
	App                         string
	Target                      string
	Namespace                   string
	Tag                         string
	Labels                      map[string]string
	ServiceLevelObjectiveLabels picchuv1alpha1.ServiceLevelObjectiveLabels
	SlothServiceLevelObjectives []*picchuv1alpha1.SlothServiceLevelObjective
}

func (p *SyncTaggedServiceLevels) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
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

func (p *SyncTaggedServiceLevelsSloth) ApplySloth(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	serviceLevels, err := p.serviceLevelsSloth(log)
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

func (p *SyncTaggedServiceLevels) serviceLevels(log logr.Logger) (*slov1alpha1.ServiceLevelList, error) {
	sll := &slov1alpha1.ServiceLevelList{}
	var sl []slov1alpha1.ServiceLevel
	var slos []slov1alpha1.SLO

	for i := range p.ServiceLevelObjectives {
		if p.ServiceLevelObjectives[i].Enabled {
			config := SLOConfig{
				SLO:    p.ServiceLevelObjectives[i],
				App:    p.App,
				Name:   sanitizeName(p.ServiceLevelObjectives[i].Name),
				Tag:    p.Tag,
				Labels: p.ServiceLevelObjectiveLabels,
			}
			serviceLevelObjective := config.serviceLevelObjective(log)
			serviceLevelObjective.ServiceLevelIndicator.SLISource.Prometheus = config.taggedSLISource()

			slos = append(slos, *serviceLevelObjective)
		}
	}

	if len(slos) > 0 {
		serviceLevel := &slov1alpha1.ServiceLevel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      p.taggedServiceLevelName(),
				Namespace: p.Namespace,
				Labels:    p.Labels,
			},
			Spec: slov1alpha1.ServiceLevelSpec{
				ServiceLevelName:       p.App,
				ServiceLevelObjectives: slos,
			},
		}
		sl = append(sl, *serviceLevel)
	}

	sll.Items = sl
	return sll, nil
}

func (p *SyncTaggedServiceLevelsSloth) serviceLevelsSloth(log logr.Logger) (*slov1.PrometheusServiceLevelList, error) {
	sll := &slov1.PrometheusServiceLevelList{}
	var sl []slov1.PrometheusServiceLevel
	var slos []slov1.SLO

	for i := range p.SlothServiceLevelObjectives {
		if p.SlothServiceLevelObjectives[i].Enabled {
			config := SlothSLOConfig{
				SLO:    p.SlothServiceLevelObjectives[i],
				App:    p.App,
				Name:   sanitizeName(p.SlothServiceLevelObjectives[i].Name),
				Tag:    p.Tag,
				Labels: p.ServiceLevelObjectiveLabels,
			}
			SLO := config.serviceLevelObjectiveSloth(log)
			SLO.SLI.Events = config.taggedSLIEvents()

			slos = append(slos, *SLO)
		}
	}

	if len(slos) > 0 {
		serviceLevel := &slov1.PrometheusServiceLevel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      p.taggedServiceLevelName(),
				Namespace: p.Namespace,
				Labels:    p.Labels,
			},
			Spec: slov1.PrometheusServiceLevelSpec{
				Service: p.App,
				SLOs:    slos,
			},
		}
		sl = append(sl, *serviceLevel)
	}

	sll.Items = sl
	return sll, nil
}

func (s *SLOConfig) taggedSLISource() *slov1alpha1.PrometheusSLISource {
	source := &slov1alpha1.PrometheusSLISource{
		ErrorQuery: s.serviceLevelTaggedErrorQuery(),
		TotalQuery: s.serviceLevelTaggedTotalQuery(),
	}
	return source
}

func (s *SlothSLOConfig) taggedSLIEvents() *slov1.SLIEvents {
	source := &slov1.SLIEvents{
		ErrorQuery: s.serviceLevelTaggedErrorQuery(),
		TotalQuery: s.serviceLevelTaggedTotalQuery(),
	}
	return source
}

func (p *SyncTaggedServiceLevels) taggedServiceLevelName() string {
	return fmt.Sprintf("%s-%s-%s-servicelevels", p.App, p.Target, p.Tag)
}

func (p *SyncTaggedServiceLevelsSloth) taggedServiceLevelName() string {
	return fmt.Sprintf("%s-%s-%s-servicelevels", p.App, p.Target, p.Tag)
}

func (s *SLOConfig) serviceLevelTaggedTotalQuery() string {
	return fmt.Sprintf("sum(%s{%s=\"%s\"})", s.totalQuery(), s.SLO.ServiceLevelIndicator.TagKey, s.Tag)
}

func (s *SLOConfig) serviceLevelTaggedErrorQuery() string {
	return fmt.Sprintf("sum(%s{%s=\"%s\"})", s.errorQuery(), s.SLO.ServiceLevelIndicator.TagKey, s.Tag)
}

func (s *SlothSLOConfig) serviceLevelTaggedTotalQuery() string {
	return fmt.Sprintf("sum(%s{%s=\"%s\"})", s.totalQuery(), s.SLO.ServiceLevelIndicator.TagKey, s.Tag)
}

func (s *SlothSLOConfig) serviceLevelTaggedErrorQuery() string {
	return fmt.Sprintf("sum(%s{%s=\"%s\"})", s.errorQuery(), s.SLO.ServiceLevelIndicator.TagKey, s.Tag)
}
