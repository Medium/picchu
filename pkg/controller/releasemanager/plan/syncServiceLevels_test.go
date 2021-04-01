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

	slov1alpha1 "github.com/Medium/service-level-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/golang/mock/gomock"
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
			Enabled:                true,
			Name:                   "test-app-availability",
			Description:            "test desc",
			ObjectivePercentString: "99.999",
			ServiceLevelIndicator: picchuv1alpha1.ServiceLevelIndicator{
				Canary: picchuv1alpha1.SLICanaryConfig{
					Enabled:                true,
					AllowancePercentString: "1",
					FailAfter:              "1m",
				},
				TagKey:     "destination_workload",
				AlertAfter: "1m",
				ErrorQuery: "sum(rate(test_metric{job=\"test\"}[2m])) by (destination_workload)",
				TotalQuery: "sum(rate(test_metric2{job=\"test\"}[2m])) by (destination_workload)",
			},
			ServiceLevelObjectiveLabels: picchuv1alpha1.ServiceLevelObjectiveLabels{
				ServiceLevelLabels: map[string]string{
					"team": "test",
				},
			},
		}},
	}

	slexpected = &slov1alpha1.ServiceLevelList{
		Items: []slov1alpha1.ServiceLevel{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app-production-servicelevels",
					Namespace: "testnamespace",
					Labels: map[string]string{
						picchuv1alpha1.LabelApp:     "test-app",
						picchuv1alpha1.LabelK8sName: "test-app",
					},
				},
				Spec: slov1alpha1.ServiceLevelSpec{
					ServiceLevelName: "test-app",
					ServiceLevelObjectives: []slov1alpha1.SLO{
						{
							Name:                         "test_app_availability",
							AvailabilityObjectivePercent: 99.999,
							Description:                  "test desc",
							Disable:                      false,
							Output: slov1alpha1.Output{
								Prometheus: &slov1alpha1.PrometheusOutputSource{
									Labels: map[string]string{
										"severity": "test",
										"team":     "test",
									},
								},
							},
							ServiceLevelIndicator: slov1alpha1.SLI{
								SLISource: slov1alpha1.SLISource{
									Prometheus: &slov1alpha1.PrometheusSLISource{
										ErrorQuery: "sum(test_app:test_app_availability:errors)",
										TotalQuery: "sum(test_app:test_app_availability:total)",
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
