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

type SyncDatadogSLOs struct {
	App    string
	Target string
	// the namesapce is ALWAYS datadog
	Namespace   string
	Tag         string
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

func (p *SyncDatadogSLOs) datadogSLOs() (ddog.DatadogSLOList, error) {
	// create a new list of datadog slos
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

func (p *SyncDatadogSLOs) taggedDatadogSLOName(sloName string) string {
	// example: echo-production-main-123-example-slo-monitor3-datadogslo
	return fmt.Sprintf("%s-%s-%s-%s-datadogSLO", p.App, p.Target, p.Tag, sloName)
}
