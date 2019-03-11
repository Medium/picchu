package v1alpha1

import (
	"fmt"
	"sort"
	"time"

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

// CurrentPercentTarget returns the percent of traffic this incarnation would
// like to recieve at any given moment, with respect to max available
func (i *Incarnation) CurrentPercentTarget(lastUpdate *metav1.Time, current uint32, max uint32) uint32 {
	// TODO(bob): gate increment on current time for "humane" scheduled
	// deployments that are currently at 0 percent of traffic
	releaseSpec := i.Spec.Release
	if max <= 0 {
		return 0
	}
	delay := time.Duration(*releaseSpec.Rate.DelaySeconds) * time.Second
	increment := releaseSpec.Rate.Increment
	if releaseSpec.Max < max {
		max = releaseSpec.Max
	}
	deadline := time.Time{}
	if lastUpdate != nil {
		deadline = lastUpdate.Add(delay)
	}

	if deadline.After(time.Now()) {
		return current
	}
	current = current + increment
	if current > max {
		return max
	}
	return current
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IncarnationList contains a list of Incarnation
type IncarnationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Incarnation `json:"items"`
}

func (i *IncarnationList) LatestRelease() (*Incarnation, error) {
	releases, err := i.SortedReleases()
	if err != nil {
		return nil, err
	}
	if len(releases) > 0 {
		return &releases[0], nil
	}
	return nil, nil
}

// SortedReleases returns all release enabled incarnations in reverse
// chronological order. Requires all releases to be annotated with the
// AnnotationGitCommitterTimestamp annotation, which should be an RFC3339 timestamp
func (i *IncarnationList) SortedReleases() ([]Incarnation, error) {
	releases := []Incarnation{}
	for _, incarnation := range i.Items {
		if incarnation.Spec.Release.Eligible {
			releases = append(releases, incarnation)
		}
	}

	var err error

	sort.Slice(releases, func(i, j int) bool {
		a := releases[i].GitTimestamp()
		b := releases[j].GitTimestamp()
		if a.IsZero() || b.IsZero() {
			err = fmt.Errorf(
				"Release is missing the '%s' RFC3339 timestamp annotation",
				AnnotationGitCommitterTimestamp,
			)
		}
		return a.After(b)
	})
	return releases, err
}

// IncarnationSpec defines the desired state of Incarnation
type IncarnationSpec struct {
	App            IncarnationApp        `json:"app"`
	Assignment     IncarnationAssignment `json:"assignment"`
	Scale          ScaleInfo             `json:"scale"`
	Release        ReleaseInfo           `json:"release,omitempty"`
	Ports          []PortInfo            `json:"ports,omitempty"`
	ConfigSelector *metav1.LabelSelector `json:"configSelector,omitempty"`
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
	gt, ok := i.Annotations[AnnotationGitCommitterTimestamp]
	if !ok {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, gt)
	if err != nil {
		return time.Time{}
	}
	return t
}

func (i *Incarnation) TargetNamespace() string {
	return fmt.Sprintf("%s-%s", i.Spec.App.Name, i.Spec.Assignment.Target)
}

func init() {
	SchemeBuilder.Register(&Incarnation{}, &IncarnationList{})
}
