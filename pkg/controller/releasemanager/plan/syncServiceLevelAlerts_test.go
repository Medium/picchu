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
	slalertsplan = &SyncServiceLevelAlerts{
		App:       "test-app",
		Namespace: "testnamespace",
		Target:    "production",
		Labels: map[string]string{
			picchuv1alpha1.LabelApp:     "test-app",
			picchuv1alpha1.LabelK8sName: "test-app",
		},
		ServiceLevelObjectiveLabels: picchuv1alpha1.ServiceLevelObjectiveLabels{
			AlertLabels: map[string]string{
				"severity": "test",
			},
		},
		ServiceLevelObjectives: []*picchuv1alpha1.ServiceLevelObjective{{
			Enabled:          true,
			Name:             "test-app-availability",
			ObjectivePercent: 99.999,
			ServiceLevelIndicator: picchuv1alpha1.ServiceLevelIndicator{
				Canary: picchuv1alpha1.SLICanaryConfig{
					Enabled:          true,
					AllowancePercent: 1,
					FailAfter:        "1m",
				},
				TagKey:     "destination_workload",
				AlertAfter: "1m",
				ErrorQuery: "sum(rate(test_metric{job=\"test\"}[2m])) by (destination_workload)",
				TotalQuery: "sum(rate(test_metric2{job=\"test\"}[2m])) by (destination_workload)",
			},
			ServiceLevelObjectiveLabels: picchuv1alpha1.ServiceLevelObjectiveLabels{
				AlertLabels: map[string]string{
					"team": "test",
				},
			},
		}},
	}

	sloalertexpected = &monitoringv1.PrometheusRuleList{
		Items: []*monitoringv1.PrometheusRule{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-app-production-slo-alerts",
				Namespace: "testnamespace",
				Labels: map[string]string{
					picchuv1alpha1.LabelApp:     "test-app",
					picchuv1alpha1.LabelK8sName: "test-app",
				},
			},
			Spec: monitoringv1.PrometheusRuleSpec{
				Groups: []monitoringv1.RuleGroup{
					{
						Name: "test_app_availability_alert",
						Rules: []monitoringv1.Rule{
							{
								Alert: "SLOErrorRateTooFast1h",
								Expr: intstr.FromString("(increase(service_level_sli_result_error_ratio_total{service_level=\"test-app\", slo=\"test_app_availability\"}[1h]) " +
									"/ increase(service_level_sli_result_count_total{service_level=\"test-app\", slo=\"test_app_availability\"}[1h])) " +
									"> (1 - service_level_slo_objective_ratio{service_level=\"test-app\", slo=\"test_app_availability\"}) * 14.6"),
								For: "1m",
								Labels: map[string]string{
									"severity": "test",
									"team":     "test",
								},
							},
							{
								Alert: "SLOErrorRateTooFast6h",
								Expr: intstr.FromString("(increase(service_level_sli_result_error_ratio_total{service_level=\"test-app\", slo=\"test_app_availability\"}[6h]) " +
									"/ increase(service_level_sli_result_count_total{service_level=\"test-app\", slo=\"test_app_availability\"}[6h])) " +
									"> (1 - service_level_slo_objective_ratio{service_level=\"test-app\", slo=\"test_app_availability\"}) * 6"),
								For: "1m",
								Labels: map[string]string{
									"severity": "test",
									"team":     "test",
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

func TestServiceLevelAlerts(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	tests := []client.ObjectKey{
		client.ObjectKey{Name: "test-app-production-slo-alerts", Namespace: "testnamespace"},
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
			sloalertexpected.Items[i],
		} {
			m.
				EXPECT().
				Create(ctx, common.K8sEqual(obj)).
				Return(nil).
				Times(1)
		}
	}

	assert.NoError(t, slalertsplan.Apply(ctx, m, 1.0, log), "Shouldn't return error.")
}
