package plan

import (
	"context"
	"testing"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"go.medium.engineering/picchu/pkg/mocks"
	common "go.medium.engineering/picchu/pkg/plan/test"
	"go.medium.engineering/picchu/pkg/test"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	defaultRevisionPlan = &SyncRevision{
		App:       "testapp",
		Tag:       "testtag",
		Namespace: "testnamespace",
		Labels: map[string]string{
			"test": "label",
		},
		Configs: []runtime.Object{},
		Ports: []picchuv1alpha1.PortInfo{{
			Name:          "http",
			Protocol:      "TCP",
			ContainerPort: 8080,
		}, {
			Name:          "status",
			Protocol:      "TCP",
			ContainerPort: 4242,
		}},
		Replicas: 2,
		Image:    "docker.medium.sh/test:testtag",
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				"cpu":    mustParseQuantity("4"),
				"memory": mustParseQuantity("4352Mi"),
			},
			Requests: corev1.ResourceList{
				"cpu":    mustParseQuantity("2"),
				"memory": mustParseQuantity("4352Mi"),
			},
		},
		IAMRole:            "testrole",
		ServiceAccountName: "testaccount",
	}
	retiredRevisionPlan = &SyncRevision{
		App:       "testapp",
		Tag:       "testtag",
		Namespace: "testnamespace",
		Labels: map[string]string{
			"test": "label",
		},
		Configs: []runtime.Object{},
		Ports: []picchuv1alpha1.PortInfo{{
			Name:          "http",
			Protocol:      "TCP",
			ContainerPort: 8080,
		}, {
			Name:          "status",
			Protocol:      "TCP",
			ContainerPort: 4242,
		}},
		Replicas: 0,
		Image:    "docker.medium.sh/test:testtag",
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				"cpu":    mustParseQuantity("4"),
				"memory": mustParseQuantity("4352Mi"),
			},
			Requests: corev1.ResourceList{
				"cpu":    mustParseQuantity("2"),
				"memory": mustParseQuantity("4352Mi"),
			},
		},
		IAMRole:            "testrole",
		ServiceAccountName: "testaccount",
	}
	zero   int32 = 0
	one    int32 = 1
	oneStr       = "1"

	defaultExpectedReplicaSet = &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testtag",
			Namespace: "testnamespace",
			Labels: map[string]string{
				"test": "label",
			},
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: &one,
			Selector: metav1.SetAsLabelSelector(map[string]string{
				"test":                          "label",
				"tag.picchu.medium.engineering": "testtag",
			}),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testtag",
					Namespace: "testnamespace",
					Annotations: map[string]string{
						picchuv1alpha1.AnnotationIAMRole: "testrole",
					},
					Labels: map[string]string{
						"test":                          "label",
						"tag.picchu.medium.engineering": "testtag",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "testaccount",
					Containers: []corev1.Container{{
						EnvFrom: []corev1.EnvFromSource{},
						Image:   "docker.medium.sh/test:testtag",
						Name:    "testapp",
						Ports: []corev1.ContainerPort{{
							Name:          "http",
							Protocol:      "TCP",
							ContainerPort: 8080,
						}, {
							Name:          "status",
							Protocol:      "TCP",
							ContainerPort: 4242,
						}},
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								"cpu":    mustParseQuantity("4"),
								"memory": mustParseQuantity("4352Mi"),
							},
							Requests: corev1.ResourceList{
								"cpu":    mustParseQuantity("2"),
								"memory": mustParseQuantity("4352Mi"),
							},
						},
						LivenessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/running",
									Port: intstr.FromString("status"),
								},
							},
							InitialDelaySeconds: 10,
							PeriodSeconds:       10,
							TimeoutSeconds:      1,
							SuccessThreshold:    1,
							FailureThreshold:    7,
						},
						ReadinessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/running",
									Port: intstr.FromString("status"),
								},
							},
							InitialDelaySeconds: 10,
							PeriodSeconds:       10,
							TimeoutSeconds:      1,
							SuccessThreshold:    1,
							FailureThreshold:    3,
						},
					}},
					DNSConfig: &corev1.PodDNSConfig{
						Options: []corev1.PodDNSConfigOption{{
							Name:  "ndots",
							Value: &oneStr,
						}},
					},
				},
			},
		},
	}
	retiredExpectedReplicaSet = &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testtag",
			Namespace: "testnamespace",
			Labels: map[string]string{
				"test": "label",
			},
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: &zero,
			Selector: metav1.SetAsLabelSelector(map[string]string{
				"test":                          "label",
				"tag.picchu.medium.engineering": "testtag",
			}),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testtag",
					Namespace: "testnamespace",
					Annotations: map[string]string{
						picchuv1alpha1.AnnotationIAMRole: "testrole",
					},
					Labels: map[string]string{
						"test":                          "label",
						"tag.picchu.medium.engineering": "testtag",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "testaccount",
					Containers: []corev1.Container{{
						EnvFrom: []corev1.EnvFromSource{},
						Image:   "docker.medium.sh/test:testtag",
						Name:    "testapp",
						Ports: []corev1.ContainerPort{{
							Name:          "http",
							Protocol:      "TCP",
							ContainerPort: 8080,
						}, {
							Name:          "status",
							Protocol:      "TCP",
							ContainerPort: 4242,
						}},
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								"cpu":    mustParseQuantity("4"),
								"memory": mustParseQuantity("4352Mi"),
							},
							Requests: corev1.ResourceList{
								"cpu":    mustParseQuantity("2"),
								"memory": mustParseQuantity("4352Mi"),
							},
						},
						LivenessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/running",
									Port: intstr.FromString("status"),
								},
							},
							InitialDelaySeconds: 10,
							PeriodSeconds:       10,
							TimeoutSeconds:      1,
							SuccessThreshold:    1,
							FailureThreshold:    7,
						},
						ReadinessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/running",
									Port: intstr.FromString("status"),
								},
							},
							InitialDelaySeconds: 10,
							PeriodSeconds:       10,
							TimeoutSeconds:      1,
							SuccessThreshold:    1,
							FailureThreshold:    3,
						},
					}},
					DNSConfig: &corev1.PodDNSConfig{
						Options: []corev1.PodDNSConfigOption{{
							Name:  "ndots",
							Value: &oneStr,
						}},
					},
				},
			},
		},
	}
)

func TestSyncRevisionNoChange(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	ok := client.ObjectKey{Name: "testtag", Namespace: "testnamespace"}
	ctx := context.TODO()

	m.
		EXPECT().
		Get(ctx, mocks.ObjectKey(ok), mocks.Kind("ReplicaSet")).
		Return(nil).
		Times(1)

	m.
		EXPECT().
		Update(ctx, common.K8sEqual(defaultExpectedReplicaSet)).
		Return(nil).
		Times(1)

	assert.NoError(t, defaultRevisionPlan.Apply(ctx, m, 0.5, log), "Shouldn't return error.")
}

func TestSyncRevisionWithChange(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	ok := client.ObjectKey{Name: "testtag", Namespace: "testnamespace"}
	ctx := context.TODO()

	m.
		EXPECT().
		Get(ctx, mocks.ObjectKey(ok), common.ReplicaSetCallback(func(rs *appsv1.ReplicaSet) bool {
			rs.Spec.Template.Spec.ServiceAccountName = "updateme"
			return true
		})).
		Return(nil).
		Times(1)

	m.
		EXPECT().
		Update(ctx, common.K8sEqual(defaultExpectedReplicaSet)).
		Return(nil).
		Times(1)

	assert.NoError(t, defaultRevisionPlan.Apply(ctx, m, 0.5, log), "Shouldn't return error.")
}

func TestSyncRevisionExistingReplicasZero(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	ok := client.ObjectKey{Name: "testtag", Namespace: "testnamespace"}
	ctx := context.TODO()

	m.
		EXPECT().
		Get(ctx, mocks.ObjectKey(ok), common.ReplicaSetCallback(func(rs *appsv1.ReplicaSet) bool {
			var zero int32 = 0
			rs.Spec.Replicas = &zero
			return true
		})).
		Return(nil).
		Times(1)

	m.
		EXPECT().
		Update(ctx, common.K8sEqual(defaultExpectedReplicaSet)).
		Return(nil).
		Times(1)

	assert.NoError(t, defaultRevisionPlan.Apply(ctx, m, 0.5, log), "Shouldn't return error.")
}

func TestSyncRevisionRetirement(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	ok := client.ObjectKey{Name: "testtag", Namespace: "testnamespace"}
	ctx := context.TODO()

	m.
		EXPECT().
		Get(ctx, mocks.ObjectKey(ok), common.ReplicaSetCallback(func(rs *appsv1.ReplicaSet) bool {
			var twenty int32 = 20
			rs.Spec.Replicas = &twenty
			return true
		})).
		Return(nil).
		Times(1)

	m.
		EXPECT().
		Update(ctx, common.K8sEqual(retiredExpectedReplicaSet)).
		Return(nil).
		Times(1)

	assert.NoError(t, retiredRevisionPlan.Apply(ctx, m, 0.5, log), "Shouldn't return error.")
}

func TestSyncRevisionWithCreate(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	ok := client.ObjectKey{Name: "testtag", Namespace: "testnamespace"}
	ctx := context.TODO()

	m.
		EXPECT().
		Get(ctx, mocks.ObjectKey(ok), common.ReplicaSetCallback(func(rs *appsv1.ReplicaSet) bool {
			rs.Spec.Template.Spec.ServiceAccountName = "updateme"
			return true
		})).
		Return(common.NotFoundError).
		Times(1)

	m.
		EXPECT().
		Create(ctx, common.K8sEqual(defaultExpectedReplicaSet)).
		Return(nil).
		Times(1)

	assert.NoError(t, defaultRevisionPlan.Apply(ctx, m, 0.5, log), "Shouldn't return error.")
}

func TestSyncRevisionWithCreateAndSecret(t *testing.T) {
	log := test.MustNewLogger()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockClient(ctrl)
	defer ctrl.Finish()

	rsok := client.ObjectKey{Name: "testtag", Namespace: "testnamespace"}
	cmok := client.ObjectKey{Name: "testconfigmap", Namespace: "testnamespace"}
	ctx := context.TODO()

	copy := *defaultRevisionPlan
	plan := &copy
	plan.Configs = []runtime.Object{
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "testconfigmap",
				Namespace:       "badnamespace",
				ResourceVersion: "removeme",
			},
			Data: map[string]string{
				"name": "bob",
			},
		},
	}

	expectedReplicaSet := defaultExpectedReplicaSet.DeepCopy()
	expectedReplicaSet.Spec.Template.Spec.Containers[0].EnvFrom = []corev1.EnvFromSource{{
		ConfigMapRef: &corev1.ConfigMapEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{Name: "testconfigmap"},
		},
	}}

	expectedConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testconfigmap",
			Namespace: "testnamespace",
			Labels: map[string]string{
				"test": "label",
			},
		},
		Data: map[string]string{
			"name": "bob",
		},
	}

	m.
		EXPECT().
		Get(ctx, mocks.ObjectKey(cmok), mocks.Kind("ConfigMap")).
		Return(common.NotFoundError).
		Times(1)

	m.
		EXPECT().
		Get(ctx, mocks.ObjectKey(rsok), mocks.Kind("ReplicaSet")).
		Return(common.NotFoundError).
		Times(1)

	m.
		EXPECT().
		Create(ctx, common.K8sEqual(expectedConfigMap)).
		Return(nil).
		Times(1)

	m.
		EXPECT().
		Create(ctx, common.K8sEqual(expectedReplicaSet)).
		Return(nil).
		Times(1)

	assert.NoError(t, plan.Apply(ctx, m, 0.5, log), "Shouldn't return error.")
}

func mustParseQuantity(val string) resource.Quantity {
	r, err := resource.ParseQuantity(val)
	if err != nil {
		panic("Failed to parse Quantity")
	}
	return r
}
