package plan

import (
	"context"
	"k8s.io/apimachinery/pkg/api/resource"
	"reflect"
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
	slorplan = &SyncSLORules{
		App:       "test-app",
		Namespace: "testnamespace",
		Labels: map[string]string{
			picchuv1alpha1.LabelApp:     "test-app",
			picchuv1alpha1.LabelK8sName: "test-app",
		},
		ServiceLevelObjectiveLabels: picchuv1alpha1.ServiceLevelObjectiveLabels{
			RuleLabels: map[string]string{
				"severity": "test",
			},
		},
		ServiceLevelObjectives: []*picchuv1alpha1.ServiceLevelObjective{{
			Enabled:          true,
			Name:             "test-app-availability",
			ObjectivePercent:  resource.MustParse("99.999"),
			ServiceLevelIndicator: picchuv1alpha1.ServiceLevelIndicator{
				Canary: picchuv1alpha1.SLICanaryConfig{
					Enabled:          true,
					AllowancePercent:  resource.MustParse("1"),
					FailAfter:        "1m",
				},
				TagKey:     "destination_workload",
				AlertAfter: "1m",
				ErrorQuery: "sum(rate(test_metric{job=\"test\"}[2m])) by (destination_workload)",
				TotalQuery: "sum(rate(test_metric2{job=\"test\"}[2m])) by (destination_workload)",
			},
			ServiceLevelObjectiveLabels: picchuv1alpha1.ServiceLevelObjectiveLabels{
				RuleLabels: map[string]string{
					"team": "test",
				},
			},
		}},
	}

	slorexpected = &monitoringv1.PrometheusRuleList{
		Items: []*monitoringv1.PrometheusRule{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-app-slo",
				Namespace: "testnamespace",
				Labels: map[string]string{
					picchuv1alpha1.LabelApp:      "test-app",
					picchuv1alpha1.LabelK8sName:  "test-app",
					picchuv1alpha1.LabelRuleType: RuleTypeSLO,
					"severity":                   "test",
				},
			},
			Spec: monitoringv1.PrometheusRuleSpec{
				Groups: []monitoringv1.RuleGroup{
					{
						Name: "test_app_availability_record",
						Rules: []monitoringv1.Rule{
							{
								Record: "test_app:test_app_availability:total",
								Expr:   intstr.FromString("sum(rate(test_metric2{job=\"test\"}[2m])) by (destination_workload)"),
							},
							{
								Record: "test_app:test_app_availability:errors",
								Expr:   intstr.FromString("sum(rate(test_metric{job=\"test\"}[2m])) by (destination_workload)"),
							},
						},
					},
				},
			},
		},
		},
	}
)

func TestSLORules(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	tests := []client.ObjectKey{
		{Name: "test-app-slo", Namespace: "testnamespace"},
	}
	ctx := context.TODO()

	for i := range tests {
		m.
			EXPECT().
			Get(ctx, mocks.ObjectKey(tests[i]), gomock.Any()).
			Return(common.NotFoundError).
			Times(1)
	}

	for i := range slorexpected.Items {
		for _, obj := range []runtime.Object{
			slorexpected.Items[i],
		} {
			m.
				EXPECT().
				Create(ctx, common.K8sEqual(obj)).
				Return(nil).
				Times(1)
		}
	}

	assert.NoError(t, slorplan.Apply(ctx, m, cluster, log), "Shouldn't return error.")
}

func TestSanitizeName(t *testing.T) {
	expressions := []struct {
		expr     string
		expected string
	}{
		{"test_app", "test_app"},
		{"TEST-APP", "test_app"},
		{"TEST#APP", "test_app"},
	}

	for _, expr := range expressions {
		actual := sanitizeName(expr.expr)
		if !reflect.DeepEqual(expr.expected, actual) {
			t.Errorf("Expected did not match actual. Expected: %v. Actual: %v", expr.expected, actual)
			t.Fail()
		}
	}
}
