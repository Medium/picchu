package plan

import (
	"context"
	_ "runtime"
	"testing"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/mocks"
	"go.medium.engineering/picchu/pkg/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestDeleteServiceMonitors(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	deleteServiceMonitors := &DeleteServiceMonitors{
		App:       "testapp",
		Namespace: "testnamespace",
	}
	ctx := context.TODO()

	opts := &client.ListOptions{
		Namespace: deleteServiceMonitors.Namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{
			picchuv1alpha1.LabelApp: deleteServiceMonitors.App,
		}),
	}

	sm := []monitoringv1.ServiceMonitor{
		monitoringv1.ServiceMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "testnamespace",
			},
		},
	}

	m.
		EXPECT().
		List(ctx, mocks.InjectServiceMonitors(sm), mocks.ListOptions(opts)).
		Return(nil).
		Times(1)

	m.
		EXPECT().
		Delete(ctx, mocks.And(mocks.NamespacedName("testnamespace", "test"), mocks.Kind("ServiceMonitor"))).
		Return(nil).
		Times(1)

	assert.NoError(t, deleteServiceMonitors.Apply(ctx, m, 1.0, log), "Shouldn't return error.")
}
