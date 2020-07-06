package revision

import (
	"context"
	"fmt"
	picchu "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type revisionValidator struct {
	client  client.Client
	decoder *admission.Decoder
}

func (r *revisionValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	clog.Info("Got revision validation request", "req", req)
	rev := &picchu.Revision{}
	if err := r.decoder.Decode(req, rev); err != nil {
		clog.Error(err, "Failed to decode revision")
		return admission.Denied("internal error")
	}
	var badTargets []string
	for _, target := range rev.Spec.Targets {
		var hasPorts bool
		var defaultCount int
		for _, port := range target.Ports {
			hasPorts = true
			if port.Default {
				defaultCount++
			}
		}
		if hasPorts && defaultCount != 1 {
			badTargets = append(badTargets, target.Name)
		}
	}
	if len(badTargets) > 0 {
		return admission.Denied(fmt.Sprintf("Must specify exactly one port as default for targets %s", badTargets))
	}
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
