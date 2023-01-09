package mocks

import (
	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/golang/mock/gomock"
	wpav1 "github.com/practo/k8s-worker-pod-autoscaler/pkg/apis/workerpodautoscaler/v1"
	slov1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v2beta2"
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
func InjectHorizontalPodAutoscalers(hpas []autoscaling.HorizontalPodAutoscaler) gomock.Matcher {
	fn := func(x interface{}) bool {
		switch o := x.(type) {
		case *autoscaling.HorizontalPodAutoscalerList:
			o.Items = append(o.Items, hpas...)
			return true
		default:
			return false
		}
	}
	return Callback(fn, "injects horizontalpodautoscalers")
}

// InjectHorizontalPodAutoscalers puts hpas into a *HorizontalPodAutoscalerList
func InjectWorkerPodAutoscalers(wpas []wpav1.WorkerPodAutoScaler) gomock.Matcher {
	fn := func(x interface{}) bool {
		switch o := x.(type) {
		case *wpav1.WorkerPodAutoScalerList:
			o.Items = append(o.Items, wpas...)
			return true
		default:
			return false
		}
	}
	return Callback(fn, "injects workerpodautoscalers")
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

// InjectServiceLevels puts ServiceLevel into a *ServiceLevelList
func InjectServiceLevels(sls []slov1.PrometheusServiceLevel) gomock.Matcher {
	fn := func(x interface{}) bool {
		switch o := x.(type) {
		case *slov1.PrometheusServiceLevelList:
			for i := range sls {
				o.Items = append(o.Items, sls[i])
			}
			return true
		default:
			return false
		}
	}
	return Callback(fn, "injects ServiceLevels")
}

// InjectServiceMonitors puts ServiceMonitor into a *ServiceMonitorList
func InjectServiceMonitors(sms []monitoringv1.ServiceMonitor) gomock.Matcher {
	fn := func(x interface{}) bool {
		switch o := x.(type) {
		case *monitoringv1.ServiceMonitorList:
			for i := range sms {
				o.Items = append(o.Items, &sms[i])
			}
			return true
		default:
			return false
		}
	}
	return Callback(fn, "injects ServiceMonitors")
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
func UpdateHPASpec(hpa *autoscaling.HorizontalPodAutoscaler) gomock.Matcher {
	fn := func(x interface{}) bool {
		switch o := x.(type) {
		case *autoscaling.HorizontalPodAutoscaler:
			o.Spec = hpa.Spec
			return true
		default:
			return false
		}
	}
	return Callback(fn, "update hpa spec")
}

// UpdateHPASpec sets the spec on a *HorizontalPodAutoscaler
func UpdateWPASpec(wpa *wpav1.WorkerPodAutoScaler) gomock.Matcher {
	fn := func(x interface{}) bool {
		switch o := x.(type) {
		case *wpav1.WorkerPodAutoScaler:
			o.Spec = wpa.Spec
			return true
		default:
			return false
		}
	}
	return Callback(fn, "update wpa spec")
}

// UpdateWPAObjectMeta sets the ObjectMeta on a *HorizontalPodAutoscaler
func UpdateWPAObjectMeta(wpa *wpav1.WorkerPodAutoScaler) gomock.Matcher {
	fn := func(x interface{}) bool {
		switch o := x.(type) {
		case *wpav1.WorkerPodAutoScaler:
			o.ObjectMeta = wpa.ObjectMeta
			return true
		default:
			return false
		}
	}
	return Callback(fn, "update wpa object meta")
}
