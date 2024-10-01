package plan

import (
	"context"
	"testing"

	"go.medium.engineering/picchu/mocks"
	"go.medium.engineering/picchu/test"

	kedav1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	wpav1 "github.com/practo/k8s-worker-pod-autoscaler/pkg/apis/workerpodautoscaler/v1"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestDeleteRevision(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	plan := &DeleteRevision{
		Labels: map[string]string{
			"owner": "testcase",
		},
		Namespace: "testnamespace",
	}
	ctx := context.TODO()

	opts := &client.ListOptions{
		Namespace:     plan.Namespace,
		LabelSelector: labels.SelectorFromSet(plan.Labels),
	}

	secrets := []corev1.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testsecret",
				Namespace: "testnamespace",
			},
		},
	}

	configMaps := []corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testconfigmap",
				Namespace: "testnamespace",
			},
		},
	}

	replicaSets := []appsv1.ReplicaSet{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testreplicaset",
				Namespace: "testnamespace",
			},
		},
	}

	hpas := []autoscaling.HorizontalPodAutoscaler{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testhpa",
				Namespace: "testnamespace",
			},
		},
	}

	wpas := []wpav1.WorkerPodAutoScaler{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testwpa",
				Namespace: "testnamespace",
			},
		},
	}

	kedas := []kedav1.ScaledObject{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testkeda",
				Namespace: "testnamespace",
			},
		},
	}

	kedaAuths := []kedav1.TriggerAuthentication{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testkedatriggerauth",
				Namespace: "testnamespace",
			},
		},
	}

	m.
		EXPECT().
		List(ctx, mocks.InjectSecrets(secrets), mocks.ListOptions(opts)).
		Return(nil).
		Times(1)
	m.
		EXPECT().
		List(ctx, mocks.InjectConfigMaps(configMaps), mocks.ListOptions(opts)).
		Return(nil).
		Times(1)
	m.
		EXPECT().
		List(ctx, mocks.InjectReplicaSets(replicaSets), mocks.ListOptions(opts)).
		Return(nil).
		Times(1)
	m.
		EXPECT().
		List(ctx, mocks.InjectHorizontalPodAutoscalers(hpas), mocks.ListOptions(opts)).
		Return(nil).
		Times(1)
	m.
		EXPECT().
		List(ctx, mocks.InjectWorkerPodAutoscalers(wpas), mocks.ListOptions(opts)).
		Return(nil).
		Times(1)
	m.
		EXPECT().
		List(ctx, mocks.InjectKedaPodAutoscalers(kedas), mocks.ListOptions(opts)).
		Return(nil).
		Times(1)
	m.
		EXPECT().
		List(ctx, mocks.InjectKedaAuths(kedaAuths), mocks.ListOptions(opts)).
		Return(nil).
		Times(1)
	m.
		EXPECT().
		Delete(ctx, mocks.NamespacedName("testnamespace", "testsecret")).
		Return(nil).
		Times(1)
	m.
		EXPECT().
		Delete(ctx, mocks.NamespacedName("testnamespace", "testconfigmap")).
		Return(nil).
		Times(1)
	m.
		EXPECT().
		Delete(ctx, mocks.NamespacedName("testnamespace", "testreplicaset")).
		Return(nil).
		Times(1)
	m.
		EXPECT().
		Delete(ctx, mocks.NamespacedName("testnamespace", "testhpa")).
		Return(nil).
		Times(1)
	m.
		EXPECT().
		Delete(ctx, mocks.NamespacedName("testnamespace", "testkeda")).
		Return(nil).
		Times(1)
	m.
		EXPECT().
		Delete(ctx, mocks.NamespacedName("testnamespace", "testwpa")).
		Return(nil).
		Times(1)
	m.
		EXPECT().
		Delete(ctx, mocks.NamespacedName("testnamespace", "testkedatriggerauth")).
		Return(nil).
		Times(1)

	assert.NoError(t, plan.Apply(ctx, m, cluster, log), "Shouldn't return error.")
}
