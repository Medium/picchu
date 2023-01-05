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

	"github.com/golang/mock/gomock"
	slov1alpha1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	slplan = &SyncServiceLevels{
		App:       "test-app",
		Namespace: "testnamespace",
		Target:    "production",
		Labels: map[string]string{
			picchuv1alpha1.LabelApp:     "test-app",
			picchuv1alpha1.LabelK8sName: "test-app",
		},
		ServiceLevelObjectiveLabels: picchuv1alpha1.ServiceLevelObjectiveLabels{
			ServiceLevelLabels: map[string]string{
				"severity": "test",
			},
		},
		ServiceLevelObjectives: []*picchuv1alpha1.ServiceLevelObjective{{
			Alerting: picchuv1alpha1.Alerting{
				TicketAlert: picchuv1alpha1.Alert{
					Disable: false,
				},
			},
			Name:        "test-app-availability",
			Description: "test desc",
			Objective:   99.999,
			SLI: picchuv1alpha1.SLI{
				Canary: picchuv1alpha1.SLICanaryConfig{
					Enabled:          true,
					AllowancePercent: 1,
					FailAfter:        "1m",
				},
				TagKey:     "destination_workload",
				AlertAfter: "1m",
				Events: &picchuv1alpha1.SLIEvents{
					ErrorQuery: "sum(rate(test_metric{job=\"test\"}[2m])) by (destination_workload)",
					TotalQuery: "sum(rate(test_metric2{job=\"test\"}[2m])) by (destination_workload)",
				},
			},
			ServiceLevelObjectiveLabels: picchuv1alpha1.ServiceLevelObjectiveLabels{
				ServiceLevelLabels: map[string]string{
					"team": "test",
				},
			},
		}},
	}

	slexpected = &slov1alpha1.PrometheusServiceLevelList{
		Items: []slov1alpha1.PrometheusServiceLevel{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app-production-servicelevels",
					Namespace: "testnamespace",
					Labels: map[string]string{
						picchuv1alpha1.LabelApp:     "test-app",
						picchuv1alpha1.LabelK8sName: "test-app",
					},
				},
				Spec: slov1alpha1.PrometheusServiceLevelSpec{
					Service: "test-app",
					SLOs: []slov1alpha1.SLO{
						{
							Name:        "test_app_availability",
							Objective:   99.999,
							Description: "test desc",
							// Disable:                      false,
							Labels: map[string]string{
								"severity": "test",
								"team":     "test",
							},
							SLI: slov1alpha1.SLI{
								Events: &slov1alpha1.SLIEvents{
									ErrorQuery: "sum(test_app:test_app_availability:errors)",
									TotalQuery: "sum(test_app:test_app_availability:total)",
								},
							},
						},
					},
				},
			},
		},
	}
)

func TestServiceLevels(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	tests := []client.ObjectKey{
		{Name: "test-app-production-servicelevels", Namespace: "testnamespace"},
	}
	ctx := context.TODO()

	for i := range tests {
		m.
			EXPECT().
			Get(ctx, mocks.ObjectKey(tests[i]), gomock.Any()).
			Return(common.NotFoundError).
			Times(1)
	}

	for i := range slexpected.Items {
		for _, obj := range []runtime.Object{
			&slexpected.Items[i],
		} {
			m.
				EXPECT().
				Create(ctx, common.K8sEqual(obj)).
				Return(nil).
				Times(1)
		}
	}

	assert.NoError(t, slplan.Apply(ctx, m, cluster, log), "Shouldn't return error.")
}
