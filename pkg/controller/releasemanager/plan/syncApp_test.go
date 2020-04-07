package plan

import (
	"context"
	"go.medium.engineering/picchu/pkg/client/scheme"
	_ "runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/mocks"
	pkgplan "go.medium.engineering/picchu/pkg/plan"
	common "go.medium.engineering/picchu/pkg/plan/test"
	"go.medium.engineering/picchu/pkg/test"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
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
			Regex: "^(testnamespace\\.doki-pen\\.org|testnamespace\\.test-a\\.cluster\\.doki-pen\\.org)(:[0-9]+)?$",
		},
	}
	defaultSyncAppPlan = &SyncApp{
		App:       "testapp",
		Namespace: "testnamespace",
		Labels: map[string]string{
			"test": "label",
		},
		PublicGateway:  "public-gateway",
		PrivateGateway: "private-gateway",
		DeployedRevisions: []Revision{
			{
				Tag:              "testtag",
				Weight:           100,
				TagRoutingHeader: "MEDIUM-TAG",
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
				HttpsRedirect: true,
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
				},
			},
		},
	}
)

func TestSyncNewApp(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	ok := client.ObjectKey{Name: "testapp", Namespace: "testnamespace"}
	ctx := context.TODO()

	m.
		EXPECT().
		Get(ctx, mocks.ObjectKey(ok), gomock.Any()).
		Return(common.NotFoundError).
		Times(4)

	m.
		EXPECT().
		Create(ctx, mocks.Kind("PrometheusRule")).
		Return(nil).
		Times(1)

	for _, obj := range []runtime.Object{
		defaultExpectedService,
		defaultExpectedDestinationRule,
		defaultExpectedVirtualService,
	} {
		m.
			EXPECT().
			Create(ctx, common.K8sEqual(obj)).
			Return(nil).
			Times(1)
	}

	options := pkgplan.Options{
		ScalingFactor: 0.5,
		ClusterName:   "test-a",
		DefaultDomains: pkgplan.DefaultDomainOptions{
			Public:  []string{""},
			Private: []string{"test-a.cluster.doki-pen.org", "doki-pen.org"},
		},
	}
	assert.NoError(t, defaultSyncAppPlan.Apply(ctx, m, options, log), "Shouldn't return error.")
}

func TestDomains(t *testing.T) {
	ctx := context.TODO()
	log := test.MustNewLogger()
	istioclient.AddToScheme(scheme.Scheme)
	corev1.AddToScheme(scheme.Scheme)
	monitoringv1.AddToScheme(scheme.Scheme)
	cli := fake.NewFakeClientWithScheme(scheme.Scheme)
	options := pkgplan.Options{
		ClusterName:   "",
		ScalingFactor: 0,
		DefaultDomains: pkgplan.DefaultDomainOptions{
			Public:  []string{"doki-pen.org"},
			Private: []string{"dkpn.io"},
		},
	}
	plan := SyncApp{
		App:            "website",
		Namespace:      "website-production",
		Labels:         map[string]string{"test": "label"},
		PublicGateway:  "public-ingressgateway",
		PrivateGateway: "private-ingressgateway",
		DeployedRevisions: []Revision{
			{
				Tag:              "1",
				Weight:           100,
				TagRoutingHeader: "DOKI-TAG",
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
				HttpsRedirect: true,
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
	assert.NoError(t, plan.Apply(ctx, cli, options, log))
	vs := &istioclient.VirtualService{}
	assert.NoError(t, cli.Get(ctx, client.ObjectKey{"website-production", "website"}, vs))
	assert.Equal(t, map[string]string{
		"test": "label",
	}, vs.Labels)
	assert.ElementsMatch(t, []string{
		"public-ingressgateway",
		"private-ingressgateway",
		"mesh",
	}, vs.Spec.Gateways)
	assert.ElementsMatch(t, []string{
		"www.doki-pen.org",
		"website-production.doki-pen.org",
		"website-production.dkpn.io",
		"website.website-production.svc.cluster.local"}, vs.Spec.Hosts)
}

func TestHosts(t *testing.T) {
	publicPort := picchuv1alpha1.PortInfo{
		Hosts: []string{"www.doki-pen.org"},
		Mode:  picchuv1alpha1.PortPublic,
	}
	privatePort := picchuv1alpha1.PortInfo{
		Hosts: []string{"www.dkpn.io"},
		Mode:  picchuv1alpha1.PortPrivate,
	}
	options := pkgplan.Options{
		DefaultDomains: pkgplan.DefaultDomainOptions{
			Public:  []string{"doki-pen.org"},
			Private: []string{"dkpn.io"},
		},
	}
	plan := SyncApp{
		App:       "website",
		Namespace: "website-production",
	}

	assert.Equal(t, "website.website-production.svc.cluster.local", plan.serviceHost())
	assert.ElementsMatch(t, []string{
		"www.doki-pen.org",
		"website-production.doki-pen.org",
	}, plan.publicHosts(publicPort, options))
	assert.ElementsMatch(t, []string{
		"website-production.dkpn.io",
	}, plan.privateHosts(publicPort, options))
	assert.ElementsMatch(t, []string{
		"website-production.doki-pen.org",
	}, plan.publicHosts(privatePort, options))
	assert.ElementsMatch(t, []string{
		"www.dkpn.io",
		"website-production.dkpn.io",
	}, plan.privateHosts(privatePort, options))
}
