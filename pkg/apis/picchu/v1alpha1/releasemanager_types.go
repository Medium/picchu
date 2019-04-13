package v1alpha1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
)

// +genclient
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
	Revisions []ReleaseManagerRevisionStatus `json:"revisions,omitempty"`
}

type ReleaseManagerRevisionStatus struct {
	Tag            string                                 `json:"tag"`
	LastUpdated    *metav1.Time                           `json:"lastUpdated"`
	CurrentPercent uint32                                 `json:"currentPercent"`
	PeakPercent    uint32                                 `json:"peakPercent"`
	Released       bool                                   `json:"released"`
	Expired        bool                                   `json:"expired"`
	EverReleased   bool                                   `json:"everReleased"`
	Deployed       bool                                   `json:"deployed"`
	EverDeployed   bool                                   `json:"everDeployed"`
	Retired        bool                                   `json:"retired"`
	Resources      []ReleaseManagerRevisionResourceStatus `json:"resources"`
	Scale          ReleaseManagerRevisionScaleStatus      `json:"scale"`
}

type ReleaseManagerRevisionResourceStatus struct {
	ApiVersion string                `json:"apiVersion"`
	Kind       string                `json:"kind"`
	Metadata   *types.NamespacedName `json:"metadata,omitempty"`
	Status     string                `json:"status"`
}

func (r *ReleaseManagerRevisionStatus) CreateOrUpdateResourceStatus(rs ReleaseManagerRevisionResourceStatus) {
	for i, s := range r.Resources {
		if s.Metadata.Name == rs.Metadata.Name &&
			s.Metadata.Namespace == rs.Metadata.Namespace &&
			s.ApiVersion == rs.ApiVersion &&
			s.Kind == rs.Kind {
			r.Resources[i].Status = rs.Status
			return
		}

	}
	r.Resources = append(r.Resources, rs)
	return
}

type ReleaseManagerRevisionScaleStatus struct {
	Current int32
	Desired int32
	Peak    int32
}

func (r *ReleaseManager) RevisionStatus(tag string) *ReleaseManagerRevisionStatus {
	for _, s := range r.Status.Revisions {
		if s.Tag == tag {
			return &s
		}
	}
	now := metav1.Now()
	s := ReleaseManagerRevisionStatus{
		Tag:            tag,
		Expired:        false,
		LastUpdated:    &now,
		CurrentPercent: 0,
		PeakPercent:    0,
	}
	r.Status.Revisions = append(r.Status.Revisions, s)
	return &s
}

func (r *ReleaseManager) UpdateRevisionStatus(u *ReleaseManagerRevisionStatus) {
	for i, s := range r.Status.Revisions {
		if s.Tag == u.Tag {
			r.Status.Revisions[i] = *u
		}
	}
}

func (r *ReleaseManager) TargetNamespace() string {
	return fmt.Sprintf("%s-%s", r.Spec.App, r.Spec.Target)
}

func (r *ReleaseManager) IsDeleted() bool {
	return !r.ObjectMeta.DeletionTimestamp.IsZero()
}

func (r *ReleaseManager) IsFinalized() bool {
	for _, item := range r.ObjectMeta.Finalizers {
		if item == FinalizerReleaseManager {
			return false
		}
	}
	return true
}

func (r *ReleaseManager) Finalize() {
	finalizers := []string{}
	for _, item := range r.ObjectMeta.Finalizers {
		if item == FinalizerReleaseManager {
			continue
		}
		finalizers = append(finalizers, item)
	}
	r.ObjectMeta.Finalizers = finalizers
}

func init() {
	SchemeBuilder.Register(&ReleaseManager{}, &ReleaseManagerList{})
}
