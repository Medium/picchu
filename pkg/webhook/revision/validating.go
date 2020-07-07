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
	invalidTargets := r.invalidTargets(rev)
	if len(invalidTargets) > 0 {
		msg := "Must specify exactly one port per ingress as default for targets %s, local ports shouldn't be defaulted"
		return admission.Denied(fmt.Sprintf(msg, invalidTargets))
	}
	return admission.Allowed("")
}

func (r *revisionValidator) invalidTargets(rev *picchu.Revision) []string {
	var badTargets []string
	for _, target := range rev.Spec.Targets {
		for mode, ports := range bucketIngressPorts(target) {
			var hasPorts bool
			var defaultCount int
			for _, port := range ports {
				hasPorts = true
				if port.Default {
					defaultCount++
				}
			}
			if mode != picchu.PortLocal && hasPorts && defaultCount != 1 {
				badTargets = append(badTargets, target.Name)
				break
			}
			if mode == picchu.PortLocal && defaultCount > 0 {
				badTargets = append(badTargets, target.Name)
				break
			}
		}
	}
	return badTargets
}

func (r *revisionValidator) InjectClient(c client.Client) error {
	r.client = c
	return nil
}

func (r *revisionValidator) InjectDecoder(d *admission.Decoder) error {
	r.decoder = d
	return nil
}
