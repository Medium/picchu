package plan

import (
	"context"
	_ "runtime"
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
	defaultSyncAppPlan = &SyncApp{
		App:       "testapp",
		Namespace: "testnamespace",
		Labels: map[string]string{
			"test": "label",
		},
		DefaultDomains: []string{"doki-pen.org"},
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
				"testnamespace-grpc.doki-pen.org",
				"testnamespace-http.doki-pen.org",
				"testnamespace.doki-pen.org",
			},
			Gateways: []string{
				"mesh",
				"public-gateway",
				"private-gateway",
			},
			Http: []*istio.HTTPRoute{
				{
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
							Port: uint32(443),
							Uri:  &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}},
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
				{
					Match: []*istio.HTTPMatchRequest{
						{
							Gateways: []string{"mesh"},
							Headers: map[string]*istio.StringMatch{
								"MEDIUM-TAG": {
									MatchType: &istio.StringMatch_Exact{Exact: "testtag"},
								},
							},
							Port: uint32(8080),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
						{
							Gateways: []string{"private-gateway"},
							Headers: map[string]*istio.StringMatch{
								"MEDIUM-TAG": {
									MatchType: &istio.StringMatch_Exact{Exact: "testtag"},
								},
							},
							Port: uint32(443),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
					},
					Route: []*istio.HTTPRouteDestination{
						{
							Destination: &istio.Destination{
								Host:   "testapp.testnamespace.svc.cluster.local",
								Port:   &istio.PortSelector{Number: uint32(8080)},
								Subset: "testtag",
							},
							Weight: 100,
						},
					},
				},
				{
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
				{
					Match: []*istio.HTTPMatchRequest{
						{
							Gateways: []string{"mesh"},
							Port:     uint32(80),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
						{
							Authority: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "testnamespace.doki-pen.org"},
							},
							Gateways: []string{"private-gateway"},
							Port:     uint32(443),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
						{
							Authority: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "testnamespace.doki-pen.org"},
							},
							Gateways: []string{"private-gateway"},
							Port:     uint32(80),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
						{
							Authority: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "testnamespace-http.doki-pen.org"},
							},
							Gateways: []string{"private-gateway"},
							Port:     uint32(443),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
						{
							Authority: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "testnamespace-http.doki-pen.org"},
							},
							Gateways: []string{"private-gateway"},
							Port:     uint32(80),
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
				{
					Match: []*istio.HTTPMatchRequest{
						{
							Gateways: []string{"mesh"},
							Port:     uint32(8080),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
						{
							Authority: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "testnamespace.doki-pen.org"},
							},
							Gateways: []string{"private-gateway"},
							Port:     uint32(443),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
						{
							Authority: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "testnamespace-grpc.doki-pen.org"},
							},
							Gateways: []string{"private-gateway"},
							Port:     uint32(443),
							Uri: &istio.StringMatch{
								MatchType: &istio.StringMatch_Prefix{Prefix: "/"},
							},
						},
					},
					Route: []*istio.HTTPRouteDestination{
						{
							Destination: &istio.Destination{
								Host:   "testapp.testnamespace.svc.cluster.local",
								Port:   &istio.PortSelector{Number: uint32(8080)},
								Subset: "testtag",
							},
							Weight: 100,
						},
					},
				},
				{
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
					Name:       "grpc",
					Port:       8080,
					Protocol:   "TCP",
					TargetPort: intstr.FromString("grpc"),
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
		//ClusterName: "test-a",
	}
	assert.NoError(t, defaultSyncAppPlan.Apply(ctx, m, options, log), "Shouldn't return error.")
}
