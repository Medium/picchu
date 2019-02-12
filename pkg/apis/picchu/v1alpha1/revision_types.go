package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RevisionSpec defines the desired state of Revision
// +k8s:openapi-gen=true
type RevisionSpec struct {
	App             AppSpec             `json:"app"`
	ReleaseEligible ReleaseEligibleSpec `json:"releaseEligible"`
	Ports           []PortSpec          `json:"ports"`
	Targets         []TargetSpec        `json:"targets"`
}

// +k8s:openapi-gen=true
type AppSpec struct {
	Name string `json:"name"`
	Ref  string `json:"ref"`
	Tag  string `json:"tag"`
}

// +k8s:openapi-gen=true
type ReleaseEligibleSpec struct {
	Eligible bool `json:"eligible"`
}

// +k8s:openapi-gen=true
type PortSpec struct {
	Name          string `json:"name"`
	ContainerPort string `json:"containerPort"`
}

// +k8s:openapi-gen=true
type TargetSpec struct {
	Name    string        `json:"name"`
	Fleet   string        `json:"fleet"`
	Scale   ScaleSpec     `json:"scale"`
	Release *ReleaseSpec  `json:"releaseSpec"`
	Metrics *[]MetricSpec `json:"metrics"`
}

// +k8s:openapi-gen=true
type ScaleSpec struct {
	Min       int           `json:"min"`
	Max       int           `json:"max"`
	Resources ResourcesSpec `json:"resources"`
}

// +k8s:openapi-gen=true
type ResourcesSpec struct {
	CPU string `json:"cpu"`
}

// +k8s:openapi-gen=true
type ReleaseSpec struct {
	Max      string `json:"max"`
	Rate     string `json:"rate"`
	Schedule string `json:"schedule"`
}

// +k8s:openapi-gen=true
type MetricSpec struct {
	Name      string      `json:"name"`
	Queries   QueriesSpec `json:"queries"`
	Objective float64     `json:"objective"`
}

// +k8s:openapi-gen=true
type QueriesSpec struct {
	Acceptable string `json:"acceptable"`
	Total      string `json:"total"`
}

// RevisionStatus defines the observed state of Revision
// +k8s:openapi-gen=true
type RevisionStatus struct {
	Targets []TargetStatus `json:"targets"`
}

// +k8s:openapi-gen=true
type TargetStatus struct {
	Name        string             `json:"name"`
	Deployments []DeploymentStatus `json:"deployments"`
}

// +k8s:openapi-gen=true
type DeploymentStatus struct {
	Name    string `json:"name"`
	Cluster string `json:"cluster"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Revision is the Schema for the revisions API
// +k8s:openapi-gen=true
type Revision struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RevisionSpec   `json:"spec,omitempty"`
	Status RevisionStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RevisionList contains a list of Revision
type RevisionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Revision `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Revision{}, &RevisionList{})
}
