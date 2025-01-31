package plan

import (
	"context"
	"fmt"

	ddog "github.com/DataDog/datadog-operator/api/datadoghq/v1alpha1"
	"github.com/go-logr/logr"
	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/plan"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SyncDatadogMonitors struct {
	App    string
	Target string
	// the namesapce is ALWAYS datadog
	Namespace       string
	Tag             string
	Labels          map[string]string
	DatadogMonitors []*picchuv1alpha1.DatadogMonitor
}

// only issue - we need the SLO hash from the cluster
// idk if i can get that from the ddog api?YES
// "https://api.datadoghq.com/api/v1/slo" can list and pull the id?
// how do i ensure that the SLO is created before the monitor?

func (p *SyncDatadogMonitors) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	datadogMonitors, err := p.datadogMonitors()
	if err != nil {
		return err
	}
	if len(datadogMonitors.Items) > 0 {
		for i := range datadogMonitors.Items {
			if err := plan.CreateOrUpdate(ctx, log, cli, &datadogMonitors.Items[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *SyncDatadogMonitors) datadogMonitors() (*ddog.DatadogMonitorList, error) {
	// create a new list of datadog slos
	datadogMonitorList := &ddog.DatadogMonitorList{}
	var ddogMonitors []ddog.DatadogMonitor

	for i := range p.DatadogMonitors {
		// update the DatadogSLO name so that it is the <service-name>-<slo-name>
		ddogmonitor_name := p.App + "-" + p.DatadogMonitors[i].Name
		ddogslo := &ddog.DatadogMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      p.datadogMonitorName(p.DatadogMonitors[i].Name),
				Namespace: p.Namespace,
				Labels:    p.Labels,
			},
			Spec: ddog.DatadogMonitorSpec{
				Name:     ddogmonitor_name,
				Message:  p.DatadogMonitors[i].Message,
				Priority: p.DatadogMonitors[i].Priority,
				Query:    p.DatadogMonitors[i].Query,
				// RestrictedRoles
				// Tags
				// Type
				// Options
				// ControllerOptions
			},
		}
		ddogslo.Spec.Tags = append(ddogslo.Spec.Tags, p.DatadogMonitors[i].Tags...)

		ddogMonitors = append(ddogMonitors, *ddogslo)
	}
	datadogMonitorList.Items = ddogMonitors

	return datadogMonitorList, nil
}

func (p *SyncDatadogMonitors) datadogMonitorName(sloName string) string {
	// example: echo-production-example-slo-monitor3-datadogslo
	// EXCLUDE TAG FOR NOW
	// lowercase
	// at most 63 characters
	// start and end with alphanumeric
	return fmt.Sprintf("%s-%s-%s-datadogmonitor", p.App, p.Target, sloName)
}
