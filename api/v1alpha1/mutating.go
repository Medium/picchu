package v1alpha1

import (
	"context"
	"fmt"

	picchu "go.medium.engineering/picchu/api/v1alpha1"
	"gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type revisionMutator struct {
	client  client.Client
	decoder *admission.Decoder
}

func (r *revisionMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	resp := &admission.Response{
		AdmissionResponse: admissionv1.AdmissionResponse{
			UID:     req.UID,
			Allowed: false,
		},
	}
	clog.Info("Got revision mutation request", "req", req)
	rev := &picchu.Revision{}
	if err := r.decoder.Decode(req, rev); err != nil {
		clog.Error(err, "Failed to decode revision")
		resp.Result = &metav1.Status{Message: "internal error"}
		return *resp
	}
	patches := r.getPatches(rev)
	if len(patches) > 0 {
		clog.Info("Patching revision", "patches", patches)
		resp.Result = &metav1.Status{Message: "patching revision"}
		resp.Allowed = true
		resp.Patches = patches
		return *resp
	}
	clog.Info("No patches needed")
	resp.Result = &metav1.Status{Message: "ok"}
	resp.Allowed = true
	return *resp
}

func (r *revisionMutator) getPatches(rev *picchu.Revision) []jsonpatch.JsonPatchOperation {
	return append(r.getIngressesPatches(rev), r.getIngressDefaultPortPatches(rev)...)
}

func (r *revisionMutator) getIngressesPatches(rev *picchu.Revision) []jsonpatch.JsonPatchOperation {
	var patches []jsonpatch.JsonPatchOperation
	for i := range rev.Spec.Targets {
		for j := range rev.Spec.Targets[i].Ports {
			port := rev.Spec.Targets[i].Ports[j]
			var ingresses []string
			if len(port.Ingresses) > 0 {
				continue
			}
			if port.Mode == picchu.PortPrivate || port.Mode == picchu.PortPublic {
				ingresses = append(ingresses, "private")
			}
			if port.Mode == picchu.PortPublic {
				ingresses = append(ingresses, "public")
			}
			if len(ingresses) > 0 {
				patches = append(patches, jsonpatch.JsonPatchOperation{
					Operation: "add",
					Path:      fmt.Sprintf("/spec/targets/%d/ports/%d/ingresses", i, j),
					Value:     ingresses,
				})
			}
		}

	}
	return patches
}

func (r *revisionMutator) getIngressDefaultPortPatches(rev *picchu.Revision) []jsonpatch.JsonPatchOperation {
	// if a single public or private ingress port is defined in a target and it's not set to default, it will be set as
	// the default.
	// if multiple ingress ports are defined and none are set to default and there is a port called 'http', it will be
	// set to default
	// internal ports will never be set as default
	var patches []jsonpatch.JsonPatchOperation
	for i := range rev.Spec.Targets {
		ingressDefaultPorts := map[string]string{}
		for ingress, ports := range bucketIngressPorts(rev.Spec.Targets[i]) {
			if len(ports) == 1 {
				ingressDefaultPorts[ingress] = ports[0].Name
			}

			var httpFound bool
			var defaultFound bool
			for j := range ports {
				port := ports[j]
				defaultFound = defaultFound || port.Default
				if port.Name == "http" {
					httpFound = true
				}
			}
			if !defaultFound && httpFound {
				ingressDefaultPorts[ingress] = "http"
			}
		}
		existing := rev.Spec.Targets[i].DefaultIngressPorts
		if (existing == nil || len(existing) <= 0) && len(ingressDefaultPorts) > 0 {
			patches = append(patches, jsonpatch.JsonPatchOperation{
				Operation: "add",
				Path:      fmt.Sprintf("/spec/targets/%d/defaultIngressPorts", i),
				Value:     ingressDefaultPorts,
			})
			continue
		}
		for ingress, def := range ingressDefaultPorts {
			if _, ok := existing[ingress]; !ok {
				patches = append(patches, jsonpatch.JsonPatchOperation{
					Operation: "add",
					Path:      fmt.Sprintf("/spec/targets/%d/defaultIngressPorts[%s]", i, ingress),
					Value:     def,
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
