package plan

import (
	"context"
	"strings"

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

// so we will get this from the app.yml
// total events: "per_minute(sum:istio.mesh.request.count.total{destination_service:tutu.tutu-production.svc.cluster.local, reporter:destination}.as_count())"
// good events: "per_minute(sum:istio.mesh.request.count.total{(response_code:2* OR response_code:3* OR response_code:4*) AND destination_service:tutu.tutu-production.svc.cluster.local AND reporter:destination}.as_count())"
// bad events; "per_minute(sum:istio.mesh.request.count.total{destination_service:tutu.tutu-production.svc.cluster.local, reporter:destination, response_code:5*}.as_count())"

func (p *SyncDatadogSLOs) datadogSLOs() (*ddog.DatadogSLOList, error) {
	// create a new list of datadog slos
	ddogSLOList := &ddog.DatadogSLOList{}
	var ddogSlOs []ddog.DatadogSLO

	for i := range p.DatadogSLOs {
		// update the DatadogSLO name so that it is the <service-name>-<target>-<tag>-<slo-name>
		ddogslo_name := p.App + "-" + p.Target + "-" + p.Tag + "-" + p.DatadogSLOs[i].Name
		ddogslo := &ddog.DatadogSLO{
			ObjectMeta: metav1.ObjectMeta{
				Name:      p.datadogSLOName(p.DatadogSLOs[i].Name),
				Namespace: p.Namespace,
				Labels:    p.Labels,
			},
			Spec: ddog.DatadogSLOSpec{
				// defaulted
				Name:        ddogslo_name,
				Description: &p.DatadogSLOs[i].Description,
				Query: &ddog.DatadogSLOQuery{
					Numerator:   p.injectTag(p.DatadogSLOs[i].Query.GoodEvents),
					Denominator: p.injectTag(p.DatadogSLOs[i].Query.TotalEvents),
				},
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

// jubilee-prod-sched-ssmps-main-20250226-235320-6c2cfd92bc

func (p *SyncDatadogSLOs) datadogSLOName(sloName string) string {
	// example: <service-name>-<condensed-target>-<condensed-slo-name>-<tag>
	// lowercase - at most 63 characters - start and end with alphanumeric

	target := ""
	if strings.Contains(p.Target, "-") {
		t := strings.LastIndex(p.Target, "-")
		first_target := p.Target[:4]
		second_target := p.Target[t+1 : t+5]
		target = first_target + "-" + second_target
	} else {
		target = p.Target[:4]
	}

	slo_name_end := strings.LastIndex(sloName, "-")
	new_slo_name := string(sloName[0])
	for i := range slo_name_end + 1 {
		if string(sloName[i]) == "-" {
			new_slo_name = new_slo_name + string(sloName[i+1])
		}
	}

	front_tag := p.App + "-" + target + "-" + new_slo_name + "-" + p.Tag
	if len(front_tag) > 63 {
		front_tag = front_tag[:63]
	}
	return front_tag
}

// inject tag into good events query with AND syntax
func (p *SyncDatadogSLOs) injectTag(query string) string {
	bracket_index := strings.Index(query, "{")
	tag_string := "destination_version:" + p.Tag + " AND "
	return query[:bracket_index+1] + tag_string + query[bracket_index+1:]
}
