package v1alpha1

import (
	"encoding/json"
	"errors"
	"k8s.io/apimachinery/pkg/api/resource"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientapi "k8s.io/client-go/tools/clientcmd/api"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Cluster is the Schema for the clusters API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:resource:categories=all;picchu
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

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {
	Enabled             bool               `json:"enabled"`
	HotStandby          bool               `json:"hotStandby,omitempty"`
	Config              *ClusterConfig     `json:"config,omitempty"`
	ScalingFactorString *string            `json:"scalingFactorString,omitempty"`
	ScalingFactor       *resource.Quantity `json:"scalingFactor,omitempty"`
	Ingresses           ClusterIngresses   `json:"ingresses"`
	EnableDevRoutes     bool               `json:"enableDevRoutes,omitempty"`
	DevRouteTagTemplate string             `json:"devRouteTagTemplate,omitempty"`
}

type ClusterConfig struct {
	Server                   string `json:"server"`
	CertificateAuthorityData []byte `json:"certificate-authority-data"`
}

const (
	PrivateIngressName = "private"
	PublicIngressName  = "public"
)

type ClusterIngresses struct {
	Public  IngressInfo `json:"public"`
	Private IngressInfo `json:"private"`
}

type IngressInfo struct {
	DNSName        string   `json:"dnsName"`
	Gateway        string   `json:"gateway,omitempty"`
	DefaultDomains []string `json:"defaultDomains"`
}

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	Kubernetes ClusterKubernetesStatus `json:"kubernetes,omitempty"`
}

type ClusterKubernetesStatus struct {
	Version string `json:"version"`
	Ready   bool   `json:"ready"`
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
			"cluster": {
				CertificateAuthorityData: cert,
				Server:                   server,
			},
		},
		Contexts: map[string]*clientapi.Context{
			"cluster": {
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

func (c *Cluster) Fleet() string {
	fleet, _ := c.Labels[LabelFleet]
	return fleet
}

func (c *Cluster) IsDeleted() bool {
	return !c.ObjectMeta.DeletionTimestamp.IsZero()
}

func (c *Cluster) IsFinalized() bool {
	for _, item := range c.ObjectMeta.Finalizers {
		if item == FinalizerCluster {
			return false
		}
	}
	return true
}

func (c *Cluster) Finalize() {
	finalizers := []string{}
	for _, item := range c.ObjectMeta.Finalizers {
		if item == FinalizerCluster {
			continue
		}
		finalizers = append(finalizers, item)
	}
	c.ObjectMeta.Finalizers = finalizers
}

func (c *Cluster) AddFinalizer() {
	c.ObjectMeta.Finalizers = append(c.ObjectMeta.Finalizers, FinalizerCluster)
}
