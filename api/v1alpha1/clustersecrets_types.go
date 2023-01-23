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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClusterSecretsSpec defines the desired state of ClusterSecrets
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:resource:categories=picchu
type ClusterSecretsSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of ClusterSecrets. Edit ClusterSecrets_types.go to remove/update
	Source ClusterSecretSource `json:"source"`
	Target ClusterSecretTarget `json:"target"`
}

type ClusterSecretSource struct {
	Namespace     string `json:"namespace"`
	LabelSelector string `json:"labelSelector,omitempty"`
	FieldSelector string `json:"fieldSelector,omitempty"`
}

type ClusterSecretTarget struct {
	// Namespace to copy secrets to
	Namespace string `json:"namespace"`

	// LabelSelector of clusters to copy secrets to
	LabelSelector string `json:"labelSelector,omitempty"`

	// FieldSelector of clusters to copy secrets to
	FieldSelector string `json:"fieldSelector,omitempty"`

	// Labels to add to the copied secrets
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations to add to the copied secrets
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ClusterSecretsStatus defines the observed state of ClusterSecrets
type ClusterSecretsStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Secrets []string `json:"secrets,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ClusterSecrets is the Schema for the clustersecrets API
type ClusterSecrets struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSecretsSpec   `json:"spec,omitempty"`
	Status ClusterSecretsStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterSecretsList contains a list of ClusterSecrets
type ClusterSecretsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterSecrets `json:"items"`
}

func (c *ClusterSecrets) IsDeleted() bool {
	return !c.ObjectMeta.DeletionTimestamp.IsZero()
}

func (c *ClusterSecrets) IsFinalized() bool {
	for _, item := range c.ObjectMeta.Finalizers {
		if item == FinalizerClusterSecrets {
			return false
		}
	}
	return true
}

func (c *ClusterSecrets) Finalize() {
	finalizers := []string{}
	for _, item := range c.ObjectMeta.Finalizers {
		if item == FinalizerClusterSecrets {
			continue
		}
		finalizers = append(finalizers, item)
	}
	c.ObjectMeta.Finalizers = finalizers
}

func (c *ClusterSecrets) AddFinalizer() {
	c.ObjectMeta.Finalizers = append(c.ObjectMeta.Finalizers, FinalizerClusterSecrets)
}

func init() {
	SchemeBuilder.Register(&ClusterSecrets{}, &ClusterSecretsList{})
}
