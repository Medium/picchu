package plan

import (
	"go.medium.engineering/picchu/pkg/controller/utils"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
