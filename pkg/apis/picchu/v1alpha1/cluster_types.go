package v1alpha1

import (
	"encoding/json"
	"errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientapi "k8s.io/client-go/tools/clientcmd/api"
)

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {
	Enabled bool          `json:"enabled"`
	Config  *CSpecConfig  `json:"config,omitempty"`
	Weight  float64       `json:"weight"`
	Account *CSpecAccount `json:"account,omitempty"`
}

type CSpecConfig struct {
	Server                   string `json:"server"`
	CertificateAuthorityData []byte `json:"certificate-authority-data"`
}

// Account is needed for Cluster to provision EKS clusters
type CSpecAccount struct {
	ID     string `json:"id"`
	Region string `json:"region"`
	AZ     string `json:"az,omitempty"`
}

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	Kubernetes CStatusKubernetes  `json:"kubernetes"`
	Conditions []CStatusCondition `json:"conditions"`
}

type CStatusKubernetes struct {
	Version string `json:"version"`
}

type CStatusCondition struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Cluster is the Schema for the clusters API
// +k8s:openapi-gen=true
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}

// Config creates a rest.Config for the Cluster, whether it be remote or
// incluster. Secret is expected to contain "auth-info" if Config is
// specified.
func (c *Cluster) Config(secret *corev1.Secret) (*rest.Config, error) {
	if secret == nil && c.Spec.Config == nil {
		return rest.InClusterConfig()
	}
	if secret == nil || c.Spec.Config == nil {
		return nil, errors.New("Illegal state, Secret and Config must both be specified for remote clusters")
	}
	server := c.Spec.Config.Server
	cert := c.Spec.Config.CertificateAuthorityData
	authInfoJSON, ok := secret.Data["auth-info"]
	if !ok {
		return nil, errors.New("auth-info not found in Secret")
	}
	var authInfo clientapi.AuthInfo
	if err := json.Unmarshal(authInfoJSON, &authInfo); err != nil {
		return nil, err
	}
	conf := clientcmd.NewDefaultClientConfig(clientapi.Config{
		Clusters: map[string]*clientapi.Cluster{
			"cluster": &clientapi.Cluster{
				CertificateAuthorityData: cert,
				Server:                   server,
			},
		},
		Contexts: map[string]*clientapi.Context{
			"cluster": &clientapi.Context{
				Cluster:   "cluster",
				AuthInfo:  "auth",
				Namespace: "picchu",
			},
		},
		AuthInfos: map[string]*clientapi.AuthInfo{
			"auth": &authInfo,
		},
		CurrentContext: "cluster",
	}, &clientcmd.ConfigOverrides{})
	return conf.ClientConfig()
}
