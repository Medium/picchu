package plan

import (
	"context"
	_ "runtime"
	"testing"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/controller/releasemanager/mocks"
	"go.medium.engineering/picchu/pkg/test"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/golang/mock/gomock"
	istiocommonv1alpha1 "github.com/knative/pkg/apis/istio/common/v1alpha1"
	istiov1alpha3 "github.com/knative/pkg/apis/istio/v1alpha3"
	"github.com/stretchr/testify/assert"
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
		DefaultDomain:  "doki-pen.org",
		PublicGateway:  "public-gateway",
		PrivateGateway: "private-gateway",
		DeployedRevisions: []Revision{{
			Tag:    "testtag",
			Weight: 100,
		}},
		AlertRules: []monitoringv1.Rule{{
			Expr: intstr.FromString("hello world"),
		}},
		Ports: []picchuv1alpha1.PortInfo{{
			Name:          "http",
			IngressPort:   443,
			Port:          80,
			ContainerPort: 5000,
			Protocol:      corev1.ProtocolTCP,
			Mode:          picchuv1alpha1.PortPrivate,
		}, {
			Name:          "status",
			IngressPort:   443,
			Port:          4242,
			ContainerPort: 4444,
			Protocol:      corev1.ProtocolTCP,
			Mode:          picchuv1alpha1.PortLocal,
		}},
		TagRoutingHeader: "MEDIUM-TAG",
		UseNewTagStyle:   true,
		TrafficPolicy: &istiov1alpha3.TrafficPolicy{
			ConnectionPool: &istiov1alpha3.ConnectionPoolSettings{
				Http: &istiov1alpha3.HTTPSettings{
					MaxRequestsPerConnection: 1,
				},
			},
		},
	}

	defaultExpectedVirtualService = &istiov1alpha3.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testapp",
			Namespace: "testnamespace",
			Labels: map[string]string{
				"test": "label",
			},
		},
		Spec: istiov1alpha3.VirtualServiceSpec{
			Hosts: []string{
				"testapp.doki-pen.org",
				"testapp.testnamespace.svc.cluster.local",
			},
			Gateways: []string{
				"mesh",
				"public-gateway",
				"private-gateway",
			},
			Http: []istiov1alpha3.HTTPRoute{{
				Match: []istiov1alpha3.HTTPMatchRequest{{
					Gateways: []string{"mesh"},
					Headers: map[string]istiocommonv1alpha1.StringMatch{
						"MEDIUM-TAG": {Exact: "testtag"},
					},
					Port: uint32(80),
				}, {
					Gateways: []string{"private-gateway"},
					Headers: map[string]istiocommonv1alpha1.StringMatch{
						"MEDIUM-TAG": {Exact: "testtag"},
					},
					Port: uint32(443),
				}},
				Route: []istiov1alpha3.DestinationWeight{{
					Destination: istiov1alpha3.Destination{
						Host:   "testapp.testnamespace.svc.cluster.local",
						Port:   istiov1alpha3.PortSelector{Number: uint32(80)},
						Subset: "testtag",
					},
					Weight: 100,
				}},
			}, {
				Match: []istiov1alpha3.HTTPMatchRequest{{
					Gateways: []string{"mesh"},
					Headers: map[string]istiocommonv1alpha1.StringMatch{
						"MEDIUM-TAG": {Exact: "testtag"},
					},
					Port: uint32(4242),
				}},
				Route: []istiov1alpha3.DestinationWeight{{
					Destination: istiov1alpha3.Destination{
						Host:   "testapp.testnamespace.svc.cluster.local",
						Port:   istiov1alpha3.PortSelector{Number: uint32(4242)},
						Subset: "testtag",
					},
					Weight: 100,
				}},
			}, {
				Match: []istiov1alpha3.HTTPMatchRequest{{
					Gateways: []string{"mesh"},
					Port:     uint32(80),
				}, {
					Authority: &istiocommonv1alpha1.StringMatch{Prefix: "testapp.doki-pen.org"},
					Gateways:  []string{"private-gateway"},
					Port:      uint32(443),
				}},
				Route: []istiov1alpha3.DestinationWeight{{
					Destination: istiov1alpha3.Destination{
						Host:   "testapp.testnamespace.svc.cluster.local",
						Port:   istiov1alpha3.PortSelector{Number: uint32(80)},
						Subset: "testtag",
					},
					Weight: 100,
				}},
			}, {
				Match: []istiov1alpha3.HTTPMatchRequest{{
					Gateways: []string{"mesh"},
					Port:     uint32(4242),
				}},
				Route: []istiov1alpha3.DestinationWeight{{
					Destination: istiov1alpha3.Destination{
						Host:   "testapp.testnamespace.svc.cluster.local",
						Port:   istiov1alpha3.PortSelector{Number: uint32(4242)},
						Subset: "testtag",
					},
					Weight: 100,
				}},
			}},
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
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Port:       80,
				Protocol:   "TCP",
				TargetPort: intstr.FromString("http"),
			}, {
				Name:       "status",
				Port:       4242,
				Protocol:   "TCP",
				TargetPort: intstr.FromString("status"),
			}},
			Selector: map[string]string{
				"picchu.medium.engineering/app": "testapp",
			},
		},
	}

	defaultExpectedDestinationRule = &istiov1alpha3.DestinationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testapp",
			Namespace: "testnamespace",
			Labels: map[string]string{
				"test": "label",
			},
		},
		Spec: istiov1alpha3.DestinationRuleSpec{
			Host: "testapp.testnamespace.svc.cluster.local",
			Subsets: []istiov1alpha3.Subset{{
				Name:   "testtag",
				Labels: map[string]string{"tag.picchu.medium.engineering": "testtag"},
			}},
			TrafficPolicy: &istiov1alpha3.TrafficPolicy{
				ConnectionPool: &istiov1alpha3.ConnectionPoolSettings{
					Http: &istiov1alpha3.HTTPSettings{
						MaxRequestsPerConnection: 1,
					},
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
		Return(notFoundError).
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
			Create(ctx, k8sEqual(obj)).
			Return(nil).
			Times(1)
	}

	assert.NoError(t, defaultSyncAppPlan.Apply(ctx, m, log), "Shouldn't return error.")
}
