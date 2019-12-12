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
	sltaggedplan = &SyncTaggedServiceLevels{
		App:       "testapp",
		Namespace: "testnamespace",
		Tag:       "v1",
		ServiceLevelObjectives: []picchuv1alpha1.ServiceLevelObjective{{
			Enabled:          true,
			Name:             "test-app-availability",
			Description:      "test desc",
			ObjectivePercent: 99.999,
			ServiceLevelIndicator: picchuv1alpha1.ServiceLevelIndicator{
				UseForCanary:    true,
				CanaryAllowance: 0.5,
				TagKey:          "destination_workload",
				AlertAfter:      "1m",
				ErrorQuery:      "sum(rate(test_metric{job=\"test\"}[2m])) by (destination_workload)",
				TotalQuery:      "sum(rate(test_metric2{job=\"test\"}[2m])) by (destination_workload)",
			},
			Labels: map[string]string{
				"severity": "test",
			},
			Annotations: map[string]string{
				"test": "true",
			},
		}},
	}

	sltaggedexpected = &slov1alpha1.ServiceLevelList{
		Items: []slov1alpha1.ServiceLevel{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testapp-v1",
					Namespace: "testnamespace",
					Labels: map[string]string{
						picchuv1alpha1.LabelApp: "testapp",
						picchuv1alpha1.LabelTag: "v1",
					},
				},
				Spec: slov1alpha1.ServiceLevelSpec{
					ServiceLevelName: "testapp",
					ServiceLevelObjectives: []slov1alpha1.SLO{
						{
							Name:                         "testapp_test_app_availability",
							AvailabilityObjectivePercent: 99.999,
							Description:                  "test desc",
							Disable:                      false,
							Output: slov1alpha1.Output{
								Prometheus: &slov1alpha1.PrometheusOutputSource{
									Labels: map[string]string{
										"severity": "test",
										"tag":      "v1",
									},
								},
							},
							ServiceLevelIndicator: slov1alpha1.SLI{
								SLISource: slov1alpha1.SLISource{
									Prometheus: &slov1alpha1.PrometheusSLISource{
										ErrorQuery: "testapp:test_app_availability:errors{destination_workload=\"v1\"}",
										TotalQuery: "testapp:test_app_availability:total{destination_workload=\"v1\"}",
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

func TestTaggedServiceLevels(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	tests := []client.ObjectKey{
		client.ObjectKey{Name: "testapp-v1", Namespace: "testnamespace"},
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
			&sltaggedexpected.Items[i],
		} {
			m.
				EXPECT().
				Create(ctx, common.K8sEqual(obj)).
				Return(nil).
				Times(1)
		}
	}

	assert.NoError(t, sltaggedplan.Apply(ctx, m, 1.0, log), "Shouldn't return error.")
}
