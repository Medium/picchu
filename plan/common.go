package plan

import (
	"context"
	"errors"
	"fmt"

	picchu "go.medium.engineering/picchu/api/v1alpha1"
	"go.medium.engineering/picchu/controllers/utils"

	es "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/go-logr/logr"
	kedav1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	wpav1 "github.com/practo/k8s-worker-pod-autoscaler/pkg/apis/workerpodautoscaler/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	slov1alpha1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	appsv1 "k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type NoChangeNeededError error

var ErrNoChangeNeeded = NoChangeNeededError(errors.New("no change needed"))

func LogSync(log logr.Logger, op controllerutil.OperationResult, err error, resource runtime.Object) {
	kind := utils.MustGetKind(resource).Kind
	switch obj := resource.(type) {
	case *autoscaling.HorizontalPodAutoscaler:
		if err != nil {
			log.Error(err, "Sync resource", "Result", "failure", "Kind", kind, "Audit", true, "Resource", obj.Spec, "Op", op)
			return
		}
	case *corev1.Service:
		if err != nil {
			log.Error(err, "Sync resource", "Result", "failure", "Kind", kind, "Audit", true, "Resource", obj.Spec, "Op", op)
			return
		}
	case *appsv1.ReplicaSet:
		if err != nil {
			log.Error(err, "Sync resource", "Result", "failure", "Kind", kind, "Audit", true, "Resource", obj.Spec, "Op", op)
			return
		}
	default:
		if err != nil {
			log.Error(err, "Sync resource", "Result", "failure", "Kind", kind, "Audit", true, "Resource", resource, "Op", op)
			return
		}
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
	isIgnored := func(meta metav1.ObjectMeta) bool {
		for label := range meta.Labels {
			if label == picchu.LabelIgnore {
				return true
			}
		}
		return false
	}
	switch orig := obj.(type) {
	case *corev1.Namespace:
		typed := orig.DeepCopy()
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   typed.Name,
				Labels: CopyStringMap(typed.Labels),
			},
		}
		op, err := controllerutil.CreateOrUpdate(ctx, cli, namespace, func() error {
			if isIgnored(namespace.ObjectMeta) {
				kind := utils.MustGetKind(namespace).Kind
				log.Info("Resource is ignored", "name", namespace.Name, "kind", kind)
				return nil
			}
			namespace.Labels = CopyStringMap(typed.Labels)
			namespace.Annotations = CopyStringMap(typed.Annotations)
			return nil
		})
		LogSync(log, op, err, namespace)
		if err != nil {
			return err
		}
	case *corev1.Service:
		typed := orig.DeepCopy()
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      typed.Name,
				Namespace: typed.Namespace,
			},
		}
		op, err := controllerutil.CreateOrUpdate(ctx, cli, service, func() error {
			if isIgnored(service.ObjectMeta) {
				kind := utils.MustGetKind(service).Kind
				log.Info("Resource is ignored", "namespace", service.Namespace, "name", service.Name, "kind", kind)
				return nil
			}
			service.Spec.Ports = typed.Spec.Ports
			service.Spec.Selector = typed.Spec.Selector
			service.Labels = CopyStringMap(typed.Labels)
			return nil
		})
		LogSync(log, op, err, service)
		if err != nil {
			return err
		}
	case *corev1.ServiceAccount:
		typed := orig.DeepCopy()
		serviceAccount := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      typed.Name,
				Namespace: typed.Namespace,
			},
		}
		op, err := controllerutil.CreateOrUpdate(ctx, cli, serviceAccount, func() error {
			if isIgnored(serviceAccount.ObjectMeta) {
				kind := utils.MustGetKind(serviceAccount).Kind
				log.Info("Resource is ignored", "namespace", serviceAccount.Namespace, "name", serviceAccount.Name, "kind", kind)
				return nil
			}
			serviceAccount.Labels = CopyStringMap(typed.Labels)
			serviceAccount.Annotations = CopyStringMap(typed.Annotations)
			return nil
		})
		LogSync(log, op, err, serviceAccount)
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
		op, err := controllerutil.CreateOrUpdate(ctx, cli, dr, func() error {
			if isIgnored(dr.ObjectMeta) {
				kind := utils.MustGetKind(dr).Kind
				log.Info("Resource is ignored", "namespace", dr.Namespace, "name", dr.Name, "kind", kind)
				return nil
			}
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
		op, err := controllerutil.CreateOrUpdate(ctx, cli, vs, func() error {
			if isIgnored(vs.ObjectMeta) {
				kind := utils.MustGetKind(vs).Kind
				log.Info("Resource is ignored", "namespace", vs.Namespace, "name", vs.Name, "kind", kind)
				return nil
			}
			vs.Spec = typed.Spec
			vs.Labels = CopyStringMap(typed.Labels)
			return nil
		})
		LogSync(log, op, err, vs)
		if err != nil {
			return err
		}
	case *istiov1alpha3.Sidecar:
		typed := orig.DeepCopy()
		sidecar := &istiov1alpha3.Sidecar{
			ObjectMeta: metav1.ObjectMeta{
				Name:      typed.Name,
				Namespace: typed.Namespace,
			},
		}
		op, err := controllerutil.CreateOrUpdate(ctx, cli, sidecar, func() error {
			if isIgnored(sidecar.ObjectMeta) {
				kind := utils.MustGetKind(sidecar).Kind
				log.Info("Resource is ignored", "namespace", sidecar.Namespace, "name", sidecar.Name, "kind", kind)
				return nil
			}
			sidecar.Spec = typed.Spec
			sidecar.Labels = CopyStringMap(typed.Labels)
			return nil
		})
		LogSync(log, op, err, sidecar)
		if err != nil {
			return err
		}
	case *istiov1alpha3.Gateway:
		typed := orig.DeepCopy()
		gateway := &istiov1alpha3.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      typed.Name,
				Namespace: typed.Namespace,
			},
		}
		op, err := controllerutil.CreateOrUpdate(ctx, cli, gateway, func() error {
			if isIgnored(gateway.ObjectMeta) {
				kind := utils.MustGetKind(gateway).Kind
				log.Info("Resource is ignored", "namespace", gateway.Namespace, "name", gateway.Name, "kind", kind)
				return nil
			}
			gateway.Spec = typed.Spec
			gateway.Labels = CopyStringMap(typed.Labels)
			return nil
		})
		LogSync(log, op, err, gateway)
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
		op, err := controllerutil.CreateOrUpdate(ctx, cli, pm, func() error {
			if isIgnored(pm.ObjectMeta) {
				kind := utils.MustGetKind(pm).Kind
				log.Info("Resource is ignored", "namespace", pm.Namespace, "name", pm.Name, "kind", kind)
				return nil
			}
			pm.Spec = typed.Spec
			pm.Labels = CopyStringMap(typed.Labels)
			return nil
		})
		LogSync(log, op, err, pm)
		if err != nil {
			return err
		}
	case *monitoringv1.ServiceMonitor:
		typed := orig.DeepCopy()
		sm := &monitoringv1.ServiceMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      typed.Name,
				Namespace: typed.Namespace,
			},
		}
		op, err := controllerutil.CreateOrUpdate(ctx, cli, sm, func() error {
			if isIgnored(sm.ObjectMeta) {
				kind := utils.MustGetKind(sm).Kind
				log.Info("Resource is ignored", "namespace", sm.Namespace, "name", sm.Name, "kind", kind)
				return nil
			}
			sm.Spec = typed.Spec
			sm.Labels = CopyStringMap(typed.Labels)
			return nil
		})
		LogSync(log, op, err, sm)
		if err != nil {
			return err
		}
	case *slov1alpha1.PrometheusServiceLevel:
		typed := orig.DeepCopy()
		sl := &slov1alpha1.PrometheusServiceLevel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      typed.Name,
				Namespace: typed.Namespace,
			},
		}
		op, err := controllerutil.CreateOrUpdate(ctx, cli, sl, func() error {
			if isIgnored(sl.ObjectMeta) {
				kind := utils.MustGetKind(sl).Kind
				log.Info("Resource is ignored", "namespace", sl.Namespace, "name", sl.Name, "kind", kind)
				return nil
			}
			sl.Spec = typed.Spec
			sl.Labels = CopyStringMap(typed.Labels)
			return nil
		})
		LogSync(log, op, err, sl)
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
		op, err := controllerutil.CreateOrUpdate(ctx, cli, secret, func() error {
			if isIgnored(secret.ObjectMeta) {
				kind := utils.MustGetKind(secret).Kind
				log.Info("Resource is ignored", "namespace", secret.Namespace, "name", secret.Name, "kind", kind)
				return nil
			}
			secret.Data = typed.Data
			secret.Labels = CopyStringMap(typed.Labels)
			return nil
		})
		LogSync(log, op, err, secret)
		if err != nil {
			return err
		}
		//add case for ExternalSecret
	case *es.ExternalSecret:
		typed := orig.DeepCopy()
		externalSecret := &es.ExternalSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      typed.Name,
				Namespace: typed.Namespace,
			},
		}
		op, err := controllerutil.CreateOrUpdate(ctx, cli, externalSecret, func() error {
			if isIgnored(externalSecret.ObjectMeta) {
				kind := utils.MustGetKind(externalSecret).Kind
				log.Info("Resource is ignored", "namespace", externalSecret.Namespace, "name", externalSecret.Name, "kind", kind)
				return nil
			}
			externalSecret.Spec = typed.Spec
			externalSecret.Labels = CopyStringMap(typed.Labels)
			return nil
		})
		LogSync(log, op, err, externalSecret)
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
		op, err := controllerutil.CreateOrUpdate(ctx, cli, configMap, func() error {
			if isIgnored(configMap.ObjectMeta) {
				kind := utils.MustGetKind(configMap).Kind
				log.Info("Resource is ignored", "namespace", configMap.Namespace, "name", configMap.Name, "kind", kind)
				return nil
			}
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
				Name:        typed.Name,
				Namespace:   typed.Namespace,
				Annotations: typed.Annotations,
			},
		}
		op, err := controllerutil.CreateOrUpdate(ctx, cli, rs, func() error {
			if isIgnored(rs.ObjectMeta) {
				kind := utils.MustGetKind(rs).Kind
				log.Info("Resource is ignored", "namespace", rs.Namespace, "name", rs.Name, "kind", kind)
				return nil
			}
			if typed.Annotations[picchu.AnnotationAutoscaler] != picchu.AutoscalerTypeWPA { // Allow WorkerPodAutoScaler to manipulate Replicas on its own
				// Only update replicas if we are changing to/from zero, which means the replicaset is being retired/deployed
				var zero int32 = 0
				if rs.Spec.Replicas == nil {
					rs.Spec.Replicas = &zero
				}
				if *typed.Spec.Replicas == 0 || *rs.Spec.Replicas == 0 {
					*rs.Spec.Replicas = *typed.Spec.Replicas
				}
			}
			// end replicas logic

			if len(rs.Spec.Template.Labels) == 0 {
				rs.Spec.Template = typed.Spec.Template
			}
			rs.Spec.Selector = typed.Spec.Selector
			rs.Labels = typed.Labels
			return nil
		})
		if errors.Is(err, ErrNoChangeNeeded) {
			op = "unchanged"
			err = nil
		}
		LogSync(log, op, err, rs)
		if err != nil {
			return err
		}
	case *autoscaling.HorizontalPodAutoscaler:
		typed := orig.DeepCopy()
		hpa := &autoscaling.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      typed.Name,
				Namespace: typed.Namespace,
				Labels:    typed.Labels,
			},
		}
		op, err := controllerutil.CreateOrUpdate(ctx, cli, hpa, func() error {
			if isIgnored(hpa.ObjectMeta) {
				kind := utils.MustGetKind(hpa).Kind
				log.Info("Resource is ignored", "namespace", hpa.Namespace, "name", hpa.Name, "kind", kind)
				return nil
			}
			hpa.Spec = typed.Spec
			return nil
		})
		LogSync(log, op, err, hpa)
		if err != nil {
			return err
		}
	case *wpav1.WorkerPodAutoScaler:
		typed := orig.DeepCopy()
		wpa := &wpav1.WorkerPodAutoScaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      typed.Name,
				Namespace: typed.Namespace,
				Labels:    typed.Labels,
			},
		}
		op, err := controllerutil.CreateOrUpdate(ctx, cli, wpa, func() error {
			if isIgnored(wpa.ObjectMeta) {
				kind := utils.MustGetKind(wpa).Kind
				log.Info("Resource is ignored", "namespace", wpa.Namespace, "name", wpa.Name, "kind", kind)
				return nil
			}
			wpa.Spec = typed.Spec
			return nil
		})
		LogSync(log, op, err, wpa)
		if err != nil {
			return err
		}

	case *kedav1.ScaledObject:
		typed := orig.DeepCopy()
		keda := &kedav1.ScaledObject{
			ObjectMeta: metav1.ObjectMeta{
				Name:      typed.Name,
				Namespace: typed.Namespace,
				Labels:    typed.Labels,
			},
		}
		op, err := controllerutil.CreateOrUpdate(ctx, cli, keda, func() error {
			if isIgnored(keda.ObjectMeta) {
				kind := utils.MustGetKind(keda).Kind
				log.Info("Resource is ignored", "namespace", keda.Namespace, "name", keda.Name, "kind", kind)
				return nil
			}
			keda.Spec = typed.Spec
			return nil
		})
		LogSync(log, op, err, keda)
		if err != nil {
			return err
		}
	case *kedav1.TriggerAuthentication:
		typed := orig.DeepCopy()
		keda := &kedav1.TriggerAuthentication{
			ObjectMeta: metav1.ObjectMeta{
				Name:      typed.Name,
				Namespace: typed.Namespace,
				Labels:    typed.Labels,
			},
		}
		op, err := controllerutil.CreateOrUpdate(ctx, cli, keda, func() error {
			if isIgnored(keda.ObjectMeta) {
				kind := utils.MustGetKind(keda).Kind
				log.Info("Resource is ignored", "namespace", keda.Namespace, "name", keda.Name, "kind", kind)
				return nil
			}
			keda.Spec = typed.Spec
			return nil
		})
		LogSync(log, op, err, keda)
		if err != nil {
			return err
		}
	case *policyv1.PodDisruptionBudget:
		typed := orig.DeepCopy()
		pdb := &policyv1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{
				Name:      typed.Name,
				Namespace: typed.Namespace,
			},
		}
		op, err := controllerutil.CreateOrUpdate(ctx, cli, pdb, func() error {
			if isIgnored(pdb.ObjectMeta) {
				kind := utils.MustGetKind(pdb).Kind
				log.Info("Resource is ignored", "name", pdb.Name, "kind", kind)
				return nil
			}
			pdb.Labels = CopyStringMap(typed.Labels)
			pdb.Annotations = CopyStringMap(typed.Annotations)
			pdb.Spec = typed.Spec
			return nil
		})
		LogSync(log, op, err, pdb)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported type")
	}
	return nil
}
