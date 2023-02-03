/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var revisionlog = logf.Log.WithName("revision-resource")

func (r *Revision) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-picchu-medium-engineering-picchu-medium-engineering-v1alpha1-revision,mutating=true,failurePolicy=fail,groups=medium.engineering.medium.engineering,resources=revisions,verbs=create;update,versions=v1alpha1,name=mrevision.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Revision{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Revision) Default() {
	revisionlog.Info("default", "name", r.Name)
	r.getPatches()
	// TODO(user): fill in your defaulting logic.
}

func (r *Revision) getPatches() error {
	if err := r.getIngressesPatches(); err != nil {
		return err
	}
	if err := r.getIngressDefaultPortPatches(); err != nil {
		return err
	}
	return nil
}

func (r *Revision) getIngressesPatches() error {

	for i := range r.Spec.Targets {
		for j := range r.Spec.Targets[i].Ports {
			port := r.Spec.Targets[i].Ports[j]
			if len(port.Ingresses) > 0 {
				continue
			}
			if port.Mode == PortPrivate || port.Mode == PortPublic {
				r.Spec.Targets[i].Ports[j].Ingresses = append(r.Spec.Targets[i].Ports[j].Ingresses, "private")

			}
			if port.Mode == PortPublic {
				r.Spec.Targets[i].Ports[j].Ingresses = append(r.Spec.Targets[i].Ports[j].Ingresses, "public")

			}

		}

	}
	return nil

}

func (r *Revision) getIngressDefaultPortPatches() error {
	// if a single public or private ingress port is defined in a target and it's not set to default, it will be set as
	// the default.
	// if multiple ingress ports are defined and none are set to default and there is a port called 'http', it will be
	// set to default
	// internal ports will never be set as default
	for i := range r.Spec.Targets {
		ingressDefaultPorts := map[string]string{}
		for ingress, ports := range bucketIngressPorts(r.Spec.Targets[i]) {
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
		existing := r.Spec.Targets[i].DefaultIngressPorts
		if (existing == nil || len(existing) <= 0) && len(ingressDefaultPorts) > 0 {
			r.Spec.Targets[i].DefaultIngressPorts = ingressDefaultPorts
			continue
		}
		for ingress, def := range ingressDefaultPorts {
			if _, ok := existing[ingress]; !ok {
				r.Spec.Targets[i].DefaultIngressPorts[ingress] = def
			}
		}
	}
	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-picchu-medium-engineering-picchu-medium-engineering-v1alpha1-revision,mutating=false,failurePolicy=fail,groups=medium.engineering.medium.engineering,resources=revisions,versions=v1alpha1,name=vrevision.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Revision{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Revision) ValidateCreate() error {
	revisionlog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return r.validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Revision) ValidateUpdate(old runtime.Object) error {
	revisionlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return r.validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Revision) ValidateDelete() error {
	revisionlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

func (r *Revision) validate() error {
	var allErrors field.ErrorList
	for _, target := range r.Spec.Targets {
		found := target.Release.ScalingStrategy == ""
		for _, s := range ScalingStrategies {
			if target.Release.ScalingStrategy == s {
				found = true
			}
		}
		if !found {
			err := field.Invalid(field.NewPath("spec"), target.Release.ScalingStrategy, "Invalid scaling strategy")
			allErrors = append(allErrors, err)
		}
		buckets := bucketIngressPorts(target)
		for ingress, ports := range buckets {
			if target.DefaultIngressPorts == nil {
				err := field.Required(field.NewPath("spec"), fmt.Sprintf("Default ingress ports not specified for %s", ingress))
				allErrors = append(allErrors, err)
			}
			found := false
			for _, port := range ports {
				if target.DefaultIngressPorts[ingress] == port.Name {
					found = true
					break
				}
			}
			if !found {
				err := field.NotFound(field.NewPath("spec"), fmt.Sprintf("Specified default for ingress %s that doesn't exist", ingress))
				allErrors = append(allErrors, err)
			}
		}
		for ingress := range target.DefaultIngressPorts {
			if _, ok := buckets[ingress]; !ok {
				err := field.NotFound(field.NewPath("spec"), fmt.Sprintf("Specified default for ingress %s that doesn't exist", ingress))
				allErrors = append(allErrors, err)
			}
		}
	}
	return apierrors.NewInvalid(schema.GroupKind{Group: GroupVersion.Group, Kind: r.Kind}, r.Name, allErrors)
}

type portInfo struct {
	PortInfo
	index int
}

// returns a map of ingresses and ports mapped to ingresses
func bucketIngressPorts(target RevisionTarget) map[string][]portInfo {
	// bucket ports by mode
	track := map[string][]portInfo{}
	for i := range target.Ports {
		pi := portInfo{
			PortInfo: target.Ports[i],
			index:    i,
		}
		if pi.Ingresses != nil {
			for _, ingress := range pi.Ingresses {
				track[ingress] = append(track[ingress], pi)
			}
			continue
		}

		switch pi.Mode {
		case PortPublic:
			track["public"] = append(track["public"], pi)
			track["private"] = append(track["private"], pi)
		case PortPrivate:
			track["private"] = append(track["private"], pi)
		}
	}
	return track
}
