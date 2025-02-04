package plan

import (
	"context"
	"fmt"
	"os"

	ddog "github.com/DataDog/datadog-operator/api/datadoghq/v1alpha1"
	"github.com/go-logr/logr"
	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/plan"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
)

type SyncDatadogMonitors struct {
	App    string
	Target string
	// the namesapce is ALWAYS datadog
	Namespace string
	Tag       string
	Labels    map[string]string
	// Use DatadogSLOs to define each monitor
	DatadogSLOs []*picchuv1alpha1.DatadogSLO
}

func (p *SyncDatadogMonitors) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	datadogMonitors, err := p.datadogMonitors(log)

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

func (p *SyncDatadogMonitors) datadogMonitors(log logr.Logger) (*ddog.DatadogMonitorList, error) {
	datadogMonitorList := &ddog.DatadogMonitorList{}
	var ddogMonitors []ddog.DatadogMonitor

	for i := range p.DatadogSLOs {
		ddogmonitor_name := p.App + "-" + p.DatadogSLOs[i].Name

		slo_id := p.getDatadogSLOIDs(p.DatadogSLOs[i], log)
		// error if slo not found

		query := "error_budget(\"" + slo_id + "\").over(\"7d\") > 10"
		// send to testing?
		message := "@slack-eng-watch-alerts-testing"
		// for right now, if no id, create the monitor anyway

		warning := "10"
		delay := int64(300)
		nodata := int64(30)
		renotif := int64(1440)
		options_true := true
		options_false := true
		ddogmonitor := &ddog.DatadogMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      p.datadogMonitorName(p.DatadogSLOs[i].Name),
				Namespace: p.Namespace,
				Labels:    p.Labels,
			},
			Spec: ddog.DatadogMonitorSpec{
				Name:    ddogmonitor_name,
				Message: message,
				// lowest priority for now
				Priority: 5,
				Query:    query,
				Type:     ddog.DatadogMonitorTypeSLO,
				// RestrictedRoles
				Options: ddog.DatadogMonitorOptions{
					EvaluationDelay:        &delay,
					IncludeTags:            &options_true,
					Locked:                 &options_false,
					NewGroupDelay:          &delay,
					NotificationPresetName: "show_all",
					NoDataTimeframe:        &nodata,
					RenotifyInterval:       &renotif,
					Thresholds: &ddog.DatadogMonitorOptionsThresholds{
						Warning: &warning,
					},
				},
				// ControllerOptions
			},
		}
		ddogmonitor.Spec.Tags = append(ddogmonitor.Spec.Tags, p.DatadogSLOs[i].Tags...)

		ddogMonitors = append(ddogMonitors, *ddogmonitor)
	}
	datadogMonitorList.Items = ddogMonitors

	return datadogMonitorList, nil
}

func (p *SyncDatadogMonitors) getDatadogSLOIDs(datadogSLO *picchuv1alpha1.DatadogSLO, log logr.Logger) string {
	// get the SLO ID from the datadog API
	if p.App == "echo" {
		ctx := datadog.NewDefaultContext(context.Background())

		configuration := datadog.NewConfiguration()
		apiClient := datadog.NewAPIClient(configuration)
		api := datadogV1.NewServiceLevelObjectivesApi(apiClient)
		resp, r, err := api.ListSLOs(ctx, *datadogV1.NewListSLOsOptionalParameters())

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error when calling `ServiceLevelObjectivesApi.ListSLOs`: %v\n", err)
			fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
		}

		for _, slo := range resp.Data {
			ddogslo_name := p.App + "-" + datadogSLO.Name
			if slo.Name == ddogslo_name {
				return *slo.Id
			}
		}
	}
	// if app is not echo, return empty string for now
	return ""
}

func (p *SyncDatadogMonitors) datadogMonitorName(sloName string) string {
	// example: echo-production-example-slo-monitor3-datadogslo
	// lowercase
	// at most 63 characters
	// start and end with alphanumeric
	return fmt.Sprintf("%s-%s-%s-datadogmonitor", p.App, p.Target, sloName)
}
