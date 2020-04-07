package plan

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

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
	virtualService := p.virtualService(log, options)
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

func (p *SyncApp) publicHosts(port picchuv1alpha1.PortInfo, options plan.Options) []string {
	return p.ingressHosts(picchuv1alpha1.PortPublic, port, options.DefaultDomains.Public)
}

func (p *SyncApp) privateHosts(port picchuv1alpha1.PortInfo, options plan.Options) []string {
	return p.ingressHosts(picchuv1alpha1.PortPrivate, port, options.DefaultDomains.Private)
}

func (p *SyncApp) authorityMatch(hosts []string) *istio.StringMatch {
	hostPatterns := make([]string, 0, len(hosts))
	for _, host := range hosts {
		hostPatterns = append(hostPatterns, regexp.QuoteMeta(host))
	}
	sort.Strings(hostPatterns)
	regex := fmt.Sprintf(`^(%s)(:[0-9]+)?$`, strings.Join(hostPatterns, "|"))
	return &istio.StringMatch{
		MatchType: &istio.StringMatch_Regex{Regex: regex},
	}
}

func (p *SyncApp) publicAuthorityMatch(port picchuv1alpha1.PortInfo, options plan.Options) *istio.StringMatch {
	return p.authorityMatch(p.publicHosts(port, options))
}

func (p *SyncApp) privateAuthorityMatch(port picchuv1alpha1.PortInfo, options plan.Options) *istio.StringMatch {
	return p.authorityMatch(p.privateHosts(port, options))
}

func (p *SyncApp) ingressHosts(
	mode picchuv1alpha1.PortMode,
	port picchuv1alpha1.PortInfo,
	defaultDomains []string,
) []string {
	hostMap := map[string]bool{}
	defaultDomainsMap := map[string]bool{}
	for _, defaultDomain := range defaultDomains {
		defaultDomainsMap[defaultDomain] = true
	}
	for domain := range defaultDomainsMap {
		hostMap[fmt.Sprintf("%s.%s", p.Namespace, domain)] = true
	}

	if port.Mode == mode {
		for _, host := range port.Hosts {
			hostMap[host] = true
		}
	}

	hosts := make([]string, 0, len(hostMap))
	for host := range hostMap {
		hosts = append(hosts, host)
	}
	return hosts
}

func (p *SyncApp) releaseMatches(
	port picchuv1alpha1.PortInfo,
	options plan.Options,
) ([]*istio.HTTPMatchRequest, []string) {
	return p.portHeaderMatches(port, nil, options)
}

func (p *SyncApp) taggedMatches(
	port picchuv1alpha1.PortInfo,
	revision Revision,
	options plan.Options,
) ([]*istio.HTTPMatchRequest, []string) {
	headers := map[string]*istio.StringMatch{
		revision.TagRoutingHeader: {MatchType: &istio.StringMatch_Exact{Exact: revision.Tag}},
	}
	return p.portHeaderMatches(port, headers, options)
}

func (p *SyncApp) portHeaderMatches(
	port picchuv1alpha1.PortInfo,
	headers map[string]*istio.StringMatch,
	options plan.Options,
) ([]*istio.HTTPMatchRequest, []string) {
	portNumber := uint32(port.Port)
	ingressPort := uint32(port.IngressPort)
	hostMap := map[string]bool{
		p.serviceHost(): true,
	}

	matches := []*istio.HTTPMatchRequest{{
		Headers:  headers,
		Port:     portNumber,
		Gateways: []string{"mesh"},
		Uri:      &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}},
	}}

	if port.Mode == picchuv1alpha1.PortPublic {
		hosts := p.publicHosts(port, options)
		if len(hosts) > 0 {
			for _, host := range hosts {
				hostMap[host] = true
			}
			matches = append(matches, &istio.HTTPMatchRequest{
				Headers:   headers,
				Authority: p.authorityMatch(hosts),
				Port:      ingressPort,
				Gateways:  []string{p.PublicGateway},
				Uri:       &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}},
			})
			if port.IngressPort == 443 {
				matches = append(matches, &istio.HTTPMatchRequest{
					Headers:   headers,
					Authority: p.publicAuthorityMatch(port, options),
					Port:      uint32(80),
					Gateways:  []string{p.PublicGateway},
					Uri:       &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}},
				})
			}
		}
	}
	if port.Mode == picchuv1alpha1.PortPublic || port.Mode == picchuv1alpha1.PortPrivate {
		hosts := p.privateHosts(port, options)
		for _, host := range hosts {
			hostMap[host] = true
		}
		if len(hosts) > 0 {
			matches = append(matches, &istio.HTTPMatchRequest{
				Headers:   headers,
				Authority: p.privateAuthorityMatch(port, options),
				Port:      ingressPort,
				Gateways:  []string{p.PrivateGateway},
				Uri:       &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}},
			})
			if port.IngressPort == 443 {
				matches = append(matches, &istio.HTTPMatchRequest{
					Headers:   headers,
					Authority: p.privateAuthorityMatch(port, options),
					Port:      uint32(80),
					Gateways:  []string{p.PrivateGateway},
					Uri:       &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}},
				})
			}
		}
	}

	hosts := make([]string, 0, len(hostMap))
	for host := range hostMap {
		hosts = append(hosts, host)
	}

	return matches, hosts
}

func (p *SyncApp) releaseRoute(port picchuv1alpha1.PortInfo) []*istio.HTTPRouteDestination {
	var weights []*istio.HTTPRouteDestination
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

func (p *SyncApp) releaseRoutes(options plan.Options) ([]*istio.HTTPRoute, []string) {
	var routes []*istio.HTTPRoute
	hostMap := map[string]bool{}

	for _, port := range p.Ports {
		if len(p.releaseRoute(port)) == 0 {
			continue
		}
		releaseMatches, hosts := p.releaseMatches(port, options)
		for _, host := range hosts {
			hostMap[host] = true
		}
		routes = append(routes, p.makeRoute(port, releaseMatches, p.releaseRoute(port)))
	}

	hosts := make([]string, 0, len(hostMap))
	for host := range hostMap {
		hosts = append(hosts, host)
	}
	return routes, hosts
}

func (p *SyncApp) taggedRoutes(options plan.Options) ([]*istio.HTTPRoute, []string) {
	var routes []*istio.HTTPRoute
	hostMap := map[string]bool{}

	for _, revision := range p.DeployedRevisions {
		if revision.TagRoutingHeader == "" {
			continue
		}
		for _, port := range p.Ports {
			taggedMatches, hosts := p.taggedMatches(port, revision, options)
			for _, host := range hosts {
				hostMap[host] = true
			}
			routes = append(routes, p.makeRoute(port, taggedMatches, p.taggedRoute(port, revision)))
		}
	}

	hosts := make([]string, 0, len(hostMap))
	for host := range hostMap {
		hosts = append(hosts, host)
	}
	return routes, hosts
}

func (p *SyncApp) service() *corev1.Service {
	var ports []corev1.ServicePort
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
	var subsets []*istio.Subset
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
func (p *SyncApp) virtualService(log logr.Logger, options plan.Options) *istioclient.VirtualService {
	hostMap := map[string]bool{}

	taggedRoutes, taggedHosts := p.taggedRoutes(options)
	for _, host := range taggedHosts {
		hostMap[host] = true
	}

	releaseRoutes, releaseHosts := p.releaseRoutes(options)
	for _, host := range releaseHosts {
		hostMap[host] = true
	}

	http := append(taggedRoutes, releaseRoutes...)
	if len(http) < 1 {
		log.Info("Not syncing VirtualService, there are no available deployments")
		return nil
	}

	gatewayMap := map[string]bool{
		"mesh": true,
	}
	for _, httpRoute := range http {
		for _, httpMatch := range httpRoute.Match {
			for _, gateway := range httpMatch.Gateways {
				gatewayMap[gateway] = true
			}
		}
	}
	gateways := make([]string, 0, len(gatewayMap))
	for gateway := range gatewayMap {
		gateways = append(gateways, gateway)
	}
	sort.Strings(gateways)

	hosts := make([]string, 0, len(hostMap))
	for host := range hostMap {
		hosts = append(hosts, host)
	}

	sort.Strings(hosts)

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
