package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
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
	HumaneSchedule                      = "humane"
	AlwaysSchedule                      = "always"
	LabelApp                            = "picchu.medium.engineering/app"
	LabelTag                            = "picchu.medium.engineering/tag"
	LabelCluster                        = "picchu.medium.engineering/cluster"
	LabelFleet                          = "picchu.medium.engineering/fleet"
	LabelRevision                       = "picchu.medium.engineering/revision"
	LabelTarget                         = "picchu.medium.engineering/target"
	LabelOwnerName                      = "picchu.medium.engineering/ownerName"
	LabelOwnerType                      = "picchu.medium.engineering/ownerType"
	LabelFleetPrefix                    = "fleet.picchu.medium.engineering/"
	FinalizerReleaseManager             = "picchu.medium.engineering/releasemanager"
	FinalizerCluster                    = "picchu.medium.engineering/cluster"
	OwnerReleaseManager                 = "releasemanager"
	AnnotationGitCommitterTimestamp     = "git-scm.com/committer-timestamp"
	AnnotationRevisionCreationTimestamp = "revisionCreationTimestamp"
	AnnotationIAMRole                   = "iam.amazonaws.com/role"
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
}

type ReleaseInfo struct {
	Eligible bool     `json:"eligible,omitempty"`
	Max      uint32   `json:"max,omitempty"`
	Rate     RateInfo `json:"rate,omitempty"`
	Schedule string   `json:"schedule,omitempty"`
	TTL      int64    `json:"ttl,omitempty"`
	GcBuffer int      `json:"gc_buffer,omitempty"`
}

type RateInfo struct {
	Increment    uint32 `json:"increment,omitempty"`
	DelaySeconds *int64 `json:"delay,omitempty"`
}

// TODO(lyra): PodTemplate
type AWSInfo struct {
	IAM IAMInfo `json:"iam,omitempty"`
}

type IAMInfo struct {
	RoleARN string `json:"role_arn,omitempty"`
}
