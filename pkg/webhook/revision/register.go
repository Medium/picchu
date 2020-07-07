package revision

import (
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	clog = logf.Log.WithName("webhook_revision")
)

func Register(mgr manager.Manager) {
	clog.Info("Registering revision webhook")
	mgr.GetWebhookServer().Register("/validate-revisions", &webhook.Admission{Handler: &revisionValidator{}})
	mgr.GetWebhookServer().Register("/mutate-revisions", &webhook.Admission{Handler: &revisionMutator{}})
}
