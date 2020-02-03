package plan

import (
	"context"
	_ "runtime"
	"testing"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/mocks"
	common "go.medium.engineering/picchu/pkg/plan/test"
	"go.medium.engineering/picchu/pkg/test"
	"sigs.k8s.io/controller-runtime/pkg/client"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	crplan = &SyncCanaryRules{
		App:       "test-app",
		Namespace: "testnamespace",
		Tag:       "tag",
		ServiceLevelObjectives: []*picchuv1alpha1.ServiceLevelObjective{{
			Enabled:          true,
			Name:             "test-app-availability",
			ObjectivePercent: 0.99999,
			ServiceLevelIndicator: picchuv1alpha1.ServiceLevelIndicator{
				Canary: picchuv1alpha1.SLICanaryConfig{
					Enabled:          true,
					AllowancePercent: 0.5,
					FailAfter:        "1m",
				},
				TagKey:     "destination_workload",
				AlertAfter: "1m",
				ErrorQuery: "sum(rate(test_metric{job=\"test\"}[2m])) by (destination_workload)",
				TotalQuery: "sum(rate(test_metric2{job=\"test\"}[2m])) by (destination_workload)",
			},
			Labels: map[string]string{
				"severity": "test",
			},
			Annotations: map[string]string{
				"test": "true",
			},
		}},
	}

	crexpected = monitoringv1.PrometheusRuleList{
		Items: []*monitoringv1.PrometheusRule{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-app-canary-tag",
				Namespace: "testnamespace",
				Labels: map[string]string{
					picchuv1alpha1.LabelApp: "test-app",
					picchuv1alpha1.LabelTag: "tag",
					"prometheus":            "slo",
				},
			},
			Spec: monitoringv1.PrometheusRuleSpec{
				Groups: []monitoringv1.RuleGroup{
					{
						Name: "test_app_availability_canary",
						Rules: []monitoringv1.Rule{
							{
								Alert: "test_app_availability_canary",
								Expr: intstr.FromString("test_app:test_app_availability:errors{destination_workload=\"tag\"} / test_app:test_app_availability:total{destination_workload=\"tag\"} + 0.5 " +
									"< ignoring(destination_workload) sum(test_app:test_app_availability:errors) / sum(test_app:test_app_availability:total)"),
								For: "1m",
								Labels: map[string]string{
									"app":      "test-app",
									"canary":   "true",
									"slo":      "true",
									"severity": "test",
									"tag":      "tag",
								},
								Annotations: map[string]string{
									"test": "true",
								},
							},
						},
					},
				},
			},
		},
		},
	}
)

func TestSyncCanaryRules(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	tests := []client.ObjectKey{
		client.ObjectKey{Name: "test-app-canary-tag", Namespace: "testnamespace"},
	}
	ctx := context.TODO()

	for i := range tests {
		m.
			EXPECT().
			Get(ctx, mocks.ObjectKey(tests[i]), gomock.Any()).
			Return(common.NotFoundError).
			Times(1)
	}

	for i := range crexpected.Items {
		for _, obj := range []runtime.Object{
			crexpected.Items[i],
		} {
			m.
				EXPECT().
				Create(ctx, common.K8sEqual(obj)).
				Return(nil).
				Times(1)
		}
	}

	assert.NoError(t, crplan.Apply(ctx, m, 1.0, log), "Shouldn't return error.")
}