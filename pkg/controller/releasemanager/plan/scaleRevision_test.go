package plan

import (
	"context"
	"testing"

	"go.medium.engineering/picchu/pkg/controller/releasemanager/mocks"
	"go.medium.engineering/picchu/pkg/test"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestScaleRevision(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	var thirty int32 = 30
	plan := &ScaleRevision{
		Tag:       "testtag",
		Namespace: "testnamespace",
		Min:       2,
		Max:       5,
		CPUTarget: &thirty,
		Labels:    map[string]string{},
	}
	ok := client.ObjectKey{Name: "testtag", Namespace: "testnamespace"}
	ctx := context.TODO()

	hpa := &autoscalingv1.HorizontalPodAutoscaler{
		Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
			MaxReplicas: 0,
		},
	}

	maxReplicasFive := mocks.Callback(func(x interface{}) bool {
		switch o := x.(type) {
		case *autoscalingv1.HorizontalPodAutoscaler:
			return o.Spec.MaxReplicas == 5
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
		Update(ctx, maxReplicasFive).
		Return(nil).
		Times(1)

	assert.NoError(t, plan.Apply(ctx, m, log), "Shouldn't return error.")
}
