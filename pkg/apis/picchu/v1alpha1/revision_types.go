package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
	Release RevisionRelease  `json:"release"`
	Ports   []RevisionPort   `json:"ports"`
	Targets []RevisionTarget `json:"targets"`
}

type RevisionApp struct {
	Name string `json:"name"`
	Ref  string `json:"ref"`
	Tag  string `json:"tag"`
}

type RevisionRelease struct {
	Eligible bool `json:"eligible"`
}

type PortMode string

var (
	// PortPublic sets a port to be published to the Internet.
	PortPublic PortMode = "public"

	// PortPrivate sets a port to be published on a private gateway, making it available to other
	// clusters and users on our private networks (including VPN), but not on the Internet.
	PortPrivate PortMode = "private"

	// PortInternal sets the port to not be published to any gateway, making it only available within
	// the local Kubernetes cluster.
	PortLocal PortMode = "local"
)

type RevisionPort struct {
	Name          string          `json:"name,omitempty"`
	Port          int32           `json:"port,omitempty"`
	ContainerPort int32           `json:"containerPort,omitempty"`
	Protocol      corev1.Protocol `json:"protocol,omitempty"`
	Mode          PortMode        `json:"mode"`
}

type RevisionTarget struct {
	Name    string                 `json:"name"`
	Fleet   string                 `json:"fleet"`
	Scale   RevisionTargetScale    `json:"scale"`
	Release *RevisionTargetRelease `json:"release,omitempty"`
	Metrics []RevisionTargetMetric `json:"metrics,omitempty"`
}

type RevisionTargetScale struct {
	Min       int32          `json:"min,omitempty"`
	Default   int32          `json:"default,omitempty"`
	Max       int32          `json:"max,omitempty"`
	Resources ScaleResources `json:"resources,omitempty"`
}

type ScaleResources struct {
	CPU resource.Quantity `json:"cpu,omitempty"`
}

type RevisionTargetRelease struct {
	Max      int    `json:"max"`
	Rate     string `json:"rate"`
	Schedule string `json:"schedule"`
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
	Targets []RevisionTargetStatus `json:"targets"`
}

type RevisionTargetStatus struct {
	Name        string                           `json:"name"`
	Deployments []RevisionTargetDeploymentStatus `json:"deployments"`
}

type RevisionTargetDeploymentStatus struct {
	Name    string `json:"name"`
	Cluster string `json:"cluster"`
}

func init() {
	SchemeBuilder.Register(&Revision{}, &RevisionList{})
}
