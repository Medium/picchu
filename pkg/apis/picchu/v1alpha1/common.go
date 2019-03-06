package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"time"
)

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

const (
	HumaneSchedule = "humane"

	RateIncrementDefault int32  = 10
	RateDelayDefault       = time.Duration(6) * time.Second
	ReleaseMaxDefault int32     = 100
	ReleaseScheduleDefault = HumaneSchedule
)

type PortInfo struct {
	Name          string          `json:"name,omitempty"`
	Hosts         []string        `json:"hosts,omitempty"`
	Port          int32           `json:"port,omitempty"`
	ContainerPort int32           `json:"containerPort,omitempty"`
	Protocol      corev1.Protocol `json:"protocol,omitempty"`
	Mode          PortMode        `json:"mode"`
}

type ScaleInfo struct {
	Min       int32        `json:"min,omitempty"`
	Default   int32        `json:"default,omitempty"`
	Max       int32        `json:"max,omitempty"`
	Resources ResourceInfo `json:"resources,omitempty"`
}

type ResourceInfo struct {
	CPU string `json:"cpu,omitempty"`
}

type ReleaseInfo struct {
	Eligible bool     `json:"eligible,omitempty"`
	Max      int32    `json:"max,omitempty"`
	Rate     RateInfo `json:"rate,omitempty"`
	Schedule string   `json:"schedule,omitempty"`
}

func (r *ReleaseInfo) GetMax() int32 {
	if r.Max == 0 {
		return ReleaseMaxDefault
	}
	return r.Max
}

type RateInfo struct {
	Increment    int32 `json:"increment,omitempty"`
	Delay meta.Duration `json:"delay,omitempty"`
}

func (r *RateInfo) GetIncrement() int32 {
	if r.Increment == 0 {
		return RateIncrementDefault
	}
	return r.Increment
}

func (r *RateInfo) GetDelay() time.Duration {
	if r.DelaySeconds == 0 {
		return RateDelayDefault
	}
	return time.Duration(r.DelaySeconds) * time.Second
}
