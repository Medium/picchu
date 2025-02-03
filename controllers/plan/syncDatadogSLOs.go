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

func (p *SyncDatadogSLOs) datadogSLOs() (*ddog.DatadogSLOList, error) {
	// create a new list of datadog slos
	ddogSLOList := &ddog.DatadogSLOList{}
	var ddogSlOs []ddog.DatadogSLO

	for i := range p.DatadogSLOs {
		// update the DatadogSLO name so that it is the <service-name>-<slo-name>
		ddogslo_name := p.App + "-" + p.DatadogSLOs[i].Name
		ddogslo := &ddog.DatadogSLO{
			ObjectMeta: metav1.ObjectMeta{
				Name:      p.datadogSLOName(p.DatadogSLOs[i].Name),
				Namespace: p.Namespace,
				Labels:    p.Labels,
			},
			Spec: ddog.DatadogSLOSpec{
				Name:        ddogslo_name,
				Description: &p.DatadogSLOs[i].Description,
				Query: &ddog.DatadogSLOQuery{
					Numerator:   p.DatadogSLOs[i].Query.Numerator,
					Denominator: p.DatadogSLOs[i].Query.Denominator,
				},
				Type:            ddog.DatadogSLOTypeMetric,
				Timeframe:       ddog.DatadogSLOTimeFrame7d,
				TargetThreshold: resource.MustParse("99.9"),
			},
		}
		ddogslo.Spec.Tags = append(ddogslo.Spec.Tags, p.DatadogSLOs[i].Tags...)

		ddogSlOs = append(ddogSlOs, *ddogslo)
	}
	ddogSLOList.Items = ddogSlOs

	return ddogSLOList, nil
}

func (p *SyncDatadogSLOs) datadogSLOName(sloName string) string {
	// example: echo-production-example-slo-monitor3-datadogslo
	// EXCLUDE TAG FOR NOW
	// lowercase
	// at most 63 characters
	// start and end with alphanumeric
	return fmt.Sprintf("%s-%s-%s-datadogslo", p.App, p.Target, sloName)
}
