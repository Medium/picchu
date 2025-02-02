package controllers

import (
	es "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v2"
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

type ExternalSecretList struct {
	Item *es.ExternalSecretList
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

func NewSecretList() *SecretList {
	return &SecretList{&corev1.SecretList{}}
}

func NewExternalSecretList() *ExternalSecretList {
	return &ExternalSecretList{&es.ExternalSecretList{}}
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

func (s *SecretList) GetItems() (r []runtime.Object) {
	for _, i := range s.Item.Items {
		r = append(r, &i)
	}
	return
}

func (s *ExternalSecretList) GetItems() (r []runtime.Object) {
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
	for i := range s.Item.Items {
		r = append(r, &s.Item.Items[i])
	}
	return
}

func (s *HorizontalPodAutoscalerList) GetItems() (r []runtime.Object) {
	for i := range s.Item.Items {
		r = append(r, &s.Item.Items[i])
	}
	return
}

func (s *SecretList) GetList() (r runtime.Object) {
	return s.Item
}

func (s *ExternalSecretList) GetList() (r runtime.Object) {
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
