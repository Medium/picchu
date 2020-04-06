package v1alpha1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ReleaseManager is the Schema for the releasemanagers API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:resource:categories=all;picchu
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
	Fleet  string `json:"fleet"`
	App    string `json:"app"`
	Target string `json:"target"`
}

// ReleaseManagerStatus defines the observed state of ReleaseManager
// +k8s:openapi-gen=true
type ReleaseManagerStatus struct {
	// +listType=set
	Revisions   []ReleaseManagerRevisionStatus `json:"revisions,omitempty"`
	LastUpdated *metav1.Time                   `json:"lastUpdated"`
}

type ReleaseManagerRevisionStatus struct {
	Tag                          string                              `json:"tag"`
	State                        ReleaseManagerRevisionStateStatus   `json:"state,omitempty"`
	CurrentPercent               uint32                              `json:"currentPercent"`
	PeakPercent                  uint32                              `json:"peakPercent"`
	ReleaseEligible              bool                                `json:"releaseEligible"`
	TriggeredAlarms              []string                            `json:"triggeredAlerts,omitempty"`
	LastUpdated                  *metav1.Time                        `json:"lastUpdated"`
	GitTimestamp                 *metav1.Time                        `json:"gitTimestamp,omitempty"`
	RevisionTimestamp            *metav1.Time                        `json:"revisionTimestamp,omitempty"`
	DeployingStartTimestamp      *metav1.Time                        `json:"deployingStartTimestamp,omitempty"`
	CanaryStartTimestamp         *metav1.Time                        `json:"canaryStartTimestamp,omitempty"`
	PendingReleaseStartTimestamp *metav1.Time                        `json:"pendingReleaseStartTimestamp,omitempty"`
	ReleaseStartTimestamp        *metav1.Time                        `json:"releaseStartTimestamp,omitempty"`
	TTL                          int64                               `json:"ttl,omitempty"`
	Metrics                      ReleaseManagerRevisionMetricsStatus `json:"metrics,omitempty"`
	Scale                        ReleaseManagerRevisionScaleStatus   `json:"scale"`
	Deleted                      bool                                `json:"deleted,omitempty"`
}

// ReleaseManagerRevisionMetricsStatus defines the observed state of ReleaseManagerRevisionMetrics
type ReleaseManagerRevisionMetricsStatus struct {
	GitCreateSeconds              *float64 `json:"gitCreateSeconds,omitempty"`
	GitDeploySeconds              *float64 `json:"gitDeploySeconds,omitempty"`
	GitCanarySeconds              *float64 `json:"gitCanarySeconds,omitempty"`
	GitPendingReleaseSeconds      *float64 `json:"gitPendingReleaseSeconds,omitempty"`
	GitReleaseSeconds             *float64 `json:"gitReleaseSeconds,omitempty"`
	RevisionDeploySeconds         *float64 `json:"revisionDeploySeconds,omitempty"`
	RevisionCanarySeconds         *float64 `json:"revisionCanarySeconds,omitempty"`
	RevisionReleaseSeconds        *float64 `json:"revisionReleaseSeconds,omitempty"`
	ReivisonPendingReleaseSeconds *float64 `json:"revisionPendingReleaseSeconds,omitempty"`
	RevisionRollbackSeconds       *float64 `json:"revisionRollbackSeconds,omitempty"`
	DeploySeconds                 *float64 `json:"deploySeconds,omitempty"`
	CanarySeconds                 *float64 `json:"canarySeconds,omitempty"`
	ReleaseSeconds                *float64 `json:"releaseSeconds,omitempty"`
}

type ReleaseManagerRevisionStateStatus struct {
	Current     string       `json:"current"`
	Target      string       `json:"target"`
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`
}

func (r *ReleaseManagerRevisionStateStatus) EqualTo(other *ReleaseManagerRevisionStateStatus) bool {
	return r.Target == other.Target && r.Current == other.Current
}

type ReleaseManagerRevisionScaleStatus struct {
	Current int32 `json:"Current,omitempty"`
	Desired int32 `json:"Desired,omitempty"`
	Peak    int32 `json:"Peak,omitempty"`
}

func (r *ReleaseManager) RevisionStatus(tag string) *ReleaseManagerRevisionStatus {
	for _, s := range r.Status.Revisions {
		if s.Tag == tag {
			return &s
		}
	}
	now := metav1.Now()
	s := ReleaseManagerRevisionStatus{
		Tag: tag,
		State: ReleaseManagerRevisionStateStatus{
			Current: "created",
			Target:  "created",
		},
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
