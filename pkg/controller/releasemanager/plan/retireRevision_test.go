package plan

import (
	"context"
	"testing"

	"go.medium.engineering/picchu/pkg/mocks"
	common "go.medium.engineering/picchu/pkg/plan/test"
	"go.medium.engineering/picchu/pkg/test"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestRetireMissingRevision(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	rr := &RetireRevision{
		Tag:       "testtag",
		Namespace: "testnamespace",
	}
	ok := client.ObjectKey{Name: "testtag", Namespace: "testnamespace"}
	ctx := context.TODO()

	m.
		EXPECT().
		Get(ctx, mocks.ObjectKey(ok), mocks.Kind("ReplicaSet")).
		Return(common.NotFoundError).
		Times(1)

	assert.NoError(t, rr.Apply(ctx, m, 1.0, log), "Shouldn't return error.")
}

func TestRetireExistingRevision(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	rr := &RetireRevision{
		Tag:       "testtag",
		Namespace: "testnamespace",
	}
	ok := client.ObjectKey{Name: "testtag", Namespace: "testnamespace"}
	ctx := context.TODO()

	var one int32 = 1
	rs := &appsv1.ReplicaSet{
		Spec: appsv1.ReplicaSetSpec{
			Replicas: &one,
		},
	}

	m.
		EXPECT().
		Get(ctx, mocks.ObjectKey(ok), mocks.UpdateReplicaSetSpec(rs)).
		Return(nil).
		Times(1)

	noreplicas := mocks.Callback(func(x interface{}) bool {
		switch o := x.(type) {
		case *appsv1.ReplicaSet:
			return *o.Spec.Replicas == 0
		default:
			return false
		}
	}, "ensure Spec.Replicas == 0")

	m.
		EXPECT().
		Update(ctx, noreplicas).
		Return(nil).
		Times(1)

	assert.NoError(t, rr.Apply(ctx, m, 1.0, log), "Shouldn't return error.")
}
