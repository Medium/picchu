package plan

import (
	"context"
	_ "runtime"
	"testing"
	"time"

	ktest "go.medium.engineering/kubernetes/pkg/test"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/test"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
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
				TagRoutingHeader: "MEDIUM-TAG",
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
					Match: []*istio.HTTPMatchRequest{
						{
							Gateways: []string{"mesh"},
							Headers: map[string]*istio.StringMatch{
								"MEDIUM-TAG": {
									MatchType: &istio.StringMatch_Exact{Exact: "testtag"},
								},
							},
							Port: uint32(80),
							Uri:  &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}},
						},
						{
							Gateways: []string{"private-gateway"},
							Headers: map[string]*istio.StringMatch{
								"MEDIUM-TAG": {
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
								"MEDIUM-TAG": {
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
					Match: []*istio.HTTPMatchRequest{
						{
							Gateways: []string{"mesh"},
							Headers: map[string]*istio.StringMatch{
								"MEDIUM-TAG": {
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
				TagRoutingHeader: "MEDIUM-TAG",
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
				Default:       true,
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
					Match: []*istio.HTTPMatchRequest{
						{
							Gateways: []string{"mesh"},
							Headers: map[string]*istio.StringMatch{
								"MEDIUM-TAG": {
									MatchType: &istio.StringMatch_Exact{Exact: "testtag"},
								},
							},
							Port: uint32(80),
							Uri:  &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}},
						},
						{
							Gateways: []string{"private-gateway"},
							Headers: map[string]*istio.StringMatch{
								"MEDIUM-TAG": {
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
								"MEDIUM-TAG": {
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
					Match: []*istio.HTTPMatchRequest{
						{
							Gateways: []string{"mesh"},
							Headers: map[string]*istio.StringMatch{
								"MEDIUM-TAG": {
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
