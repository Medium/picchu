package v1alpha1

import (
	"time"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
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
	Ports   []PortInfo       `json:"ports"`
	Targets []RevisionTarget `json:"targets"`
	Failed  bool             `json:"failed"`
}

type RevisionApp struct {
	Name  string `json:"name"`
	Ref   string `json:"ref"`
	Tag   string `json:"tag"`
	Image string `json:"image"`
}

type RevisionTarget struct {
	Name               string                      `json:"name"`
	Fleet              string                      `json:"fleet"`
	Resources          corev1.ResourceRequirements `json:"resources,omitempty"`
	Scale              ScaleInfo                   `json:"scale"`
	Release            ReleaseInfo                 `json:"release,omitempty"`
	Metrics            []RevisionTargetMetric      `json:"metrics,omitempty"`
	ConfigSelector     *metav1.LabelSelector       `json:"configSelector,omitempty"`
	AWS                AWSInfo                     `json:"aws,omitempty"`
	AlertRules         []monitoringv1.Rule         `json:"alertRules,omitempty"`
	ServiceAccountName string                      `json:"serviceAccountName,omitempty"`
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
