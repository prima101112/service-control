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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ApplicationSpec defines the desired state of Application.
// i am on purpose make it simple and easy to use where we will set sensible default values if not provided.
type ApplicationSpec struct {
	// Required: Container image
	Image string `json:"image"`

	// Required: Image tag
	Tag string `json:"tag"`

	// Optional: Host for ingress rule
	Host string `json:"host,omitempty"`

	// Optional: Number of replicas
	Replicas *int32 `json:"replicas,omitempty"`

	// Optional: Container port
	Port *int32 `json:"port,omitempty"`

	// Optional: Environment variables
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Optional: Resource requirements
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// Optional: Service configuration
	Service *ServiceConfig `json:"service,omitempty"`

	// Optional: Ingress configuration
	Ingress *IngressConfig `json:"ingress,omitempty"`

	HPA         *HPAConfig         `json:"hpa,omitempty"`
	HealthCheck *HealthCheckConfig `json:"healthCheck,omitempty"`
}

type ServiceConfig struct {
	// Service type
	Type corev1.ServiceType `json:"type,omitempty"`

	// Service port it should be 80
	Port *int32 `json:"port,omitempty"`

	// NodePort for NodePort service type
	NodePort *int32 `json:"nodePort,omitempty"`

	// Additional annotations
	Annotations map[string]string `json:"annotations,omitempty"`
}

type IngressConfig struct {
	// Host for ingress rule
	Host string `json:"host"`

	// Path (defaults to "/")
	Path string `json:"path,omitempty"`

	// TLS secret name (optional)
	TLSSecretName string `json:"tlsSecretName,omitempty"`

	// Ingress class name (optional)
	IngressClassName *string `json:"ingressClassName,omitempty"`

	// Additional annotations
	Annotations map[string]string `json:"annotations,omitempty"`
}

type HPAConfig struct {
	// Minimum replicas
	MinReplicas *int32 `json:"minReplicas,omitempty"`

	// Maximum replicas
	MaxReplicas int32 `json:"maxReplicas"`

	// Target CPU utilization percentage
	TargetCPUUtilizationPercentage *int32 `json:"targetCPUUtilizationPercentage,omitempty"`

	// Target memory utilization percentage
	TargetMemoryUtilizationPercentage *int32 `json:"targetMemoryUtilizationPercentage,omitempty"`
}

type HealthCheckConfig struct {
	// Health check path (defaults to "/health")
	Path string `json:"path,omitempty"`

	// Health check port (defaults to container port)
	Port *int32 `json:"port,omitempty"`

	// Initial delay seconds (defaults to 10)
	InitialDelaySeconds *int32 `json:"initialDelaySeconds,omitempty"`

	// Period seconds (defaults to 10)
	PeriodSeconds *int32 `json:"periodSeconds,omitempty"`
}

// TODO should add all resource thtat holds by the application
type ApplicationStatus struct {
	// Phase of the application (Pending, Running, Failed)
	Phase string `json:"phase,omitempty"`

	// Ready replicas
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// Total replicas
	Replicas int32 `json:"replicas,omitempty"`

	// Deployment status
	DeploymentReady bool `json:"deploymentReady,omitempty"`

	// Service status
	ServiceReady bool `json:"serviceReady,omitempty"`

	// Ingress status
	IngressReady bool `json:"ingressReady,omitempty"`

	// HPA status
	HPAReady bool `json:"hpaReady,omitempty"`

	// Service endpoint
	ServiceEndpoint string `json:"serviceEndpoint,omitempty"`

	// Ingress endpoint
	IngressEndpoint string `json:"ingressEndpoint,omitempty"`

	// Conditions
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Message
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Image",type="string",JSONPath=".spec.image"
// +kubebuilder:printcolumn:name="Tag",type="string",JSONPath=".spec.tag"
// +kubebuilder:printcolumn:name="Host",type="string",JSONPath=".spec.host"
// +kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".spec.replicas"
// +kubebuilder:printcolumn:name="Ready",type="integer",JSONPath=".status.readyReplicas"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Application is the Schema for the applications API.
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"` // include metadata like name, namespace, labels, etc

	Spec   ApplicationSpec   `json:"spec,omitempty"`
	Status ApplicationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApplicationList contains a list of Application.
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Application `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Application{}, &ApplicationList{})
}
