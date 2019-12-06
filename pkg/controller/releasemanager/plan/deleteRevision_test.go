package plan

import (
	"context"
	"testing"

	"go.medium.engineering/picchu/pkg/mocks"
	"go.medium.engineering/picchu/pkg/test"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
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
		corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testsecret",
				Namespace: "testnamespace",
			},
		},
	}

	configMaps := []corev1.ConfigMap{
		corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testconfigmap",
				Namespace: "testnamespace",
			},
		},
	}

	replicaSets := []appsv1.ReplicaSet{
		appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testreplicaset",
				Namespace: "testnamespace",
			},
		},
	}

	hpas := []autoscalingv1.HorizontalPodAutoscaler{
		autoscalingv1.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testhpa",
				Namespace: "testnamespace",
			},
		},
	}

	m.
		EXPECT().
		List(ctx, mocks.ListOptions(opts), mocks.InjectSecrets(secrets)).
		Return(nil).
		Times(1)
	m.
		EXPECT().
		List(ctx, mocks.ListOptions(opts), mocks.InjectConfigMaps(configMaps)).
		Return(nil).
		Times(1)
	m.
		EXPECT().
		List(ctx, mocks.ListOptions(opts), mocks.InjectReplicaSets(replicaSets)).
		Return(nil).
		Times(1)
	m.
		EXPECT().
		List(ctx, mocks.ListOptions(opts), mocks.InjectHorizontalPodAutoscalers(hpas)).
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

	assert.NoError(t, plan.Apply(ctx, m, 1.0, log), "Shouldn't return error.")
}
