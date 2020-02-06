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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	smplan = &SyncServiceMonitors{
		App:       "testapp",
		Namespace: "testnamespace",
		ServiceMonitors: []*picchuv1alpha1.ServiceMonitor{
			{
				Name:     "test1",
				SLORegex: true,
				Labels: map[string]string{
					"test": "test",
				},
				Spec: monitoringv1.ServiceMonitorSpec{
					Selector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test": "test",
						},
					},
					Endpoints: []monitoringv1.Endpoint{{
						Interval: "15s",
						MetricRelabelConfigs: []*monitoringv1.RelabelConfig{{
							Action:       "drop",
							SourceLabels: []string{"__name__"},
						}},
					}},
				},
			},
			{
				Name: "test2",
				Labels: map[string]string{
					"test": "test",
				},
				Annotations: map[string]string{
					"test": "test",
				},
				Spec: monitoringv1.ServiceMonitorSpec{
					Selector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test": "test",
						},
					},
					Endpoints: []monitoringv1.Endpoint{{
						Interval: "15s",
						MetricRelabelConfigs: []*monitoringv1.RelabelConfig{{
							Action:       "keep",
							Regex:        "(.*)",
							SourceLabels: []string{"__name__"},
						}},
					}},
				},
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
		}},
	}

	smexpected = monitoringv1.ServiceMonitorList{
		Items: []*monitoringv1.ServiceMonitor{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test1",
				Namespace: "testnamespace",
				Labels: map[string]string{
					"test": "test",
				},
			},
			Spec: monitoringv1.ServiceMonitorSpec{
				Endpoints: []monitoringv1.Endpoint{{
					Interval: "15s",
					MetricRelabelConfigs: []*monitoringv1.RelabelConfig{{
						Action:       "drop",
						Regex:        "test_metric|test_metric2",
						SourceLabels: []string{"__name__"},
					}},
				}},
				NamespaceSelector: monitoringv1.NamespaceSelector{
					MatchNames: []string{"testnamespace"},
				},
				Selector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"test": "test",
					},
				},
			},
		},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test2",
					Namespace: "testnamespace",
					Labels: map[string]string{
						"test": "test",
					},
				},
				Spec: monitoringv1.ServiceMonitorSpec{
					Endpoints: []monitoringv1.Endpoint{{
						Interval: "15s",
						MetricRelabelConfigs: []*monitoringv1.RelabelConfig{{
							Action:       "keep",
							Regex:        "(.*)",
							SourceLabels: []string{"__name__"},
						}},
					}},
					NamespaceSelector: monitoringv1.NamespaceSelector{
						MatchNames: []string{"testnamespace"},
					},
					Selector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test": "test",
						},
					},
				},
			},
		}}
)

func TestSyncServiceMonitors(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	tests := []client.ObjectKey{
		client.ObjectKey{Name: "test1", Namespace: "testnamespace"},
		client.ObjectKey{Name: "test2", Namespace: "testnamespace"},
	}
	ctx := context.TODO()

	for i := range tests {
		m.
			EXPECT().
			Get(ctx, mocks.ObjectKey(tests[i]), gomock.Any()).
			Return(common.NotFoundError).
			Times(1)
	}

	for i := range smexpected.Items {
		for _, obj := range []runtime.Object{
			smexpected.Items[i],
		} {
			m.
				EXPECT().
				Create(ctx, common.K8sEqual(obj)).
				Return(nil).
				Times(1)
		}
	}

	assert.NoError(t, smplan.Apply(ctx, m, 1.0, log), "Shouldn't return error.")
}
