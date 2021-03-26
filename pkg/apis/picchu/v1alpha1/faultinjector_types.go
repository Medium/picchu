package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HTTPPortFault allows injecting faults into apps by port number
type HTTPPortFault struct {
	PortSelector *PortSelector       `json:"portSelector,omitempty"`
	HTTPFault    *HTTPFaultInjection `json:"fault,omitempty"`
}

type HTTPFaultInjection struct {
	Delay *HTTPFaultInjection_Delay `json:"delay,omitempty"`
	Abort *HTTPFaultInjection_Abort `json:"abort,omitempty"`
}

type HTTPFaultInjection_Delay struct {
	Percent       int32                               `json:"percent,omitempty"`
	HttpDelayType *HTTPFaultInjection_Delay_DelayType `json:"http_delay_type,omitempty"`
	Percentage    *Percent                            `json:"percentage,omitempty"`
}

type Percent struct {
	Value resource.Quantity `json:"value,omitempty"`
}

type HTTPFaultInjection_Abort struct {
	ErrorType  *HTTPFaultInjection_Abort_ErrorType `json:"error_type,omitempty"`
	Percentage *Percent                            `json:"percentage,omitempty"`
}

type HTTPFaultInjection_Abort_ErrorType struct {
	HttpStatus int32  `json:"http_status,omitempty"`
	GrpcStatus string `json:"grpc_status,omitempty"`
	Http2Error string `json:"http2_error,omitempty"`
}

type HTTPFaultInjection_Delay_DelayType struct {
	FixedDelay       *Duration `json:"fixed_delay,omitempty"`
	ExponentialDelay *Duration `json:"exponential_delay,omitempty"`
}

// FaultInjectorSpec defines the desired state of FaultInjector
type FaultInjectorSpec struct {
	HTTPPortFaults []HTTPPortFault `json:"httpPortFaults"`
}

// FaultInjectorStatus defines the observed state of FaultInjector
type FaultInjectorStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FaultInjector is the Schema for the faultinjectors API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=faultinjectors,scope=Namespaced
type FaultInjector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FaultInjectorSpec   `json:"spec,omitempty"`
	Status FaultInjectorStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FaultInjectorList contains a list of FaultInjector
type FaultInjectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FaultInjector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FaultInjector{}, &FaultInjectorList{})
}
