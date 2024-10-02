package plan

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"go.medium.engineering/picchu/mocks"
	common "go.medium.engineering/picchu/plan/test"
	"go.medium.engineering/picchu/test"

	kedav1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	wpav1 "github.com/practo/k8s-worker-pod-autoscaler/pkg/apis/workerpodautoscaler/v1"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
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
	keda := &kedav1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rr.Tag,
			Namespace: rr.Namespace,
		},
	}
	kedaTriggerAuth := &kedav1.TriggerAuthentication{
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
		Delete(ctx, mocks.UpdateKEDAObjectMeta(keda)).
		Return(common.NotFoundError).
		Times(1)
	m.
		EXPECT().
		Delete(ctx, mocks.UpdateKEDATriggerAuthObjectMeta(kedaTriggerAuth)).
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
	keda := &kedav1.ScaledObject{
		Spec: kedav1.ScaledObjectSpec{
			MinReplicaCount: &one,
			MaxReplicaCount: &one,
		},
	}
	kedaTriggerAuth := &kedav1.TriggerAuthentication{
		Spec: kedav1.TriggerAuthenticationSpec{
			PodIdentity: &kedav1.AuthPodIdentity{
				Provider: kedav1.PodIdentityProviderAwsKiam,
			},
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
		Delete(ctx, mocks.UpdateKEDASpec(keda)).
		Return(nil).
		Times(1)
	m.
		EXPECT().
		Delete(ctx, mocks.UpdateKEDATriggerAuthSpec(kedaTriggerAuth)).
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
