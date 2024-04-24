package plan

import (
	kedav1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	wpav1 "github.com/practo/k8s-worker-pod-autoscaler/pkg/apis/workerpodautoscaler/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type List interface {
	GetItems() []client.Object
	GetList() client.ObjectList
}

type SecretList struct {
	Item *corev1.SecretList
}

type ConfigMapList struct {
	Item *corev1.ConfigMapList
}

type ReplicaSetList struct {
	Item *appsv1.ReplicaSetList
}

type HorizontalPodAutoscalerList struct {
	Item *autoscaling.HorizontalPodAutoscalerList
}

type WorkerPodAutoscalerList struct {
	Item *wpav1.WorkerPodAutoScalerList
}

type KedaPodAutoscalerList struct {
	Item *kedav1.ScaledObjectList
}

type KedaAuthList struct {
	Item *kedav1.TriggerAuthenticationList
}

func NewSecretList() *SecretList {
	return &SecretList{&corev1.SecretList{}}
}

func NewConfigMapList() *ConfigMapList {
	return &ConfigMapList{&corev1.ConfigMapList{}}
}

func NewReplicaSetList() *ReplicaSetList {
	return &ReplicaSetList{&appsv1.ReplicaSetList{}}
}

func NewHorizontalPodAutoscalerList() *HorizontalPodAutoscalerList {
	return &HorizontalPodAutoscalerList{&autoscaling.HorizontalPodAutoscalerList{}}
}

func NewWorkerPodAutoscalerList() *WorkerPodAutoscalerList {
	return &WorkerPodAutoscalerList{&wpav1.WorkerPodAutoScalerList{}}
}

func NewKedaPodAutoscalerList() *KedaPodAutoscalerList {
	return &KedaPodAutoscalerList{&kedav1.ScaledObjectList{}}
}

func NewKedaAuthList() *KedaAuthList {
	return &KedaAuthList{&kedav1.TriggerAuthenticationList{}}
}

func (s *SecretList) GetItems() (r []client.Object) {
	for _, i := range s.Item.Items {
		r = append(r, &i)
	}
	return
}

func (s *ConfigMapList) GetItems() (r []client.Object) {
	for _, i := range s.Item.Items {
		r = append(r, &i)
	}
	return
}

func (s *ReplicaSetList) GetItems() (r []client.Object) {
	for _, i := range s.Item.Items {
		r = append(r, &i)
	}
	return
}

func (s *HorizontalPodAutoscalerList) GetItems() (r []client.Object) {
	for _, i := range s.Item.Items {
		r = append(r, &i)
	}
	return
}

func (s *WorkerPodAutoscalerList) GetItems() (r []client.Object) {
	for _, i := range s.Item.Items {
		r = append(r, &i)
	}
	return
}

func (s *KedaPodAutoscalerList) GetItems() (r []client.Object) {
	for _, i := range s.Item.Items {
		r = append(r, &i)
	}
	return
}

func (s *KedaAuthList) GetItems() (r []client.Object) {
	for _, i := range s.Item.Items {
		r = append(r, &i)
	}
	return
}

func (s *SecretList) GetList() (r client.ObjectList) {
	return s.Item
}

func (s *ConfigMapList) GetList() (r client.ObjectList) {
	return s.Item
}

func (s *ReplicaSetList) GetList() (r client.ObjectList) {
	return s.Item
}

func (s *HorizontalPodAutoscalerList) GetList() (r client.ObjectList) {
	return s.Item
}

func (s *WorkerPodAutoscalerList) GetList() (r client.ObjectList) {
	return s.Item
}

func (s *KedaPodAutoscalerList) GetList() (r client.ObjectList) {
	return s.Item
}

func (s *KedaAuthList) GetList() (r client.ObjectList) {
	return s.Item
}
