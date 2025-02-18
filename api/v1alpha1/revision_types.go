/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"time"

	ddogv1alpha1 "github.com/DataDog/datadog-operator/api/datadoghq/v1alpha1"

	istio "istio.io/api/networking/v1alpha3"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:pruning:PreserveUnknownFields
type Istio struct {
	TrafficPolicy *istio.TrafficPolicy `json:"trafficPolicy,omitempty"`
	Sidecar       *IstioSidecar        `json:"sidecar,omitempty"`
}

type IstioSidecar struct {
	EgressHosts []string `json:"egressHosts,omitempty"`
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

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
	Name                        string                        `json:"name"`
	Fleet                       string                        `json:"fleet"`
	Scale                       ScaleInfo                     `json:"scale"`
	Release                     ReleaseInfo                   `json:"release,omitempty"`
	ServiceMonitors             []*ServiceMonitor             `json:"serviceMonitors,omitempty"`
	SlothServiceLevelObjectives []*SlothServiceLevelObjective `json:"serviceLevelObjectives,omitempty"`
	DatadogSLOs                 []*DatadogSLO                 `json:"datadogServiceLevelObjectives,omitempty"`
	ServiceLevelObjectiveLabels ServiceLevelObjectiveLabels   `json:"serviceLevelObjectiveLabels,omitempty"`
	AcceptanceTarget            bool                          `json:"acceptanceTarget,omitempty"`
	ConfigSelector              *metav1.LabelSelector         `json:"configSelector,omitempty"`
	AWS                         AWSInfo                       `json:"aws,omitempty"`
	AlertRules                  []monitoringv1.Rule           `json:"alertRules,omitempty"`
	Sidecars                    []corev1.Container            `json:"sidecars,omitempty"`
	VolumeMounts                []corev1.VolumeMount          `json:"volumeMounts,omitempty"`
	Volumes                     []corev1.Volume               `json:"volumes,omitempty"`

	PodAnnotations     map[string]string `json:"podAnnotations,omitempty"`
	ServiceAccountName string            `json:"serviceAccountName,omitempty"`

	Resources      corev1.ResourceRequirements `json:"resources,omitempty"`
	LivenessProbe  *corev1.Probe               `json:"livenessProbe,omitempty"`
	ReadinessProbe *corev1.Probe               `json:"readinessProbe,omitempty"`
	Lifecycle      *corev1.Lifecycle           `json:"lifecycle,omitempty"`

	Affinity          *corev1.Affinity    `json:"affinity,omitempty"`
	PriorityClassName string              `json:"priorityClassName,omitempty"`
	Tolerations       []corev1.Toleration `json:"tolerations,omitempty"`

	ExternalTest        ExternalTest      `json:"externalTest,omitempty"`
	Canary              Canary            `json:"canary,omitempty"`
	DefaultIngressPorts map[string]string `json:"defaultIngressPorts,omitempty"`
	Ports               []PortInfo        `json:"ports,omitempty"`
	Env                 []corev1.EnvVar   `json:"env,omitempty"`

	Istio               *Istio                        `json:"istio,omitempty"`
	PodDisruptionBudget *policyv1.PodDisruptionBudget `json:"podDisruptionBudget,omitempty"`
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

type SlothServiceLevelObjective struct {
	Name                        string                      `json:"name,omitempty"`
	Description                 string                      `json:"description,omitempty"`
	Enabled                     bool                        `json:"enabled"`
	Objective                   string                      `json:"objectivePercentString,omitempty"`
	ServiceLevelIndicator       ServiceLevelIndicator       `json:"serviceLevelIndicator,omitempty"`
	ServiceLevelObjectiveLabels ServiceLevelObjectiveLabels `json:"serviceLevelObjectiveLabels,omitempty"`
	Alerting                    Alerting                    `json:"alerting,omitempty"`
}

type ServiceLevelObjectiveLabels struct {
	AlertLabels        map[string]string `json:"alertLabels,omitempty"`
	RuleLabels         map[string]string `json:"ruleLabels,omitempty"`
	ServiceLevelLabels map[string]string `json:"serviceLevelLabels,omitempty"`
}

type ServiceLevelIndicator struct {
	Canary     SLICanaryConfig `json:"canary,omitempty"`
	TagKey     string          `json:"tagKey,omitempty"`
	AlertAfter string          `json:"alertAfter,omitempty"`
	TotalQuery string          `json:"totalQuery,omitempty"`
	ErrorQuery string          `json:"errorQuery,omitempty"`
}

type SLICanaryConfig struct {
	Enabled                bool    `json:"enabled"`
	AllowancePercentString string  `json:"allowancePercentString,omitempty"`
	AllowancePercent       float64 `json:"allowancePercent,omitempty"`
	FailAfter              string  `json:"failAfter,omitempty"`
}

type DatadogMonitor struct {
	// if enabled, use these values
	Enabled bool `json:"enabled,omitempty"`
	// defaulted - <app>-<slo name>-<target>-datadogmonitor
	Name string `json:"name,omitempty"`
	// defaulted
	Message string `json:"message,omitempty"`
	// defaulted
	Query string `json:"query,omitempty"`
	// both dev and defaulted - target name
	Tags []string `json:"tags,omitempty"`
	// defaulted - "slo alert"
	Type    ddogv1alpha1.DatadogMonitorType `json:"type,omitempty"`
	Options DatadogMonitorOptions           `json:"options,omitempty"`
}

// double check this matches the one in kbfd
type DatadogMonitorOptions struct {
	// A Boolean indicating whether to send a log sample when the log monitor triggers.
	// defaulted - true
	EnableLogsSample *bool `json:"enableLogsSample,omitempty"`

	// A message to include with a re-notification.
	// defaulted - "renotifying for <slo>" or something similar
	EscalationMessage *string `json:"escalationMessage,omitempty"`

	// Time (in seconds) to delay evaluation, as a non-negative integer. For example, if the value is set to 300 (5min),
	// the timeframe is set to last_5m and the time is 7:00, the monitor evaluates data from 6:50 to 6:55.
	// This is useful for AWS CloudWatch and other backfilled metrics to ensure the monitor always has data during evaluation.
	// defaulted - 5m for now
	EvaluationDelay *int64 `json:"evaluationDelay,omitempty"`

	// A Boolean indicating whether notifications from this monitor automatically inserts its triggering tags into the title.
	// default - true
	IncludeTags *bool `json:"includeTags,omitempty"`

	// The number of minutes before a monitor notifies after data stops reporting. Datadog recommends at least 2x the
	// monitor timeframe for metric alerts or 2 minutes for service checks. If omitted, 2x the evaluation timeframe
	// is used for metric alerts, and 24 hours is used for service checks.
	// default - 3m for now
	NoDataTimeframe *int64 `json:"noDataTimeframe,omitempty"`

	// An enum that toggles the display of additional content sent in the monitor notification.
	// defaulted - `show_all`
	NotificationPresetName string `json:"notificationPresetName,omitempty"`

	// A Boolean indicating whether this monitor notifies when data stops reporting.
	// defaulted - true
	NotifyNoData *bool `json:"notifyNoData,omitempty"`

	// The types of statuses for which re-notification messages should be sent. Valid values are alert, warn, no data.
	// +listType=set
	// defaulted - "alert" and "no data"
	RenotifyStatuses []string `json:"renotifyStatuses,omitempty"`

	// The number of minutes after the last notification before a monitor re-notifies on the current status.
	// It only re-notifies if it’s not resolved.
	// defaulted - 5min
	RenotifyInterval *int64 `json:"renotifyInterval,omitempty"`

	// The number of times re-notification messages should be sent on the current status at the provided re-notification interval.
	// defaulted - 2 times
	RenotifyOccurrences *int64 `json:"renotifyOccurrences,omitempty"`

	// A Boolean indicating whether this monitor needs a full window of data before it’s evaluated. We highly
	// recommend you set this to false for sparse metrics, otherwise some evaluations are skipped. Default is false.
	// defaulted - true for now
	RequireFullWindow *bool `json:"requireFullWindow,omitempty"`

	// A struct of the different monitor threshold values.
	Thresholds *DatadogMonitorOptionsThresholds `json:"thresholds,omitempty"`

	// A struct of the alerting time window options.
	ThresholdWindows *DatadogMonitorOptionsThresholdWindows `json:"thresholdWindows,omitempty"`
}

// Values read from the monitor query
type DatadogMonitorOptionsThresholds struct {
	// The monitor CRITICAL threshold.
	Critical *string `json:"critical,omitempty"`
	// The monitor CRITICAL recovery threshold.
	CriticalRecovery *string `json:"criticalRecovery,omitempty"`
	// The monitor OK threshold.
	OK *string `json:"ok,omitempty"`
	// The monitor UNKNOWN threshold.
	Unknown *string `json:"unknown,omitempty"`
}

// defaulted - 5m
type DatadogMonitorOptionsThresholdWindows struct {
	// Describes how long an anomalous metric must be normal before the alert recovers.
	RecoveryWindow *string `json:"recoveryWindow,omitempty"`
	// Describes how long a metric must be anomalous before an alert triggers.
	TriggerWindow *string `json:"triggerWindow,omitempty"`
}

// ddog specific slos
type DatadogSLO struct {
	// defaulted
	Name            string          `json:"name,omitempty"`
	Description     string          `json:"description,omitempty"`
	Query           DatadogSLOQuery `json:"query,omitempty"`
	Tags            []string        `json:"tags,omitempty"`
	TargetThreshold string          `json:"targetThreshold,omitempty"`
	Timeframe       string          `json:"timeframe,omitempty"`
	// defaulted - metric
	Type string `json:"type,omitempty"`
	// not implemented yet
	Canary         DatadogSLOCanaryConfig `json:"canary,omitempty"`
	DatadogMonitor DatadogMonitor         `json:"datadogMonitor,omitempty"`
}

type DatadogSLOQuery struct {
	Numerator   string `json:"numerator"`
	Denominator string `json:"denominator"`
}

type DatadogSLOCanaryConfig struct {
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

type Alerting struct {
	Name        string            `json:"name,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	PageAlert   AlertMeta         `json:"pageAlert,omitempty"`
	TicketAlert AlertMeta         `json:"ticketAlert,omitempty"`
}

type AlertMeta struct {
	Disable     bool              `json:"disable,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// RevisionStatus defines the observed state of Revision
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

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// Revision is the Schema for the revisions API
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

// +kubebuilder:object:root=true

// RevisionList contains a list of Revision
type RevisionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Revision `json:"items"`
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
	// if canary values aren't set or no SLOs set up for service
	if r.Canary.Percent == 0 || r.Canary.TTL == 0 || len(r.SlothServiceLevelObjectives) == 0 {
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
