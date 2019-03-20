package v1alpha1

import (
	"fmt"
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
	Releases []ReleaseManagerReleaseStatus `json:"releases,omitempty"`
}

type ReleaseManagerReleaseStatus struct {
	Tag            string       `json:"tag"`
	LastUpdate     *metav1.Time `json:"lastUpdated"`
	CurrentPercent uint32       `json:"currentPercent"`
	PeakPercent    uint32       `json:"peakPercent"`
}

func (r *ReleaseManager) ReleaseStatus(tag string) *ReleaseManagerReleaseStatus {
	for _, s := range r.Status.Releases {
		if s.Tag == tag {
			return &s
		}
	}
	now := metav1.Now()
	s := ReleaseManagerReleaseStatus{
		Tag:            tag,
		LastUpdate:     &now,
		CurrentPercent: 0,
		PeakPercent:    0,
	}
	r.Status.Releases = append(r.Status.Releases, s)
	return &s
}

func (r *ReleaseManager) UpdateReleaseStatus(u *ReleaseManagerReleaseStatus) {
	for i, s := range r.Status.Releases {
		if s.Tag == u.Tag {
			r.Status.Releases[i].LastUpdate = u.LastUpdate
			r.Status.Releases[i].CurrentPercent = u.CurrentPercent
			r.Status.Releases[i].PeakPercent = u.PeakPercent
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
