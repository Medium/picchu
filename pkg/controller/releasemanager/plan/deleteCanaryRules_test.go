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

func TestDeleteCanaryRules(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	deleteCanaryRules := &DeleteCanaryRules{
		App:       "testapp",
		Namespace: "testnamespace",
		Tag:       "tag",
	}
	ctx := context.TODO()

	opts := &client.ListOptions{
		Namespace: deleteCanaryRules.Namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{
			picchuv1alpha1.LabelApp:      deleteCanaryRules.App,
			picchuv1alpha1.LabelTag:      deleteCanaryRules.Tag,
			picchuv1alpha1.LabelRuleType: RuleTypeCanary,
		}),
	}

	pr := []monitoringv1.PrometheusRule{
		{
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

	assert.NoError(t, deleteCanaryRules.Apply(ctx, m, 1.0, log), "Shouldn't return error.")
}
