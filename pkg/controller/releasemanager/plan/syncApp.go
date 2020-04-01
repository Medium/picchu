package plan

import (
	"context"
	"fmt"
	"sort"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/plan"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/go-logr/logr"
	"github.com/gogo/protobuf/types"
	istio "istio.io/api/networking/v1alpha3"
	istioclient "istio.io/client-go/pkg/apis/networking/v1alpha3"
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
	DefaultDomains    []string
	PublicGateway     string
	PrivateGateway    string
	DeployedRevisions []Revision
	AlertRules        []monitoringv1.Rule
	Ports             []picchuv1alpha1.PortInfo
}

func (p *SyncApp) Apply(ctx context.Context, cli client.Client, options plan.Options, log logr.Logger) error {
	if len(p.Ports) == 0 {
		log.Info("Not syncing app", "Reason", "there are no exposed ports")
		return nil
	}

	// Create the Service even before the Deployment is ready as a workaround to an Istio bug:
	// https://github.com/istio/istio/issues/11979
	service := p.service()
	if err := plan.CreateOrUpdate(ctx, log, cli, service); err != nil {
		return err
	}

	if len(p.DeployedRevisions) == 0 {
		log.Info("Not syncing app", "Reason", "there are no deployed revisions")
		return nil
	}

	destinationRule := p.destinationRule()
	virtualService := p.virtualService(options.ClusterName, log)
	prometheusRule := p.prometheusRule()

	if err := plan.CreateOrUpdate(ctx, log, cli, destinationRule); err != nil {
		return err
	}

	if len(prometheusRule.Spec.Groups) == 0 {
		err := cli.Delete(ctx, prometheusRule)
		if err != nil && !errors.IsNotFound(err) {
			plan.LogSync(log, "deleted", err, prometheusRule)
			return nil
		}
		if err == nil {
			plan.LogSync(log, "deleted", err, prometheusRule)
		}
	} else {
		if err := plan.CreateOrUpdate(ctx, log, cli, prometheusRule); err != nil {
			return err
		}
	}

	if virtualService == nil {
		log.Info("No virtualService")
		return nil
	}

	return plan.CreateOrUpdate(ctx, log, cli, virtualService)
}

func (p *SyncApp) serviceHost() string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", p.App, p.Namespace)
}

func (p *SyncApp) releaseMatches(log logr.Logger, clusterName string, port picchuv1alpha1.PortInfo) []*istio.HTTPMatchRequest {
	matches := []*istio.HTTPMatchRequest{{
		Port:     uint32(port.Port),
		Gateways: []string{"mesh"},
		Uri:      &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}},
	}}

	portNumber := uint32(port.Port)
	gateways := []string{"mesh"}
	hosts := port.Hosts

	switch port.Mode {
	case picchuv1alpha1.PortPublic:
		if p.PublicGateway == "" {
			log.Info("publicGateway undefined in Cluster for port", "port", port)
			return matches
		}
		gateways = []string{p.PublicGateway}
		portNumber = uint32(port.IngressPort)
	case picchuv1alpha1.PortPrivate:
		if p.PrivateGateway == "" {
			log.Info("privateGateway undefined in Cluster for port", "port", port)
			return matches
		}
		gateways = []string{p.PrivateGateway}

		defaultDomainsMap := map[string]bool{}
		for _, defaultDomain := range p.DefaultDomains {
			defaultDomainsMap[defaultDomain] = true
			if clusterName != "" {
				clusterDomain := fmt.Sprintf("%s.cluster.%s", clusterName, defaultDomain)
				defaultDomainsMap[clusterDomain] = true
			}
		}
		for domain := range defaultDomainsMap {
			hosts = append(hosts, fmt.Sprintf("%s.%s", p.Namespace, domain))
			if port.Name != "" {
				hosts = append(hosts, fmt.Sprintf("%s-%s.%s", p.Namespace, port.Name, domain))
			}
		}

		portNumber = uint32(port.IngressPort)
	}

	for _, host := range hosts {
		matches = append(matches, &istio.HTTPMatchRequest{
			Authority: &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: host}},
			Port:      portNumber,
			Gateways:  gateways,
			Uri:       &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}},
		})
		if port.IngressPort == 443 && port.HttpsRedirect {
			matches = append(matches, &istio.HTTPMatchRequest{
				Authority: &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: host}},
				Port:      uint32(80),
				Gateways:  gateways,
				Uri:       &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}},
			})
		}
	}
	return matches
}

func (p *SyncApp) taggedMatches(port picchuv1alpha1.PortInfo, revision Revision) []*istio.HTTPMatchRequest {
	headers := map[string]*istio.StringMatch{
		revision.TagRoutingHeader: {MatchType: &istio.StringMatch_Exact{Exact: revision.Tag}},
	}
	matches := []*istio.HTTPMatchRequest{{
		Headers:  headers,
		Port:     uint32(port.Port),
		Gateways: []string{"mesh"},
		Uri:      &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}},
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
		matches = append(matches, &istio.HTTPMatchRequest{
			Headers:  headers,
			Port:     uint32(port.IngressPort),
			Gateways: gateways,
			Uri:      &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}},
		})
	}
	return matches
}

func (p *SyncApp) releaseRoute(port picchuv1alpha1.PortInfo) []*istio.HTTPRouteDestination {
	weights := []*istio.HTTPRouteDestination{}
	for _, revision := range p.DeployedRevisions {
		if revision.Weight == 0 {
			continue
		}
		weights = append(weights, &istio.HTTPRouteDestination{
			Destination: &istio.Destination{
				Host:   p.serviceHost(),
				Port:   &istio.PortSelector{Number: uint32(port.Port)},
				Subset: revision.Tag,
			},
			Weight: int32(revision.Weight),
		})
	}
	return weights
}

func (p *SyncApp) taggedRoute(port picchuv1alpha1.PortInfo, revision Revision) []*istio.HTTPRouteDestination {
	return []*istio.HTTPRouteDestination{{
		Destination: &istio.Destination{
			Host:   p.serviceHost(),
			Port:   &istio.PortSelector{Number: uint32(port.Port)},
			Subset: revision.Tag,
		},
		Weight: 100,
	}}
}

func (p *SyncApp) makeRoute(
	port picchuv1alpha1.PortInfo,
	match []*istio.HTTPMatchRequest,
	route []*istio.HTTPRouteDestination,
) *istio.HTTPRoute {
	var retries *istio.HTTPRetry

	http := port.Istio.HTTP

	if http.Retries != nil {
		retries = &istio.HTTPRetry{
			Attempts: http.Retries.Attempts,
		}
		if http.Retries.PerTryTimeout != nil {
			retries.PerTryTimeout = types.DurationProto(http.Retries.PerTryTimeout.Duration)
		}
	}
	return &istio.HTTPRoute{
		Retries: retries,
		Match:   match,
		Route:   route,
	}
}

func (p *SyncApp) releaseRoutes(clusterName string, log logr.Logger) []*istio.HTTPRoute {
	routes := []*istio.HTTPRoute{}
	for _, port := range p.Ports {
		if len(p.releaseRoute(port)) == 0 {
			continue
		}
		routes = append(routes, p.makeRoute(port, p.releaseMatches(log, clusterName, port), p.releaseRoute(port)))
	}
	return routes
}

func (p *SyncApp) taggedRoutes() []*istio.HTTPRoute {
	routes := []*istio.HTTPRoute{}
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

func (p *SyncApp) service() *corev1.Service {
	ports := []corev1.ServicePort{}
	for _, port := range p.Ports {
		ports = append(ports, corev1.ServicePort{
			Name:       port.Name,
			Protocol:   port.Protocol,
			Port:       port.Port,
			TargetPort: intstr.FromString(port.Name),
		})
	}

	labels := map[string]string{
		PrometheusScrapeLabel: PrometheusScrapeLabelValue,
	}
	for k, v := range p.Labels {
		labels[k] = v
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

func (p *SyncApp) destinationRule() *istioclient.DestinationRule {
	labelTag := "tag.picchu.medium.engineering"
	subsets := []*istio.Subset{}
	for _, revision := range p.DeployedRevisions {
		subsets = append(subsets, &istio.Subset{
			Name:   revision.Tag,
			Labels: map[string]string{labelTag: revision.Tag},
		})
	}
	return &istioclient.DestinationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.App,
			Namespace: p.Namespace,
			Labels:    p.Labels,
		},
		Spec: istio.DestinationRule{
			Host:    p.serviceHost(),
			Subsets: subsets,
		},
	}

}

// virtualService will return nil if there are not configured routes
func (p *SyncApp) virtualService(clusterName string, log logr.Logger) *istioclient.VirtualService {
	http := append(p.taggedRoutes(), p.releaseRoutes(clusterName, log)...)
	if len(http) < 1 {
		log.Info("Not syncing VirtualService, there are no available deployments")
		return nil
	}

	hostsMap := map[string]bool{
		p.serviceHost(): true,
	}
	for _, httpRoute := range http {
		for _, httpMatch := range httpRoute.Match {
			prefix := httpMatch.Authority.GetPrefix()
			if prefix != "" {
				hostsMap[prefix] = true
			}
		}
	}
	hosts := make([]string, 0, len(hostsMap))
	for host := range hostsMap {
		hosts = append(hosts, host)
	}
	sort.Strings(hosts)

	gateways := []string{"mesh"}
	if p.PublicGateway != "" {
		gateways = append(gateways, p.PublicGateway)
	}
	if p.PrivateGateway != "" {
		gateways = append(gateways, p.PrivateGateway)
	}

	return &istioclient.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.App,
			Namespace: p.Namespace,
			Labels:    p.Labels,
		},
		Spec: istio.VirtualService{
			Hosts:    hosts,
			Gateways: gateways,
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
