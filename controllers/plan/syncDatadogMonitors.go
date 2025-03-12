package plan

import (
	"context"
	"strings"

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
	// update the DatadogMonitor name so that it is the <service-name>-<target>-<tag>-<slo-name>-error-budget
	ddogmonitor_name := p.App + "-" + p.Target + "-" + p.Tag + "-" + datadogslo.Name + "-error-budget"

	slo_id, err := p.getDatadogSLOIDs(datadogslo, log)
	if err != nil {
		log.Error(err, "Error Budget: Error getting Datadog SLO id", "DatadogSLO Name:", datadogslo.Name)
		return ddog.DatadogMonitor{}
	}
	if slo_id == "" {
		log.Info("Error Budget: No SLOs found", "DatadogSLO Name:", datadogslo.Name)
		return ddog.DatadogMonitor{}
	}

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

	ddogmonitor.Spec.Options.Thresholds.Critical = &datadogslo.TargetThreshold

	// taken from datadogslo
	ddogmonitor.Spec.Tags = append(ddogmonitor.Spec.Tags, datadogslo.Tags...)

	return ddogmonitor
}

func (p *SyncDatadogMonitors) burnRate(datadogslo *picchuv1alpha1.DatadogSLO, log logr.Logger) ddog.DatadogMonitor {
	// update the DatadogMonitor name so that it is the <service-name>-<target>-<tag>-<slo-name>-burn-rate
	ddogmonitor_name := p.App + "-" + p.Target + "-" + p.Tag + "-" + datadogslo.Name + "-burn-rate"

	slo_id, err := p.getDatadogSLOIDs(datadogslo, log)
	if err != nil {
		log.Error(err, "Burn Rate: Error getting Datadog SLO id", "DatadogSLO Name:", datadogslo.Name)
		return ddog.DatadogMonitor{}
	}

	if slo_id == "" {
		log.Info("Error Budget: No SLOs found", "DatadogSLO Name:", datadogslo.Name)
		return ddog.DatadogMonitor{}
	}

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

	ddogmonitor.Spec.Tags = append(ddogmonitor.Spec.Tags, datadogslo.Tags...)

	return ddogmonitor
}

func (p *SyncDatadogMonitors) getDatadogSLOIDs(datadogSLO *picchuv1alpha1.DatadogSLO, log logr.Logger) (string, error) {
	// get the SLO ID from the datadog API
	if p.App == "echo" {
		ctx := datadog.NewDefaultContext(context.Background())

		configuration := datadog.NewConfiguration()
		apiClient := datadog.NewAPIClient(configuration)
		api := datadogV1.NewServiceLevelObjectivesApi(apiClient)
		ddogslo_name := "\"" + p.App + "-" + p.Target + "-" + p.Tag + "-" + datadogSLO.Name + "\""
		resp, r, err := api.SearchSLO(ctx, *datadogV1.NewSearchSLOOptionalParameters().WithQuery(ddogslo_name))

		if err != nil {
			log.Error(err, "Error when calling `ServiceLevelObjectivesApi.NewSearchSLOOptionalParameters`:", "response", r)
			return "", err
		}

		if len(resp.Data.Attributes.Slos) == 0 || resp.Data.Attributes.Slos == nil {
			log.Info("No SLOs found when calling `ServiceLevelObjectivesApi.NewSearchSLOOptionalParameters` for service", "App", p.App)
			return "", nil
		}

		return *resp.Data.Attributes.Slos[0].Data.Id, nil
	}
	// if app is not echo, return empty string for now
	return "", nil
}

func (p *SyncDatadogMonitors) datadogMonitorName(sloName string, monitor_type string) string {
	// example: <service-name>-<condensed-target>-<condensed-slo-name>-<tag>-<monitor-type>
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

	front_tag := p.App + "-" + target + "-" + new_slo_name + "-" + p.Tag + "-" + monitor_type
	if len(front_tag) > 63 {
		front_tag = front_tag[:63]
	}
	return front_tag
}
