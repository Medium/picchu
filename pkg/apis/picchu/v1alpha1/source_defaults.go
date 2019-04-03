package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	defaultReleaseMax              = 100
	defaultReleaseSchedule         = HumaneSchedule
	defaultReleaseRateIncrement    = 5
	defaultReleaseRateDelaySeconds = int64(10)
	defaultReleaseGcTTLSeconds     = int64(5 * 24 * 60 * 60)
	defaultGcBuffer                = 5

	defaultPortIngressPort   = int32(443)
	defaultPortContainerPort = int32(80)
	defaultPortPort          = int32(80)
	defaultPortProtocol      = corev1.ProtocolTCP
	defaultPortMode          = PortPrivate
)

func SetDefaults_RevisionSpec(spec *RevisionSpec) {
	for _, target := range spec.Targets {
		SetReleaseDefaults(&target.Release)
		if target.Scale.Resources.CPU == (resource.Quantity{}) {
			target.Scale.Resources.CPU = resource.MustParse("500m")
		}
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
	if spec.Scale.Resources.CPU == (resource.Quantity{}) {
		spec.Scale.Resources.CPU = resource.MustParse("500m")
	}
	if spec.Scale.Min == nil {
		var one int32 = 1
		spec.Scale.Min = &one
	}
	if spec.Scale.Max == 0 {
		spec.Scale.Max = 10
	}
	if spec.Scale.Default == 0 {
		spec.Scale.Default = *spec.Scale.Min
	}
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
	if release.Rate.DelaySeconds == nil {
		delay := defaultReleaseRateDelaySeconds
		release.Rate.DelaySeconds = &delay
	}
	if release.TTL == 0 {
		release.TTL = defaultReleaseGcTTLSeconds
	}
	if release.GcBuffer == 0 {
		release.GcBuffer = defaultGcBuffer
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
