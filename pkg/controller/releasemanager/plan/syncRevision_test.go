package plan

import (
	"context"
	"testing"

	picchuv1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	common "go.medium.engineering/picchu/pkg/plan/test"
	"go.medium.engineering/picchu/pkg/test"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
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
			Name:          "status",
			Protocol:      "TCP",
			ContainerPort: 4242,
		}, {
			Name:          "http",
			Protocol:      "TCP",
			ContainerPort: 8080,
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
		Affinity: &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchFields: []corev1.NodeSelectorRequirement{
								{Key: "node-role.kubernetes.io/infrastructure", Operator: corev1.NodeSelectorOpExists},
							},
						},
					},
				},
			},
		},
		PriorityClassName: "default",
		Tolerations: []corev1.Toleration{
			{
				Key:    "infrastructure",
				Effect: corev1.TaintEffectNoExecute,
			},
		},
		IAMRole: "testrole",
		PodAnnotations: map[string]string{
			"sidecar.istio.io/statsInclusionPrefixes": "listener,cluster.outbound",
		},
		ServiceAccountName: "testaccount",
		EnvVars: []corev1.EnvVar{{
			Name: "NODE_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.hostIP",
				},
			},
		}},
		Lifecycle: &corev1.Lifecycle{
			PreStop: &corev1.Handler{
				Exec: &corev1.ExecAction{
					Command: []string{"/bin/sh", "-c", "sleep 20"},
				},
			},
		},
		Sidecars: []corev1.Container{
			{
				Name:  "test-1",
				Image: "test-1:latest",
			},
		},
		Volumes: []corev1.Volume{
			{
				Name: "shm",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{
						Medium: corev1.StorageMediumMemory,
					},
				},
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "shm",
				MountPath: "/dev/shm",
			},
		},
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
			Name:          "status",
			Protocol:      "TCP",
			ContainerPort: 4242,
		}, {
			Name:          "http",
			Protocol:      "TCP",
			ContainerPort: 8080,
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
		TypeMeta: metav1.TypeMeta{
			Kind:       "ReplicaSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testtag",
			Namespace: "testnamespace",
			Labels: map[string]string{
				"test": "label",
			},
			Annotations: map[string]string{
				"picchu.medium.engineering/autoscaler": "hpa",
			},
			ResourceVersion: "1",
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
						"sidecar.istio.io/statsInclusionPrefixes": "listener,cluster.outbound",
						picchuv1alpha1.AnnotationIAMRole:          "testrole",
						annotationDatadogTolerateUnready:          "true",
					},
					Labels: map[string]string{
						"test":                          "label",
						"tag.picchu.medium.engineering": "testtag",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "testaccount",
					Containers: []corev1.Container{{
						Env: []corev1.EnvVar{{
							Name: "NODE_IP",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "status.hostIP",
								},
							},
						}},
						EnvFrom: []corev1.EnvFromSource{},
						Image:   "docker.medium.sh/test:testtag",
						Name:    "testapp",
						Ports: []corev1.ContainerPort{{
							Name:          "status",
							Protocol:      "TCP",
							ContainerPort: 4242,
						}, {
							Name:          "http",
							Protocol:      "TCP",
							ContainerPort: 8080,
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
						Lifecycle: &corev1.Lifecycle{
							PreStop: &corev1.Handler{
								Exec: &corev1.ExecAction{
									Command: []string{"/bin/sh", "-c", "sleep 20"},
								},
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "shm",
								MountPath: "/dev/shm",
							},
						},
					},
						{
							Image: "test-1:latest",
							Name:  "test-1",
							Env: []corev1.EnvVar{{
								Name: "NODE_IP",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "status.hostIP",
									},
								},
							}},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "shm",
									MountPath: "/dev/shm",
								},
							},
						},
					},
					DNSConfig: &corev1.PodDNSConfig{
						Options: []corev1.PodDNSConfigOption{{
							Name:  "ndots",
							Value: &oneStr,
						}},
					},
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{
									{
										MatchFields: []corev1.NodeSelectorRequirement{
											{Key: "node-role.kubernetes.io/infrastructure", Operator: corev1.NodeSelectorOpExists},
										},
									},
								},
							},
						},
					},
					PriorityClassName: "default",
					Tolerations: []corev1.Toleration{
						{
							Key:    "infrastructure",
							Effect: corev1.TaintEffectNoExecute,
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "shm",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									Medium: corev1.StorageMediumMemory,
								},
							},
						},
					},
				},
			},
		},
	}
	retiredExpectedReplicaSet = &appsv1.ReplicaSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ReplicaSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testtag",
			Namespace: "testnamespace",
			Labels: map[string]string{
				"test": "label",
			},
			Annotations: map[string]string{
				"picchu.medium.engineering/autoscaler": "hpa",
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
						annotationDatadogTolerateUnready: "true",
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
							Name:          "status",
							Protocol:      "TCP",
							ContainerPort: 4242,
						}, {
							Name:          "http",
							Protocol:      "TCP",
							ContainerPort: 8080,
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
	ctx := context.TODO()
	log := test.MustNewLogger()
	cli := fakeClient(defaultExpectedReplicaSet)

	rsl := &appsv1.ReplicaSetList{}
	assert.NoError(t, defaultRevisionPlan.Apply(ctx, cli, halfCluster, log), "Shouldn't return error.")
	assert.NoError(t, cli.List(ctx, rsl))
	assert.Equal(t, 1, len(rsl.Items))
	common.ResourcesEqual(t, defaultExpectedReplicaSet, &rsl.Items[0])
}

func TestSyncRevisionWithChange(t *testing.T) {
	ctx := context.TODO()
	log := test.MustNewLogger()
	existing := defaultExpectedReplicaSet.DeepCopy()
	existing.Labels["name"] = "updateme"
	expected := defaultExpectedReplicaSet.DeepCopy()
	expected.ObjectMeta.ResourceVersion = "2"
	cli := fakeClient(existing)

	rsl := &appsv1.ReplicaSetList{}
	assert.NoError(t, defaultRevisionPlan.Apply(ctx, cli, halfCluster, log), "Shouldn't return error.")
	assert.NoError(t, cli.List(ctx, rsl))
	assert.Equal(t, 1, len(rsl.Items))

	common.ResourcesEqual(t, expected, &rsl.Items[0])
}

func TestSyncRevisionExistingReplicasZero(t *testing.T) {
	log := test.MustNewLogger()
	ctx := context.TODO()
	existing := defaultExpectedReplicaSet.DeepCopy()
	var zero int32 = 0
	existing.Spec.Replicas = &zero
	expected := defaultExpectedReplicaSet.DeepCopy()
	expected.ObjectMeta.ResourceVersion = "2"
	cli := fakeClient(existing)

	rsl := &appsv1.ReplicaSetList{}
	assert.NoError(t, defaultRevisionPlan.Apply(ctx, cli, halfCluster, log), "Shouldn't return error.")
	assert.NoError(t, cli.List(ctx, rsl))
	assert.Equal(t, 1, len(rsl.Items))
	common.ResourcesEqual(t, expected, &rsl.Items[0])
}

func TestSyncRevisionRetirement(t *testing.T) {
	log := test.MustNewLogger()
	ctx := context.TODO()
	existing := retiredExpectedReplicaSet.DeepCopy()
	var twenty int32 = 20
	existing.Spec.Replicas = &twenty
	expected := retiredExpectedReplicaSet.DeepCopy()
	expected.ObjectMeta.ResourceVersion = "1000"
	cli := fakeClient(existing)

	rsl := &appsv1.ReplicaSetList{}
	assert.NoError(t, retiredRevisionPlan.Apply(ctx, cli, halfCluster, log), "Shouldn't return error.")
	assert.NoError(t, cli.List(ctx, rsl))
	assert.Equal(t, 1, len(rsl.Items))
	common.ResourcesEqual(t, expected, &rsl.Items[0])
}

func TestSyncRevisionWithCreate(t *testing.T) {
	log := test.MustNewLogger()
	ctx := context.TODO()
	expected := defaultExpectedReplicaSet.DeepCopy()
	expected.ObjectMeta.ResourceVersion = "1"
	expected.TypeMeta = metav1.TypeMeta{}
	cli := fakeClient()

	rsl := &appsv1.ReplicaSetList{}
	assert.NoError(t, defaultRevisionPlan.Apply(ctx, cli, halfCluster, log), "Shouldn't return error.")
	assert.NoError(t, cli.List(ctx, rsl))
	assert.Equal(t, 1, len(rsl.Items))
	common.ResourcesEqual(t, expected, &rsl.Items[0])
}

func TestSyncRevisionWithCreateAndSecret(t *testing.T) {
	log := test.MustNewLogger()
	ctx := context.TODO()
	cli := fakeClient()

	expectedRs := defaultExpectedReplicaSet.DeepCopy()
	expectedRs.TypeMeta = metav1.TypeMeta{}
	for i := range expectedRs.Spec.Template.Spec.Containers {
		expectedRs.Spec.Template.Spec.Containers[i].EnvFrom = []corev1.EnvFromSource{{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: "testconfigmap"},
			},
		}}
	}

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

	expectedCm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testconfigmap",
			Namespace: "testnamespace",
			Labels: map[string]string{
				"test": "label",
			},
			ResourceVersion: "1",
		},
		Data: map[string]string{
			"name": "bob",
		},
	}

	assert.NoError(t, plan.Apply(ctx, cli, halfCluster, log), "Shouldn't return error.")

	rsl := &appsv1.ReplicaSetList{}
	assert.NoError(t, cli.List(ctx, rsl))
	assert.Equal(t, 1, len(rsl.Items))
	common.ResourcesEqual(t, expectedRs, &rsl.Items[0])

	cfm := &corev1.ConfigMapList{}
	assert.NoError(t, cli.List(ctx, cfm))
	assert.Equal(t, 1, len(cfm.Items))
	common.ResourcesEqual(t, expectedCm, &cfm.Items[0])
}

func mustParseQuantity(val string) resource.Quantity {
	r, err := resource.ParseQuantity(val)
	if err != nil {
		panic("Failed to parse Quantity")
	}
	return r
}
