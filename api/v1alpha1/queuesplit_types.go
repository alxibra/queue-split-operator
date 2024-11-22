/*
Copyright 2024.

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

// QueueSplitSpec defines the desired state of QueueSplit.
type QueueSplitSpec struct {
	// Replicas defines the number of replicas for the QueueSplit.
	// +kubebuilder:validation:Minimum=1
	Replicas int `json:"replicas"`

	// SecretName specifies the name of the Kubernetes Secret containing configuration.
	// +kubebuilder:validation:Optional
	SecretName string `json:"secretName,omitempty"`

	// QueueName specifies the name of the primary queue to split.
	// +kubebuilder:validation:Required
	QueueName string `json:"queueName"`

	// PrefetchCount defines the number of messages to prefetch. Defaults to 0.
	// +kubebuilder:default=0
	// +kubebuilder:validation:Minimum=0
	PrefetchCount int `json:"prefetchCount"`

	// Destinations lists the destinations and their corresponding weights for splitting traffic.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=2
	// +kubebuilder:validation:MaxItems=2
	Destinations []Destination `json:"destinations"`
}

// Destination defines the structure for each destination in the queue split.
type Destination struct {
	// Name represents the name of the destination queue or consumer.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Weight determines the percentage of traffic directed to this destination.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	Weight int `json:"weight"`
}

// QueueSplitStatus defines the observed state of QueueSplit.
type QueueSplitStatus struct {
	// Example status field - You can replace this with actual fields to track the status.
	// Example: TotalMessagesProcessed int `json:"totalMessagesProcessed"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// QueueSplit is the Schema for the queuesplits API.
// +kubebuilder:resource:shortName=qs
// +kubebuilder:metadata:labels:app.kubernetes.io/name="queuesplit"
// +kubebuilder:metadata:labels:app.kubernetes.io/part-of="messaging-system"
// +kubebuilder:metadata:labels:app.kubernetes.io/managed-by="queuesplit-operator"
// +kubebuilder:metadata:labels:app.kubernetes.io/version="v1alpha1"
type QueueSplit struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   QueueSplitSpec   `json:"spec,omitempty"`
	Status QueueSplitStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// QueueSplitList contains a list of QueueSplit.
type QueueSplitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []QueueSplit `json:"items"`
}

func init() {
	SchemeBuilder.Register(&QueueSplit{}, &QueueSplitList{})
}
