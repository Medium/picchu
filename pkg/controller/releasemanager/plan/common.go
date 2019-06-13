package plan

import (
	"go.medium.engineering/picchu/pkg/controller/utils"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func LogSync(log logr.Logger, op controllerutil.OperationResult, err error, resource runtime.Object) {
	kind := utils.MustGetKind(resource).Kind
	if err != nil {
		log.Error(err, "Sync resource", "Result", "failure", "Kind", kind, "Audit", true, "Resource", resource, "Op", op)
		return
	}
	log.Info("Sync resource", "Result", "success", "Kind", kind, "Audit", true, "Resource", resource, "Op", op)
}
