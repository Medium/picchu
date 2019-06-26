package plan

import (
	"context"
	"fmt"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/go-logr/logr"
	istiocommonv1alpha1 "github.com/knative/pkg/apis/istio/common/v1alpha1"
	istiov1alpha3 "github.com/knative/pkg/apis/istio/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	StatusPort                  = "status"
	PrometheusScrapeLabel       = "prometheus.io/scrape"
	PrometheusScrapeLabelValue  = "true"
	TrafficPolicyMaxConnections = 10000
)

type Revision struct {
	Tag    string
	Weight uint32
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
	TagRoutingHeader  string
	UseNewTagStyle    bool
	TrafficPolicy     *istiov1alpha3.TrafficPolicy
}

func (p *SyncApp) Apply(ctx context.Context, cli client.Client, log logr.Logger) error {
	if len(p.DeployedRevisions) == 0 {
		log.Info("Not syncing app", "Reason", "there are no deployed revisions")
		return nil
	}
	if len(p.Ports) == 0 {
		log.Info("Not syncing app", "Reason", "there are no exposed ports")
		return nil
	}
	service := p.service()
	destinationRule := p.destinationRule()
	virtualService := p.virtualService(log)
	prometheusRule := p.prometheusRule()

	serviceSpec := *service.Spec.DeepCopy()
	serviceLabels := service.Labels
	op, err := controllerutil.CreateOrUpdate(ctx, cli, service, func(runtime.Object) error {
		service.Spec = serviceSpec
		service.Labels = serviceLabels
		return nil
	})
	LogSync(log, op, err, service)
	if err != nil {
		return err
	}

	drSpec := *destinationRule.Spec.DeepCopy()
	drLabels := destinationRule.Labels
	op, err = controllerutil.CreateOrUpdate(ctx, cli, destinationRule, func(runtime.Object) error {
		destinationRule.Spec = drSpec
		destinationRule.Labels = drLabels
		return nil
	})
	LogSync(log, op, err, destinationRule)
	if err != nil {
		return err
	}
	return nil

	if len(prometheusRule.Spec.Groups) == 0 {
		err = cli.Delete(ctx, prometheusRule)
		if err != nil && !errors.IsNotFound(err) {
			LogSync(log, "deleted", err, prometheusRule)
			return nil
		}
		if err == nil {
			LogSync(log, "deleted", err, prometheusRule)
		}
	} else {
		prSpec := *prometheusRule.Spec.DeepCopy()
		prLabels := prometheusRule.Labels
		op, err = controllerutil.CreateOrUpdate(ctx, cli, prometheusRule, func(runtime.Object) error {
			prometheusRule.Spec = prSpec
			prometheusRule.Labels = prLabels
			return nil
		})
		LogSync(log, op, err, prometheusRule)
		if err != nil {
			return err
		}
	}

	if virtualService == nil {
		return nil
	}

	vsSpec := *virtualService.Spec.DeepCopy()
	vsLabels := virtualService.Labels
	op, err = controllerutil.CreateOrUpdate(ctx, cli, virtualService, func(runtime.Object) error {
		virtualService.Spec = vsSpec
		virtualService.Labels = vsLabels
		return nil
	})
	LogSync(log, op, err, virtualService)
	if err != nil {
		return err
	}
	return nil
}

func (p *SyncApp) serviceHost() string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", p.App, p.Namespace)
}

func (p *SyncApp) defaultHost() string {
	return fmt.Sprintf("%s.%s", p.App, p.DefaultDomain)
}

func (p *SyncApp) releaseMatches(log logr.Logger, port picchuv1alpha1.PortInfo) []istiov1alpha3.HTTPMatchRequest {
	matches := []istiov1alpha3.HTTPMatchRequest{{
		Port:     uint32(port.Port),
		Gateways: []string{"mesh"},
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

	for _, host := range port.Hosts {
		matches = append(matches, istiov1alpha3.HTTPMatchRequest{
			Authority: &istiocommonv1alpha1.StringMatch{Prefix: host},
			Port:      portNumber,
			Gateways:  gateway,
		})
	}
	return matches
}

func (p *SyncApp) taggedMatches(port picchuv1alpha1.PortInfo, tag string) []istiov1alpha3.HTTPMatchRequest {
	headers := map[string]istiocommonv1alpha1.StringMatch{
		p.TagRoutingHeader: {Exact: tag},
	}
	matches := []istiov1alpha3.HTTPMatchRequest{{
		Headers:  headers,
		Port:     uint32(port.Port),
		Gateways: []string{"mesh"},
	}}

	gateways := []string{}
	if p.PublicGateway != "" {
		gateways = append(gateways, p.PublicGateway)
	}
	if p.PrivateGateway != "" {
		gateways = append(gateways, p.PrivateGateway)
	}

	if len(gateways) > 0 {
		matches = append(matches, istiov1alpha3.HTTPMatchRequest{
			Headers:  headers,
			Port:     uint32(port.IngressPort),
			Gateways: gateways,
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

func (p *SyncApp) taggedRoute(port picchuv1alpha1.PortInfo, tag string) []istiov1alpha3.DestinationWeight {
	return []istiov1alpha3.DestinationWeight{{
		Destination: istiov1alpha3.Destination{
			Host:   p.serviceHost(),
			Port:   istiov1alpha3.PortSelector{Number: uint32(port.Port)},
			Subset: tag,
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
		routes = append(routes, p.makeRoute(port, p.releaseMatches(log, port), p.releaseRoute(port)))
	}
	return routes
}

func (p *SyncApp) taggedRoutes() []istiov1alpha3.HTTPRoute {
	if p.TagRoutingHeader == "" {
		return []istiov1alpha3.HTTPRoute{}
	}
	routes := []istiov1alpha3.HTTPRoute{}
	for _, revision := range p.DeployedRevisions {
		for _, port := range p.Ports {
			tag := revision.Tag
			routes = append(routes, p.makeRoute(port, p.taggedMatches(port, tag), p.taggedRoute(port, tag)))
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
	labelTag := picchuv1alpha1.LabelTag
	if p.UseNewTagStyle {
		labelTag = "tag.picchu.medium.engineering"
	}
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
				Name:      "prometheus-rule",
				Namespace: p.Namespace,
				Labels:    p.Labels,
			},
		}
	}
	return &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-rule",
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
