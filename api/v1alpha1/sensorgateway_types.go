/*
Copyright 2025.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SensorGatewaySpec defines the desired state of SensorGateway.
type SensorGatewaySpec struct {
	Image      string `json:"image,omitempty"`
	BrokerURL  string `json:"brokerURL,omitempty"`
	BrokerPort string `json:"brokerPort,omitempty"`
	Topic      string `json:"topic,omitempty"`
	SensorType string `json:"sensorType,omitempty"`
}

// SensorGatewayStatus defines the observed state of SensorGateway.
type SensorGatewayStatus struct {
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SensorGateway is the Schema for the sensorgateways API.
type SensorGateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SensorGatewaySpec   `json:"spec,omitempty"`
	Status SensorGatewayStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SensorGatewayList contains a list of SensorGateway.
type SensorGatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SensorGateway `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SensorGateway{}, &SensorGatewayList{})
}
