package v1alpha1

import (
	istio "istio.io/api/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HTTPPortFault allows injecting faults into apps by port number
// +k8s:deepcopy-gen=false
type HTTPPortFault struct {
	PortSelector *istio.PortSelector       `json:"portSelector,omitempty"`
	HTTPFault    *istio.HTTPFaultInjection `json:"fault,omitempty"`
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
