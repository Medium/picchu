package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterDeploymentSpec defines the desired state of ClusterDeployment
type ClusterDeploymentSpec struct {
	App     CDSpecApp     `json:"app"`
	Cluster CDSpecCluster `json:"cluster"`
	Scale   CDSpecScale   `json:"scale"`
	Release CDSpecRelease `json:"release"`
}

type CDSpecApp struct {
	Name string `json:"name"`
	Tag  string `json:"tag"`
}

type CDSpecCluster struct {
	Name string `json:"name"`
}

type CDSpecScale struct {
	Min       int             `json:"min"`
	Max       int             `json:"max"`
	Resources CDSpecResources `json:"resources"`
}

type CDSpecResources struct {
	CPU string `json:"cpu"`
}

type CDSpecRelease struct {
	Max      int    `json:"max"`
	Rate     string `json:"rate"`
	Schedule string `json:"schedule"`
}

// ClusterDeploymentStatus defines the observed state of ClusterDeployment
type ClusterDeploymentStatus struct {
	Health    CDStatusHealth     `json:"health"`
	Release   CDStatusRelease    `json:"release"`
	Scale     CDStatusScale      `json:"scale"`
	Resources []CDStatusResource `json:"resources"`
}

type CDStatusHealth struct {
	Healthy bool             `json:"healthy"`
	Metrics []CDStatusMetric `json:"metrics"`
}

type CDStatusMetric struct {
	Name      string  `json:"name"`
	Objective float64 `json:"objective"`
	Actual    float64 `json:"actual"`
}

type CDStatusRelease struct {
	PeakPercent    int `json:"peakPercent"`
	CurrentPercent int `json:"currentPercent"`
}

type CDStatusScale struct {
	Current int `json:"current"`
	Desired int `json:"desired"`
}

type CDStatusResource struct {
	ApiVersion string           `json:"apiVersion"`
	Kind       string           `json:"kind"`
	Metadata   CDStatusMetadata `json:"metadata"`
	Status     string           `json:"status"`
}

type CDStatusMetadata struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterDeployment is the Schema for the clusterdeployments API
// +k8s:openapi-gen=true
type ClusterDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterDeploymentSpec   `json:"spec,omitempty"`
	Status ClusterDeploymentStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterDeploymentList contains a list of ClusterDeployment
type ClusterDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterDeployment{}, &ClusterDeploymentList{})
}
