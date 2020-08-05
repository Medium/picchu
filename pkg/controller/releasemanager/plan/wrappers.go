package plan

import (
	wpav1 "github.com/practo/k8s-worker-pod-autoscaler/pkg/apis/workerpodautoscaler/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type List interface {
	GetItems() []runtime.Object
	GetList() runtime.Object
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

func (s *SecretList) GetItems() (r []runtime.Object) {
	for _, i := range s.Item.Items {
		r = append(r, &i)
	}
	return
}

func (s *ConfigMapList) GetItems() (r []runtime.Object) {
	for _, i := range s.Item.Items {
		r = append(r, &i)
	}
	return
}

func (s *ReplicaSetList) GetItems() (r []runtime.Object) {
	for _, i := range s.Item.Items {
		r = append(r, &i)
	}
	return
}

func (s *HorizontalPodAutoscalerList) GetItems() (r []runtime.Object) {
	for _, i := range s.Item.Items {
		r = append(r, &i)
	}
	return
}

func (s *WorkerPodAutoscalerList) GetItems() (r []runtime.Object) {
	for _, i := range s.Item.Items {
		r = append(r, &i)
	}
	return
}

func (s *SecretList) GetList() (r runtime.Object) {
	return s.Item
}

func (s *ConfigMapList) GetList() (r runtime.Object) {
	return s.Item
}

func (s *ReplicaSetList) GetList() (r runtime.Object) {
	return s.Item
}

func (s *HorizontalPodAutoscalerList) GetList() (r runtime.Object) {
	return s.Item
}

func (s *WorkerPodAutoscalerList) GetList() (r runtime.Object) {
	return s.Item
}
