package v1alpha1

import (
	"sort"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	annotationGitTimestamp = "git-scm.com/committer-timestamp"
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

// CurrentPercentTarget returns the percent of traffic this incarnation would
// like to recieve at any given moment, with respect to max available
func (i *Incarnation) CurrentPercentTarget(max int32) int32 {
	releaseStatus := i.Status.Release
	releaseSpec := i.Spec.Release
	current := releaseStatus.CurrentPercent
	if max > 0 {
		delay := releaseSpec.Rate.GetDelay()
		increment := releaseSpec.Rate.GetIncrement()
		specMax := releaseSpec.GetMax()
		if specMax < max {
			max = specMax
		}
		deadline := releaseStatus.LastUpdateTime().Add(delay)

		if deadline.Before(time.Now()) {
			current = current + increment
			if current > max {
				return max
			}
			return current
		}
	}
	return 0
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IncarnationList contains a list of Incarnation
type IncarnationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Incarnation `json:"items"`
}

func (i *IncarnationList) LatestRelease() *Incarnation {
	releases := i.SortedReleases()
	if len(releases) > 0 {
		return &releases[0]
	}
	return nil
}

func (i *IncarnationList) SortedReleases() []Incarnation {
	releases := []Incarnation{}
	for _, incarnation := range i.Items {
		if incarnation.Spec.Release.Eligible {
			releases = append(releases, incarnation)
		}
	}
	sort.Slice(releases, func(i, j int) bool {
		return releases[i].GitTimestamp().After(releases[j].GitTimestamp())
	})
	return releases
}

// IncarnationSpec defines the desired state of Incarnation
type IncarnationSpec struct {
	App        IncarnationApp        `json:"app"`
	Assignment IncarnationAssignment `json:"assignment"`
	Scale      ScaleInfo             `json:"scale"`
	Release    ReleaseInfo           `json:"release,omitempty"`
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

// IncarnationStatus defines the observed state of Incarnation
type IncarnationStatus struct {
	Health    IncarnationHealthStatus     `json:"health,omitempty"`
	Release   IncarnationReleaseStatus    `json:"release,omitempty"`
	Scale     IncarnationScaleStatus      `json:"scale,omitempty"`
	Resources []IncarnationResourceStatus `json:"resources,omitempty"`
}

type IncarnationHealthStatus struct {
	Healthy bool                            `json:"healthy,omitempty"`
	Metrics []IncarnationHealthMetricStatus `json:"metrics,omitempty"`
}

type IncarnationHealthMetricStatus struct {
	Name      string  `json:"name,omitempty"`
	Objective float64 `json:"objective,omitempty"`
	Actual    float64 `json:"actual,omitempty"`
}

type IncarnationReleaseStatus struct {
	PeakPercent    int32  `json:"peakPercent,omitempty"`
	CurrentPercent int32  `json:"currentPercent,omitempty"`
	LastUpdate     string `json:"lastUpdate,omitempty"`
}

func (irs *IncarnationReleaseStatus) LastUpdateTime() time.Time {
	lu, err := time.Parse(time.RFC3339, irs.LastUpdate)
	if err != nil {
		return time.Unix(0, 0)
	}
	return lu
}

type IncarnationScaleStatus struct {
	Current int32 `json:"current"`
	Desired int32 `json:"desired"`
}

type IncarnationResourceStatus struct {
	ApiVersion string                `json:"apiVersion"`
	Kind       string                `json:"kind"`
	Metadata   *types.NamespacedName `json:"metadata,omitempty"`
	Status     string                `json:"status"`
}

func (i *Incarnation) GitTimestamp() time.Time {
	gt, ok := i.Annotations[annotationGitTimestamp]
	if !ok {
		return time.Unix(0, 0)
	}
	t, err := time.Parse(time.RFC3339, gt)
	if err != nil {
		return time.Unix(0, 0)
	}
	return t
}

func init() {
	SchemeBuilder.Register(&Incarnation{}, &IncarnationList{})
}
