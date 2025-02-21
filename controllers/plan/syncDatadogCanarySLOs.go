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

type SyncDatadogCanarySLOs struct {
	App    string
	Target string
	// the namesapce is ALWAYS datadog
	Namespace   string
	Tag         string
	Labels      map[string]string
	DatadogSLOs []*picchuv1alpha1.DatadogSLO
}

func (p *SyncDatadogCanarySLOs) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	datadogCanarySLOs, err := p.datadogCanarySLOs()
	if err != nil {
		return err
	}
	if len(datadogCanarySLOs.Items) > 0 {
		for i := range datadogCanarySLOs.Items {
			if err := plan.CreateOrUpdate(ctx, log, cli, &datadogCanarySLOs.Items[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *SyncDatadogCanarySLOs) datadogCanarySLOs() (*ddog.DatadogSLOList, error) {
	// create a new list of datadog slos
	ddogSLOList := &ddog.DatadogSLOList{}
	var ddogSlOs []ddog.DatadogSLO

	for i := range p.DatadogSLOs {
		// update the DatadogSLO name so that it is the <service-name>-<slo-name>
		ddogslo_name := p.App + "-" + p.DatadogSLOs[i].Name
		ddogslo := &ddog.DatadogSLO{
			ObjectMeta: metav1.ObjectMeta{
				Name:      p.datadogCanarySLOName(p.DatadogSLOs[i].Name),
				Namespace: p.Namespace,
				Labels:    p.Labels,
			},
			Spec: ddog.DatadogSLOSpec{
				// defaulted
				Name:        ddogslo_name,
				Description: &p.DatadogSLOs[i].Description,
				// QUERY is going to be different

				// Query: &ddog.DatadogSLOQuery{
				// 	Numerator:   p.DatadogSLOs[i].Query.Numerator,
				// 	Denominator: p.DatadogSLOs[i].Query.Denominator,
				// },
				// defaulted
				Type: ddog.DatadogSLOTypeMetric,
				// defaulted 30d for now
				Timeframe:       ddog.DatadogSLOTimeFrame30d,
				TargetThreshold: resource.MustParse(p.DatadogSLOs[i].TargetThreshold),
			},
		}

		target := "target:" + p.Target
		ddogslo.Spec.Tags = append(ddogslo.Spec.Tags, p.DatadogSLOs[i].Tags...)
		// add target tag
		ddogslo.Spec.Tags = append(ddogslo.Spec.Tags, target)

		ddogSlOs = append(ddogSlOs, *ddogslo)
	}
	ddogSLOList.Items = ddogSlOs

	return ddogSLOList, nil
}

func (p *SyncDatadogCanarySLOs) datadogCanarySLOName(sloName string) string {
	// example: echo-example-slo-<tag>-canary-datadogslo
	// do we need the target?
	// lowercase - at most 63 characters - start and end with alphanumeric
	return fmt.Sprintf("%s-%s-%s-canary-datadogslo", p.App, sloName, p.Tag)
}
