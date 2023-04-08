package plan

import (
	"context"
	"testing"

	picchuv1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/mocks"
	"go.medium.engineering/picchu/test"

	"github.com/golang/mock/gomock"
	wpav1 "github.com/practo/k8s-worker-pod-autoscaler/pkg/apis/workerpodautoscaler/v1"
	"github.com/stretchr/testify/assert"
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
