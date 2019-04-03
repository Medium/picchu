package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	defaultReleaseMax              = 100
	defaultReleaseSchedule         = HumaneSchedule
	defaultReleaseRateIncrement    = 5
	defaultReleaseRateDelaySeconds = int64(10)
	defaultReleaseGcTTLSeconds     = int64(5 * 24 * 60 * 60)
	defaultGcBuffer                = 5
	defaultScaleDefault            = int32(1)
	defaultScaleMax                = int32(1)

	defaultPortIngressPort   = int32(443)
	defaultPortContainerPort = int32(80)
	defaultPortPort          = int32(80)
	defaultPortProtocol      = corev1.ProtocolTCP
	defaultPortMode          = PortPrivate
)

func SetDefaults_RevisionSpec(spec *RevisionSpec) {
	for i, _ := range spec.Targets {
		SetReleaseDefaults(&spec.Targets[i].Release)
		SetScaleDefaults(&spec.Targets[i].Scale)
	}
	for i, _ := range spec.Ports {
		SetPortDefaults(&spec.Ports[i])
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

func SetScaleDefaults(scale *ScaleInfo) {
	if scale.Default == 0 {
		scale.Default = defaultScaleDefault
	}
	if scale.Max == 0 {
		scale.Max = defaultScaleMax
	}
}
