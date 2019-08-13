package v1alpha1

import (
	"time"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	istiov1alpha3 "github.com/knative/pkg/apis/istio/v1alpha3"
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
	App              RevisionApp                  `json:"app"`
	Ports            []PortInfo                   `json:"ports"`
	Targets          []RevisionTarget             `json:"targets"`
	TrafficPolicy    *istiov1alpha3.TrafficPolicy `json:"trafficPolicy,omitempty"`
	Failed           bool                         `json:"failed"`
	IgnoreSLOs       bool                         `json:"ignoreSLOs,omitempty"`
	TagRoutingHeader string                       `json:"tagRoutingHeader,omitempty"`
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

	ExternalTest ExternalTest `json:"externalTest"`
}

type ExternalTest struct {
	Enabled   bool `json:"enabled"`
	Started   bool `json:"started"`
	Completed bool `json:"completed"`
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
	Name    string                `json:"name"`
	Scale   RevisionScaleStatus   `json:"scale"`
	Release RevisionReleaseStatus `json:"release"`
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

func (r *RevisionTargetStatus) AddReleaseManagerStatus(status ReleaseManagerRevisionStatus) {
	r.Release.CurrentPercent = status.CurrentPercent
	r.Release.PeakPercent = status.PeakPercent
	r.Scale.Current = uint32(status.Scale.Current)
	r.Scale.Desired = uint32(status.Scale.Desired)
	r.Scale.Peak = uint32(status.Scale.Peak)
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

func (r *RevisionTarget) IsExternalTestPending() bool {
	return r.ExternalTest.Enabled && !r.ExternalTest.Completed
}

func init() {
	SchemeBuilder.Register(&Revision{}, &RevisionList{})
}
