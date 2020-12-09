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
	Tag                 string
	Weight              uint32
	TagRoutingHeader    string
	DevTagRoutingHeader string
	TrafficPolicy       *istio.TrafficPolicy
}

type SyncApp struct {
	App                  string
	Target               string
	Fleet                string
	Namespace            string
	Labels               map[string]string
	DeployedRevisions    []Revision
	AlertRules           []monitoringv1.Rule
	Ports                []picchuv1alpha1.PortInfo
	HTTPPortFaults       []picchuv1alpha1.HTTPPortFault
	IstioSidecarConfig   *picchuv1alpha1.IstioSidecar
	DefaultVariant       bool
	IngressesVariant     bool
	DefaultIngressPorts  map[string]string
	DevRoutesServiceHost string
	DevRoutesServicePort int
}

func (p *SyncApp) Apply(
	ctx context.Context,
	cli client.Client,
	cluster *picchuv1alpha1.Cluster,
	log logr.Logger,
) error {
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

	sidecarResource := p.sidecarResource()
	if sidecarResource != nil {
		if err := plan.CreateOrUpdate(ctx, log, cli, sidecarResource); err != nil {
			return err
		}
	}

	if len(p.DeployedRevisions) == 0 {
		log.Info("Not syncing app", "Reason", "there are no deployed revisions")
		return nil
	}

	destinationRule := p.destinationRule()
	if err := plan.CreateOrUpdate(ctx, log, cli, destinationRule); err != nil {
		return err
	}

	prometheusRule := p.prometheusRule()
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

	virtualService := p.virtualService(log, cluster)
	if virtualService == nil {
		log.Info("No virtualService")
		return nil
	}

	return plan.CreateOrUpdate(ctx, log, cli, virtualService)
}

func (p *SyncApp) serviceHost() string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", p.App, p.Namespace)
}

func (p *SyncApp) publicHosts(port picchuv1alpha1.PortInfo, cluster *picchuv1alpha1.Cluster) []string {
	return p.ingressHosts(port, cluster.Spec.Ingresses.Public.DefaultDomains, "public")
}

func (p *SyncApp) privateHosts(port picchuv1alpha1.PortInfo, cluster *picchuv1alpha1.Cluster) []string {
	return p.ingressHosts(port, cluster.Spec.Ingresses.Private.DefaultDomains, "private")
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

func (p *SyncApp) publicAuthorityMatch(port picchuv1alpha1.PortInfo, cluster *picchuv1alpha1.Cluster) *istio.StringMatch {
	return p.authorityMatch(p.publicHosts(port, cluster))
}

func (p *SyncApp) privateAuthorityMatch(port picchuv1alpha1.PortInfo, cluster *picchuv1alpha1.Cluster) *istio.StringMatch {
	return p.authorityMatch(p.privateHosts(port, cluster))
}

func (p *SyncApp) isUserDefined(host string) bool {
	userDefinedHosts := map[string]bool{}
	for _, port := range p.Ports {
		for _, host := range port.Hosts {
			userDefinedHosts[host] = true
		}
	}
	return userDefinedHosts[host]
}

func (p *SyncApp) ingressHosts(
	port picchuv1alpha1.PortInfo,
	defaultDomains []string,
	ingressName string,
) []string {
	hostMap := map[string]bool{}
	fleetSuffix := fmt.Sprintf("-%s", p.Fleet)

	addDefaultHost := func(host string) {
		if p.isUserDefined(host) {
			return
		}
		if p.DefaultVariant {
			defaultPort, ok := p.DefaultIngressPorts[ingressName]
			if !ok {
				return
			}
			if defaultPort != port.Name {
				return
			}
		}
		hostMap[host] = true
	}

	for _, domain := range defaultDomains {
		addDefaultHost(fmt.Sprintf("%s.%s", p.Namespace, domain))

		if p.Target == p.Fleet {
			addDefaultHost(fmt.Sprintf("%s.%s", p.App, domain))
		} else if strings.HasSuffix(p.Namespace, fleetSuffix) {
			basename := strings.TrimSuffix(p.Namespace, fleetSuffix)
			addDefaultHost(fmt.Sprintf("%s.%s", basename, domain))
		}
	}

	for _, host := range port.Hosts {
		hostMap[host] = true
	}

	hosts := make([]string, 0, len(hostMap))
	for host := range hostMap {
		hosts = append(hosts, host)
	}
	return hosts
}

func (p *SyncApp) releaseMatches(
	port picchuv1alpha1.PortInfo,
	cluster *picchuv1alpha1.Cluster,
) ([]*istio.HTTPMatchRequest, []string) {
	return p.portHeaderMatches(port, nil, cluster)
}

func (p *SyncApp) taggedMatches(
	port picchuv1alpha1.PortInfo,
	revision Revision,
	cluster *picchuv1alpha1.Cluster,
) ([]*istio.HTTPMatchRequest, []string) {
	headers := map[string]*istio.StringMatch{
		revision.TagRoutingHeader: {MatchType: &istio.StringMatch_Exact{Exact: revision.Tag}},
	}
	return p.portHeaderMatches(port, headers, cluster)
}

func (p *SyncApp) portHeaderMatches(
	port picchuv1alpha1.PortInfo,
	headers map[string]*istio.StringMatch,
	cluster *picchuv1alpha1.Cluster,
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

	publicEnabled, privateEnabled := false, false
	if p.IngressesVariant {
		for _, ingress := range port.Ingresses {
			switch ingress {
			case "public":
				publicEnabled = true
			case "private":
				privateEnabled = true
			}
		}

	} else {
		switch port.Mode {
		case picchuv1alpha1.PortPublic:
			publicEnabled = true
			privateEnabled = true
		case picchuv1alpha1.PortPrivate:
			privateEnabled = true
		}
	}

	if publicEnabled {
		hosts := p.publicHosts(port, cluster)
		if len(hosts) > 0 {
			for _, host := range hosts {
				hostMap[host] = true
			}
			matches = append(matches, &istio.HTTPMatchRequest{
				Headers:   headers,
				Authority: p.authorityMatch(hosts),
				Port:      ingressPort,
				Gateways:  []string{cluster.Spec.Ingresses.Public.Gateway},
				Uri:       &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}},
			})
			if port.IngressPort == 443 {
				matches = append(matches, &istio.HTTPMatchRequest{
					Headers:   headers,
					Authority: p.publicAuthorityMatch(port, cluster),
					Port:      uint32(80),
					Gateways:  []string{cluster.Spec.Ingresses.Public.Gateway},
					Uri:       &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}},
				})
			}
		}
	}
	if privateEnabled {
		hosts := p.privateHosts(port, cluster)
		if len(hosts) > 0 {
			for _, host := range hosts {
				hostMap[host] = true
			}
			matches = append(matches, &istio.HTTPMatchRequest{
				Headers:   headers,
				Authority: p.privateAuthorityMatch(port, cluster),
				Port:      ingressPort,
				Gateways:  []string{cluster.Spec.Ingresses.Private.Gateway},
				Uri:       &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}},
			})
			if port.IngressPort == 443 {
				matches = append(matches, &istio.HTTPMatchRequest{
					Headers:   headers,
					Authority: p.privateAuthorityMatch(port, cluster),
					Port:      uint32(80),
					Gateways:  []string{cluster.Spec.Ingresses.Private.Gateway},
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

func (p *SyncApp) devMatches(
	revision Revision,
) []*istio.HTTPMatchRequest {
	headers := map[string]*istio.StringMatch{
		revision.DevTagRoutingHeader: {MatchType: &istio.StringMatch_Regex{Regex: "(.+)"}},
	}

	return []*istio.HTTPMatchRequest{{
		Headers: headers,
	}}
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

func (p *SyncApp) devRoute() []*istio.HTTPRouteDestination {
	return []*istio.HTTPRouteDestination{{
		Destination: &istio.Destination{
			Host: p.DevRoutesServiceHost,
			Port: &istio.PortSelector{Number: uint32(p.DevRoutesServicePort)},
		},
		Weight: 100,
	}}
}

func (p *SyncApp) makeRoute(
	name string,
	port picchuv1alpha1.PortInfo,
	match []*istio.HTTPMatchRequest,
	route []*istio.HTTPRouteDestination,
) *istio.HTTPRoute {
	var retries *istio.HTTPRetry
	var fault *istio.HTTPFaultInjection

	http := port.Istio.HTTP

	for _, f := range p.HTTPPortFaults {
		if int32(f.PortSelector.Number) == port.Port {
			fault = f.HTTPFault
		}
	}

	if http.Retries != nil {
		retries = &istio.HTTPRetry{
			Attempts: http.Retries.Attempts,
		}
		if http.Retries.PerTryTimeout != nil {
			retries.PerTryTimeout = types.DurationProto(http.Retries.PerTryTimeout.Duration)
		}
		if http.Retries.RetryOn != nil {
			retries.RetryOn = *http.Retries.RetryOn
		}
	}
	var timeout *types.Duration
	if http.Timeout != nil {
		timeout = types.DurationProto(http.Timeout.Duration)
	}
	return &istio.HTTPRoute{
		Name:    name,
		Timeout: timeout,
		Retries: retries,
		Match:   match,
		Route:   route,
		Fault:   fault,
	}
}

func (p *SyncApp) releaseRoutes(cluster *picchuv1alpha1.Cluster) ([]*istio.HTTPRoute, []string) {
	var routes []*istio.HTTPRoute
	hostMap := map[string]bool{}

	for _, port := range p.Ports {
		if len(p.releaseRoute(port)) == 0 {
			continue
		}
		releaseMatches, hosts := p.releaseMatches(port, cluster)
		for _, host := range hosts {
			hostMap[host] = true
		}
		name := fmt.Sprintf("01_release-%s", port.Name)
		routes = append(routes, p.makeRoute(name, port, releaseMatches, p.releaseRoute(port)))
	}

	hosts := make([]string, 0, len(hostMap))
	for host := range hostMap {
		hosts = append(hosts, host)
	}
	return routes, hosts
}

func (p *SyncApp) taggedRoutes(cluster *picchuv1alpha1.Cluster) ([]*istio.HTTPRoute, []string) {
	var routes []*istio.HTTPRoute
	hostMap := map[string]bool{}

	for _, revision := range p.DeployedRevisions {
		if revision.TagRoutingHeader == "" {
			continue
		}
		for _, port := range p.Ports {
			taggedMatches, hosts := p.taggedMatches(port, revision, cluster)
			for _, host := range hosts {
				hostMap[host] = true
			}
			name := fmt.Sprintf("00_tagged-%s-%s", revision.Tag, port.Name)
			routes = append(routes, p.makeRoute(name, port, taggedMatches, p.taggedRoute(port, revision)))
		}
	}

	hosts := make([]string, 0, len(hostMap))
	for host := range hostMap {
		hosts = append(hosts, host)
	}
	return routes, hosts
}

func (p *SyncApp) devRoutes(cluster *picchuv1alpha1.Cluster) []*istio.HTTPRoute {
	var routes []*istio.HTTPRoute

	for _, revision := range p.DeployedRevisions {
		if revision.DevTagRoutingHeader == "" {
			continue
		}
		if cluster.Spec.EnableDevRoutes {
			route := &istio.HTTPRoute{
				Name:  fmt.Sprintf("00_dev-%s", p.App),
				Match: p.devMatches(revision),
				Route: p.devRoute(),
			}
			routes = append(routes, route)
		}
	}

	return routes
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

	// Don't needlessly update services
	sort.Slice(ports, func(i, j int) bool {
		return ports[i].Port < ports[j].Port
	})
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
			Name:          revision.Tag,
			Labels:        map[string]string{labelTag: revision.Tag},
			TrafficPolicy: revision.TrafficPolicy,
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
func (p *SyncApp) virtualService(log logr.Logger, cluster *picchuv1alpha1.Cluster) *istioclient.VirtualService {
	hostMap := map[string]bool{}

	taggedRoutes, taggedHosts := p.taggedRoutes(cluster)
	for _, host := range taggedHosts {
		hostMap[host] = true
	}

	releaseRoutes, releaseHosts := p.releaseRoutes(cluster)
	for _, host := range releaseHosts {
		hostMap[host] = true
	}

	devRoutes := p.devRoutes(cluster)

	http := append(devRoutes, taggedRoutes...)
	http = append(http, releaseRoutes...)
	if len(http) < 1 {
		log.Info("Not syncing VirtualService, there are no available deployments")
		return nil
	}

	sort.Slice(http, func(i, j int) bool {
		return http[i].Name < http[j].Name
	})

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

func (p *SyncApp) sidecarResource() *istioclient.Sidecar {
	if p.IstioSidecarConfig == nil {
		return nil
	}

	return &istioclient.Sidecar{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.App,
			Namespace: p.Namespace,
			Labels:    p.Labels,
		},
		Spec: istio.Sidecar{
			Egress: []*istio.IstioEgressListener{
				{
					Hosts: p.IstioSidecarConfig.EgressHosts,
				},
			},
		},
	}
}
