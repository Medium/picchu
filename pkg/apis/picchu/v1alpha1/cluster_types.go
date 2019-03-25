package v1alpha1

import (
	"encoding/json"
	"errors"
	"fmt"

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
	Enabled       bool              `json:"enabled"`
	Config        *ClusterConfig    `json:"config,omitempty"`
	Weight        float64           `json:"weight"`
	AWS           *ClusterAWSInfo   `json:"aws,omitempty"`
	DNS           []ClusterDNSGroup `json:"dns,omitempty"`
	Ingresses     ClusterIngresses  `json:"ingresses"`
	DefaultDomain string            `json:"defaultDomain"`
}

type ClusterConfig struct {
	Server                   string `json:"server"`
	CertificateAuthorityData []byte `json:"certificate-authority-data"`
}

type ClusterAWSInfo struct {
	AccountID string `json:"accountId,id"`
	Region    string `json:"region"`
	AZ        string `json:"az,omitempty"`
}

const (
	Route53Provider    = "route53"
	CloudflareProvider = "cloudflare"
	PrivateIngressName = "private"
	PublicIngressName  = "public"
)

type ClusterDNSGroup struct {
	Hosts    []string `json:"hosts,omitempty"`
	Provider string   `json:"provider,omitempty"`
	Ingress  string   `json:"ingress,omitempty"`
}

type ClusterIngresses struct {
	Public  IngressInfo `json:"public"`
	Private IngressInfo `json:"private"`
}

type IngressInfo struct {
	HostedZoneId string `json:"hostedZoneId"`
	DNSName      string `json:"dnsName"`
	Gateway      string `json:"gateway,omitempty"`
}

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	Kubernetes ClusterKubernetesStatus  `json:"kubernetes,omitempty"`
	AWS        *ClusterAWSInfo          `json:"aws,omitempty"`
	Conditions []ClusterConditionStatus `json:"conditions,omitempty"`
}

type ClusterKubernetesStatus struct {
	Version string `json:"version"`
}

type ClusterConditionStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
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

// Region gets the region/az the cluster is in based on spec and status. This
// shouldn't be used to *discover* the region/az of a InClusterConfig, but only
// see what region was already discovered by some other process.
func (c *Cluster) RegionAZ() string {
	region, az := "", ""
	if c.Spec.AWS != nil {
		region = c.Spec.AWS.Region
		az = c.Spec.AWS.AZ
	}
	if c.Status.AWS != nil {
		region = c.Status.AWS.Region
		az = c.Status.AWS.AZ
	}
	return fmt.Sprintf("%s%s", region, az)
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
