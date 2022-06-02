package v1alpha1

import (
	"time"

	istio "istio.io/api/networking/v1alpha3"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// see custom_deepcopy.go
// +k8s:deepcopy-gen=false
type Istio struct {
	TrafficPolicy *istio.TrafficPolicy `json:"trafficPolicy,omitempty"`
	Sidecar       *IstioSidecar        `json:"sidecar,omitempty"`
}

type IstioSidecar struct {
	EgressHosts []string `json:"egressHosts,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Revision is the Schema for the revisions API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:resource:categories=all;picchu
type Revision struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RevisionSpec   `json:"spec,omitempty"`
	Status RevisionStatus `json:"status,omitempty"`
}

func (r *Revision) Fail() {
	if !r.Spec.Failed {
		r.Spec.Failed = true
		t := time.Now()
		if r.Annotations == nil {
			r.Annotations = map[string]string{
				AnnotationFailedAt: t.Format(time.RFC3339),
			}
		} else {
			r.Annotations[AnnotationFailedAt] = t.Format(time.RFC3339)
		}
	}
}

func (r *Revision) SinceFailed() time.Duration {
	ft, ok := r.Annotations[AnnotationFailedAt]
	if !ok {
		return time.Duration(0)
	}
	t, err := time.Parse(time.RFC3339, ft)
	if err != nil {
		return time.Duration(0)
	}
	return time.Since(t)
}

func (r *Revision) Failed() bool {
	return r.Annotations != nil && r.Annotations[AnnotationFailedAt] != ""
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RevisionList contains a list of Revision
type RevisionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Revision `json:"items"`
}

// RevisionSpec defines the desired state of Revision
type RevisionSpec struct {
	App                RevisionApp      `json:"app"`
	Targets            []RevisionTarget `json:"targets"`
	Failed             bool             `json:"failed"`
	IgnoreSLOs         bool             `json:"ignoreSLOs,omitempty"`
	CanaryWithSLIRules bool             `json:"canaryWithSLIRules,omitempty"`
	Sentry             SentryInfo       `json:"sentry,omitempty"`
	TagRoutingHeader   string           `json:"tagRoutingHeader,omitempty"`
	DisableMirroring   bool             `json:"disableMirroring,omitempty"`
}

type RevisionApp struct {
	Name  string `json:"name"`
	Ref   string `json:"ref"`
	Tag   string `json:"tag"`
	Image string `json:"image"`
}

type RevisionTarget struct {
	Name                        string                      `json:"name"`
	Fleet                       string                      `json:"fleet"`
	Scale                       ScaleInfo                   `json:"scale"`
	Release                     ReleaseInfo                 `json:"release,omitempty"`
	ServiceMonitors             []*ServiceMonitor           `json:"serviceMonitors,omitempty"`
	ServiceLevelObjectives      []*ServiceLevelObjective    `json:"serviceLevelObjectives,omitempty"`
	ServiceLevelObjectiveLabels ServiceLevelObjectiveLabels `json:"serviceLevelObjectiveLabels,omitempty"`
	AcceptanceTarget            bool                        `json:"acceptanceTarget,omitempty"`
	ConfigSelector              *metav1.LabelSelector       `json:"configSelector,omitempty"`
	AWS                         AWSInfo                     `json:"aws,omitempty"`
	AlertRules                  []monitoringv1.Rule         `json:"alertRules,omitempty"`
	Sidecars                    []corev1.Container          `json:"sidecars,omitempty"`
	VolumeMounts                []corev1.VolumeMount        `json:"volumeMounts,omitempty"`
	Volumes                     []corev1.Volume             `json:"volumes,omitempty"`

	PodAnnotations     map[string]string `json:"podAnnotations,omitempty"`
	ServiceAccountName string            `json:"serviceAccountName,omitempty"`

	Resources      corev1.ResourceRequirements `json:"resources,omitempty"`
	LivenessProbe  *corev1.Probe               `json:"livenessProbe,omitempty"`
	ReadinessProbe *corev1.Probe               `json:"readinessProbe,omitempty"`
	Lifecycle      *corev1.Lifecycle           `json:"lifecycle,omitempty"`

	Affinity    *corev1.Affinity    `json:"affinity,omitempty"`
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	ExternalTest        ExternalTest      `json:"externalTest,omitempty"`
	Canary              Canary            `json:"canary,omitempty"`
	DefaultIngressPorts map[string]string `json:"defaultIngressPorts,omitempty"`
	Ports               []PortInfo        `json:"ports,omitempty"`
	Env                 []corev1.EnvVar   `json:"env,omitempty"`

	Istio *Istio `json:"istio,omitempty"`
}

type ExternalTest struct {
	Enabled     bool             `json:"enabled"`
	Started     bool             `json:"started"`
	Completed   bool             `json:"completed"`
	Succeeded   bool             `json:"succeeded,omitempty"`
	Timeout     *metav1.Duration `json:"timeout,omitempty"`
	LastUpdated *metav1.Time     `json:"lastUpdated,omitempty"`
}

type Canary struct {
	Percent uint32 `json:"percent"`
	TTL     int64  `json:"ttl"`
}

type ServiceLevelObjective struct {
	Name                        string                      `json:"name,omitempty"`
	Description                 string                      `json:"description,omitempty"`
	Objective                   float64                     `json:"objective"`
	SLI                         SLI                         `json:"serviceLevelIndicator,omitempty"`
	Labels                      map[string]string           `json:"labels,omitempty"`
	ServiceLevelObjectiveLabels ServiceLevelObjectiveLabels `json:"serviceLevelObjectiveLabels,omitempty"`
	Alerting                    Alerting                    `json:"alerting"`
}

type Alerting struct {
	Name        string            `json:"name,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	PageAlert   Alert             `json:"pageAlert,omitempty"`
	TicketAlert Alert             `json:"ticketAlert,omitempty"`
}

type Alert struct {
	Disable     bool              `json:"disable,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type ServiceLevelObjectiveLabels struct {
	AlertLabels        map[string]string `json:"alertLabels,omitempty"`
	RuleLabels         map[string]string `json:"ruleLabels,omitempty"`
	ServiceLevelLabels map[string]string `json:"serviceLevelLabels,omitempty"`
}

type SLI struct {
	Canary     SLICanaryConfig `json:"canary,omitempty"`
	TagKey     string          `json:"tagKey,omitempty"`
	AlertAfter string          `json:"alertAfter,omitempty"`
	// TotalQuery string          `json:"totalQuery,omitempty"` Events ->
	// ErrorQuery string          `json:"errorQuery,omitempty"` Events ->
	Raw    *SLIRaw    `json:"raw,omitempty"`
	Events *SLIEvents `json:"events,omitempty"`
	Plugin *SLIPlugin `json:"plugin,omitempty"`
}

type SLIRaw struct {
	ErrorRatioQuery string `json:"errorRatioQuery"`
}

type SLIEvents struct {
	ErrorQuery string `json:"errorQuery,omitempty"`
	TotalQuery string `json:"totalQuery,omitempty"`
}

type SLIPlugin struct {
	ID      string            `json:"id"`
	Options map[string]string `json:"options,omitempty"`
}

type SLICanaryConfig struct {
	Enabled                bool    `json:"enabled"`
	AllowancePercentString string  `json:"allowancePercentString,omitempty"`
	AllowancePercent       float64 `json:"allowancePercent,omitempty"`
	FailAfter              string  `json:"failAfter,omitempty"`
}

type ServiceMonitor struct {
	Name string `json:"name"`
	// if true, and the Spec.Endpoints.MetricRelabelConfigs does not specify a regex, will replace the regex with a list of SLO metric names
	SLORegex    bool                            `json:"sloRegex"`
	Annotations map[string]string               `json:"annotations,omitempty"`
	Labels      map[string]string               `json:"labels,omitempty"`
	Spec        monitoringv1.ServiceMonitorSpec `json:"spec,omitempty"`
}

type RevisionStatus struct {
	Sentry  SentryInfo             `json:"sentry"`
	Targets []RevisionTargetStatus `json:"targets"`
}

func (r *RevisionStatus) AddTarget(ts RevisionTargetStatus) {
	r.Targets = append(r.Targets, ts)
}

func (r *RevisionStatus) GetTarget(name string) *RevisionTargetStatus {
	for _, ts := range r.Targets {
		if ts.Name == name {
			return &ts
		}
	}
	return nil
}

// RevisionStatus defines the observed state of Revision
type RevisionTargetStatus struct {
	Name    string                `json:"name"`
	Scale   RevisionScaleStatus   `json:"scale"`
	Release RevisionReleaseStatus `json:"release"`
	State   string                `json:"state,omitempty"`
}

type RevisionScaleStatus struct {
	Current uint32 `json:"current"`
	Desired uint32 `json:"desired"`
	Peak    uint32 `json:"peak"`
}

type RevisionReleaseStatus struct {
	CurrentPercent uint32 `json:"currentPercent"`
	PeakPercent    uint32 `json:"peakPercent"`
}

func (r *RevisionTargetStatus) AddReleaseManagerStatus(status ReleaseManagerRevisionStatus) {
	r.Release.CurrentPercent = status.CurrentPercent
	r.Release.PeakPercent = status.PeakPercent
	r.Scale.Current = uint32(status.Scale.Current)
	r.Scale.Desired = uint32(status.Scale.Desired)
	r.Scale.Peak = uint32(status.Scale.Peak)
	r.State = status.State.Current
}

func (r *Revision) GitTimestamp() time.Time {
	gt, ok := r.Annotations[AnnotationGitCommitterTimestamp]
	if !ok {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, gt)
	if err != nil {
		return time.Time{}
	}
	return t
}

func (r *Revision) HasTarget(name string) bool {
	for _, target := range r.Spec.Targets {
		if target.Name == name {
			return true
		}
	}
	return false
}

func (r *RevisionTarget) IsExternalTestPending() bool {
	return r.ExternalTest.Enabled && !r.ExternalTest.Completed
}

func (r *RevisionTarget) IsExternalTestSuccessful() bool {
	t := &r.ExternalTest
	return t.Enabled && t.Completed && t.Succeeded
}

func (r *RevisionTarget) IsCanaryPending(startTime *metav1.Time) bool {
	if r.Canary.Percent == 0 || r.Canary.TTL == 0 {
		return false
	}
	if startTime == nil {
		return true
	}
	return startTime.Time.Add(time.Duration(r.Canary.TTL) * time.Second).After(time.Now())
}

func init() {
	SchemeBuilder.Register(&Revision{}, &RevisionList{})
}
