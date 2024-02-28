package plan

import (
	"context"
	_ "runtime"
	"testing"

	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/mocks"
	common "go.medium.engineering/picchu/plan/test"
	"go.medium.engineering/picchu/test"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/golang/mock/gomock"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	dpplan = &SyncDeploymentRules{
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
		ServiceLevelObjectives: []*picchuv1alpha1.SlothServiceLevelObjective{
			{
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
			},
		},
	}

	dpexpected = monitoringv1.PrometheusRuleList{
		Items: []*monitoringv1.PrometheusRule{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-app-deployment-tag",
				Namespace: "testnamespace",
				Labels: map[string]string{
					picchuv1alpha1.LabelApp:        "test-app",
					picchuv1alpha1.LabelTag:        "v1",
					picchuv1alpha1.LabelK8sName:    "test-app",
					picchuv1alpha1.LabelK8sVersion: "v1",
					picchuv1alpha1.LabelRuleType:   RuleTypeDeployment,
				},
			},
			Spec: monitoringv1.PrometheusRuleSpec{
				Groups: []monitoringv1.RuleGroup{
					{
						Name: "test_app_deployment_crashloop",
						Rules: []monitoringv1.Rule{
							{
								Alert: "test_app_deployment_crashloop",
								Expr:  intstr.FromString("sum by (reason) (kube_pod_container_status_waiting_reason{reason=\"CrashLoopBackOff\", container=\"test-app\", pod=~\"tag-.*\"}) > 0"),
								For:   "1m",
								Annotations: map[string]string{
									DeploymentMessageAnnotation: "There is at least one pod in state `CrashLoopBackOff`",
									DeploymentSummaryAnnotation: "test-app - Deployment is failing CrashLoopBackOff SLO - there is at least one pod in state `CrashLoopBackOff`",
								},
								Labels: map[string]string{
									DeploymentAppLabel: "test-app",
									DeploymentTagLabel: "tag",
									DeploymentSLOLabel: "true",
									"severity":         "test",
									"channel":          "#eng-releases",
								},
							},
						},
					},
					{
						Name: "test_app_deployment_imagepullbackoff",
						Rules: []monitoringv1.Rule{
							{
								Alert: "test_app_deployment_imagepullbackoff",
								Expr:  intstr.FromString("sum by (reason) (kube_pod_container_status_waiting_reason{reason=\"ImagePullBackOff\", container=\"test-app\", pod=~\"tag-.*\"}) > 0"),
								For:   "1m",
								Annotations: map[string]string{
									DeploymentMessageAnnotation: "There is at least one pod in state `ImagePullBackOff`",
									DeploymentSummaryAnnotation: "test-app - Deployment is failing ImagePullBackOff SLO - there is at least one pod in state `ImagePullBackOff`",
								},
								Labels: map[string]string{
									DeploymentAppLabel: "test-app",
									DeploymentTagLabel: "tag",
									DeploymentSLOLabel: "true",
									"severity":         "test",
									"channel":          "#eng-releases",
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

func TestSyncDeploymentRules(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	tests := []client.ObjectKey{
		{Name: "test-app-deployment-tag", Namespace: "testnamespace"},
	}
	ctx := context.TODO()

	for i := range tests {
		m.
			EXPECT().
			Get(ctx, mocks.ObjectKey(tests[i]), gomock.Any()).
			Return(common.NotFoundError).
			Times(1)
	}

	for i := range dpexpected.Items {
		for _, obj := range []runtime.Object{
			dpexpected.Items[i],
		} {
			m.
				EXPECT().
				Create(ctx, common.K8sEqual(obj)).
				Return(nil).
				Times(1)
		}
	}

	assert.NoError(t, dpplan.Apply(ctx, m, cluster, log), "Shouldn't return error.")
}
