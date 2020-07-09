package revision

import (
	"context"
	"fmt"
	picchu "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type failure struct {
	target string
	reason string
}

func (f *failure) string() string {
	return fmt.Sprintf("Failure to validate target %s because %s", f.target, f.reason)
}

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
	failures := r.failures(rev)
	for _, failure := range failures {
		clog.Error(nil, failure.string(), "revision", rev)
	}
	/*
		TODO(bob): turn on validation and format error messages
		if len(failures) > 0 {
			msg := "Must specify existing port for defaultIngressPort for all ingresses"
			return admission.Denied(fmt.Sprintf(msg, invalidTargets))
		}
	*/
	return admission.Allowed("")
}

func (r *revisionValidator) failures(rev *picchu.Revision) (failures []failure) {
	// exactly one port per target ingress should be set as default.
	for _, target := range rev.Spec.Targets {
		buckets := bucketIngressPorts(target)
		for ingress, ports := range buckets {
			if len(ports) == 0 {
				continue
			}
			if target.DefaultIngressPorts == nil {
				msg := fmt.Sprintf("Default ingress ports not specified for %s", ingress)
				failures = append(failures, failure{target.Name, msg})
			}
			found := false
			for _, port := range ports {
				if target.DefaultIngressPorts[ingress] == port.Name {
					found = true
					break
				}
			}
			if !found {
				msg := fmt.Sprintf("Specified default ingress port for %s doesn't exist", ingress)
				failures = append(failures, failure{target.Name, msg})
			}
		}
		for ingress := range target.DefaultIngressPorts {
			if _, ok := buckets[ingress]; !ok {
				msg := fmt.Sprintf("Specified default for ingress %s that doesn't exist", ingress)
				failures = append(failures, failure{target.Name, msg})
			}
		}
	}
	return
}

func (r *revisionValidator) InjectClient(c client.Client) error {
	r.client = c
	return nil
}

func (r *revisionValidator) InjectDecoder(d *admission.Decoder) error {
	r.decoder = d
	return nil
}
