package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Incarnation is the Schema for the incarnation API
// +k8s:openapi-gen=true
type Incarnation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IncarnationSpec   `json:"spec,omitempty"`
	Status IncarnationStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IncarnationList contains a list of Incarnation
type IncarnationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Incarnation `json:"items"`
}

// IncarnationSpec defines the desired state of Incarnation
type IncarnationSpec struct {
	App        IncarnationApp        `json:"app"`
	Assignment IncarnationAssignment `json:"assignment"`
	Scale      IncarnationScale      `json:"scale"`
	Release    IncarnationRelease    `json:"release"`
	Ports      []PortInfo            `json:"ports,omitempty"`
}

type IncarnationApp struct {
	Name  string `json:"name"`
	Tag   string `json:"tag"`
	Ref   string `json:"ref"`
	Image string `json:"image"`
}

type IncarnationAssignment struct {
	Name   string `json:"name"`
	Target string `json:"target"`
}

type IncarnationScale struct {
	Min       int32                      `json:"min"`
	Max       int32                      `json:"max"`
	Resources []IncarnationScaleResource `json:"resources"`
}

type IncarnationScaleResource struct {
	CPU string `json:"cpu"`
}

type IncarnationRelease struct {
	Max      int    `json:"max"`
	Rate     string `json:"rate"`
	Schedule string `json:"schedule"`
}

// IncarnationStatus defines the observed state of Incarnation
type IncarnationStatus struct {
	Health    IncarnationHealthStatus     `json:"health"`
	Release   *IncarnationReleaseStatus   `json:"release,omitempty"`
	Scale     IncarnationScaleStatus      `json:"scale"`
	Resources []IncarnationResourceStatus `json:"resources"`
}

type IncarnationHealthStatus struct {
	Healthy bool                            `json:"healthy"`
	Metrics []IncarnationHealthMetricStatus `json:"metrics"`
}

type IncarnationHealthMetricStatus struct {
	Name      string  `json:"name"`
	Objective float64 `json:"objective"`
	Actual    float64 `json:"actual"`
}

type IncarnationReleaseStatus struct {
	PeakPercent    int32 `json:"peakPercent"`
	CurrentPercent int32 `json:"currentPercent"`
}

type IncarnationScaleStatus struct {
	Current int32 `json:"current"`
	Desired int32 `json:"desired"`
}

type IncarnationResourceStatus struct {
	ApiVersion string                `json:"apiVersion"`
	Kind       string                `json:"kind"`
	Metadata   *types.NamespacedName `json:"metadata, omitempty"`
	Status     string                `json:"status"`
}

func init() {
	SchemeBuilder.Register(&Incarnation{}, &IncarnationList{})
}
