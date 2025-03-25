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

	ddog "github.com/DataDog/datadog-operator/api/datadoghq/v1alpha1"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	ddogsloplan = &SyncDatadogSLOs{
		App:       "echo",
		Target:    "prod",
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
				Enabled:     true,
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
			},
			{
				Name:        "http-availability",
				Enabled:     true,
				Description: "test create example datadogSLO two",
				Query: picchuv1alpha1.DatadogSLOQuery{
					GoodEvents:  "per_minute(sum:istio.mesh.request.count.total{(response_code:2* OR response_code:3* OR response_code:4*) AND destination_service:echo.echo-production.svc.cluster.local AND reporter:destination}.as_count())",
					TotalEvents: "per_minute(sum:istio.mesh.request.count.total{destination_service:echo.echo-production.svc.cluster.local AND reporter:destination}.as_count())",
					BadEvents:   "per_minute(sum:istio.mesh.request.count.total{destination_service:echo.echo-production.svc.cluster.local AND reporter:destination AND response_code:5*}.as_count())",
				},
				Tags: []string{
					"service:example",
					"env:prod",
				},
				TargetThreshold: "99.9",
				Timeframe:       "7d",
				Type:            "metric",
			},
			{
				Name:        "disabled-http-availability",
				Enabled:     false,
				Description: "test create example datadogSLO two",
				Query: picchuv1alpha1.DatadogSLOQuery{
					GoodEvents:  "per_minute(sum:istio.mesh.request.count.total{(response_code:2* OR response_code:3* OR response_code:4*) AND destination_service:echo.echo-production.svc.cluster.local AND reporter:destination}.as_count())",
					TotalEvents: "per_minute(sum:istio.mesh.request.count.total{destination_service:echo.echo-production.svc.cluster.local AND reporter:destination}.as_count())",
					BadEvents:   "per_minute(sum:istio.mesh.request.count.total{destination_service:echo.echo-production.svc.cluster.local AND reporter:destination AND response_code:5*}.as_count())",
				},
				Tags: []string{
					"service:example",
					"env:prod",
				},
				TargetThreshold: "99.9",
				Timeframe:       "7d",
				Type:            "metric",
			},
		},
	}

	descrption_one  = "test create example datadogSLO one"
	descrption_two  = "test create example datadogSLO two"
	ddogsloexpected = &ddog.DatadogSLOList{
		Items: []ddog.DatadogSLO{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "echo-prod-irs-main-123-456",
					Namespace: "datadog",
					Labels: map[string]string{
						picchuv1alpha1.LabelApp:        "echo",
						picchuv1alpha1.LabelTag:        "main-123-456",
						picchuv1alpha1.LabelK8sName:    "echo",
						picchuv1alpha1.LabelK8sVersion: "main-123-456",
					},
				},
				Spec: ddog.DatadogSLOSpec{
					Name:        "echo-prod-main-123-456-istio-request-success",
					Description: &descrption_one,
					Query: &ddog.DatadogSLOQuery{
						Numerator:   "per_minute(sum:istio.mesh.request.count.total{destination_version:main-123-456 AND (response_code:2* OR response_code:3* OR response_code:4*) AND destination_service:tutu.tutu-production.svc.cluster.local AND reporter:destination}.as_count())",
						Denominator: "per_minute(sum:istio.mesh.request.count.total{destination_version:main-123-456 AND destination_service:tutu.tutu-production.svc.cluster.local AND reporter:destination}.as_count())",
					},
					Tags: []string{
						"service:example",
						"env:prod",
						"target:prod",
					},
					TargetThreshold: resource.MustParse("99.9"),
					Timeframe:       "7d",
					Type:            "metric",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					// 	return fmt.Sprintf("%s-%s-%s-%s-datadogSLO", p.App, p.Target, p.Tag, sloName)
					Name:      "echo-prod-ha-main-123-456",
					Namespace: "datadog",
					Labels: map[string]string{
						picchuv1alpha1.LabelApp:        "echo",
						picchuv1alpha1.LabelTag:        "main-123-456",
						picchuv1alpha1.LabelK8sName:    "echo",
						picchuv1alpha1.LabelK8sVersion: "main-123-456",
					},
				},
				Spec: ddog.DatadogSLOSpec{
					Name:        "echo-prod-main-123-456-http-availability",
					Description: &descrption_two,
					Query: &ddog.DatadogSLOQuery{
						Numerator:   "per_minute(sum:istio.mesh.request.count.total{destination_version:main-123-456 AND (response_code:2* OR response_code:3* OR response_code:4*) AND destination_service:echo.echo-production.svc.cluster.local AND reporter:destination}.as_count())",
						Denominator: "per_minute(sum:istio.mesh.request.count.total{destination_version:main-123-456 AND destination_service:echo.echo-production.svc.cluster.local AND reporter:destination}.as_count())",
					},
					Tags: []string{
						"service:example",
						"env:prod",
						"target:prod",
					},
					TargetThreshold: resource.MustParse("99.9"),
					Timeframe:       "7d",
					Type:            "metric",
				},
			},
		},
	}

	ddogmonitorplan = &SyncDatadogMonitors{
		App:       "echo",
		Target:    "prod",
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
			},
		},
	}

	ddogmonitorexpected = &ddog.DatadogMonitorList{
		Items: []ddog.DatadogMonitor{
			{
				ObjectMeta: metav1.ObjectMeta{
					// 	return fmt.Sprintf("%s-%s-%s-%s-datadogSLO", p.App, p.Target, p.Tag, sloName)
					Name:      "production-newsletterv3preview-production-newsletterv3previewsqsmessagesprocessedsuccess-456-eb",
					Namespace: "datadog",
					Labels: map[string]string{
						picchuv1alpha1.LabelApp:        "echo",
						picchuv1alpha1.LabelTag:        "main-123-456",
						picchuv1alpha1.LabelK8sName:    "echo",
						picchuv1alpha1.LabelK8sVersion: "main-123-456",
					},
				},
				Spec: ddog.DatadogMonitorSpec{
					Name:  "example-prod-main-123-456-example-slo-error-budget",
					Query: "error_budget(\"" + "echo-slo1" + "\").over(\"7d\") > 10",
					Type:  ddog.DatadogMonitorTypeSLO,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					// 	return fmt.Sprintf("%s-%s-%s-%s-datadogSLO", p.App, p.Target, p.Tag, sloName)
					Name:      "example-prod-exampleslo-456-br",
					Namespace: "datadog",
					Labels: map[string]string{
						picchuv1alpha1.LabelApp:        "echo",
						picchuv1alpha1.LabelTag:        "main-123-456",
						picchuv1alpha1.LabelK8sName:    "echo",
						picchuv1alpha1.LabelK8sVersion: "main-123-456",
					},
				},
				Spec: ddog.DatadogMonitorSpec{
					Name:  "example-prod-main-123-456-example-slo-burn-rate",
					Query: "burn_rate(\"" + "echo-slo1" + "\").over(\"7d\") > 10",
					Type:  ddog.DatadogMonitorTypeSLO,
				},
			},
		},
	}
)

func TestDatadogSLOs(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)

	defer ctrl.Finish()

	tests := []client.ObjectKey{
		{Name: "echo-prod-irs-main-123-456", Namespace: "datadog"},
		{Name: "echo-prod-ha-main-123-456", Namespace: "datadog"},
	}

	// tests_monitor := []client.ObjectKey{
	// 	{Name: "echo-prod-irs-main-123-456-br", Namespace: "datadog"},
	// 	{Name: "echo-prod-irs-main-123-456-eb", Namespace: "datadog"},
	// }

	ctx := context.TODO()

	for i := range tests {
		m.
			EXPECT().
			Get(ctx, mocks.ObjectKey(tests[i]), gomock.Any()).
			Return(common.NotFoundError).
			Times(1)
	}

	// for i := range tests_monitor {
	// 	m.
	// 		EXPECT().
	// 		Get(ctx, mocks.ObjectKey(tests_monitor[i]), gomock.Any()).
	// 		Return(common.NotFoundError).
	// 		Times(1)
	// }

	for i := range ddogsloexpected.Items {
		for _, obj := range []runtime.Object{
			&ddogsloexpected.Items[i],
		} {
			m.
				EXPECT().
				Create(ctx, common.K8sEqual(obj)).
				Return(nil).
				AnyTimes()
		}
	}

	// for i := range ddogmonitorexpected.Items {
	// 	for _, obj := range []runtime.Object{
	// 		&ddogmonitorexpected.Items[i],
	// 	} {
	// 		m.
	// 			EXPECT().
	// 			Create(ctx, common.K8sEqual(obj)).
	// 			Return(nil).
	// 			AnyTimes()
	// 	}
	// }

	assert.NoError(t, ddogsloplan.Apply(ctx, m, cluster, log), "Shouldn't return error.")
	// assert.NoError(t, ddogmonitorplan.Apply(ctx, m, cluster, log), "Shouldn't return error.")
}
