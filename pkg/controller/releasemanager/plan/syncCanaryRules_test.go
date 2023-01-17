package plan

import (
	"context"
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
	crplan = &SyncCanaryRules{
		App:       "test-app",
		Namespace: "testnamespace",
		Tag:       "tag",
		Labels: map[string]string{
			picchuv1alpha1.LabelApp:        "test-app",
			picchuv1alpha1.LabelTag:        "v1",
			picchuv1alpha1.LabelK8sName:    "test-app",
			picchuv1alpha1.LabelK8sVersion: "v1",
		},
		ServiceLevelObjectiveLabels: picchuv1alpha1.ServiceLevelObjectiveLabels{
			AlertLabels: map[string]string{
				"severity": "test",
			},
		},
		ServiceLevelObjectives: []*picchuv1alpha1.ServiceLevelObjective{{
			Enabled:                true,
			Name:                   "test-app-availability",
			ObjectivePercentString: "99.999",
			Description:            "Test description",
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

	crexpected = monitoringv1.PrometheusRuleList{
		Items: []*monitoringv1.PrometheusRule{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-app-canary-tag",
				Namespace: "testnamespace",
				Labels: map[string]string{
					picchuv1alpha1.LabelApp:        "test-app",
					picchuv1alpha1.LabelTag:        "v1",
					picchuv1alpha1.LabelK8sName:    "test-app",
					picchuv1alpha1.LabelK8sVersion: "v1",
					picchuv1alpha1.LabelRuleType:   RuleTypeCanary,
				},
			},
			Spec: monitoringv1.PrometheusRuleSpec{
				Groups: []monitoringv1.RuleGroup{
					{
						Name: "test_app_availability_canary",
						Rules: []monitoringv1.Rule{
							{
								Alert: "test_app_availability_canary",
								Expr: intstr.FromString("test_app:test_app_availability:errors{destination_workload=\"tag\"} / test_app:test_app_availability:total{destination_workload=\"tag\"} - 0.01 " +
									"> ignoring(destination_workload) sum(test_app:test_app_availability:errors) / sum(test_app:test_app_availability:total)"),
								For: "1m",
								Annotations: map[string]string{
									CanaryMessageAnnotation: "Test description",
									CanarySummaryAnnotation: "Canary is failing SLO",
								},
								Labels: map[string]string{
									CanaryAppLabel: "test-app",
									CanaryTagLabel: "tag",
									CanaryLabel:    "true",
									CanarySLOLabel: "true",
									"severity":     "test",
									"team":         "test",
								},
							},
						},
					},
				},
			},
		},
		},
	}

	slothcrplan = &SyncCanaryRules{
		App:       "test-app",
		Namespace: "testnamespace",
		Tag:       "tag",
		Labels: map[string]string{
			picchuv1alpha1.LabelApp:        "test-app",
			picchuv1alpha1.LabelTag:        "v1",
			picchuv1alpha1.LabelK8sName:    "test-app",
			picchuv1alpha1.LabelK8sVersion: "v1",
		},
		ServiceLevelObjectiveLabels: picchuv1alpha1.ServiceLevelObjectiveLabels{
			AlertLabels: map[string]string{
				"severity": "test",
			},
		},
		SlothServiceLevelObjectives: []*picchuv1alpha1.SlothServiceLevelObjective{{
			Enabled:     true,
			Name:        "test-app-availability",
			Objective:   "99.999",
			Description: "Test description",
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
)

func TestSyncCanaryRules(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	tests := []client.ObjectKey{
		{Name: "test-app-canary-tag", Namespace: "testnamespace"},
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

	assert.NoError(t, crplan.Apply(ctx, m, cluster, log), "Shouldn't return error.")
}

func TestSyncCanaryRulesSloth(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	tests := []client.ObjectKey{
		{Name: "test-app-canary-tag", Namespace: "testnamespace"},
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

	assert.NoError(t, slothcrplan.Apply(ctx, m, cluster, log), "Shouldn't return error.")
}

func TestFormatAllowancePercent(t *testing.T) {
	log := test.MustNewLogger()

	inputs := []struct {
		float    float64
		expected string
	}{
		{1, "0.01"},
		{100, "1"},
		{2, "0.02"},
		{0.1, "0.001"},
		{0.01, "0.0001"},
	}

	for _, i := range inputs {
		c := SLOConfig{
			SLO: &picchuv1alpha1.ServiceLevelObjective{
				ServiceLevelIndicator: picchuv1alpha1.ServiceLevelIndicator{
					Canary: picchuv1alpha1.SLICanaryConfig{
						AllowancePercent: i.float,
					},
				},
			},
		}
		actual := c.formatAllowancePercent(log)
		if !reflect.DeepEqual(i.expected, actual) {
			t.Errorf("Expected did not match actual. Expected: %v. Actual: %v", i.expected, actual)
			t.Fail()
		}
	}
}

func TestFormatAllowancePercentSloth(t *testing.T) {
	log := test.MustNewLogger()

	inputs := []struct {
		float    float64
		expected string
	}{
		{1, "0.01"},
		{100, "1"},
		{2, "0.02"},
		{0.1, "0.001"},
		{0.01, "0.0001"},
	}

	for _, i := range inputs {
		c := SlothSLOConfig{
			SLO: &picchuv1alpha1.SlothServiceLevelObjective{
				ServiceLevelIndicator: picchuv1alpha1.ServiceLevelIndicator{
					Canary: picchuv1alpha1.SLICanaryConfig{
						AllowancePercent: i.float,
					},
				},
			},
		}
		actual := c.formatAllowancePercent(log)
		if !reflect.DeepEqual(i.expected, actual) {
			t.Errorf("Expected did not match actual. Expected: %v. Actual: %v", i.expected, actual)
			t.Fail()
		}
	}
}
