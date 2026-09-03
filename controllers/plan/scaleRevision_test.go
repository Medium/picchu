package plan

import (
	"context"
	"strings"
	"testing"

	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/mocks"
	"go.medium.engineering/picchu/test"

	ddogv1alpha1 "github.com/DataDog/datadog-operator/api/datadoghq/v1alpha1"
	kedav1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	wpav1 "github.com/practo/k8s-worker-pod-autoscaler/pkg/apis/workerpodautoscaler/v1"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	autoscaling "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestScaleRevisionByCPU(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	var thirty int32 = 30
	plan := &ScaleRevision{
		Tag:       "testtag",
		Namespace: "testnamespace",
		Min:       4,
		Max:       10,
		CPUTarget: &thirty,
		Labels:    map[string]string{},
	}
	ok := client.ObjectKey{Name: "testtag", Namespace: "testnamespace"}
	ctx := context.TODO()

	hpa := &autoscaling.HorizontalPodAutoscaler{
		Spec: autoscaling.HorizontalPodAutoscalerSpec{
			MaxReplicas: 0,
		},
	}

	expected := mocks.Callback(func(x interface{}) bool {
		switch o := x.(type) {
		case *autoscaling.HorizontalPodAutoscaler:
			return o.Spec.MaxReplicas == 5 &&
				*o.Spec.Metrics[0].Resource.Target.AverageUtilization == 30 &&
				len(o.Spec.Metrics) == 1
		default:
			return false
		}
	}, "match expected hpa")

	m.
		EXPECT().
		Get(ctx, mocks.ObjectKey(ok), mocks.UpdateHPASpec(hpa)).
		Return(nil).
		Times(1)

	m.
		EXPECT().
		Update(ctx, expected).
		Return(nil).
		Times(1)

	assert.NoError(t, plan.Apply(ctx, m, halfCluster, log), "Shouldn't return error.")
}

func TestScaleRevisionByMemory(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	var thirty int32 = 30
	plan := &ScaleRevision{
		Tag:          "testtag",
		Namespace:    "testnamespace",
		Min:          4,
		Max:          10,
		MemoryTarget: &thirty,
		Labels:       map[string]string{},
	}
	ok := client.ObjectKey{Name: "testtag", Namespace: "testnamespace"}
	ctx := context.TODO()

	hpa := &autoscaling.HorizontalPodAutoscaler{
		Spec: autoscaling.HorizontalPodAutoscalerSpec{
			MaxReplicas: 0,
		},
	}

	expected := mocks.Callback(func(x interface{}) bool {
		switch o := x.(type) {
		case *autoscaling.HorizontalPodAutoscaler:
			return o.Spec.MaxReplicas == 5 &&
				*o.Spec.Metrics[0].Resource.Target.AverageUtilization == 30 &&
				len(o.Spec.Metrics) == 1
		default:
			return false
		}
	}, "match expected hpa")

	m.
		EXPECT().
		Get(ctx, mocks.ObjectKey(ok), mocks.UpdateHPASpec(hpa)).
		Return(nil).
		Times(1)

	m.
		EXPECT().
		Update(ctx, expected).
		Return(nil).
		Times(1)

	assert.NoError(t, plan.Apply(ctx, m, halfCluster, log), "Shouldn't return error.")
}

func TestScaleRevisionByRequestsRate(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	quantity := resource.NewQuantity(5, resource.DecimalSI)
	plan := &ScaleRevision{
		Tag:                "testtag",
		Namespace:          "testnamespace",
		Min:                4,
		Max:                10,
		RequestsRateMetric: "request_rate",
		RequestsRateTarget: quantity,
		Labels:             map[string]string{},
	}
	ok := client.ObjectKey{Name: "testtag", Namespace: "testnamespace"}
	ctx := context.TODO()

	hpa := &autoscaling.HorizontalPodAutoscaler{
		Spec: autoscaling.HorizontalPodAutoscalerSpec{
			MaxReplicas: 0,
		},
	}

	expected := mocks.Callback(func(x interface{}) bool {
		switch o := x.(type) {
		case *autoscaling.HorizontalPodAutoscaler:
			return o.Spec.MaxReplicas == 5 &&
				o.Spec.Metrics[0].Pods.Target.AverageValue.String() == "5" &&
				o.Spec.Metrics[0].Pods.Metric.Name == "request_rate" &&
				len(o.Spec.Metrics) == 1
		case *ddogv1alpha1.DatadogMetric:
			return true
		default:
			return false
		}
	}, "match Spec.MaxReplicas == 5")

	m.
		EXPECT().
		Get(ctx, mocks.ObjectKey(ok), mocks.UpdateHPASpec(hpa)).
		Return(nil).
		Times(1)

	m.
		EXPECT().
		Update(ctx, expected).
		Return(nil).
		Times(1)

	assert.NoError(t, plan.Apply(ctx, m, halfCluster, log), "Shouldn't return error.")
}

func TestScaleRevisionByRequestsRateAmbientMesh(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	quantity := resource.NewQuantity(5, resource.DecimalSI)
	plan := &ScaleRevision{
		Tag:                "testtag",
		Namespace:          "testnamespace",
		Min:                4,
		Max:                10,
		RequestsRateMetric: "istio_requests_rate_2m",
		RequestsRateTarget: quantity,
		AmbientMesh:        true,
		PrometheusAddress:  "http://prometheus-slo.monitoring-slo.svc.cluster.local:9090",
		Labels:             map[string]string{},
	}
	ok := client.ObjectKey{Name: "testtag", Namespace: "testnamespace"}
	ctx := context.TODO()

	var kedaMaxReplicas int32 = 0
	keda := &kedav1.ScaledObject{
		Spec: kedav1.ScaledObjectSpec{
			MaxReplicaCount: &kedaMaxReplicas,
		},
	}

	expected := mocks.Callback(func(x interface{}) bool {
		switch o := x.(type) {
		case *kedav1.ScaledObject:
			return *o.Spec.MaxReplicaCount == 5 &&
				o.Spec.ScaleTargetRef.Name == "testtag" &&
				o.Spec.ScaleTargetRef.Kind == "ReplicaSet" &&
				len(o.Spec.Triggers) == 1 &&
				o.Spec.Triggers[0].Type == "prometheus" &&
				o.Spec.Triggers[0].Metadata["threshold"] == "5" &&
				o.Spec.Triggers[0].Metadata["serverAddress"] == "http://prometheus-slo.monitoring-slo.svc.cluster.local:9090"
		default:
			return false
		}
	}, "match KEDA ScaledObject with prometheus trigger")

	m.
		EXPECT().
		Delete(ctx, mocks.NamespacedName("testnamespace", "testtag")).
		Return(nil).
		Times(1)

	m.
		EXPECT().
		Get(ctx, mocks.ObjectKey(ok), mocks.UpdateKEDASpec(keda)).
		Return(nil).
		Times(1)

	m.
		EXPECT().
		Update(ctx, expected).
		Return(nil).
		Times(1)

	assert.NoError(t, plan.Apply(ctx, m, halfCluster, log), "Shouldn't return error.")
}

func TestScaleRevisionWithWPA(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	plan := &ScaleRevision{
		Tag:       "testtag",
		Namespace: "testnamespace",
		Min:       4,
		Max:       10,
		Worker:    &picchuv1alpha1.WorkerScaleInfo{},
		Labels:    map[string]string{},
	}
	ok := client.ObjectKey{Name: "testtag", Namespace: "testnamespace"}
	ctx := context.TODO()

	var wpaMaxReplicas int32 = 0
	wpa := &wpav1.WorkerPodAutoScaler{
		Spec: wpav1.WorkerPodAutoScalerSpec{
			MaxReplicas: &wpaMaxReplicas,
		},
	}

	expected := mocks.Callback(func(x interface{}) bool {
		switch o := x.(type) {
		case *wpav1.WorkerPodAutoScaler:
			return *o.Spec.MaxReplicas == 5 &&
				o.Spec.ReplicaSetName == "testtag"
		default:
			return false
		}
	}, "match Spec.MaxReplicas == 5 and Spec.ReplicaSetName == testtag")

	m.
		EXPECT().
		Get(ctx, mocks.ObjectKey(ok), mocks.UpdateWPASpec(wpa)).
		Return(nil).
		Times(1)

	m.
		EXPECT().
		Update(ctx, expected).
		Return(nil).
		Times(1)

	assert.NoError(t, plan.Apply(ctx, m, halfCluster, log), "Shouldn't return error.")
}

func TestScaleRevisionWithKEDA(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	plan := &ScaleRevision{
		Tag:        "testtag",
		Namespace:  "testnamespace",
		Min:        4,
		Max:        10,
		KedaWorker: &picchuv1alpha1.KedaScaleInfo{},
		Labels:     map[string]string{},
	}
	ok := client.ObjectKey{Name: "testtag", Namespace: "testnamespace"}
	ctx := context.TODO()

	var kedaMaxReplicas int32 = 0
	keda := &kedav1.ScaledObject{
		Spec: kedav1.ScaledObjectSpec{
			MaxReplicaCount: &kedaMaxReplicas,
		},
	}

	expected := mocks.Callback(func(x interface{}) bool {
		switch o := x.(type) {
		case *kedav1.ScaledObject:
			return *o.Spec.MaxReplicaCount == 5 &&
				o.Spec.ScaleTargetRef.Name == "testtag"
		case *kedav1.TriggerAuthentication:
			return true
		default:
			return false
		}
	}, "match Spec.MaxReplicaCount == 5 and Spec.ScaleTargetRef.Name == testtag")

	m.
		EXPECT().
		Get(ctx, mocks.ObjectKey(ok), mocks.UpdateKEDASpec(keda)).
		Return(nil).
		Times(2)

	m.
		EXPECT().
		Update(ctx, expected).
		Return(nil).
		Times(2)

	assert.NoError(t, plan.Apply(ctx, m, halfCluster, log), "Shouldn't return error.")
}

func TestDontScaleRevision(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	plan := &ScaleRevision{
		Tag:       "testtag",
		Namespace: "testnamespace",
		Min:       4,
		Max:       10,
		Labels:    map[string]string{},
	}
	ctx := context.TODO()

	m.
		EXPECT().
		Delete(ctx, mocks.NamespacedName("testnamespace", "testtag")).
		Return(nil).
		Times(1)

	assert.NoError(t, plan.Apply(ctx, m, halfCluster, log), "Shouldn't return error.")
}

// TestScaleRevisionHPANameTruncatedOverLimit verifies that when Tag is longer than the
// HPA name limit, applyHPA truncates the HPA's own name (keeping the trailing portion,
// since that's what carries the date-time-commit info that needs to stay unique) while
// Spec.ScaleTargetRef.Name keeps the full, untruncated Tag so it still matches the real
// ReplicaSet name (created elsewhere with the untruncated Tag).
func TestScaleRevisionHPANameTruncatedOverLimit(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	longTag := "auto-109151-add-design-system-typescript-support-20260903-075707-40a3c27da1"
	assert.Greater(t, len(longTag), 54, "sanity check: fixture tag should be over the HPA name limit")

	expectedHPAName := longTag[len(longTag)-54:]

	var thirty int32 = 30
	plan := &ScaleRevision{
		Tag:       longTag,
		Namespace: "testnamespace",
		Min:       4,
		Max:       10,
		CPUTarget: &thirty,
		Labels:    map[string]string{},
	}
	ok := client.ObjectKey{Name: expectedHPAName, Namespace: "testnamespace"}
	ctx := context.TODO()

	hpa := &autoscaling.HorizontalPodAutoscaler{
		Spec: autoscaling.HorizontalPodAutoscalerSpec{
			MaxReplicas: 0,
		},
	}

	expected := mocks.Callback(func(x interface{}) bool {
		switch o := x.(type) {
		case *autoscaling.HorizontalPodAutoscaler:
			return o.ObjectMeta.Name == expectedHPAName &&
				o.Spec.ScaleTargetRef.Name == longTag &&
				o.Spec.MaxReplicas == 5 &&
				*o.Spec.Metrics[0].Resource.Target.AverageUtilization == 30
		default:
			return false
		}
	}, "match truncated hpa name with untruncated ScaleTargetRef")

	m.
		EXPECT().
		Get(ctx, mocks.ObjectKey(ok), mocks.UpdateHPASpec(hpa)).
		Return(nil).
		Times(1)

	m.
		EXPECT().
		Update(ctx, expected).
		Return(nil).
		Times(1)

	assert.NoError(t, plan.Apply(ctx, m, halfCluster, log), "Shouldn't return error.")
}

// TestScaleRevisionWithKEDANameTruncatedOverLimit verifies that applyKeda truncates the
// ScaledObject's own name (via hpaName) when Tag is over the limit, since KEDA's operator
// prepends "keda-hpa-" to it when generating the underlying HPA. ScaleTargetRef.Name must
// stay the full, untruncated Tag to match the real ReplicaSet name.
func TestScaleRevisionWithKEDANameTruncatedOverLimit(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	longTag := "auto-109151-add-design-system-typescript-support-20260903-075707-40a3c27da1"
	assert.Greater(t, len(longTag), 54, "sanity check: fixture tag should be over the HPA name limit")

	expectedScaledObjectName := longTag[len(longTag)-54:]

	plan := &ScaleRevision{
		Tag:        longTag,
		Namespace:  "testnamespace",
		Min:        4,
		Max:        10,
		KedaWorker: &picchuv1alpha1.KedaScaleInfo{},
		Labels:     map[string]string{},
	}
	// TriggerAuthentication keeps the full, untruncated Tag as its name (it's
	// not the object KEDA prepends "keda-hpa-" to), while the ScaledObject's
	// own name is truncated via hpaName - so each Get uses a different key.
	triggerAuthKey := client.ObjectKey{Name: longTag, Namespace: "testnamespace"}
	scaledObjectKey := client.ObjectKey{Name: expectedScaledObjectName, Namespace: "testnamespace"}
	ctx := context.TODO()

	var kedaMaxReplicas int32 = 0
	keda := &kedav1.ScaledObject{
		Spec: kedav1.ScaledObjectSpec{
			MaxReplicaCount: &kedaMaxReplicas,
		},
	}

	expected := mocks.Callback(func(x interface{}) bool {
		switch o := x.(type) {
		case *kedav1.ScaledObject:
			return o.ObjectMeta.Name == expectedScaledObjectName &&
				o.Spec.ScaleTargetRef.Name == longTag &&
				*o.Spec.MaxReplicaCount == 5
		case *kedav1.TriggerAuthentication:
			return true
		default:
			return false
		}
	}, "match truncated ScaledObject name with untruncated ScaleTargetRef")

	m.
		EXPECT().
		Get(ctx, mocks.ObjectKey(triggerAuthKey), mocks.UpdateKEDASpec(keda)).
		Return(nil).
		Times(1)

	m.
		EXPECT().
		Get(ctx, mocks.ObjectKey(scaledObjectKey), mocks.UpdateKEDASpec(keda)).
		Return(nil).
		Times(1)

	m.
		EXPECT().
		Update(ctx, expected).
		Return(nil).
		Times(2)

	assert.NoError(t, plan.Apply(ctx, m, halfCluster, log), "Shouldn't return error.")
}

// TestHpaNameTrimsLeadingDash verifies hpaName trims a leading "-" left behind when the
// 54-char cut happens to land right at a separator, since a k8s object name can't start
// with "-".
func TestHpaNameTrimsLeadingDash(t *testing.T) {
	// Constructed so the last 54 characters begin exactly on a "-": a 10-char prefix,
	// then "-", then 53 more characters (10 + 1 + 53 = 64 total, so len-54 == 10, which
	// is exactly where the "-" sits).
	tag := "auto123456" + "-" + strings.Repeat("y", 53)
	plan := &ScaleRevision{Tag: tag}

	name := plan.hpaName()

	assert.False(t, strings.HasPrefix(name, "-"), "hpaName should never start with a separator")
	assert.Equal(t, strings.Repeat("y", 53), name)
}
