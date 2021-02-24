package plan

import (
	"context"
	_ "runtime"
	"testing"
	"time"

	ktest "go.medium.engineering/kubernetes/pkg/test"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/test"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/gogo/protobuf/types"
	testify "github.com/stretchr/testify/assert"
	istio "istio.io/api/networking/v1alpha3"
	istioclient "istio.io/client-go/pkg/apis/networking/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	authorityRegex = &istio.StringMatch{
		MatchType: &istio.StringMatch_Regex{
			Regex: "^(testapp\\.doki-pen\\.org|testapp\\.test-a\\.cluster\\.doki-pen\\.org|testnamespace\\.doki-pen\\.org|testnamespace\\.test-a\\.cluster\\.doki-pen\\.org)(:[0-9]+)?$",
		},
	}
	defaultSyncAppPlan = &SyncApp{
		App:       "testapp",
		Fleet:     "production",
		Target:    "production",
		Namespace: "testnamespace",
		Labels: map[string]string{
			"test": "label",
		},
		DeployedRevisions: []Revision{
			{
				Tag:              "testtag",
				Weight:           100,
				TagRoutingHeader: "TEST-TAG",
				TrafficPolicy: &istio.TrafficPolicy{
					ConnectionPool: &istio.ConnectionPoolSettings{
						Tcp: &istio.ConnectionPoolSettings_TCPSettings{
							MaxConnections: 100,
						},
					},
				},
			},
		},
		AlertRules: []monitoringv1.Rule{
			{
				Expr: intstr.FromString("hello world"),
			},
		},
		Ports: []picchuv1alpha1.PortInfo{
			{
				Name:          "http",
				IngressPort:   443,
				Port:          80,
				ContainerPort: 5000,
				Protocol:      corev1.ProtocolTCP,
				Mode:          picchuv1alpha1.PortPrivate,
				Istio: picchuv1alpha1.IstioPortConfig{
					HTTP: picchuv1alpha1.IstioHTTPPortConfig{
						Retries: &picchuv1alpha1.Retries{
							RetryOn: &defaultRetryOn,
							PerTryTimeout: &metav1.Duration{
								Duration: time.Duration(3) * time.Millisecond,
							},
							Attempts: 2,
						},
						Timeout: &metav1.Duration{
							Duration: time.Duration(3) * time.Millisecond,
						},
					},
				},
			},
			{
				Name:          "status",
				IngressPort:   443,
				Port:          4242,
				ContainerPort: 4444,
				Protocol:      corev1.ProtocolTCP,
				Mode:          picchuv1alpha1.PortLocal,
			},
		},
		IstioSidecarConfig: &picchuv1alpha1.IstioSidecar{
			EgressHosts: []string{"istio-system/*"},
		},
	}

	defaultExpectedVirtualService = &istioclient.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testapp",
			Namespace: "testnamespace",
			Labels: map[string]string{
				"test": "label",
			},
		},
		Spec: istio.VirtualService{
			Hosts: []string{
				"testapp.doki-pen.org",
				"testapp.test-a.cluster.doki-pen.org",
				"testapp.testnamespace.svc.cluster.local",
				"testnamespace.doki-pen.org",
				"testnamespace.test-a.cluster.doki-pen.org",
			},
			Gateways: []string{
				"mesh",
				"private-gateway",
			},
			Http: []*istio.HTTPRoute{
				{ // Tagged http route
					Name: "00_tagged-testtag-http",
					Match: []*istio.HTTPMatchRequest{
						{
							Gateways: []string{"mesh"},
							Headers: map[string]*istio.StringMatch{
								"TEST-TAG": {
									MatchType: &istio.StringMatch_Exact{Exact: "testtag"},
								},
							},
							Port: uint32(80),
							Uri:  &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}},
						},
						{
							Gateways: []string{"private-gateway"},
							Headers: map[string]*istio.StringMatch{
								"TEST-TAG": {
									MatchType: &istio.StringMatch_Exact{Exact: "testtag"},
								},
							},
							Authority: authorityRegex,
							Port:      uint32(443),
							Uri:       &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}},
						},
						{
							Gateways: []string{"private-gateway"},
							Headers: map[string]*istio.StringMatch{
								"TEST-TAG": {
									MatchType: &istio.StringMatch_Exact{Exact: "testtag"},
								},
							},
							Authority: authorityRegex,
							Port:      uint32(80),
							Uri:       &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}},
						},
					},
					Route: []*istio.HTTPRouteDestination{
						{
							Destination: &istio.Destination{
								Host:   "testapp.testnamespace.svc.cluster.local",
								Port:   &istio.PortSelector{Number: uint32(80)},
								Subset: "testtag",
							},
							Weight: 100,
						},
					},
					Retries: &istio.HTTPRetry{
						Attempts:      2,
						RetryOn:       defaultRetryOn,
						PerTryTimeout: types.DurationProto(time.Duration(3000000) * time.Nanosecond),
					},
					Timeout: types.DurationProto(time.Duration(3000000) * time.Nanosecond),
				},
				{ // Tagged status route
					Name: "00_tagged-testtag-status",
					Match: []*istio.HTTPMatchRequest{
						{
							Gateways: []string{"mesh"},
							Headers: map[string]*istio.StringMatch{
								"TEST-TAG": {
									MatchType: &istio.StringMatch_Exact{Exact: "testtag"},
								},
							},
							Port: uint32(4242),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
					},
					Route: []*istio.HTTPRouteDestination{
						{
							Destination: &istio.Destination{
								Host:   "testapp.testnamespace.svc.cluster.local",
								Port:   &istio.PortSelector{Number: uint32(4242)},
								Subset: "testtag",
							},
							Weight: 100,
						},
					},
				},
				{ // Release http route
					Name: "01_release-http",
					Match: []*istio.HTTPMatchRequest{
						{
							Gateways: []string{"mesh"},
							Port:     uint32(80),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
						{
							Authority: authorityRegex,
							Gateways:  []string{"private-gateway"},
							Port:      uint32(443),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
						{
							Authority: authorityRegex,
							Gateways:  []string{"private-gateway"},
							Port:      uint32(80),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
					},
					Route: []*istio.HTTPRouteDestination{
						{
							Destination: &istio.Destination{
								Host:   "testapp.testnamespace.svc.cluster.local",
								Port:   &istio.PortSelector{Number: uint32(80)},
								Subset: "testtag",
							},
							Weight: 100,
						},
					},
					Retries: &istio.HTTPRetry{
						Attempts:      2,
						RetryOn:       defaultRetryOn,
						PerTryTimeout: types.DurationProto(time.Duration(3000000) * time.Nanosecond),
					},
					Timeout: types.DurationProto(time.Duration(3000000) * time.Nanosecond),
				},
				{ // release status route
					Name: "01_release-status",
					Match: []*istio.HTTPMatchRequest{
						{
							Gateways: []string{"mesh"},
							Port:     uint32(4242),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
					},
					Route: []*istio.HTTPRouteDestination{
						{
							Destination: &istio.Destination{
								Host:   "testapp.testnamespace.svc.cluster.local",
								Port:   &istio.PortSelector{Number: uint32(4242)},
								Subset: "testtag",
							},
							Weight: 100,
						},
					},
				},
			},
		},
	}

	defaultExpectedService = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testapp",
			Namespace: "testnamespace",
			Labels: map[string]string{
				"prometheus.io/scrape": "true",
				"test":                 "label",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					Protocol:   "TCP",
					TargetPort: intstr.FromString("http"),
				},
				{
					Name:       "status",
					Port:       4242,
					Protocol:   "TCP",
					TargetPort: intstr.FromString("status"),
				},
			},
			Selector: map[string]string{
				"picchu.medium.engineering/app": "testapp",
			},
		},
	}

	defaultExpectedDestinationRule = &istioclient.DestinationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testapp",
			Namespace: "testnamespace",
			Labels: map[string]string{
				"test": "label",
			},
		},
		Spec: istio.DestinationRule{
			Host: "testapp.testnamespace.svc.cluster.local",
			Subsets: []*istio.Subset{
				{
					Name:   "testtag",
					Labels: map[string]string{"tag.picchu.medium.engineering": "testtag"},
					TrafficPolicy: &istio.TrafficPolicy{
						ConnectionPool: &istio.ConnectionPoolSettings{
							Tcp: &istio.ConnectionPoolSettings_TCPSettings{
								MaxConnections: 100,
							},
						},
					},
				},
			},
		},
	}

	defaultExpectedIstioSidecar = &istioclient.Sidecar{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testapp",
			Namespace: "testnamespace",
			Labels: map[string]string{
				"test": "label",
			},
		},
		Spec: istio.Sidecar{
			Egress: []*istio.IstioEgressListener{
				{
					Hosts: []string{"istio-system/*"},
				},
			},
		},
	}

	defaultPrometheusRule = &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testapp",
			Namespace: "testnamespace",
			Labels: map[string]string{
				picchuv1alpha1.LabelApp: "testapp",
				"test":                  "label",
			},
		},
		Spec: monitoringv1.PrometheusRuleSpec{Groups: []monitoringv1.RuleGroup{
			{
				Name: "picchu.rules",
				Rules: []monitoringv1.Rule{
					{
						Expr: intstr.IntOrString{
							Type:   intstr.String,
							StrVal: "hello world",
						},
					},
				},
			},
		}},
	}

	defaultRetryOn = "5xx"
)

func TestSyncNewApp(t *testing.T) {
	assert := testify.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	log := test.MustNewLogger()
	cli := fakeClient()
	expected := []runtime.Object{
		defaultPrometheusRule,
		defaultExpectedService,
		defaultExpectedDestinationRule,
		defaultExpectedVirtualService,
		defaultExpectedIstioSidecar,
	}

	scalingFactor := 0.5
	cluster := &picchuv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-a",
		},
		Spec: picchuv1alpha1.ClusterSpec{
			ScalingFactor: &scalingFactor,
			Ingresses: picchuv1alpha1.ClusterIngresses{
				Public: picchuv1alpha1.IngressInfo{
					Gateway: "public-gateway",
				},
				Private: picchuv1alpha1.IngressInfo{
					Gateway:        "private-gateway",
					DefaultDomains: []string{"test-a.cluster.doki-pen.org", "doki-pen.org"},
				},
			},
		},
	}
	assert.NoError(defaultSyncAppPlan.Apply(ctx, cli, cluster, log), "Shouldn't return error.")

	for _, e := range expected {
		ktest.AssertMatch(ctx, t, cli, e)
	}
}

func TestDomains(t *testing.T) {
	assert := testify.New(t)
	ctx := context.TODO()
	log := test.MustNewLogger()
	cli := fakeClient()
	scalingFactor := 1.0
	cluster := &picchuv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-a",
		},
		Spec: picchuv1alpha1.ClusterSpec{
			ScalingFactor: &scalingFactor,
			Ingresses: picchuv1alpha1.ClusterIngresses{
				Public: picchuv1alpha1.IngressInfo{
					Gateway:        "public-ingressgateway",
					DefaultDomains: []string{"doki-pen.org"},
				},
				Private: picchuv1alpha1.IngressInfo{
					Gateway:        "private-ingressgateway",
					DefaultDomains: []string{"dkpn.io"},
				},
			},
		},
	}
	plan := SyncApp{
		App:       "website",
		Namespace: "website-production",
		Labels:    map[string]string{"test": "label"},
		DeployedRevisions: []Revision{
			{
				Tag:              "1",
				Weight:           100,
				TagRoutingHeader: "DOKI-TAG",
				TrafficPolicy: &istio.TrafficPolicy{
					ConnectionPool: &istio.ConnectionPoolSettings{
						Tcp: &istio.ConnectionPoolSettings_TCPSettings{
							MaxConnections: 100,
						},
					},
				},
			},
		},
		AlertRules: []monitoringv1.Rule{
			{
				Expr: intstr.FromString("hello world"),
			},
		},
		Ports: []picchuv1alpha1.PortInfo{
			{
				Name:          "http",
				IngressPort:   443,
				Port:          80,
				ContainerPort: 5000,
				Protocol:      corev1.ProtocolTCP,
				Mode:          picchuv1alpha1.PortPublic,
				Hosts:         []string{"www.doki-pen.org"},
			},
			{
				Name:          "grpc",
				IngressPort:   443,
				Port:          8080,
				ContainerPort: 5001,
				Protocol:      corev1.ProtocolTCP,
				Mode:          picchuv1alpha1.PortPrivate,
			},
			{
				Name:          "status",
				IngressPort:   443,
				Port:          4242,
				ContainerPort: 4444,
				Protocol:      corev1.ProtocolTCP,
				Mode:          picchuv1alpha1.PortLocal,
			},
		},
	}
	assert.NoError(plan.Apply(ctx, cli, cluster, log))
	vs := &istioclient.VirtualService{}
	key := client.ObjectKey{
		Namespace: "website-production",
		Name:      "website",
	}
	assert.NoError(cli.Get(ctx, key, vs))
	assert.Equal(map[string]string{
		"test": "label",
	}, vs.Labels)
	assert.ElementsMatch([]string{
		"public-ingressgateway",
		"private-ingressgateway",
		"mesh",
	}, vs.Spec.Gateways)
	assert.ElementsMatch([]string{
		"www.doki-pen.org",
		"website.doki-pen.org",
		"website-production.doki-pen.org",
		"website.dkpn.io",
		"website-production.dkpn.io",
		"website.website-production.svc.cluster.local"}, vs.Spec.Hosts)
}

func TestHosts(t *testing.T) {
	assert := testify.New(t)
	publicPort := picchuv1alpha1.PortInfo{
		Hosts: []string{"www.doki-pen.org", "website.doki-pen.org"},
		Mode:  picchuv1alpha1.PortPublic,
	}
	privatePort := picchuv1alpha1.PortInfo{
		Hosts: []string{"www.dkpn.io"},
		Mode:  picchuv1alpha1.PortPrivate,
	}
	scalingFactor := 1.0
	cluster := &picchuv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-a",
		},
		Spec: picchuv1alpha1.ClusterSpec{
			ScalingFactor: &scalingFactor,
			Ingresses: picchuv1alpha1.ClusterIngresses{
				Public: picchuv1alpha1.IngressInfo{
					Gateway:        "public-ingressgateway",
					DefaultDomains: []string{"doki-pen.org"},
				},
				Private: picchuv1alpha1.IngressInfo{
					Gateway:        "private-ingressgateway",
					DefaultDomains: []string{"dkpn.io"},
				},
			},
		},
	}
	plan := SyncApp{
		App:       "website",
		Target:    "staging",
		Fleet:     "internal",
		Namespace: "website-internal",
		Ports:     []picchuv1alpha1.PortInfo{publicPort, privatePort},
	}

	assert.Equal("website.website-internal.svc.cluster.local", plan.serviceHost())
	assert.ElementsMatch([]string{
		"www.doki-pen.org",
		"website-internal.doki-pen.org",
		"website.doki-pen.org",
	}, plan.publicHosts(publicPort, cluster))
	assert.ElementsMatch([]string{
		"www.doki-pen.org",
		"website-internal.dkpn.io",
		"website.dkpn.io",
		"website.doki-pen.org",
	}, plan.privateHosts(publicPort, cluster))
	assert.ElementsMatch([]string{
		"www.dkpn.io",
		"website-internal.doki-pen.org",
	}, plan.publicHosts(privatePort, cluster))
	assert.ElementsMatch([]string{
		"www.dkpn.io",
		"website-internal.dkpn.io",
		"website.dkpn.io",
	}, plan.privateHosts(privatePort, cluster))
}

func TestHostsWithVariantsEnabled(t *testing.T) {
	assert := testify.New(t)
	scalingFactor := 0.5
	cluster := &picchuv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-a",
		},
		Spec: picchuv1alpha1.ClusterSpec{
			ScalingFactor: &scalingFactor,
			Ingresses: picchuv1alpha1.ClusterIngresses{
				Public: picchuv1alpha1.IngressInfo{
					Gateway: "public-gateway",
				},
				Private: picchuv1alpha1.IngressInfo{
					Gateway:        "private-gateway",
					DefaultDomains: []string{"test-a.cluster.doki-pen.org", "doki-pen.org"},
				},
			},
		},
	}
	ctx := context.TODO()
	log := test.MustNewLogger()
	cli := fakeClient()

	plan := &SyncApp{
		App:       "testapp",
		Fleet:     "production",
		Target:    "production",
		Namespace: "testnamespace",
		Labels: map[string]string{
			"test": "label",
		},
		DeployedRevisions: []Revision{
			{
				Tag:              "testtag",
				Weight:           100,
				TagRoutingHeader: "TEST-TAG",
				TrafficPolicy: &istio.TrafficPolicy{
					ConnectionPool: &istio.ConnectionPoolSettings{
						Tcp: &istio.ConnectionPoolSettings_TCPSettings{
							MaxConnections: 100,
						},
					},
				},
			},
		},
		AlertRules: []monitoringv1.Rule{
			{
				Expr: intstr.FromString("hello world"),
			},
		},
		DefaultIngressPorts: map[string]string{
			"public":  "http",
			"private": "http",
		},
		Ports: []picchuv1alpha1.PortInfo{
			{
				Name:          "http",
				IngressPort:   443,
				Port:          80,
				ContainerPort: 5000,
				Protocol:      corev1.ProtocolTCP,
				Ingresses:     []string{"private"},
				Istio: picchuv1alpha1.IstioPortConfig{
					HTTP: picchuv1alpha1.IstioHTTPPortConfig{
						Retries: &picchuv1alpha1.Retries{
							RetryOn: &defaultRetryOn,
							PerTryTimeout: &metav1.Duration{
								Duration: time.Duration(3) * time.Millisecond,
							},
							Attempts: 2,
						},
						Timeout: &metav1.Duration{
							Duration: time.Duration(3) * time.Millisecond,
						},
					},
				},
			},
			{
				Name:          "status",
				IngressPort:   443,
				Port:          4242,
				ContainerPort: 4444,
				Protocol:      corev1.ProtocolTCP,
				Mode:          picchuv1alpha1.PortLocal,
			},
		},
		DefaultVariant:   true,
		IngressesVariant: true,
	}

	expected := &istioclient.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testapp",
			Namespace: "testnamespace",
			Labels: map[string]string{
				"test": "label",
			},
		},
		Spec: istio.VirtualService{
			Hosts: []string{
				"testapp.doki-pen.org",
				"testapp.test-a.cluster.doki-pen.org",
				"testapp.testnamespace.svc.cluster.local",
				"testnamespace.doki-pen.org",
				"testnamespace.test-a.cluster.doki-pen.org",
			},
			Gateways: []string{
				"mesh",
				"private-gateway",
			},
			Http: []*istio.HTTPRoute{
				{ // Tagged http route
					Name: "00_tagged-testtag-http",
					Match: []*istio.HTTPMatchRequest{
						{
							Gateways: []string{"mesh"},
							Headers: map[string]*istio.StringMatch{
								"TEST-TAG": {
									MatchType: &istio.StringMatch_Exact{Exact: "testtag"},
								},
							},
							Port: uint32(80),
							Uri:  &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}},
						},
						{
							Gateways: []string{"private-gateway"},
							Headers: map[string]*istio.StringMatch{
								"TEST-TAG": {
									MatchType: &istio.StringMatch_Exact{Exact: "testtag"},
								},
							},
							Authority: authorityRegex,
							Port:      uint32(443),
							Uri:       &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}},
						},
						{
							Gateways: []string{"private-gateway"},
							Headers: map[string]*istio.StringMatch{
								"TEST-TAG": {
									MatchType: &istio.StringMatch_Exact{Exact: "testtag"},
								},
							},
							Authority: authorityRegex,
							Port:      uint32(80),
							Uri:       &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}},
						},
					},
					Route: []*istio.HTTPRouteDestination{
						{
							Destination: &istio.Destination{
								Host:   "testapp.testnamespace.svc.cluster.local",
								Port:   &istio.PortSelector{Number: uint32(80)},
								Subset: "testtag",
							},
							Weight: 100,
						},
					},
					Retries: &istio.HTTPRetry{
						Attempts:      2,
						RetryOn:       defaultRetryOn,
						PerTryTimeout: types.DurationProto(time.Duration(3000000) * time.Nanosecond),
					},
					Timeout: types.DurationProto(time.Duration(3000000) * time.Nanosecond),
				},
				{ // Tagged status route
					Name: "00_tagged-testtag-status",
					Match: []*istio.HTTPMatchRequest{
						{
							Gateways: []string{"mesh"},
							Headers: map[string]*istio.StringMatch{
								"TEST-TAG": {
									MatchType: &istio.StringMatch_Exact{Exact: "testtag"},
								},
							},
							Port: uint32(4242),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
					},
					Route: []*istio.HTTPRouteDestination{
						{
							Destination: &istio.Destination{
								Host:   "testapp.testnamespace.svc.cluster.local",
								Port:   &istio.PortSelector{Number: uint32(4242)},
								Subset: "testtag",
							},
							Weight: 100,
						},
					},
				},
				{ // Release http route
					Name: "01_release-http",
					Match: []*istio.HTTPMatchRequest{
						{
							Gateways: []string{"mesh"},
							Port:     uint32(80),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
						{
							Authority: authorityRegex,
							Gateways:  []string{"private-gateway"},
							Port:      uint32(443),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
						{
							Authority: authorityRegex,
							Gateways:  []string{"private-gateway"},
							Port:      uint32(80),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
					},
					Route: []*istio.HTTPRouteDestination{
						{
							Destination: &istio.Destination{
								Host:   "testapp.testnamespace.svc.cluster.local",
								Port:   &istio.PortSelector{Number: uint32(80)},
								Subset: "testtag",
							},
							Weight: 100,
						},
					},
					Retries: &istio.HTTPRetry{
						Attempts:      2,
						RetryOn:       defaultRetryOn,
						PerTryTimeout: types.DurationProto(time.Duration(3000000) * time.Nanosecond),
					},
					Timeout: types.DurationProto(time.Duration(3000000) * time.Nanosecond),
				},
				{ // release status route
					Name: "01_release-status",
					Match: []*istio.HTTPMatchRequest{
						{
							Gateways: []string{"mesh"},
							Port:     uint32(4242),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
					},
					Route: []*istio.HTTPRouteDestination{
						{
							Destination: &istio.Destination{
								Host:   "testapp.testnamespace.svc.cluster.local",
								Port:   &istio.PortSelector{Number: uint32(4242)},
								Subset: "testtag",
							},
							Weight: 100,
						},
					},
				},
			},
		},
	}
	assert.NoError(plan.Apply(ctx, cli, cluster, log))
	ktest.AssertMatch(ctx, t, cli, expected)
}

func TestProductionEcho(t *testing.T) {
	assert := testify.New(t)
	scalingFactor := 1.0
	cluster := &picchuv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "production-reef-a",
			Labels: map[string]string{
				"picchu.medium.engineering/fleet": "production",
				"shunt.medium.build/edition":      "reef",
			},
		},
		Spec: picchuv1alpha1.ClusterSpec{
			Enabled:       true,
			ScalingFactor: &scalingFactor,
			Ingresses: picchuv1alpha1.ClusterIngresses{
				Public: picchuv1alpha1.IngressInfo{
					Gateway: "public-ingressgateway.istio-system.svc.cluster.local",
				},
				Private: picchuv1alpha1.IngressInfo{
					Gateway: "private-ingressgateway.istio-system.svc.cluster.local",
					DefaultDomains: []string{
						"medm.io",
						"production.medm.io",
						"production.medm.cloud",
						"production-reef.medm.io",
						"production-reef.medm.cloud",
					},
				},
			},
		},
	}
	ctx := context.TODO()
	log := test.MustNewLogger()
	cli := fakeClient(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "echo-production",
		},
	})

	plan := &SyncApp{
		App:       "echo",
		Target:    "production",
		Fleet:     "production",
		Namespace: "echo-production",
		Labels: map[string]string{
			"app.kubernetes.io/name":              "echo",
			"picchu.medium.engineering/ownerName": "echo-production",
			"picchu.medium.engineering/ownerType": "releasemanager",
			"picchu.medium.engineering/app":       "echo",
		},
		DeployedRevisions: []Revision{
			{
				TrafficPolicy: &istio.TrafficPolicy{
					PortLevelSettings: []*istio.TrafficPolicy_PortTrafficPolicy{
						{
							Port: &istio.PortSelector{
								Number: 4242,
							},
						},
					},
					ConnectionPool: &istio.ConnectionPoolSettings{
						Http: &istio.ConnectionPoolSettings_HTTPSettings{
							Http2MaxRequests: 10,
						},
					},
				},
				Tag:              "main-20200529-144642-c3d06a9828",
				Weight:           100,
				TagRoutingHeader: "echo-tag",
			},
		},
		Ports: []picchuv1alpha1.PortInfo{
			{
				Name:          "http",
				IngressPort:   443,
				Port:          80,
				ContainerPort: 8080,
				Protocol:      corev1.ProtocolTCP,
				Ingresses:     []string{"private", "public"},
				Mode:          picchuv1alpha1.PortPublic,
			},
			{
				Name:          "status",
				IngressPort:   443,
				Port:          4242,
				ContainerPort: 8081,
				Protocol:      corev1.ProtocolTCP,
				Mode:          picchuv1alpha1.PortLocal,
				Istio: picchuv1alpha1.IstioPortConfig{
					HTTP: picchuv1alpha1.IstioHTTPPortConfig{
						Retries: &picchuv1alpha1.Retries{
							Attempts: 2,
						},
					},
				},
			},
		},
		DefaultIngressPorts: map[string]string{
			"public":  "http",
			"private": "http",
		},
		DefaultVariant:   true,
		IngressesVariant: true,
	}

	regex := &istio.StringMatch{
		MatchType: &istio.StringMatch_Regex{
			Regex: "^(echo-production\\.medm\\.io|echo-production\\.production-reef\\.medm\\.cloud|echo-production\\.production-reef\\.medm\\.io|echo-production\\.production\\.medm\\.cloud|echo-production\\.production\\.medm\\.io|echo\\.medm\\.io|echo\\.production-reef\\.medm\\.cloud|echo\\.production-reef\\.medm\\.io|echo\\.production\\.medm\\.cloud|echo\\.production\\.medm\\.io)(:[0-9]+)?$",
		},
	}

	expected := &istioclient.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "echo",
			Namespace: "echo-production",
			Labels: map[string]string{
				"app.kubernetes.io/name":              "echo",
				"picchu.medium.engineering/ownerName": "echo-production",
				"picchu.medium.engineering/ownerType": "releasemanager",
				"picchu.medium.engineering/app":       "echo",
			},
		},
		Spec: istio.VirtualService{
			Hosts: []string{
				"echo-production.medm.io",
				"echo-production.production-reef.medm.cloud",
				"echo-production.production-reef.medm.io",
				"echo-production.production.medm.cloud",
				"echo-production.production.medm.io",
				"echo.echo-production.svc.cluster.local",
				"echo.medm.io",
				"echo.production-reef.medm.cloud",
				"echo.production-reef.medm.io",
				"echo.production.medm.cloud",
				"echo.production.medm.io",
			},
			Gateways: []string{
				"mesh",
				"private-ingressgateway.istio-system.svc.cluster.local",
			},
			Http: []*istio.HTTPRoute{
				{ // Tagged http route
					Name: "00_tagged-main-20200529-144642-c3d06a9828-http",
					Match: []*istio.HTTPMatchRequest{
						{
							Gateways: []string{"mesh"},
							Headers: map[string]*istio.StringMatch{
								"echo-tag": {
									MatchType: &istio.StringMatch_Exact{Exact: "main-20200529-144642-c3d06a9828"},
								},
							},
							Port: uint32(80),
							Uri:  &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}},
						},
						{
							Gateways: []string{"private-ingressgateway.istio-system.svc.cluster.local"},
							Headers: map[string]*istio.StringMatch{
								"echo-tag": {
									MatchType: &istio.StringMatch_Exact{Exact: "main-20200529-144642-c3d06a9828"},
								},
							},
							Authority: regex,
							Port:      uint32(443),
							Uri:       &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}},
						},
						{
							Gateways: []string{"private-ingressgateway.istio-system.svc.cluster.local"},
							Headers: map[string]*istio.StringMatch{
								"echo-tag": {
									MatchType: &istio.StringMatch_Exact{Exact: "main-20200529-144642-c3d06a9828"},
								},
							},
							Authority: regex,
							Port:      uint32(80),
							Uri:       &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}},
						},
					},
					Route: []*istio.HTTPRouteDestination{
						{
							Destination: &istio.Destination{
								Host:   "echo.echo-production.svc.cluster.local",
								Port:   &istio.PortSelector{Number: uint32(80)},
								Subset: "main-20200529-144642-c3d06a9828",
							},
							Weight: 100,
						},
					},
				},
				{ // Tagged status route
					Name: "00_tagged-main-20200529-144642-c3d06a9828-status",
					Match: []*istio.HTTPMatchRequest{
						{
							Gateways: []string{"mesh"},
							Headers: map[string]*istio.StringMatch{
								"echo-tag": {
									MatchType: &istio.StringMatch_Exact{Exact: "main-20200529-144642-c3d06a9828"},
								},
							},
							Port: uint32(4242),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
					},
					Route: []*istio.HTTPRouteDestination{
						{
							Destination: &istio.Destination{
								Host:   "echo.echo-production.svc.cluster.local",
								Port:   &istio.PortSelector{Number: uint32(4242)},
								Subset: "main-20200529-144642-c3d06a9828",
							},
							Weight: 100,
						},
					},
					Retries: &istio.HTTPRetry{
						Attempts: 2,
					},
				},
				{ // Release http route
					Name: "01_release-http",
					Match: []*istio.HTTPMatchRequest{
						{
							Gateways: []string{"mesh"},
							Port:     uint32(80),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
						{
							Authority: regex,
							Gateways:  []string{"private-ingressgateway.istio-system.svc.cluster.local"},
							Port:      uint32(443),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
						{
							Authority: regex,
							Gateways:  []string{"private-ingressgateway.istio-system.svc.cluster.local"},
							Port:      uint32(80),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
					},
					Route: []*istio.HTTPRouteDestination{
						{
							Destination: &istio.Destination{
								Host:   "echo.echo-production.svc.cluster.local",
								Port:   &istio.PortSelector{Number: uint32(80)},
								Subset: "main-20200529-144642-c3d06a9828",
							},
							Weight: 100,
						},
					},
				},
				{ // release status route
					Name: "01_release-status",
					Match: []*istio.HTTPMatchRequest{
						{
							Gateways: []string{"mesh"},
							Port:     uint32(4242),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
					},
					Route: []*istio.HTTPRouteDestination{
						{
							Destination: &istio.Destination{
								Host:   "echo.echo-production.svc.cluster.local",
								Port:   &istio.PortSelector{Number: uint32(4242)},
								Subset: "main-20200529-144642-c3d06a9828",
							},
							Weight: 100,
						},
					},
					Retries: &istio.HTTPRetry{
						Attempts: 2,
					},
				},
			},
		},
	}
	assert.NoError(plan.Apply(ctx, cli, cluster, log))
	ktest.AssertMatch(ctx, t, cli, expected)
}

func TestDevRoutes(t *testing.T) {
	assert := testify.New(t)
	scalingFactor := 1.0
	cluster := &picchuv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "staging-reef-a",
			Labels: map[string]string{
				"picchu.medium.engineering/fleet": "staging",
				"shunt.medium.build/edition":      "reef",
			},
		},
		Spec: picchuv1alpha1.ClusterSpec{
			Enabled:             true,
			EnableDevRoutes:     true,               // what this function is testing
			DevRouteTagTemplate: "dev-{{.App}}-tag", // what this function is testing
			ScalingFactor:       &scalingFactor,
			Ingresses: picchuv1alpha1.ClusterIngresses{
				Public: picchuv1alpha1.IngressInfo{
					Gateway: "public-ingressgateway.istio-system.svc.cluster.local",
				},
				Private: picchuv1alpha1.IngressInfo{
					Gateway: "private-ingressgateway.istio-system.svc.cluster.local",
					DefaultDomains: []string{
						"medm.io",
						"staging.medm.io",
					},
				},
			},
		},
	}
	ctx := context.TODO()
	log := test.MustNewLogger()
	cli := fakeClient(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "echo-staging",
		},
	})

	plan := &SyncApp{
		App:       "echo",
		Target:    "staging",
		Fleet:     "staging",
		Namespace: "echo-staging",
		Labels: map[string]string{
			"app.kubernetes.io/name":              "echo",
			"picchu.medium.engineering/ownerName": "echo-staging",
			"picchu.medium.engineering/ownerType": "releasemanager",
			"picchu.medium.engineering/app":       "echo",
		},
		DeployedRevisions: []Revision{
			{
				TrafficPolicy: &istio.TrafficPolicy{
					PortLevelSettings: []*istio.TrafficPolicy_PortTrafficPolicy{
						{
							Port: &istio.PortSelector{
								Number: 4242,
							},
						},
					},
					ConnectionPool: &istio.ConnectionPoolSettings{
						Http: &istio.ConnectionPoolSettings_HTTPSettings{
							Http2MaxRequests: 10,
						},
					},
				},
				Tag:              "main-20200529-144642-c3d06a9828",
				Weight:           100,
				TagRoutingHeader: "echo-tag",
			},
		},
		Ports: []picchuv1alpha1.PortInfo{
			{
				Name:          "http",
				IngressPort:   443,
				Port:          80,
				ContainerPort: 8080,
				Protocol:      corev1.ProtocolTCP,
				Ingresses:     []string{"private", "public"},
				Mode:          picchuv1alpha1.PortPublic,
			},
			{
				Name:          "status",
				IngressPort:   443,
				Port:          4242,
				ContainerPort: 8081,
				Protocol:      corev1.ProtocolTCP,
				Mode:          picchuv1alpha1.PortLocal,
				Istio: picchuv1alpha1.IstioPortConfig{
					HTTP: picchuv1alpha1.IstioHTTPPortConfig{
						Retries: &picchuv1alpha1.Retries{
							Attempts: 2,
						},
					},
				},
			},
		},
		DefaultIngressPorts: map[string]string{
			"public":  "http",
			"private": "http",
		},
		DefaultVariant:       true,
		IngressesVariant:     true,
		DevRoutesServiceHost: "dev-routes-service-host",
		DevRoutesServicePort: 80,
	}

	regex := &istio.StringMatch{
		MatchType: &istio.StringMatch_Regex{
			Regex: "^(echo-staging\\.medm\\.io|echo-staging\\.staging\\.medm\\.io|echo\\.medm\\.io|echo\\.staging\\.medm\\.io)(:[0-9]+)?$",
		},
	}

	expected := &istioclient.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "echo",
			Namespace: "echo-staging",
			Labels: map[string]string{
				"app.kubernetes.io/name":              "echo",
				"picchu.medium.engineering/ownerName": "echo-staging",
				"picchu.medium.engineering/ownerType": "releasemanager",
				"picchu.medium.engineering/app":       "echo",
			},
		},
		Spec: istio.VirtualService{
			Hosts: []string{
				"echo-staging.medm.io",
				"echo-staging.staging.medm.io",
				"echo.echo-staging.svc.cluster.local",
				"echo.medm.io",
				"echo.staging.medm.io",
			},
			Gateways: []string{
				"mesh",
				"private-ingressgateway.istio-system.svc.cluster.local",
			},
			Http: []*istio.HTTPRoute{
				{ // dev route
					Name: "00_dev-echo",
					Match: []*istio.HTTPMatchRequest{
						{
							Headers: map[string]*istio.StringMatch{
								"dev-echo-tag": {
									MatchType: &istio.StringMatch_Regex{Regex: "(.+)"},
								},
							},
						},
					},
					Route: []*istio.HTTPRouteDestination{
						{
							Destination: &istio.Destination{
								Host: "dev-routes-service-host",
								Port: &istio.PortSelector{Number: uint32(80)},
							},
							Weight: 100,
						},
					},
				},
				{ // Tagged http route
					Name: "00_tagged-main-20200529-144642-c3d06a9828-http",
					Match: []*istio.HTTPMatchRequest{
						{
							Gateways: []string{"mesh"},
							Headers: map[string]*istio.StringMatch{
								"echo-tag": {
									MatchType: &istio.StringMatch_Exact{Exact: "main-20200529-144642-c3d06a9828"},
								},
							},
							Port: uint32(80),
							Uri:  &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}},
						},
						{
							Gateways: []string{"private-ingressgateway.istio-system.svc.cluster.local"},
							Headers: map[string]*istio.StringMatch{
								"echo-tag": {
									MatchType: &istio.StringMatch_Exact{Exact: "main-20200529-144642-c3d06a9828"},
								},
							},
							Authority: regex,
							Port:      uint32(443),
							Uri:       &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}},
						},
						{
							Gateways: []string{"private-ingressgateway.istio-system.svc.cluster.local"},
							Headers: map[string]*istio.StringMatch{
								"echo-tag": {
									MatchType: &istio.StringMatch_Exact{Exact: "main-20200529-144642-c3d06a9828"},
								},
							},
							Authority: regex,
							Port:      uint32(80),
							Uri:       &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}},
						},
					},
					Route: []*istio.HTTPRouteDestination{
						{
							Destination: &istio.Destination{
								Host:   "echo.echo-staging.svc.cluster.local",
								Port:   &istio.PortSelector{Number: uint32(80)},
								Subset: "main-20200529-144642-c3d06a9828",
							},
							Weight: 100,
						},
					},
				},
				{ // Tagged status route
					Name: "00_tagged-main-20200529-144642-c3d06a9828-status",
					Match: []*istio.HTTPMatchRequest{
						{
							Gateways: []string{"mesh"},
							Headers: map[string]*istio.StringMatch{
								"echo-tag": {
									MatchType: &istio.StringMatch_Exact{Exact: "main-20200529-144642-c3d06a9828"},
								},
							},
							Port: uint32(4242),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
					},
					Route: []*istio.HTTPRouteDestination{
						{
							Destination: &istio.Destination{
								Host:   "echo.echo-staging.svc.cluster.local",
								Port:   &istio.PortSelector{Number: uint32(4242)},
								Subset: "main-20200529-144642-c3d06a9828",
							},
							Weight: 100,
						},
					},
					Retries: &istio.HTTPRetry{
						Attempts: 2,
					},
				},
				{ // Release http route
					Name: "01_release-http",
					Match: []*istio.HTTPMatchRequest{
						{
							Gateways: []string{"mesh"},
							Port:     uint32(80),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
						{
							Authority: regex,
							Gateways:  []string{"private-ingressgateway.istio-system.svc.cluster.local"},
							Port:      uint32(443),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
						{
							Authority: regex,
							Gateways:  []string{"private-ingressgateway.istio-system.svc.cluster.local"},
							Port:      uint32(80),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
					},
					Route: []*istio.HTTPRouteDestination{
						{
							Destination: &istio.Destination{
								Host:   "echo.echo-staging.svc.cluster.local",
								Port:   &istio.PortSelector{Number: uint32(80)},
								Subset: "main-20200529-144642-c3d06a9828",
							},
							Weight: 100,
						},
					},
				},
				{ // release status route
					Name: "01_release-status",
					Match: []*istio.HTTPMatchRequest{
						{
							Gateways: []string{"mesh"},
							Port:     uint32(4242),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
					},
					Route: []*istio.HTTPRouteDestination{
						{
							Destination: &istio.Destination{
								Host:   "echo.echo-staging.svc.cluster.local",
								Port:   &istio.PortSelector{Number: uint32(4242)},
								Subset: "main-20200529-144642-c3d06a9828",
							},
							Weight: 100,
						},
					},
					Retries: &istio.HTTPRetry{
						Attempts: 2,
					},
				},
			},
		},
	}
	assert.NoError(plan.Apply(ctx, cli, cluster, log))
	ktest.AssertMatch(ctx, t, cli, expected)
}
