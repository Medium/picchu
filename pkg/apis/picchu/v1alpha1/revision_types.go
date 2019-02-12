package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RevisionSpec defines the desired state of Revision
type RevisionSpec struct {
	App             RSpecApp             `json:"app"`
	ReleaseEligible RSpecReleaseEligible `json:"releaseEligible"`
	Ports           []RSpecPort          `json:"ports"`
	Targets         []RSpecTarget        `json:"targets"`
}

type RSpecApp struct {
	Name string `json:"name"`
	Ref  string `json:"ref"`
	Tag  string `json:"tag"`
}

type RSpecReleaseEligible struct {
	Eligible bool `json:"eligible"`
}

type RSpecPort struct {
	Name          string `json:"name"`
	ContainerPort string `json:"containerPort"`
}

type RSpecTarget struct {
	Name    string         `json:"name"`
	Fleet   string         `json:"fleet"`
	Scale   RSpecScale     `json:"scale"`
	Release *RSpecRelease  `json:"release"`
	Metrics *[]RSpecMetric `json:"metrics"`
}

type RSpecScale struct {
	Min       int            `json:"min"`
	Max       int            `json:"max"`
	Resources RSpecResources `json:"resources"`
}

type RSpecResources struct {
	CPU string `json:"cpu"`
}

type RSpecRelease struct {
	Max      string `json:"max"`
	Rate     string `json:"rate"`
	Schedule string `json:"schedule"`
}

type RSpecMetric struct {
	Name      string       `json:"name"`
	Queries   RSpecQueries `json:"queries"`
	Objective float64      `json:"objective"`
}

type RSpecQueries struct {
	Acceptable string `json:"acceptable"`
	Total      string `json:"total"`
}

// RevisionStatus defines the observed state of Revision
type RevisionStatus struct {
	Targets []RStatusTarget `json:"targets"`
}

type RStatusTarget struct {
	Name        string              `json:"name"`
	Deployments []RStatusDeployment `json:"deployments"`
}

type RStatusDeployment struct {
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
