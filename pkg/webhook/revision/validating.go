package revision

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type revisionValidator struct {
	client  client.Client
	decoder *admission.Decoder
}

func (r *revisionValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	clog.Info("Got revision validation request", "req", req)
	return admission.Allowed("")
}

func (r *revisionValidator) InjectClient(c client.Client) error {
	r.client = c
	return nil
}

func (r *revisionValidator) InjectDecoder(d *admission.Decoder) error {
	r.decoder = d
	return nil
}
