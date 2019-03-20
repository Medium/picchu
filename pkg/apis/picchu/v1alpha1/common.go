package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type PortMode string

const (
	// PortPublic sets a port to be published to the Internet.
	PortPublic PortMode = "public"

	// PortPrivate sets a port to be published on a private gateway, making it available to other
	// clusters and users on our private networks (including VPN), but not on the Internet.
	PortPrivate PortMode = "private"

	// PortInternal sets the port to not be published to any gateway, making it only available within
	// the local Kubernetes cluster.
	PortLocal PortMode = "local"
)

const (
	HumaneSchedule                  = "humane"
	AlwaysSchedule                  = "always"
	LabelApp                        = "picchu.medium.engineering/app"
	LabelTag                        = "picchu.medium.engineering/tag"
	LabelCluster                    = "picchu.medium.engineering/cluster"
	LabelFleet                      = "picchu.medium.engineering/fleet"
	LabelRevision                   = "picchu.medium.engineering/revision"
	LabelTarget                     = "picchu.medium.engineering/target"
	LabelOwnerName                  = "picchu.medium.engineering/ownerName"
	LabelOwnerType                  = "picchu.medium.engineering/ownerType"
	FinalizerIncarnation            = "picchu.medium.engineering/incarnation"
	FinalizerReleaseManager         = "picchu.medium.engineering/releasemanager"
	FinalizerCluster                = "picchu.medium.engineering/cluster"
	OwnerIncarnation                = "incarnation"
	OwnerReleaseManager             = "releasemanager"
	AnnotationGitCommitterTimestamp = "git-scm.com/committer-timestamp"
)

type PortInfo struct {
	Name          string          `json:"name"`
	Hosts         []string        `json:"hosts,omitempty"`
	IngressPort   int32           `json:"ingressPort,omitempty"`
	Port          int32           `json:"port,omitempty"`
	ContainerPort int32           `json:"containerPort,omitempty"`
	Protocol      corev1.Protocol `json:"protocol,omitempty"`
	Mode          PortMode        `json:"mode"`
}

type ScaleInfo struct {
	Min                            *int32 `json:"min,omitempty"`
	Default                        int32  `json:"default,omitempty"`
	Max                            int32  `json:"max,omitempty"`
	TargetCPUUtilizationPercentage *int32 `json:"targetCPUUtilizationPercentage,omitempty"`

	// Deprecated. Use TargetCPUUtilizationPercentage.
	Resources ResourceInfo `json:"resources,omitempty"`
}

type ResourceInfo struct {
	CPU resource.Quantity `json:"cpu,omitempty"`
}

type ReleaseInfo struct {
	Eligible bool     `json:"eligible,omitempty"`
	Max      uint32   `json:"max,omitempty"`
	Rate     RateInfo `json:"rate,omitempty"`
	Schedule string   `json:"schedule,omitempty"`
}

type RateInfo struct {
	Increment    uint32 `json:"increment,omitempty"`
	DelaySeconds *int64 `json:"delay,omitempty"`
}
