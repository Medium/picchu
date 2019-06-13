package v1alpha1

import (
	"time"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// TODO(bob): I don't like this. maybe configurable?
	// Which targets need to reach 100% to consider the rollout complete and
	// stop slo failures from rolling back.
	rolloutTargets = []string{"production"}
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

func (r *Revision) Fail() {
	if !r.Spec.Failed {
		r.Spec.Failed = true
		t := time.Now()
		if r.Annotations == nil {
			r.Annotations = map[string]string{
				AnnotationFailedAt: t.Format(time.RFC3339),
			}
		} else {
			r.Annotations[AnnotationFailedAt] = t.Format(time.RFC3339)
		}
	}
}

func (r *Revision) SinceFailed() time.Duration {
	ft, ok := r.Annotations[AnnotationFailedAt]
	if !ok {
		return time.Duration(0)
	}
	t, err := time.Parse(time.RFC3339, ft)
	if err != nil {
		return time.Duration(0)
	}
	return time.Since(t)
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
	App        RevisionApp      `json:"app"`
	Ports      []PortInfo       `json:"ports"`
	Targets    []RevisionTarget `json:"targets"`
	Failed     bool             `json:"failed"`
	IgnoreSLOs bool             `json:"ignoreSLOs,omitempty"`
	// This is immutable. Changing it has undefined behavior
	UseNewTagStyle bool `json:"useNewTagStyle,omitempty"`
}

type RevisionApp struct {
	Name  string `json:"name"`
	Ref   string `json:"ref"`
	Tag   string `json:"tag"`
	Image string `json:"image"`
}

type RevisionTarget struct {
	Name           string                 `json:"name"`
	Fleet          string                 `json:"fleet"`
	Scale          ScaleInfo              `json:"scale"`
	Release        ReleaseInfo            `json:"release,omitempty"`
	Metrics        []RevisionTargetMetric `json:"metrics,omitempty"`
	ConfigSelector *metav1.LabelSelector  `json:"configSelector,omitempty"`
	AWS            AWSInfo                `json:"aws,omitempty"`
	AlertRules     []monitoringv1.Rule    `json:"alertRules,omitempty"`

	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	Resources      corev1.ResourceRequirements `json:"resources,omitempty"`
	LivenessProbe  *corev1.Probe               `json:"livenessProbe,omitempty"`
	ReadinessProbe *corev1.Probe               `json:"readinessProbe,omitempty"`
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

type RevisionStatus struct {
	Targets []RevisionTargetStatus `json:"targets"`
}

func (r *RevisionStatus) AddTarget(ts RevisionTargetStatus) {
	r.Targets = append(r.Targets, ts)
}

// IsRolloutComplete returns true if all rolloutTargets are rollout complete
func (r *RevisionStatus) IsRolloutComplete() bool {
	for _, target := range rolloutTargets {
		ts := r.GetTarget(target)
		if ts == nil {
			return false
		}
		if !ts.IsRolloutComplete() {
			return false
		}
	}
	return true
}

func (r *RevisionStatus) GetTarget(name string) *RevisionTargetStatus {
	for _, ts := range r.Targets {
		if ts.Name == name {
			return &ts
		}
	}
	return nil
}

// RevisionStatus defines the observed state of Revision
type RevisionTargetStatus struct {
	Name     string                 `json:"name"`
	Scale    RevisionScaleStatus    `json:"scale"`
	Release  RevisionReleaseStatus  `json:"release"`
	Clusters RevisionClustersStatus `json:"clusters"`
}

type RevisionScaleStatus struct {
	Current uint32 `json:"current"`
	Desired uint32 `json:"desired"`
	Peak    uint32 `json:"peak"`
}

type RevisionReleaseStatus struct {
	CurrentPercent uint32 `json:"currentPercent"`
	PeakPercent    uint32 `json:"peakPercent"`
}

type RevisionClustersStatus struct {
	Names         []string `json:"names,omitempty"`
	MinPercent    uint32   `json:"minPercent"`
	MaxPercent    uint32   `json:"maxPercent"`
	ReleasedCount uint32   `json:"releasedCount"`
}

func (r *RevisionClustersStatus) Count() uint32 {
	return uint32(len(r.Names))
}

func (r *RevisionTargetStatus) AddReleaseManagerStatus(name string, status ReleaseManagerRevisionStatus) {
	weightedPercent := r.Release.CurrentPercent*r.Clusters.Count() + status.CurrentPercent
	r.Clusters.Names = append(r.Clusters.Names, name)
	r.Release.CurrentPercent = weightedPercent / r.Clusters.Count()
	r.Scale.Current += uint32(status.Scale.Current)
	r.Scale.Desired += uint32(status.Scale.Desired)
	if status.CurrentPercent > 0 {
		r.Clusters.ReleasedCount++
	}
	if status.CurrentPercent < r.Clusters.MinPercent {
		r.Clusters.MinPercent = status.CurrentPercent
	}
	if status.CurrentPercent > r.Clusters.MaxPercent {
		r.Clusters.MaxPercent = status.CurrentPercent
	}
	if r.Release.CurrentPercent > r.Release.PeakPercent {
		r.Release.PeakPercent = r.Release.CurrentPercent
	}
	if r.Scale.Current > r.Scale.Peak {
		r.Scale.Peak = r.Scale.Current
	}
}

func (r *RevisionTargetStatus) IsRolloutComplete() bool {
	return r.Clusters.MaxPercent >= 100
}

func (r *Revision) GitTimestamp() time.Time {
	gt, ok := r.Annotations[AnnotationGitCommitterTimestamp]
	if !ok {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, gt)
	if err != nil {
		return time.Time{}
	}
	return t
}

func init() {
	SchemeBuilder.Register(&Revision{}, &RevisionList{})
}
