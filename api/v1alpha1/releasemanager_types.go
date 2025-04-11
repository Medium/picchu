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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ReleaseManagerSpec defines the desired state of ReleaseManager
type ReleaseManagerSpec struct {
	Fleet    string    `json:"fleet"`
	App      string    `json:"app"`
	Target   string    `json:"target"`
	Variants []Variant `json:"variants,omitempty"`
}

type Variant struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// ReleaseManagerStatus defines the observed state of ReleaseManager
type ReleaseManagerStatus struct {
	Revisions   []ReleaseManagerRevisionStatus `json:"revisions,omitempty"`
	LastUpdated *metav1.Time                   `json:"lastUpdated"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +k8s:openapi-gen=true
// ReleaseManager is the Schema for the releasemanagers API
type ReleaseManager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ReleaseManagerSpec   `json:"spec,omitempty"`
	Status ReleaseManagerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ReleaseManagerList contains a list of ReleaseManager
type ReleaseManagerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ReleaseManager `json:"items"`
}

type ReleaseManagerRevisionStatus struct {
	Tag                          string                              `json:"tag"`
	State                        ReleaseManagerRevisionStateStatus   `json:"state,omitempty"`
	CurrentPercent               uint32                              `json:"currentPercent"`
	PeakPercent                  uint32                              `json:"peakPercent"`
	ReleaseEligible              bool                                `json:"releaseEligible"`
	TriggeredAlarms              []string                            `json:"triggeredAlerts,omitempty"`
	TriggeredDatadogMonitors     []string                            `json:"triggeredDatadogMonitors,omitempty"`
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
	GitCreateSecondsInt              *int `json:"gitCreateSecondsInt,omitempty"`
	GitDeploySecondsInt              *int `json:"gitDeploySecondsInt,omitempty"`
	GitCanarySecondsInt              *int `json:"gitCanarySecondsInt,omitempty"`
	GitPendingReleaseSecondsInt      *int `json:"gitPendingReleaseSecondsInt,omitempty"`
	GitReleaseSecondsInt             *int `json:"gitReleaseSecondsInt,omitempty"`
	RevisionDeploySecondsInt         *int `json:"revisionDeploySecondsInt,omitempty"`
	RevisionCanarySecondsInt         *int `json:"revisionCanarySecondsInt,omitempty"`
	RevisionReleaseSecondsInt        *int `json:"revisionReleaseSecondsInt,omitempty"`
	ReivisonPendingReleaseSecondsInt *int `json:"revisionPendingReleaseSecondsInt,omitempty"`
	RevisionRollbackSecondsInt       *int `json:"revisionRollbackSecondsInt,omitempty"`
	DeploySecondsInt                 *int `json:"deploySecondsInt,omitempty"`
	CanarySecondsInt                 *int `json:"canarySecondsInt,omitempty"`
	ReleaseSecondsInt                *int `json:"releaseSecondsInt,omitempty"`
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
