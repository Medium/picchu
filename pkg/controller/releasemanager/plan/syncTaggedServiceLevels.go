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

type SyncTaggedServiceLevels struct {
	App                    string
	Namespace              string
	Tag                    string
	ServiceLevelObjectives []*picchuv1alpha1.ServiceLevelObjective
}

func (p *SyncTaggedServiceLevels) Apply(ctx context.Context, cli client.Client, scalingFactor float64, log logr.Logger) error {
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

func (p *SyncTaggedServiceLevels) serviceLevels() (*slov1alpha1.ServiceLevelList, error) {
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

			labels["tag"] = p.Tag
			errorQuery := taggedServiceLevelQuery(s, errorQueryName(s, p.App, name), p.App, name, p.Tag)
			totalQuery := taggedServiceLevelQuery(s, totalQueryName(s, p.App, name), p.App, name, p.Tag)
			slo := serviceLevelObjective(s, name, errorQuery, totalQuery, labels)
			slos = append(slos, *slo)
		}
	}

	serviceLevel := &slov1alpha1.ServiceLevel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.taggedServiceLevelName(),
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

func (p *SyncTaggedServiceLevels) serviceLevelLabels() map[string]string {
	labels := map[string]string{
		picchuv1alpha1.LabelApp: p.App,
		picchuv1alpha1.LabelTag: p.Tag,
	}
	return labels
}
func (p *SyncTaggedServiceLevels) taggedServiceLevelName() string {
	return fmt.Sprintf("%s-%s", p.App, p.Tag)
}

func taggedServiceLevelQuery(slo *picchuv1alpha1.ServiceLevelObjective, query, app, name, tag string) string {
	return fmt.Sprintf("%s{%s=\"%s\"}", query, slo.ServiceLevelIndicator.TagKey, tag)
}
