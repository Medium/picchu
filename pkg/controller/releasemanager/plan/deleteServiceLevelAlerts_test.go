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

func TestDeleteServiceLevelAlerts(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	deleteSLOAlerts := &DeleteServiceLevelAlerts{
		App:       "testapp",
		Namespace: "testnamespace",
		Target:    "target",
	}
	ctx := context.TODO()

	opts := &client.ListOptions{
		Namespace: deleteSLOAlerts.Namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{
			picchuv1alpha1.LabelApp:    deleteSLOAlerts.App,
			picchuv1alpha1.LabelTarget: deleteSLOAlerts.Target,
		}),
	}

	pr := []monitoringv1.PrometheusRule{
		monitoringv1.PrometheusRule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "testnamespace",
			},
		},
	}

	m.
		EXPECT().
		List(ctx, mocks.InjectPrometheusRules(pr), mocks.ListOptions(opts)).
		Return(nil).
		Times(1)

	m.
		EXPECT().
		Delete(ctx, mocks.And(mocks.NamespacedName("testnamespace", "test"), mocks.Kind("PrometheusRule"))).
		Return(nil).
		Times(1)

	assert.NoError(t, deleteSLOAlerts.Apply(ctx, m, 1.0, log), "Shouldn't return error.")
}
