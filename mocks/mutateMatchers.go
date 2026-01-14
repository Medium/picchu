package mocks

import (
	es "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/golang/mock/gomock"
	kedav1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	wpav1 "github.com/practo/k8s-worker-pod-autoscaler/pkg/apis/workerpodautoscaler/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	slov1alpha1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v2"
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

// InjectExternalSecrets puts ExternalSecrets into an *ExternalSecretList
func InjectExternalSecrets(externalSecrets []es.ExternalSecret) gomock.Matcher {
	fn := func(x interface{}) bool {
		switch o := x.(type) {
		case *es.ExternalSecretList:
			o.Items = append(o.Items, externalSecrets...)
			return true
		default:
			return false
		}
	}
	return Callback(fn, "injects ExternalSecrets")
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

func InjectKedaPodAutoscalers(kedas []kedav1.ScaledObject) gomock.Matcher {
	fn := func(x interface{}) bool {
		switch o := x.(type) {
		case *kedav1.ScaledObjectList:
			o.Items = append(o.Items, kedas...)
			return true
		default:
			return false
		}
	}
	return Callback(fn, "injects kedapodautoscalers")
}

func InjectKedaAuths(kedas []kedav1.TriggerAuthentication) gomock.Matcher {
	fn := func(x interface{}) bool {
		switch o := x.(type) {
		case *kedav1.TriggerAuthenticationList:
			o.Items = append(o.Items, kedas...)
			return true
		default:
			return false
		}
	}
	return Callback(fn, "injects kedaauths")
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
				o.Items = append(o.Items, rules[i])
			}
			return true
		default:
			return false
		}
	}
	return Callback(fn, "injects PrometheusRules")
}

// InjectServiceLevels puts ServiceLevel into a *ServiceLevelList
func InjectServiceLevels(sls []slov1alpha1.PrometheusServiceLevel) gomock.Matcher {
	fn := func(x interface{}) bool {
		switch o := x.(type) {
		case *slov1alpha1.PrometheusServiceLevelList:
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
				o.Items = append(o.Items, sms[i])
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

func UpdateKEDASpec(keda *kedav1.ScaledObject) gomock.Matcher {
	fn := func(x interface{}) bool {
		switch o := x.(type) {
		case *kedav1.ScaledObject:
			o.Spec = keda.Spec
			return true
		case *kedav1.TriggerAuthentication:
			return true
		default:
			return false
		}
	}
	return Callback(fn, "update keda spec")
}

func UpdateKEDATriggerAuthSpec(keda *kedav1.TriggerAuthentication) gomock.Matcher {
	fn := func(x interface{}) bool {
		switch o := x.(type) {
		case *kedav1.TriggerAuthentication:
			o.Spec = keda.Spec
			return true
		default:
			return false
		}
	}
	return Callback(fn, "update keda trigger auth spec")
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

// UpdateKEDAObjectMeta sets the ObjectMeta on a *Keda Scaled Object
func UpdateKEDAObjectMeta(keda *kedav1.ScaledObject) gomock.Matcher {
	fn := func(x interface{}) bool {
		switch o := x.(type) {
		case *kedav1.ScaledObject:
			o.ObjectMeta = keda.ObjectMeta
			return true
		default:
			return false
		}
	}
	return Callback(fn, "update keda object meta")
}

func UpdateKEDATriggerAuthObjectMeta(keda *kedav1.TriggerAuthentication) gomock.Matcher {
	fn := func(x interface{}) bool {
		switch o := x.(type) {
		case *kedav1.TriggerAuthentication:
			o.ObjectMeta = keda.ObjectMeta
			return true
		default:
			return false
		}
	}
	return Callback(fn, "update keda trigger auth object meta")
}

// UpdateWPAObjectMeta sets the ObjectMeta on a *WorkerPodAutoscaler
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
