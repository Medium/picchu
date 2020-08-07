package plan

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"go.medium.engineering/picchu/pkg/mocks"
	common "go.medium.engineering/picchu/pkg/plan/test"
	"go.medium.engineering/picchu/pkg/test"

	"github.com/golang/mock/gomock"
	wpav1 "github.com/practo/k8s-worker-pod-autoscaler/pkg/apis/workerpodautoscaler/v1"
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

	wpa := &wpav1.WorkerPodAutoScaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rr.Tag,
			Namespace: rr.Namespace,
		},
	}

	m.
		EXPECT().
		Delete(ctx, mocks.UpdateWPAObjectMeta(wpa)).
		Return(common.NotFoundError).
		Times(1)

	m.
		EXPECT().
		Get(ctx, mocks.ObjectKey(ok), mocks.Kind("ReplicaSet")).
		Return(common.NotFoundError).
		Times(1)

	assert.NoError(t, rr.Apply(ctx, m, cluster, log), "Shouldn't return error.")
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
	wpa := &wpav1.WorkerPodAutoScaler{
		Spec: wpav1.WorkerPodAutoScalerSpec{
			MinReplicas: &one,
			MaxReplicas: &one,
		},
	}
	rs := &appsv1.ReplicaSet{
		Spec: appsv1.ReplicaSetSpec{
			Replicas: &one,
		},
	}
	noReplicas := mocks.Callback(func(x interface{}) bool {
		switch o := x.(type) {
		case *appsv1.ReplicaSet:
			return *o.Spec.Replicas == 0
		default:
			return false
		}
	}, "ensure Spec.Replicas == 0")

	m.
		EXPECT().
		Delete(ctx, mocks.UpdateWPASpec(wpa)).
		Return(nil).
		Times(1)
	m.
		EXPECT().
		Get(ctx, mocks.ObjectKey(ok), mocks.UpdateReplicaSetSpec(rs)).
		Return(nil).
		Times(1)
	m.
		EXPECT().
		Update(ctx, noReplicas).
		Return(nil).
		Times(1)

	assert.NoError(t, rr.Apply(ctx, m, cluster, log), "Shouldn't return error.")
}
