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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var revisionlog = logf.Log.WithName("revision-resource")

func (r *Revision) SetupWebhookWithManager(mgr ctrl.Manager) error {
	mgr.GetWebhookServer().Register("/validate-revisions", &webhook.Admission{Handler: &revisionValidator{}})
	mgr.GetWebhookServer().Register("/mutate-revisions", &webhook.Admission{Handler: &revisionMutator{}})
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-picchu-medium-engineering-picchu-medium-engineering-v1alpha1-revision,mutating=true,failurePolicy=fail,groups=picchu.medium.engineering.picchu.medium.engineering,resources=revisions,verbs=create;update,versions=v1alpha1,name=mrevision.kb.io

var _ webhook.Defaulter = &Revision{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Revision) Default() {
	revisionlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-picchu-medium-engineering-picchu-medium-engineering-v1alpha1-revision,mutating=false,failurePolicy=fail,groups=picchu.medium.engineering.picchu.medium.engineering,resources=revisions,versions=v1alpha1,name=vrevision.kb.io

var _ webhook.Validator = &Revision{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Revision) ValidateCreate() error {
	revisionlog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Revision) ValidateUpdate(old runtime.Object) error {
	revisionlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Revision) ValidateDelete() error {
	revisionlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
