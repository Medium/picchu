package mocks

import (
	"github.com/golang/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
)

// InjectSecrets puts secrets into a *SecretList
func InjectSecrets(secrets []corev1.Secret) gomock.Matcher {
	fn := func(x interface{}) bool {
		switch o := x.(type) {
		case *corev1.SecretList:
			for _, secret := range secrets {
				o.Items = append(o.Items, secret)
			}
			return true
		default:
			return false
		}
	}
	return Callback(fn, "injects secrets")
}

// InjectConfigMaps puts configmaps into a *ConfigMapList
func InjectConfigMaps(configMaps []corev1.ConfigMap) gomock.Matcher {
	fn := func(x interface{}) bool {
		switch o := x.(type) {
		case *corev1.ConfigMapList:
			for _, configMap := range configMaps {
				o.Items = append(o.Items, configMap)
			}
			return true
		default:
			return false
		}
	}
	return Callback(fn, "injects configMaps")
}

// InjectReplicaSets puts replicasets into a *ReplicaSetList
func InjectReplicaSets(replicaSets []appsv1.ReplicaSet) gomock.Matcher {
	fn := func(x interface{}) bool {
		switch o := x.(type) {
		case *appsv1.ReplicaSetList:
			for _, replicaSet := range replicaSets {
				o.Items = append(o.Items, replicaSet)
			}
			return true
		default:
			return false
		}
	}
	return Callback(fn, "injects replicaSets")
}

// InjectHorizontalPodAutoscalers puts hpas into a *HorizontalPodAutoscalerList
func InjectHorizontalPodAutoscalers(hpas []autoscalingv1.HorizontalPodAutoscaler) gomock.Matcher {
	fn := func(x interface{}) bool {
		switch o := x.(type) {
		case *autoscalingv1.HorizontalPodAutoscalerList:
			for _, hpa := range hpas {
				o.Items = append(o.Items, hpa)
			}
			return true
		default:
			return false
		}
	}
	return Callback(fn, "injects horizontalpodautoscalers")
}

// UpdateNamespaceLabes sets the labels on a *Namespace
func UpdateNamespaceLabels(labels map[string]string) gomock.Matcher {
	fn := func(x interface{}) bool {
		switch o := x.(type) {
		case *corev1.Namespace:
			o.ObjectMeta.Labels = labels
			return true
		default:
			return false
		}
	}
	return Callback(fn, "update namespace labels")
}

// UpdateReplicaSetSpec sets the spec on a *ReplicaSet
func UpdateReplicaSetSpec(replicaSet *appsv1.ReplicaSet) gomock.Matcher {
	fn := func(x interface{}) bool {
		switch o := x.(type) {
		case *appsv1.ReplicaSet:
			o.Spec = replicaSet.Spec
			return true
		default:
			return false
		}
	}
	return Callback(fn, "update replicaSet spec")
}

// UpdateHPASpec sets the spec on a *HorizontalPodAutoscaler
func UpdateHPASpec(hpa *autoscalingv1.HorizontalPodAutoscaler) gomock.Matcher {
	fn := func(x interface{}) bool {
		switch o := x.(type) {
		case *autoscalingv1.HorizontalPodAutoscaler:
			o.Spec = hpa.Spec
			return true
		default:
			return false
		}
	}
	return Callback(fn, "update hpa spec")
}
