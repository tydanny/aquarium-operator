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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AquariumSpec defines the desired state of Aquarium
type AquariumSpec struct {
	// +kubebuilder:validation:Minimum=1
	NumTanks int32 `json:"num_tanks,omitempty"`
	// +kubebuilder:default=pier39
	Location string `json:"location,omitempty"`
	Image    string `json:"image,omitempty"`
}

// AquariumStatus defines the observed state of Aquarium
type AquariumStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	NumTanksReady int32      `json:"num_tanks_ready,omitempty"`
	FishHealth    FishHealth `json:"fish_health,omitempty"`
}

type FishHealth string

const (
	Healthy   FishHealth = "Healthy"
	Unhealthy FishHealth = "Unhealthy"
	Unknown   FishHealth = "Unknown"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:validation:Required
// +kubebuilder:printcolumn:name="Tanks",type="integer",JSONPath=".status.num_tanks",priority=0
// +kubebuilder:printcolumn:name="Fish Health",type="string",JSONPath=".status.fish_health",priority=0
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0

// Aquarium is the Schema for the aquaria API
type Aquarium struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AquariumSpec   `json:"spec,omitempty"`
	Status AquariumStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AquariumList contains a list of Aquarium
type AquariumList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Aquarium `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Aquarium{}, &AquariumList{})
}
