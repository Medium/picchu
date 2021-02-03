package v1alpha1

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultReleaseMax                    = 100
	defaultReleaseSchedule               = InhumaneSchedule
	defaultReleaseGcTTLSeconds           = int64(5 * 24 * 60 * 60)
	defaultScaleDefault                  = int32(1)
	defaultScaleMax                      = int32(1)
	defaultRequestsRateMetric            = "istio_requests_rate"
	defaultReleaseScalingStrategy        = ScalingStrategyGeometric
	defaultReleaseGeometricScalingStart  = 10
	defaultReleaseGeometricScalingFactor = 2
	defaultReleaseLinearScalingIncrement = 5

	defaultPortIngressPort   = int32(443)
	defaultPortContainerPort = int32(80)
	defaultPortPort          = int32(80)
	defaultPortProtocol      = corev1.ProtocolTCP
	defaultPortMode          = PortPrivate

	defaultExternalTestTimeout = time.Duration(2) * time.Hour
)

var (
	defaultReleaseGeometricScalingDelay = &metav1.Duration{Duration: time.Duration(10) * time.Second}
	defaultReleaseLinearScalingDelay    = &metav1.Duration{Duration: time.Duration(10) * time.Second}
)

func SetDefaults_ClusterSpec(spec *ClusterSpec) {
	if spec.ScalingFactor == nil {
		one := float64(1.0)
		spec.ScalingFactor = &one
	}
}

func SetDefaults_RevisionSpec(spec *RevisionSpec) {
	for i := range spec.Targets {
		SetReleaseDefaults(&spec.Targets[i].Release)
		SetScaleDefaults(&spec.Targets[i].Scale)
		for j := range spec.Targets[i].Ports {
			SetPortDefaults(&spec.Targets[i].Ports[j])
		}
		SetExternalTestDefaults(&spec.Targets[i].ExternalTest)
	}
}

func SetExternalTestDefaults(externalTest *ExternalTest) {
	if externalTest.Timeout == nil {
		externalTest.Timeout = &metav1.Duration{Duration: defaultExternalTestTimeout}
	}
}

func SetReleaseDefaults(release *ReleaseInfo) {
	if release.Max == 0 {
		release.Max = defaultReleaseMax
	}
	if release.Schedule == "" {
		release.Schedule = defaultReleaseSchedule
	}
	if release.ScalingStrategy == "" {
		release.ScalingStrategy = defaultReleaseScalingStrategy
	}
	if release.TTL == 0 {
		release.TTL = defaultReleaseGcTTLSeconds
	}
	if release.GeometricScaling.Factor == 0 {
		release.GeometricScaling.Factor = defaultReleaseGeometricScalingFactor
	}
	if release.GeometricScaling.Start == 0 {
		release.GeometricScaling.Start = defaultReleaseGeometricScalingStart
	}
	if release.GeometricScaling.Delay == nil {
		release.GeometricScaling.Delay = defaultReleaseGeometricScalingDelay.DeepCopy()
	}
	if release.LinearScaling.Increment == 0 {
		release.LinearScaling.Increment = defaultReleaseLinearScalingIncrement
	}
	if release.LinearScaling.Delay == nil {
		release.LinearScaling.Delay = defaultReleaseLinearScalingDelay.DeepCopy()
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
	if scale.RequestsRateMetric == "" {
		scale.RequestsRateMetric = defaultRequestsRateMetric
	}
}
