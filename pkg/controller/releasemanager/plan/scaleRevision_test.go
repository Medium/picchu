package plan

import (
	"context"
	"testing"

	"go.medium.engineering/picchu/pkg/mocks"
	"go.medium.engineering/picchu/pkg/test"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	autoscaling "k8s.io/api/autoscaling/v2beta2"
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
