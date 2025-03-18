package plan

import (
	"context"
	"reflect"
	_ "runtime"
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	ddog "github.com/DataDog/datadog-operator/api/datadoghq/v1alpha1"
	"github.com/stretchr/testify/assert"
	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/mocks"
	common "go.medium.engineering/picchu/plan/test"
	"go.medium.engineering/picchu/test"
	"go.uber.org/mock/gomock"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	true_val     = true
	escalate_irs = "ESCALATED: The istio-request-success canary query is firing @slack-eng-watch-alerts-testing"
	escalate_ha  = "ESCALATED: The http-availability canary query is firing @slack-eng-watch-alerts-testing"

	five     = int64(5)
	renotify = []datadogV1.MonitorRenotifyStatusType{datadogV1.MONITORRENOTIFYSTATUSTYPE_ALERT, datadogV1.MONITORRENOTIFYSTATUSTYPE_NO_DATA}

	canary_thresh = "0.0"
	dcmplan       = &SyncDatadogCanaryMonitors{
		App:       "echo",
		Target:    "production",
		Namespace: "datadog",
		Tag:       "main-123-456",
		Labels: map[string]string{
			picchuv1alpha1.LabelApp:        "echo",
			picchuv1alpha1.LabelTag:        "main-123-456",
			picchuv1alpha1.LabelK8sName:    "echo",
			picchuv1alpha1.LabelK8sVersion: "main-123-456",
		},
		DatadogSLOs: []*picchuv1alpha1.DatadogSLO{
			{
				Name:        "istio-request-success",
				Description: "test create example datadogSLO one",
				Query: picchuv1alpha1.DatadogSLOQuery{
					GoodEvents:  "per_minute(sum:istio.mesh.request.count.total{(response_code:2* OR response_code:3* OR response_code:4*) AND destination_service:tutu.tutu-production.svc.cluster.local AND reporter:destination}.as_count())",
					TotalEvents: "per_minute(sum:istio.mesh.request.count.total{destination_service:tutu.tutu-production.svc.cluster.local AND reporter:destination}.as_count())",
					BadEvents:   "per_minute(sum:istio.mesh.request.count.total{destination_service:tutu.tutu-production.svc.cluster.local AND reporter:destination AND response_code:5*}.as_count())",
				},
				Tags: []string{
					"service:example",
					"env:prod",
				},
				TargetThreshold: "99.9",
				Timeframe:       "7d",
				Type:            "metric",
				Canary: picchuv1alpha1.DatadogSLOCanaryConfig{
					Enabled:          true,
					AllowancePercent: float64(1),
				},
			},
			{
				Name:        "http-availability",
				Description: "test create example datadogSLO two",
				Query: picchuv1alpha1.DatadogSLOQuery{
					GoodEvents:  "per_minute(sum:istio.mesh.request.count.total{(response_code:2* OR response_code:3* OR response_code:4*) AND destination_service:tutu.tutu-production.svc.cluster.local AND reporter:destination}.as_count())",
					TotalEvents: "per_minute(sum:istio.mesh.request.count.total{destination_service:tutu.tutu-production.svc.cluster.local AND reporter:destination}.as_count())",
					BadEvents:   "per_minute(sum:istio.mesh.request.count.total{destination_service:tutu.tutu-production.svc.cluster.local AND reporter:destination AND response_code:5*}.as_count())",
				},
				Tags: []string{
					"service:example",
					"env:prod",
				},
				TargetThreshold: "99.9",
				Timeframe:       "7d",
				Type:            "metric",
				Canary: picchuv1alpha1.DatadogSLOCanaryConfig{
					Enabled:          true,
					AllowancePercent: float64(1),
				},
			},
		},
	}

	dcmexpected = &ddog.DatadogMonitorList{
		Items: []ddog.DatadogMonitor{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "echo-prod-irs-main-123-456-canary",
					Namespace: "datadog",
					Labels: map[string]string{
						picchuv1alpha1.LabelApp:        "echo",
						picchuv1alpha1.LabelTag:        "main-123-456",
						picchuv1alpha1.LabelK8sName:    "echo",
						picchuv1alpha1.LabelK8sVersion: "main-123-456",
					},
				},
				Spec: ddog.DatadogMonitorSpec{
					Name:    "echo-production-main-123-456-istio-request-success-canary",
					Message: "The istio-request-success canary query is firing @slack-eng-watch-alerts-testing",
					Query:   "sum(last_2m):((per_minute(sum:istio.mesh.request.count.total{destination_version:main-123-456 AND destination_service:tutu.tutu-production.svc.cluster.local AND reporter:destination AND response_code:5*}.as_count()) / per_minute(sum:istio.mesh.request.count.total{destination_version:main-123-456 AND destination_service:tutu.tutu-production.svc.cluster.local AND reporter:destination}.as_count())) - 0.01) - (per_minute(sum:istio.mesh.request.count.total{destination_service:tutu.tutu-production.svc.cluster.local AND reporter:destination AND response_code:5*}.as_count()) / per_minute(sum:istio.mesh.request.count.total{destination_service:tutu.tutu-production.svc.cluster.local AND reporter:destination}.as_count())) >= 0",
					Tags: []string{
						"service:example",
						"env:prod",
					},
					Type: ddog.DatadogMonitorTypeMetric,
					Options: ddog.DatadogMonitorOptions{
						EnableLogsSample:       &true_val,
						EscalationMessage:      &escalate_irs,
						EvaluationDelay:        &five,
						IncludeTags:            &true_val,
						NoDataTimeframe:        &five,
						NotificationPresetName: "show_all",
						NotifyNoData:           &true_val,
						RenotifyInterval:       &five,
						RenotifyStatuses:       renotify,
						RequireFullWindow:      &true_val,
						Thresholds: &ddog.DatadogMonitorOptionsThresholds{
							Critical: &canary_thresh,
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "echo-prod-ha-main-123-456-canary",
					Namespace: "datadog",
					Labels: map[string]string{
						picchuv1alpha1.LabelApp:        "echo",
						picchuv1alpha1.LabelTag:        "main-123-456",
						picchuv1alpha1.LabelK8sName:    "echo",
						picchuv1alpha1.LabelK8sVersion: "main-123-456",
					},
				},
				Spec: ddog.DatadogMonitorSpec{
					Name:    "echo-production-main-123-456-http-availability-canary",
					Message: "The http-availability canary query is firing @slack-eng-watch-alerts-testing",
					Query:   "sum(last_2m):((per_minute(sum:istio.mesh.request.count.total{destination_version:main-123-456 AND destination_service:tutu.tutu-production.svc.cluster.local AND reporter:destination AND response_code:5*}.as_count()) / per_minute(sum:istio.mesh.request.count.total{destination_version:main-123-456 AND destination_service:tutu.tutu-production.svc.cluster.local AND reporter:destination}.as_count())) - 0.01) - (per_minute(sum:istio.mesh.request.count.total{destination_service:tutu.tutu-production.svc.cluster.local AND reporter:destination AND response_code:5*}.as_count()) / per_minute(sum:istio.mesh.request.count.total{destination_service:tutu.tutu-production.svc.cluster.local AND reporter:destination}.as_count())) >= 0",
					Tags: []string{
						"service:example",
						"env:prod",
					},
					Type: ddog.DatadogMonitorTypeMetric,
					Options: ddog.DatadogMonitorOptions{
						EnableLogsSample:       &true_val,
						EscalationMessage:      &escalate_ha,
						EvaluationDelay:        &five,
						IncludeTags:            &true_val,
						NoDataTimeframe:        &five,
						NotificationPresetName: "show_all",
						NotifyNoData:           &true_val,
						RenotifyInterval:       &five,
						RenotifyStatuses:       renotify,
						RequireFullWindow:      &true_val,
						Thresholds: &ddog.DatadogMonitorOptionsThresholds{
							Critical: &canary_thresh,
						},
					},
				},
			},
		},
	}
)

func TestSyncDatadogCanaryMonitors(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	tests := []client.ObjectKey{
		{Name: "echo-prod-irs-main-123-456-canary", Namespace: "datadog"},
		{Name: "echo-prod-ha-main-123-456-canary", Namespace: "datadog"},
	}
	ctx := context.TODO()

	for i := range tests {
		m.
			EXPECT().
			Get(ctx, mocks.ObjectKey(tests[i]), gomock.Any()).
			Return(common.NotFoundError).
			Times(1)
	}

	for i := range dcmexpected.Items {
		for _, obj := range []runtime.Object{
			&dcmexpected.Items[i],
		} {
			m.
				EXPECT().
				Create(ctx, common.K8sEqual(obj)).
				Return(nil).
				Times(1)
		}
	}

	assert.NoError(t, dcmplan.Apply(ctx, m, cluster, log), "Shouldn't return error.")
}

func TestFormatDDogAllowancePercent(t *testing.T) {
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
		dm := SyncDatadogCanaryMonitors{}
		d := &picchuv1alpha1.DatadogSLO{
			Canary: picchuv1alpha1.DatadogSLOCanaryConfig{
				AllowancePercent: i.float,
			},
		}

		actual := dm.formatAllowancePercent(d, log)
		if !reflect.DeepEqual(i.expected, actual) {
			t.Errorf("Expected did not match actual. Expected: %v. Actual: %v", i.expected, actual)
			t.Fail()
		}
	}
}
