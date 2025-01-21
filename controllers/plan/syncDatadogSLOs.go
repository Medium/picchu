package plan

import (
	"context"
	"fmt"

	ddog "github.com/DataDog/datadog-operator/api/datadoghq/v1alpha1"
	"github.com/go-logr/logr"
	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/plan"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// we need to apply each individual datadogSLO object
type SyncDatadogSLOs struct {
	App string
	// we need the target to know what the query should be?
	Target string
	// the namesapce is ALWAYS datadog
	Namespace string
	// do we need the tag? prob
	Tag string
	// idk labels
	Labels      map[string]string
	DatadogSLOs []*picchuv1alpha1.DatadogSLO
}

func (p *SyncDatadogSLOs) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	datadogSLOs, err := p.datadogSLOs()
	log.Info("datadogSLOs", "datadogSLOs", datadogSLOs)
	if err != nil {
		return err
	}
	if len(datadogSLOs.Items) > 0 {
		for i := range datadogSLOs.Items {
			if err := plan.CreateOrUpdate(ctx, log, cli, &datadogSLOs.Items[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

// returns list of ddog slos
func (p *SyncDatadogSLOs) datadogSLOs() (ddog.DatadogSLOList, error) {
	// names, err := p.parseMetricNames()
	// if err != nil {
	// 	return nil, err
	// }
	// metricNamesRegex := strings.Join(names, "|")

	// idk whats up with the pointers bro fix this
	ddogSLOList := ddog.DatadogSLOList{}
	var ddogSlOs []ddog.DatadogSLO

	for i := range p.DatadogSLOs {
		ddogslo := p.ddogSLO(p.DatadogSLOs[i])
		ddogSlOs = append(ddogSlOs, *ddogslo)
	}
	ddogSLOList.Items = ddogSlOs

	return ddogSLOList, nil
}

// returns individual ddog slo object
func (p *SyncDatadogSLOs) ddogSLO(ddogSLO *picchuv1alpha1.DatadogSLO) *ddog.DatadogSLO {
	newDdogSLO := &ddog.DatadogSLO{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.taggedDatadogSLOName(ddogSLO.Name),
			Namespace: p.Namespace,
			Labels:    p.Labels,
		},
		Spec: ddog.DatadogSLOSpec{
			Name:        ddogSLO.Name,
			Description: &ddogSLO.Description,
			Query: &ddog.DatadogSLOQuery{
				Numerator:   ddogSLO.Query.Numerator,
				Denominator: ddogSLO.Query.Denominator,
			},
			Type:            ddog.DatadogSLOTypeMetric,
			Timeframe:       ddog.DatadogSLOTimeFrame7d,
			TargetThreshold: resource.MustParse("99.9"),
		},
	}
	newDdogSLO.Spec.Tags = append(newDdogSLO.Spec.Tags, ddogSLO.Tags...)

	// ignore canary for now
	return newDdogSLO
}

// YO this needs to be the name of the slo not the app
func (p *SyncDatadogSLOs) taggedDatadogSLOName(sloName string) string {
	return fmt.Sprintf("%s-%s-%s-%s-datadogSLO", p.App, p.Target, p.Tag, sloName)
}
