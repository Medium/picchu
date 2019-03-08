package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"time"
)

const (
	defaultReleaseMax           = 100
	defaultReleaseSchedule      = HumaneSchedule
	defaultReleaseRateIncrement = 5
	defaultReleaseRateDelay     = time.Duration(10) * time.Second

	defaultPortIngressPort   = int32(443)
	defaultPortContainerPort = int32(80)
	defaultPortPort          = int32(80)
	defaultPortProtocol      = corev1.ProtocolTCP
	defaultPortMode          = PortPrivate
)

func SetDefaults_RevisionSpec(spec *RevisionSpec) {
	for _, target := range spec.Targets {
		SetReleaseDefaults(&target.Release)
	}
	ports := []PortInfo{}
	for _, port := range spec.Ports {
		SetPortDefaults(&port)
		ports = append(ports, port)
	}
	spec.Ports = ports
}

func SetDefaults_IncarnationSpec(spec *IncarnationSpec) {
	SetReleaseDefaults(&spec.Release)
	ports := []PortInfo{}
	for _, port := range spec.Ports {
		SetPortDefaults(&port)
		ports = append(ports, port)
	}
	spec.Ports = ports
}

func SetReleaseDefaults(release *ReleaseInfo) {
	if release.Max == 0 {
		release.Max = defaultReleaseMax
	}
	if release.Schedule == "" {
		release.Schedule = HumaneSchedule
	}
	if release.Rate.Increment == 0 {
		release.Rate.Increment = defaultReleaseRateIncrement
	}
	if release.Rate.Delay == 0 {
		release.Rate.Delay = defaultReleaseRateDelay
	}
}

func SetPortDefaults(port *PortInfo) {
	if port.IngressPort == 0 {
		port.IngressPort = defaultPortIngressPort
	}
	if port.ContainerPort == 0 {
		port.ContainerPort = defaultPortContainerPort
	}
	if port.Port == 0 {
		port.Port = defaultPortPort
	}
	if port.Protocol == "" {
		port.Protocol = defaultPortProtocol
	}
	if port.Mode == "" {
		port.Mode = defaultPortMode
	}
}
