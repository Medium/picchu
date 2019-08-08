package plan

import (
	"context"
	"fmt"

	"go.medium.engineering/picchu/pkg/controller/utils"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/go-logr/logr"
	istiov1alpha3 "github.com/knative/pkg/apis/istio/v1alpha3"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func LogSync(log logr.Logger, op controllerutil.OperationResult, err error, resource runtime.Object) {
	kind := utils.MustGetKind(resource).Kind
	switch obj := resource.(type) {
	case *autoscalingv1.HorizontalPodAutoscaler:
		if err != nil {
			log.Error(err, "Sync resource", "Result", "failure", "Kind", kind, "Audit", true, "Resource", obj.Spec, "Op", op)
			return
		}
		log.Info("Sync resource", "Result", "success", "Kind", kind, "Audit", true, "Resource", obj.Spec, "Op", op)
	case *corev1.Service:
		if err != nil {
			log.Error(err, "Sync resource", "Result", "failure", "Kind", kind, "Audit", true, "Resource", obj.Spec, "Op", op)
			return
		}
		log.Info("Sync resource", "Result", "success", "Kind", kind, "Audit", true, "Resource", obj.Spec, "Op", op)
	case *appsv1.ReplicaSet:
		if err != nil {
			log.Error(err, "Sync resource", "Result", "failure", "Kind", kind, "Audit", true, "Resource", obj.Spec, "Op", op)
			return
		}
		log.Info("Sync resource", "Result", "success", "Kind", kind, "Audit", true, "Resource", obj.Spec, "Op", op)
	default:
		if err != nil {
			log.Error(err, "Sync resource", "Result", "failure", "Kind", kind, "Audit", true, "Resource", resource, "Op", op)
			return
		}
		log.Info("Sync resource", "Result", "success", "Kind", kind, "Audit", true, "Resource", resource, "Op", op)
	}
}

func CopyStringMap(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}
	dst := map[string]string{}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// CreateOrUpdate safely performs a standard CreateOrUpdate for known types.
// my kingdom for go generics
func CreateOrUpdate(
	ctx context.Context,
	log logr.Logger,
	cli client.Client,
	obj runtime.Object,
) error {
	switch orig := obj.(type) {
	case *corev1.Service:
		typed := orig.DeepCopy()
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      typed.Name,
				Namespace: typed.Namespace,
			},
		}
		op, err := controllerutil.CreateOrUpdate(ctx, cli, service, func(runtime.Object) error {
			service.Spec.Ports = typed.Spec.Ports
			service.Spec.Selector = typed.Spec.Selector
			service.Labels = CopyStringMap(typed.Labels)
			return nil
		})
		LogSync(log, op, err, service)
		if err != nil {
			return err
		}
	case *istiov1alpha3.DestinationRule:
		typed := orig.DeepCopy()
		dr := &istiov1alpha3.DestinationRule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      typed.Name,
				Namespace: typed.Namespace,
			},
		}
		op, err := controllerutil.CreateOrUpdate(ctx, cli, dr, func(runtime.Object) error {
			dr.Spec = typed.Spec
			dr.Labels = CopyStringMap(typed.Labels)
			return nil
		})
		LogSync(log, op, err, dr)
		if err != nil {
			return err
		}
	case *istiov1alpha3.VirtualService:
		typed := orig.DeepCopy()
		vs := &istiov1alpha3.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      typed.Name,
				Namespace: typed.Namespace,
			},
		}
		op, err := controllerutil.CreateOrUpdate(ctx, cli, vs, func(runtime.Object) error {
			vs.Spec = typed.Spec
			vs.Labels = CopyStringMap(typed.Labels)
			return nil
		})
		LogSync(log, op, err, vs)
		if err != nil {
			return err
		}
	case *monitoringv1.PrometheusRule:
		typed := orig.DeepCopy()
		pm := &monitoringv1.PrometheusRule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      typed.Name,
				Namespace: typed.Namespace,
			},
		}
		op, err := controllerutil.CreateOrUpdate(ctx, cli, pm, func(runtime.Object) error {
			pm.Spec = typed.Spec
			pm.Labels = CopyStringMap(typed.Labels)
			return nil
		})
		LogSync(log, op, err, pm)
		if err != nil {
			return err
		}
	case *corev1.Secret:
		typed := orig.DeepCopy()
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      typed.Name,
				Namespace: typed.Namespace,
			},
		}
		op, err := controllerutil.CreateOrUpdate(ctx, cli, secret, func(runtime.Object) error {
			secret.Data = typed.Data
			secret.Labels = CopyStringMap(typed.Labels)
			return nil
		})
		LogSync(log, op, err, secret)
		if err != nil {
			return err
		}
	case *corev1.ConfigMap:
		typed := orig.DeepCopy()
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      typed.Name,
				Namespace: typed.Namespace,
			},
		}
		op, err := controllerutil.CreateOrUpdate(ctx, cli, configMap, func(runtime.Object) error {
			configMap.Data = typed.Data
			configMap.Labels = CopyStringMap(typed.Labels)
			return nil
		})
		LogSync(log, op, err, configMap)
		if err != nil {
			return err
		}
	case *appsv1.ReplicaSet:
		typed := orig.DeepCopy()
		rs := &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      typed.Name,
				Namespace: typed.Namespace,
			},
		}
		op, err := controllerutil.CreateOrUpdate(ctx, cli, rs, func(runtime.Object) error {
			var zero int32 = 0
			if rs.Spec.Replicas == nil {
				rs.Spec.Replicas = &zero
			}
			if 0 == *typed.Spec.Replicas || 0 == *rs.Spec.Replicas {
				*rs.Spec.Replicas = *typed.Spec.Replicas
			}
			rs.Spec.Template = typed.Spec.Template
			rs.Spec.Selector = typed.Spec.Selector
			rs.Labels = CopyStringMap(typed.Labels)
			return nil
		})
		LogSync(log, op, err, rs)
		if err != nil {
			return err
		}
	case *autoscalingv1.HorizontalPodAutoscaler:
		typed := orig.DeepCopy()
		hpa := &autoscalingv1.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      typed.Name,
				Namespace: typed.Namespace,
			},
		}
		op, err := controllerutil.CreateOrUpdate(ctx, cli, hpa, func(runtime.Object) error {
			hpa.Spec = typed.Spec
			return nil
		})
		LogSync(log, op, err, hpa)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("Unsupported type")
	}
	return nil
}
