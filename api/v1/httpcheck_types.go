/*
Copyright 2023.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HttpCheckSpec defines the desired state of HttpCheck
type HttpCheckSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Name                      string `json:"name,omitempty"`
	Uri                       string `json:"uri,omitempty"`
	Num_Retries               int    `json:"num_retries,omitempty"`
	Response_Status_Code      int    `json:"response_status_code,omitempty"`
	Check_Interval_In_Seconds int    `json:"check_interval_in_seconds,omitempty"`
}

// HttpCheckStatus defines the observed state of HttpCheck
type HttpCheckStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ObservedStatus string `json:"observedStatus,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// HttpCheck is the Schema for the httpchecks API
type HttpCheck struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HttpCheckSpec   `json:"spec,omitempty"`
	Status HttpCheckStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HttpCheckList contains a list of HttpCheck
type HttpCheckList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HttpCheck `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HttpCheck{}, &HttpCheckList{})
}
