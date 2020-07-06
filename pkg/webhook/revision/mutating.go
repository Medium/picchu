package revision

import (
	"context"
	"fmt"
	"github.com/prometheus/common/log"
	picchu "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"gomodules.xyz/jsonpatch/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type revisionMutator struct {
	client  client.Client
	decoder *admission.Decoder
}

func (r *revisionMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	clog.Info("Got revision mutation request", "req", req)
	rev := &picchu.Revision{}
	if err := r.decoder.Decode(req, rev); err != nil {
		clog.Error(err, "Failed to decode revision")
		return admission.Denied("internal error")
	}
	var patches []jsonpatch.JsonPatchOperation
	for i := range rev.Spec.Targets {
		target := rev.Spec.Targets[i]
		// if there's only one port, it will be defaulted
		if len(target.Ports) == 1 && !target.Ports[0].Default {
			patches = append(patches, jsonpatch.JsonPatchOperation{
				Operation: "add",
				Path:      fmt.Sprintf("/spec/targets/%d/ports/0/default", i),
				Value:     true,
			})
			break
		}
		var httpIndex *int
		var defaultFound bool
		// if there's multiple ports and no default is specified, the 'http' port will be defaulted
		for j := range target.Ports {
			port := rev.Spec.Targets[i].Ports[j]
			if port.Name == "http" {
				found := j
				httpIndex = &found
			}
			defaultFound = defaultFound || port.Default
		}
		if !defaultFound && httpIndex != nil {
			patches = append(patches, jsonpatch.JsonPatchOperation{
				Operation: "add",
				Path:      fmt.Sprintf("/spec/targets/%d/ports/%d/default", i, *httpIndex),
				Value:     true,
			})
		}
	}
	if len(patches) > 0 {
		log.Info("Setting default port", "patches", patches)
		return admission.Patched("setting default port", patches...)
	}
	return admission.Allowed("ok")
}

func (r *revisionMutator) InjectClient(c client.Client) error {
	r.client = c
	return nil
}

func (r *revisionMutator) InjectDecoder(d *admission.Decoder) error {
	r.decoder = d
	return nil
}
