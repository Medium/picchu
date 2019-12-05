package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterSecretsSpec defines the desired state of ClusterSecrets
// +k8s:openapi-gen=true
type ClusterSecretsSpec struct {
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
// +k8s:openapi-gen=true
type ClusterSecretsStatus struct {
	// Names of secrets copied to targets
	// +listType=set
	Secrets []string `json:"secrets,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterSecrets is the Schema for the clustersecrets API
// +k8s:openapi-gen=true
type ClusterSecrets struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSecretsSpec   `json:"spec,omitempty"`
	Status ClusterSecretsStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

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
