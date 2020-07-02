package revision

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type revisionMutator struct {
	client  client.Client
	decoder *admission.Decoder
}

func (r *revisionMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	clog.Info("Got revision mutation request", "req", req)
	return admission.Allowed("")
}

func (r *revisionMutator) InjectClient(c client.Client) error {
	r.client = c
	return nil
}

func (r *revisionMutator) InjectDecoder(d *admission.Decoder) error {
	r.decoder = d
	return nil
}
