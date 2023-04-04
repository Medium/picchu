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
	istio "istio.io/api/networking/v1alpha3"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HTTPPortFault allows injecting faults into apps by port number
// +k8s:deepcopy-gen=true
type HTTPPortFault struct {
	PortSelector *istio.PortSelector       `json:"portSelector,omitempty"`
	HTTPFault    *istio.HTTPFaultInjection `json:"fault,omitempty"`
}

// FaultInjectorSpec defines the desired state of FaultInjector
type FaultInjectorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	HTTPPortFaults []HTTPPortFault `json:"httpPortFaults"`
}

// FaultInjectorStatus defines the observed state of FaultInjector
type FaultInjectorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// FaultInjector is the Schema for the faultinjectors API
type FaultInjector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FaultInjectorSpec   `json:"spec,omitempty"`
	Status FaultInjectorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FaultInjectorList contains a list of FaultInjector
type FaultInjectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FaultInjector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FaultInjector{}, &FaultInjectorList{})
}
