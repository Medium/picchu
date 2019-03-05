package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ReleaseManager is the Schema for the releasemanagers API
// +k8s:openapi-gen=true
type ReleaseManager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ReleaseManagerSpec   `json:"spec,omitempty"`
	Status ReleaseManagerStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ReleaseManagerList contains a list of ReleaseManager
type ReleaseManagerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ReleaseManager `json:"items"`
}

// ReleaseManagerSpec defines the desired state of ReleaseManager
// +k8s:openapi-gen=true
type ReleaseManagerSpec struct {
	Cluster string `json:"cluster"`
	App     string `json:"app"`
	Target  string `json:"target"`
}

// ReleaseManagerStatus defines the observed state of ReleaseManager
// +k8s:openapi-gen=true
type ReleaseManagerStatus struct {
}

func init() {
	SchemeBuilder.Register(&ReleaseManager{}, &ReleaseManagerList{})
}
