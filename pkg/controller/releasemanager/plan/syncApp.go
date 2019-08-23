package plan

import (
	"context"
	"fmt"
	"sort"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/go-logr/logr"
	istiocommonv1alpha1 "github.com/knative/pkg/apis/istio/common/v1alpha1"
	istiov1alpha3 "github.com/knative/pkg/apis/istio/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	StatusPort                 = "status"
	PrometheusScrapeLabel      = "prometheus.io/scrape"
	PrometheusScrapeLabelValue = "true"
)

type Revision struct {
	Tag              string
	Weight           uint32
	TagRoutingHeader string
}

type SyncApp struct {
	App               string
	Namespace         string
	Labels            map[string]string
	DefaultDomain     string
	PublicGateway     string
	PrivateGateway    string
	DeployedRevisions []Revision
	AlertRules        []monitoringv1.Rule
	Ports             []picchuv1alpha1.PortInfo
	TrafficPolicy     *istiov1alpha3.TrafficPolicy
}

func (p *SyncApp) Apply(ctx context.Context, cli client.Client, log logr.Logger) error {
	if len(p.Ports) == 0 {
		log.Info("Not syncing app", "Reason", "there are no exposed ports")
		return nil
	}

	// Create the Service even before the Deployment is ready as a workaround to an Istio bug:
	// https://github.com/istio/istio/issues/11979
	service := p.service()
	if err := CreateOrUpdate(ctx, log, cli, service); err != nil {
		return err
	}

	if len(p.DeployedRevisions) == 0 {
		log.Info("Not syncing app", "Reason", "there are no deployed revisions")
		return nil
	}

	destinationRule := p.destinationRule()
	virtualService := p.virtualService(log)
	prometheusRule := p.prometheusRule()

	if err := CreateOrUpdate(ctx, log, cli, destinationRule); err != nil {
		return err
	}

	if len(prometheusRule.Spec.Groups) == 0 {
		err := cli.Delete(ctx, prometheusRule)
		if err != nil && !errors.IsNotFound(err) {
			LogSync(log, "deleted", err, prometheusRule)
			return nil
		}
		if err == nil {
			LogSync(log, "deleted", err, prometheusRule)
		}
	} else {
		if err := CreateOrUpdate(ctx, log, cli, prometheusRule); err != nil {
			return err
		}
	}

	if virtualService == nil {
		log.Info("No virtualService")
		return nil
	}

	return CreateOrUpdate(ctx, log, cli, virtualService)
}

func (p *SyncApp) serviceHost() string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", p.App, p.Namespace)
}

func (p *SyncApp) defaultHost() string {
	return fmt.Sprintf("%s.%s", p.Namespace, p.DefaultDomain)
}

func (p *SyncApp) releaseMatches(log logr.Logger, port picchuv1alpha1.PortInfo) []istiov1alpha3.HTTPMatchRequest {
	matches := []istiov1alpha3.HTTPMatchRequest{{
		Port:     uint32(port.Port),
		Gateways: []string{"mesh"},
		Uri:      &istiocommonv1alpha1.StringMatch{Prefix: "/"},
	}}

	portNumber := uint32(port.Port)
	gateway := []string{"mesh"}
	hosts := port.Hosts

	switch port.Mode {
	case picchuv1alpha1.PortPublic:
		if p.PublicGateway == "" {
			log.Info("publicGateway undefined in Cluster for port", "port", port)
			return matches
		}
		gateway = []string{p.PublicGateway}
		portNumber = uint32(port.IngressPort)
	case picchuv1alpha1.PortPrivate:
		if p.PrivateGateway == "" {
			log.Info("privateGateway undefined in Cluster for port", "port", port)
			return matches
		}
		gateway = []string{p.PrivateGateway}
		hosts = append(hosts, p.defaultHost())
		portNumber = uint32(port.IngressPort)
	}

	for _, host := range hosts {
		matches = append(matches, istiov1alpha3.HTTPMatchRequest{
			Authority: &istiocommonv1alpha1.StringMatch{Prefix: host},
			Port:      portNumber,
			Gateways:  gateway,
			Uri:       &istiocommonv1alpha1.StringMatch{Prefix: "/"},
		})
	}
	return matches
}

func (p *SyncApp) taggedMatches(port picchuv1alpha1.PortInfo, revision Revision) []istiov1alpha3.HTTPMatchRequest {
	headers := map[string]istiocommonv1alpha1.StringMatch{
		revision.TagRoutingHeader: {Exact: revision.Tag},
	}
	matches := []istiov1alpha3.HTTPMatchRequest{{
		Headers:  headers,
		Port:     uint32(port.Port),
		Gateways: []string{"mesh"},
		Uri:      &istiocommonv1alpha1.StringMatch{Prefix: "/"},
	}}

	gateways := []string{}

	switch port.Mode {
	case picchuv1alpha1.PortPublic:
		if p.PublicGateway != "" {
			gateways = append(gateways, p.PublicGateway)
		}
	case picchuv1alpha1.PortPrivate:
		if p.PrivateGateway != "" {
			gateways = append(gateways, p.PrivateGateway)
		}
	}

	if len(gateways) > 0 {
		matches = append(matches, istiov1alpha3.HTTPMatchRequest{
			Headers:  headers,
			Port:     uint32(port.IngressPort),
			Gateways: gateways,
			Uri:      &istiocommonv1alpha1.StringMatch{Prefix: "/"},
		})
	}
	return matches
}

func (p *SyncApp) releaseRoute(port picchuv1alpha1.PortInfo) []istiov1alpha3.DestinationWeight {
	weights := []istiov1alpha3.DestinationWeight{}
	for _, revision := range p.DeployedRevisions {
		if revision.Weight == 0 {
			continue
		}
		weights = append(weights, istiov1alpha3.DestinationWeight{
			Destination: istiov1alpha3.Destination{
				Host:   p.serviceHost(),
				Port:   istiov1alpha3.PortSelector{Number: uint32(port.Port)},
				Subset: revision.Tag,
			},
			Weight: int(revision.Weight),
		})
	}
	return weights
}

func (p *SyncApp) taggedRoute(port picchuv1alpha1.PortInfo, revision Revision) []istiov1alpha3.DestinationWeight {
	return []istiov1alpha3.DestinationWeight{{
		Destination: istiov1alpha3.Destination{
			Host:   p.serviceHost(),
			Port:   istiov1alpha3.PortSelector{Number: uint32(port.Port)},
			Subset: revision.Tag,
		},
		Weight: 100,
	}}
}

func (p *SyncApp) makeRoute(
	port picchuv1alpha1.PortInfo,
	match []istiov1alpha3.HTTPMatchRequest,
	route []istiov1alpha3.DestinationWeight,
) istiov1alpha3.HTTPRoute {
	return istiov1alpha3.HTTPRoute{
		Redirect:              port.Istio.HTTP.Redirect,
		Rewrite:               port.Istio.HTTP.Rewrite,
		Retries:               port.Istio.HTTP.Retries,
		Fault:                 port.Istio.HTTP.Fault,
		Mirror:                port.Istio.HTTP.Mirror,
		AppendHeaders:         port.Istio.HTTP.AppendHeaders,
		RemoveResponseHeaders: port.Istio.HTTP.RemoveResponseHeaders,
		Match:                 match,
		Route:                 route,
	}
}

func (p *SyncApp) releaseRoutes(log logr.Logger) []istiov1alpha3.HTTPRoute {
	routes := []istiov1alpha3.HTTPRoute{}
	for _, port := range p.Ports {
		if len(p.releaseRoute(port)) == 0 {
			continue
		}
		routes = append(routes, p.makeRoute(port, p.releaseMatches(log, port), p.releaseRoute(port)))
	}
	return routes
}

func (p *SyncApp) taggedRoutes() []istiov1alpha3.HTTPRoute {
	routes := []istiov1alpha3.HTTPRoute{}
	for _, revision := range p.DeployedRevisions {
		if revision.TagRoutingHeader == "" {
			continue
		}
		for _, port := range p.Ports {
			routes = append(routes, p.makeRoute(port, p.taggedMatches(port, revision), p.taggedRoute(port, revision)))
		}
	}
	return routes
}

func (p *SyncApp) hosts() []string {
	hostsMap := map[string]bool{
		p.defaultHost(): true,
		p.serviceHost(): true,
	}
	for _, port := range p.Ports {
		for _, host := range port.Hosts {
			hostsMap[host] = true
		}
	}

	hosts := make([]string, 0, len(hostsMap))
	for host, _ := range hostsMap {
		hosts = append(hosts, host)
	}
	sort.Strings(hosts)
	return hosts
}

func (p *SyncApp) gateways() []string {
	gateways := []string{"mesh"}
	if p.PublicGateway != "" {
		gateways = append(gateways, p.PublicGateway)
	}
	if p.PrivateGateway != "" {
		gateways = append(gateways, p.PrivateGateway)
	}
	return gateways
}

func (p *SyncApp) service() *corev1.Service {
	hasStatus := false
	ports := []corev1.ServicePort{}
	for _, port := range p.Ports {
		ports = append(ports, corev1.ServicePort{
			Name:       port.Name,
			Protocol:   port.Protocol,
			Port:       port.Port,
			TargetPort: intstr.FromString(port.Name),
		})
		if port.Name == "status" {
			hasStatus = true
		}
	}

	labels := map[string]string{}
	for k, v := range p.Labels {
		labels[k] = v
	}
	if hasStatus {
		labels[PrometheusScrapeLabel] = PrometheusScrapeLabelValue
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.App,
			Namespace: p.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
			Selector: map[string]string{
				picchuv1alpha1.LabelApp: p.App,
			},
		},
	}
}

func (p *SyncApp) destinationRule() *istiov1alpha3.DestinationRule {
	labelTag := "tag.picchu.medium.engineering"
	subsets := []istiov1alpha3.Subset{}
	for _, revision := range p.DeployedRevisions {
		subsets = append(subsets, istiov1alpha3.Subset{
			Name:   revision.Tag,
			Labels: map[string]string{labelTag: revision.Tag},
		})
	}
	return &istiov1alpha3.DestinationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.App,
			Namespace: p.Namespace,
			Labels:    p.Labels,
		},
		Spec: istiov1alpha3.DestinationRuleSpec{
			Host:          p.serviceHost(),
			Subsets:       subsets,
			TrafficPolicy: p.TrafficPolicy,
		},
	}

}

// virtualService will return nil if there are not configured routes
func (p *SyncApp) virtualService(log logr.Logger) *istiov1alpha3.VirtualService {
	http := append(p.taggedRoutes(), p.releaseRoutes(log)...)
	if len(http) < 1 {
		log.Info("Not syncing VirtualService, there are no available deployments")
		return nil
	}

	return &istiov1alpha3.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.App,
			Namespace: p.Namespace,
			Labels:    p.Labels,
		},
		Spec: istiov1alpha3.VirtualServiceSpec{
			Hosts:    p.hosts(),
			Gateways: p.gateways(),
			Http:     http,
		},
	}
}

// If prometheusRule ruturns len(rule.Spec.Group) == 0, it should be deleted
func (p *SyncApp) prometheusRule() *monitoringv1.PrometheusRule {
	if len(p.AlertRules) == 0 {
		return &monitoringv1.PrometheusRule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      p.App,
				Namespace: p.Namespace,
				Labels:    p.Labels,
			},
		}
	}
	return &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.App,
			Namespace: p.Namespace,
			Labels:    p.Labels,
		},
		Spec: monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{{
				Name:  "picchu.rules",
				Rules: p.AlertRules,
			}},
		},
	}
}
