package plan

import (
	"context"
	"testing"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/controller/releasemanager/mocks"
	"go.medium.engineering/picchu/pkg/test"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/golang/mock/gomock"
	istiov1alpha3 "github.com/knative/pkg/apis/istio/v1alpha3"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			IngressPort:   80,
			Port:          80,
			ContainerPort: 5000,
			Protocol:      corev1.ProtocolTCP,
			Mode:          picchuv1alpha1.PortPrivate,
		}},
		TagRoutingHeader: "MEDIUM-TAG",
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
				"testapp.testapp-production.svc.cluster.local",
			},
			Gateways: []string{
				"private-gateway",
				"public-gateway",
				"mesh",
			},
			Http: []istiov1alpha3.HTTPRoute{{}},
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
		Get(ctx, mocks.ObjectKey(ok), mocks.Kind("Service")).
		Return(notFoundError).
		Times(1)
	m.
		EXPECT().
		Create(ctx, mocks.Kind("Service")).
		Return(nil).
		Times(1)
	m.
		EXPECT().
		Get(ctx, mocks.ObjectKey(ok), mocks.Kind("DestinationRule")).
		Return(notFoundError).
		Times(1)
	m.
		EXPECT().
		Create(ctx, mocks.Kind("DestinationRule")).
		Return(nil).
		Times(1)
		/*
			m.
				EXPECT().
				Get(ctx, mocks.ObjectKey(ok), mocks.Kind("VirtualService")).
				Return(notFoundError).
				Times(1)
			m.
				EXPECT().
				Get(ctx, mocks.ObjectKey(ok), mocks.Kind("PrometheusRule")).
				Return(notFoundError).
				Times(1)
			m.
				EXPECT().
				Create(ctx, mocks.Kind("PrometheusRule")).
				Return(nil).
				Times(1)
			m.
				EXPECT().
				Create(ctx, mocks.Kind("VirtualService")).
				Return(nil).
				Times(1)
		*/

	assert.NoError(t, defaultSyncAppPlan.Apply(ctx, m, log), "Shouldn't return error.")
}
