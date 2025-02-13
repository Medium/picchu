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

	// for each ddog slo we generate two datadog monitors, one for error budeget and the other burn rate
	for i := range p.DatadogSLOs {
		// call error budget
		errorbudget_ddogmonitor := p.errorBudget(p.DatadogSLOs[i], log)
		ddogMonitors = append(ddogMonitors, errorbudget_ddogmonitor)
		// call burn rate
		burnrate_ddogmonitor := p.burnRate(p.DatadogSLOs[i], log)
		ddogMonitors = append(ddogMonitors, burnrate_ddogmonitor)

	}
	datadogMonitorList.Items = ddogMonitors

	return datadogMonitorList, nil
}

// Error Budget Monitor
func (p *SyncDatadogMonitors) errorBudget(datadogslo *picchuv1alpha1.DatadogSLO, log logr.Logger) ddog.DatadogMonitor {
	ddogmonitor_name := p.App + "-" + "error-budget-" + datadogslo.Name

	slo_id := p.getDatadogSLOIDs(datadogslo, log)
	// error if slo not found

	query := "error_budget(\"" + slo_id + "\").over(\"" + datadogslo.Timeframe + "\") > " + datadogslo.TargetThreshold
	message := "The " + datadogslo.Name + " error budget is over expected @slack-eng-watch-alerts-testing"
	escalation_message := "ESCALATED: The " + datadogslo.Name + " error budget is over expected @slack-eng-watch-alerts-testing"
	renotify := []datadogV1.MonitorRenotifyStatusType{datadogV1.MONITORRENOTIFYSTATUSTYPE_ALERT, datadogV1.MONITORRENOTIFYSTATUSTYPE_NO_DATA}

	five_min := int64(5)
	five_min_sting := "5"
	options_true := true

	ddogmonitor := ddog.DatadogMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.datadogMonitorName(datadogslo.Name, "error-budget"),
			Namespace: p.Namespace,
			Labels:    p.Labels,
		},
		Spec: ddog.DatadogMonitorSpec{
			// defaulted
			Name: ddogmonitor_name,
			// defaulted
			Message: message,
			// defaulted
			Query: query,
			// defaulted slo_type
			Type: ddog.DatadogMonitorTypeSLO,
			Options: ddog.DatadogMonitorOptions{
				EnableLogsSample:       &options_true,
				EscalationMessage:      &escalation_message,
				IncludeTags:            &options_true,
				NotificationPresetName: "show_all",
				NotifyNoData:           &options_true,
				RenotifyStatuses:       renotify,
				RenotifyInterval:       &five_min,
				// maybe false?
				RequireFullWindow: &options_true,
				EvaluationDelay:   &five_min,
				NoDataTimeframe:   &five_min,

				Thresholds: &ddog.DatadogMonitorOptionsThresholds{
					Warning:  &five_min_sting,
					Critical: &five_min_sting,
				},
			},
		},
	}

	// if the ddog monitor is enabled in app.yml, use the threshold ddog monitor values
	if datadogslo.DatadogMonitor.Enabled {
		// use the set ddog monitor values
		ddogmonitor.Spec.Options.Thresholds.Critical = datadogslo.DatadogMonitor.Options.Thresholds.Critical
		ddogmonitor.Spec.Options.Thresholds.CriticalRecovery = datadogslo.DatadogMonitor.Options.Thresholds.CriticalRecovery
		// ddogmonitor.Spec.Options.Thresholds.OK = datadogslo.DatadogMonitor.Options.Thresholds.OK
		// ddogmonitor.Spec.Options.Thresholds.Unknown = datadogslo.DatadogMonitor.Options.Thresholds.Unknown
	} else {
		ddogmonitor.Spec.Options.Thresholds.Critical = &datadogslo.TargetThreshold
	}

	// taken from datadogslo
	ddogmonitor.Spec.Tags = append(ddogmonitor.Spec.Tags, datadogslo.Tags...)

	return ddogmonitor
}

func (p *SyncDatadogMonitors) burnRate(datadogslo *picchuv1alpha1.DatadogSLO, log logr.Logger) ddog.DatadogMonitor {
	ddogmonitor_name := p.App + "-" + "burn-rate-" + datadogslo.Name

	slo_id := p.getDatadogSLOIDs(datadogslo, log)
	// error if slo not found

	// how are we defining log and short window
	// going to default to this slo for now
	// burn_rate("slo_id").over("time_window").long_window("1h").short_window("5m") > 14.4

	query := "burn_rate(\"" + slo_id + "\").over(\"" + datadogslo.Timeframe + "\").long_window(\"1h\").short_window(\"5m\") > 14.4"
	message := "The " + datadogslo.Name + " burn rate is over expected @slack-eng-watch-alerts-testing"
	escalation_message := "ESCALATED: The " + datadogslo.Name + " burn rate is over expected @slack-eng-watch-alerts-testing"
	renotify := []datadogV1.MonitorRenotifyStatusType{datadogV1.MONITORRENOTIFYSTATUSTYPE_ALERT, datadogV1.MONITORRENOTIFYSTATUSTYPE_NO_DATA}

	five_min := int64(5)
	critical_threshold := "14.4"
	options_true := true

	ddogmonitor := ddog.DatadogMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.datadogMonitorName(datadogslo.Name, "burn-rate"),
			Namespace: p.Namespace,
			Labels:    p.Labels,
		},
		Spec: ddog.DatadogMonitorSpec{
			// defaulted
			Name: ddogmonitor_name,
			// defaulted
			Message: message,
			// defaulted
			Query: query,
			// defaulted slo_type
			Type: ddog.DatadogMonitorTypeSLO,
			Options: ddog.DatadogMonitorOptions{
				EnableLogsSample:       &options_true,
				EscalationMessage:      &escalation_message,
				IncludeTags:            &options_true,
				NotificationPresetName: "show_all",
				NotifyNoData:           &options_true,
				RenotifyStatuses:       renotify,
				RenotifyInterval:       &five_min,
				// maybe false?
				RequireFullWindow: &options_true,
				EvaluationDelay:   &five_min,
				NoDataTimeframe:   &five_min,

				Thresholds: &ddog.DatadogMonitorOptionsThresholds{
					Critical: &critical_threshold,
				},
			},
		},
	}

	// // if the ddog monitor is enabled in app.yml, use the threshold ddog monitor values
	// if datadogslo.DatadogMonitor.Enabled {
	// 	// use the set ddog monitor values
	// 	ddogmonitor.Spec.Options.Thresholds.Critical = datadogslo.DatadogMonitor.Options.Thresholds.Critical
	// 	ddogmonitor.Spec.Options.Thresholds.CriticalRecovery = datadogslo.DatadogMonitor.Options.Thresholds.CriticalRecovery
	// 	// ddogmonitor.Spec.Options.Thresholds.OK = datadogslo.DatadogMonitor.Options.Thresholds.OK
	// 	// ddogmonitor.Spec.Options.Thresholds.Unknown = datadogslo.DatadogMonitor.Options.Thresholds.Unknown
	// } else {
	// 	ddogmonitor.Spec.Options.Thresholds.Critical = &datadogslo.TargetThreshold
	// }

	ddogmonitor.Spec.Tags = append(ddogmonitor.Spec.Tags, datadogslo.Tags...)

	return ddogmonitor
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

func (p *SyncDatadogMonitors) datadogMonitorName(sloName string, monitor_type string) string {
	// example: echo-production-example-slo-monitor3-datadogslo
	// lowercase
	// at most 63 characters
	// start and end with alphanumeric
	return fmt.Sprintf("%s-%s-%s-%s-datadogmonitor", p.App, monitor_type, p.Target, sloName)
}
