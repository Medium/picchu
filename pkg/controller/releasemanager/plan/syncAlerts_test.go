package plan

import (
	"context"
	_ "runtime"
	"testing"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/mocks"
	common "go.medium.engineering/picchu/pkg/plan/test"
	"go.medium.engineering/picchu/pkg/test"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	defaultSyncCanaryAlertsPlan = &SyncAlerts{
		App:       "testapp",
		Namespace: "testnamespace",
		Tag:       "testtag",
		Target:    "test",
		AlertType: Canary,
		ServiceLevelObjectives: []picchuv1alpha1.ServiceLevelObjective{{
			Enabled:          true,
			Name:             "test-app-availability",
			ObjectivePercent: 99.999,
			ServiceLevelIndicator: picchuv1alpha1.ServiceLevelIndicator{
				UseForCanary:    true,
				CanaryAllowance: 0.5,
				TagKey:          "destination_workload",
				AlertAfter:      "1m",
				ErrorQuery:      "test_app:error_requests:rate5m",
				TotalQuery:      "test_app:total_requests:rate5m",
			},
		}},
	}

	defaultSyncSLIAlertsPlan = &SyncAlerts{
		App:       "testapp",
		Namespace: "testnamespace",
		Tag:       "testtag",
		Target:    "test",
		AlertType: SLI,
		ServiceLevelObjectives: []picchuv1alpha1.ServiceLevelObjective{{
			Enabled:          true,
			Name:             "test-app-availability",
			ObjectivePercent: 99.999,
			ServiceLevelIndicator: picchuv1alpha1.ServiceLevelIndicator{
				UseForCanary:    true,
				CanaryAllowance: 0.5,
				TagKey:          "destination_workload",
				AlertAfter:      "1m",
				ErrorQuery:      "test_app:error_requests:rate5m",
				TotalQuery:      "test_app:total_requests:rate5m",
			},
		}},
	}

	defaultExpectedSyncCanariesRules = &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testapp-testtag-canary",
			Namespace: "testnamespace",
			Labels: map[string]string{
				picchuv1alpha1.LabelApp:      "testapp",
				picchuv1alpha1.LabelTag:      "testtag",
				picchuv1alpha1.LabelTarget:   "test",
				picchuv1alpha1.LabelRuleType: "canary",
			},
		},
		Spec: monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{{
				Name: "testapp-testtag-canary",
				Rules: []monitoringv1.Rule{{
					Alert: "test-app-availability-testtag-canary",
					Expr: intstr.FromString("100 * (1 - test_app:error_requests:rate5m{ destination_workload=\"testtag\" } " +
						"/ test_app:total_requests:rate5m{ destination_workload=\"testtag\" }) + 0.5 < " +
						"(100 * (1 - sum(test_app:error_requests:rate5m) / sum(test_app:total_requests:rate5m)))"),
					For: "1m",
					Labels: map[string]string{
						"alertType": "canary",
						"app":       "testapp",
						"tag":       "testtag",
					},
				}},
			}},
		},
	}

	defaultExpectedSyncSLIRules = &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testapp-testtag-sli",
			Namespace: "testnamespace",
			Labels: map[string]string{
				picchuv1alpha1.LabelApp:      "testapp",
				picchuv1alpha1.LabelTag:      "testtag",
				picchuv1alpha1.LabelTarget:   "test",
				picchuv1alpha1.LabelRuleType: "sli",
			},
		},
		Spec: monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{{
				Name: "testapp-testtag-sli",
				Rules: []monitoringv1.Rule{{
					Alert: "test-app-availability-testtag-sli",
					Expr: intstr.FromString("100 * (1 - test_app:error_requests:rate5m{ destination_workload=\"testtag\" } / " +
						"test_app:total_requests:rate5m{ destination_workload=\"testtag\" }) < 99.999"),
					For: "1m",
					Labels: map[string]string{
						"alertType": "sli",
						"app":       "testapp",
						"tag":       "testtag",
					},
				}},
			}},
		},
	}
)

func TestSyncCanaryAlerts(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	ok := client.ObjectKey{Name: "testapp-testtag-canary", Namespace: "testnamespace"}
	ctx := context.TODO()

	m.
		EXPECT().
		Get(ctx, mocks.ObjectKey(ok), gomock.Any()).
		Return(common.NotFoundError).
		Times(1)

	for _, obj := range []runtime.Object{
		defaultExpectedSyncCanariesRules,
	} {
		m.
			EXPECT().
			Create(ctx, common.K8sEqual(obj)).
			Return(nil).
			Times(1)
	}

	assert.NoError(t, defaultSyncCanaryAlertsPlan.Apply(ctx, m, 1.0, log), "Shouldn't return error.")
}

func TestSyncSLIAlerts(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	ok := client.ObjectKey{Name: "testapp-testtag-sli", Namespace: "testnamespace"}
	ctx := context.TODO()

	m.
		EXPECT().
		Get(ctx, mocks.ObjectKey(ok), gomock.Any()).
		Return(common.NotFoundError).
		Times(1)

	for _, obj := range []runtime.Object{
		defaultExpectedSyncSLIRules,
	} {
		m.
			EXPECT().
			Create(ctx, common.K8sEqual(obj)).
			Return(nil).
			Times(1)
	}

	assert.NoError(t, defaultSyncSLIAlertsPlan.Apply(ctx, m, 1.0, log), "Shouldn't return error.")
}
