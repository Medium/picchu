package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
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

// RevisionSpec defines the desired state of Revision
type RevisionSpec struct {
	App     RevisionApp      `json:"app"`
	Release RevisionRelease  `json:"release"`
	Ports   []PortInfo       `json:"ports"`
	Targets []RevisionTarget `json:"targets"`
}

type RevisionApp struct {
	Name  string `json:"name"`
	Ref   string `json:"ref"`
	Tag   string `json:"tag"`
	Image string `json:"image"`
}

type RevisionRelease struct {
	Eligible bool `json:"eligible"`
}

type RevisionTarget struct {
	Name    string                 `json:"name"`
	Fleet   string                 `json:"fleet"`
	Scale   RevisionTargetScale    `json:"scale"`
	Release *RevisionTargetRelease `json:"release,omitempty"`
	Metrics []RevisionTargetMetric `json:"metrics,omitempty"`
}

type RevisionTargetScale struct {
	Min       int32          `json:"min,omitempty"`
	Default   int32          `json:"default,omitempty"`
	Max       int32          `json:"max,omitempty"`
	Resources ScaleResources `json:"resources,omitempty"`
}

type ScaleResources struct {
	CPU string `json:"cpu,omitempty"`
}

type RevisionTargetRelease struct {
	Max      int    `json:"max"`
	Rate     string `json:"rate"`
	Schedule string `json:"schedule"`
}

type RevisionTargetMetric struct {
	Name      string                      `json:"name"`
	Queries   RevisionTargetMetricQueries `json:"queries"`
	Objective float64                     `json:"objective"`
}

type RevisionTargetMetricQueries struct {
	Acceptable string `json:"acceptable"`
	Total      string `json:"total"`
}

// RevisionStatus defines the observed state of Revision
type RevisionStatus struct {
	Targets []RevisionTargetStatus `json:"targets"`
}

type RevisionTargetStatus struct {
	Name         string                            `json:"name"`
	Incarnations []RevisionTargetIncarnationStatus `json:"incarnations"`
}

type RevisionTargetIncarnationStatus struct {
	Name    string `json:"name"`
	Cluster string `json:"cluster"`
	Status  string `json:"status"`
}

func init() {
	SchemeBuilder.Register(&Revision{}, &RevisionList{})
}
