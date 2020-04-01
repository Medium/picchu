package plan

import (
	"context"
	"sort"
	"strings"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/go-logr/logr"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/plan"
	prometheus "go.medium.engineering/picchu/pkg/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SyncServiceMonitors struct {
	App                    string
	Namespace              string
	Labels                 map[string]string
	ServiceMonitors        []*picchuv1alpha1.ServiceMonitor
	ServiceLevelObjectives []*picchuv1alpha1.ServiceLevelObjective
}

func (p *SyncServiceMonitors) Apply(ctx context.Context, cli client.Client, options plan.Options, log logr.Logger) error {
	serviceMonitors, err := p.serviceMonitors()
	if err != nil {
		return err
	}
	if len(serviceMonitors.Items) > 0 {
		for i := range serviceMonitors.Items {
			if err := plan.CreateOrUpdate(ctx, log, cli, serviceMonitors.Items[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *SyncServiceMonitors) serviceMonitors() (*monitoringv1.ServiceMonitorList, error) {

	names, err := p.parseMetricNames()
	if err != nil {
		return nil, err
	}
	metricNamesRegex := strings.Join(names, "|")

	sml := &monitoringv1.ServiceMonitorList{}
	sms := []*monitoringv1.ServiceMonitor{}

	for i := range p.ServiceMonitors {
		sm := p.serviceMonitor(p.ServiceMonitors[i], metricNamesRegex)
		sms = append(sms, sm)
	}
	sml.Items = sms

	return sml, nil
}

func (p *SyncServiceMonitors) serviceMonitor(sm *picchuv1alpha1.ServiceMonitor, metricNamesRegex string) *monitoringv1.ServiceMonitor {
	labels := make(map[string]string)

	for k, v := range sm.Labels {
		labels[k] = v
	}

	annotations := make(map[string]string)

	for k, v := range sm.Annotations {
		annotations[k] = v
	}

	serviceMonitor := &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:        sm.Name,
			Namespace:   p.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
	}

	serviceMonitor.Spec = sm.Spec

	for _, ep := range serviceMonitor.Spec.Endpoints {
		for _, r := range ep.MetricRelabelConfigs {
			// Passes in the SLO metric names as | delimited string
			// to allow automatic keep/drop handling of SLO metrics
			if r.Regex == "" && sm.SLORegex {
				r.Regex = metricNamesRegex
			}
		}
	}

	nsselector := &monitoringv1.NamespaceSelector{
		MatchNames: []string{p.Namespace},
	}

	serviceMonitor.Spec.NamespaceSelector = *nsselector

	return serviceMonitor
}

// return all unique metric names required by the ServiceLevelObjectives
func (p *SyncServiceMonitors) parseMetricNames() ([]string, error) {
	n := make(map[string]bool)
	for i := range p.ServiceLevelObjectives {
		totalQuery, err := prometheus.MetricNames(p.ServiceLevelObjectives[i].ServiceLevelIndicator.TotalQuery)
		if err != nil {
			return nil, err
		}
		for name := range totalQuery {
			n[name] = true
		}
		errorQuery, err := prometheus.MetricNames(p.ServiceLevelObjectives[i].ServiceLevelIndicator.ErrorQuery)
		if err != nil {
			return nil, err
		}
		for name := range errorQuery {
			n[name] = true
		}
	}

	m := []string{}
	for name := range n {
		m = append(m, name)
	}
	sort.Strings(m)
	return m, nil
}
