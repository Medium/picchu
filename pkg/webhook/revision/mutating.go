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
	patches := r.getPatches(rev)
	if len(patches) > 0 {
		log.Info("Setting default port", "patches", patches)
		return admission.Patched("setting default port", patches...)
	}
	return admission.Allowed("ok")
}

func (r *revisionMutator) getPatches(rev *picchu.Revision) []jsonpatch.JsonPatchOperation {
	var patches []jsonpatch.JsonPatchOperation
	for i := range rev.Spec.Targets {
		buckets := bucketIngressPorts(rev.Spec.Targets[i])
		for mode, ports := range buckets {
			if mode == picchu.PortLocal {
				continue
			}

			if len(ports) == 1 && !ports[0].Default {
				patches = append(patches, jsonpatch.JsonPatchOperation{
					Operation: "add",
					Path:      fmt.Sprintf("/spec/targets/%d/ports/%d/default", i, ports[0].index),
					Value:     true,
				})
			}

			var httpIndex *int
			var defaultFound bool
			for j := range ports {
				port := ports[j]
				defaultFound = defaultFound || port.Default
				if port.Name == "http" {
					httpIndex = &port.index
				}
			}
			if !defaultFound && httpIndex != nil {
				patches = append(patches, jsonpatch.JsonPatchOperation{
					Operation: "add",
					Path:      fmt.Sprintf("/spec/targets/%d/ports/%d/default", i, *httpIndex),
					Value:     true,
				})
			}
		}
	}
	return patches
}

func (r *revisionMutator) InjectClient(c client.Client) error {
	r.client = c
	return nil
}

func (r *revisionMutator) InjectDecoder(d *admission.Decoder) error {
	r.decoder = d
	return nil
}
