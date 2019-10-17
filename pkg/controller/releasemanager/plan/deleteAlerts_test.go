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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestDeleteAlerts(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	deleteAlerts := &DeleteAlerts{
		App:       "testapp",
		Namespace: "testnamespace",
		Tag:       "testtag",
		AlertType: Canary,
	}
	ctx := context.TODO()

	opts := client.
		MatchingLabels(map[string]string{
			picchuv1alpha1.LabelApp:      deleteAlerts.App,
			picchuv1alpha1.LabelTag:      deleteAlerts.Tag,
			picchuv1alpha1.LabelRuleType: string(deleteAlerts.AlertType),
		}).
		InNamespace(deleteAlerts.Namespace)

	rules := []monitoringv1.PrometheusRule{
		monitoringv1.PrometheusRule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-rules",
				Namespace: "testnamespace",
			},
		},
	}

	m.
		EXPECT().
		List(ctx, mocks.ListOptions(opts), mocks.InjectPrometheusRules(rules)).
		Return(nil).
		Times(1)

	m.
		EXPECT().
		Delete(ctx, mocks.And(mocks.NamespacedName("testnamespace", "test-rules"), mocks.Kind("PrometheusRule"))).
		Return(nil).
		Times(1)

	assert.NoError(t, deleteAlerts.Apply(ctx, m, 1.0, log), "Shouldn't return error.")
}
