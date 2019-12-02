package mocks

import (
	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
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
			o.Items = append(o.Items, secrets...)
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
			o.Items = append(o.Items, configMaps...)
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
			o.Items = append(o.Items, replicaSets...)
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
			o.Items = append(o.Items, hpas...)
			return true
		default:
			return false
		}
	}
	return Callback(fn, "injects horizontalpodautoscalers")
}

// InjectPrometheusRules puts PrometheusRules into a *PrometheusRuleList
func InjectPrometheusRules(rules []monitoringv1.PrometheusRule) gomock.Matcher {
	fn := func(x interface{}) bool {
		switch o := x.(type) {
		case *monitoringv1.PrometheusRuleList:
			for i := range rules {
				o.Items = append(o.Items, &rules[i])
			}
			return true
		default:
			return false
		}
	}
	return Callback(fn, "injects PrometheusRules")
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
