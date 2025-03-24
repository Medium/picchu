package plan

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	ddog "github.com/DataDog/datadog-operator/api/datadoghq/v1alpha1"
	"github.com/go-logr/logr"
	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/plan"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	MonitorTypeCanary = "canary"
)

type SyncDatadogCanaryMonitors struct {
	App    string
	Target string
	// the namesapce is ALWAYS datadog
	Namespace string
	Tag       string
	Labels    map[string]string
	// Use DatadogSLOs to define each monitor
	DatadogSLOs []*picchuv1alpha1.DatadogSLO
}

func (p *SyncDatadogCanaryMonitors) Apply(ctx context.Context, cli client.Client, cluster *picchuv1alpha1.Cluster, log logr.Logger) error {
	datadogMonitors, err := p.datadogCanaryMonitors(log)

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

func (p *SyncDatadogCanaryMonitors) datadogCanaryMonitors(log logr.Logger) (*ddog.DatadogMonitorList, error) {
	datadogMonitorList := &ddog.DatadogMonitorList{}
	var ddogCanaryMonitors []ddog.DatadogMonitor

	// for each ddog slo we generate a datadog canary monitors - type is metric
	for i := range p.DatadogSLOs {
		// canary monitor
		if p.DatadogSLOs[i].Canary.Enabled {
			ddogCanaryMonitor := p.canaryMonitor(p.DatadogSLOs[i], log)
			ddogCanaryMonitors = append(ddogCanaryMonitors, ddogCanaryMonitor)
		}
	}
	datadogMonitorList.Items = ddogCanaryMonitors

	return datadogMonitorList, nil
}

func (p *SyncDatadogCanaryMonitors) canaryMonitor(datadogslo *picchuv1alpha1.DatadogSLO, log logr.Logger) ddog.DatadogMonitor {
	// update the DatadogMonitor name so that it is the <service-name>-<target>-<tag>-<slo-name>-error-budget
	ddogmonitor_name := p.App + "-" + p.Target + "-" + p.Tag + "-" + datadogslo.Name + "-canary"

	message := "The " + datadogslo.Name + " canary query is firing @slack-eng-watch-alerts-testing"
	escalation_message := "ESCALATED: The " + datadogslo.Name + " canary query is firing @slack-eng-watch-alerts-testing"
	renotify := []datadogV1.MonitorRenotifyStatusType{datadogV1.MONITORRENOTIFYSTATUSTYPE_ALERT, datadogV1.MONITORRENOTIFYSTATUSTYPE_NO_DATA}

	five_min := int64(5)
	options_true := true
	canary_threshold := "0.0"

	allowancePercent := p.formatAllowancePercent(datadogslo, log)
	query_first := "((" + p.injectTag(datadogslo.Query.BadEvents) + " / " + p.injectTag(datadogslo.Query.TotalEvents) + ") - " + allowancePercent + ") - "
	query_second := "(" + datadogslo.Query.BadEvents + " / " + datadogslo.Query.TotalEvents + ") >= 0"
	query := "sum(last_2m):" + query_first + query_second

	p.Labels[picchuv1alpha1.LabelMonitorType] = MonitorTypeCanary

	ddogmonitor := ddog.DatadogMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.datadogCanaryMonitorName(datadogslo.Name),
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
			// metric type monitor
			Type: ddog.DatadogMonitorTypeMetric,
			// these will change bc its a metric monitor type
			Options: ddog.DatadogMonitorOptions{
				EnableLogsSample:       &options_true,
				EscalationMessage:      &escalation_message,
				IncludeTags:            &options_true,
				NotificationPresetName: "show_all",
				NotifyNoData:           &options_true,
				RenotifyStatuses:       renotify,
				RenotifyInterval:       &five_min,
				RequireFullWindow:      &options_true,
				EvaluationDelay:        &five_min,
				NoDataTimeframe:        &five_min,

				Thresholds: &ddog.DatadogMonitorOptionsThresholds{
					Critical: &canary_threshold,
				},
			},
		},
	}

	// taken from datadogslo
	ddogmonitor.Spec.Tags = append(ddogmonitor.Spec.Tags, datadogslo.Tags...)

	return ddogmonitor
}

func (p *SyncDatadogCanaryMonitors) datadogCanaryMonitorName(sloName string) string {
	// example: <service-name>-<condensed-target>-<condensed-slo-name>-<tag>-canary
	// lowercase - at most 63 characters - start and end with alphanumeric

	target := p.Target
	if strings.Contains(p.Target, "-") {
		t := strings.LastIndex(p.Target, "-")
		first_target := p.Target[:4]
		second_target := p.Target[t+1 : t+5]
		target = first_target + "-" + second_target
	} else if len(p.Target) >= 4 {
		target = p.Target[:4]
	}

	slo_name_end := strings.LastIndex(sloName, "-")
	new_slo_name := string(sloName[0])
	for i := range slo_name_end + 1 {
		if string(sloName[i]) == "-" {
			new_slo_name = new_slo_name + string(sloName[i+1])
		}
	}

	front_tag := p.App + "-" + target + "-" + new_slo_name + "-" + p.Tag + "-canary"
	if len(front_tag) > 63 {
		front_tag = front_tag[:63]
	}
	return front_tag
}

func (p *SyncDatadogCanaryMonitors) injectTag(query string) string {
	bracket_index := strings.Index(query, "{")
	tag_string := "destination_version:" + p.Tag + " AND "
	return query[:bracket_index+1] + tag_string + query[bracket_index+1:]
}

func (p *SyncDatadogCanaryMonitors) formatAllowancePercent(datadogslo *picchuv1alpha1.DatadogSLO, log logr.Logger) string {
	allowancePercent := datadogslo.Canary.AllowancePercent
	if datadogslo.Canary.AllowancePercentString != "" {
		f, err := strconv.ParseFloat(datadogslo.Canary.AllowancePercentString, 64)
		if err != nil {
			log.Error(err, "Could not parse %v to float", datadogslo.Canary.AllowancePercentString)
		} else {
			allowancePercent = f
		}
	}
	r := allowancePercent / 100
	return fmt.Sprintf("%.10g", r)
}
