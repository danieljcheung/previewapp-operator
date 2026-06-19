/*
Copyright 2026.

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
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PreviewAppSpec defines the desired state of PreviewApp.
type PreviewAppSpec struct {
	// image is the immutable container image reference to run for this preview.
	// V1 previews are intentionally limited to GHCR-hosted images.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^ghcr\.io/.+`
	Image string `json:"image"`

	// appPort is the TCP port exposed by the preview container.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	AppPort int32 `json:"appPort"`

	// ttlSeconds is the maximum lifetime of the preview before cleanup.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=60
	TTLSeconds int32 `json:"ttlSeconds"`

	// route configures the private Tailscale preview route.
	// +kubebuilder:validation:Required
	Route PreviewAppRouteSpec `json:"route"`
}

// PreviewAppRouteSpec defines private Tailscale routing for a preview.
type PreviewAppRouteSpec struct {
	// host is the short MagicDNS hostname requested through Tailscale Ingress.
	// Use a single DNS label, not a dotted FQDN.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	// +kubebuilder:validation:MaxLength=63
	Host string `json:"host"`
}

// PreviewAppStatus defines the observed state of PreviewApp.
type PreviewAppStatus struct {
	// phase is a compact summary for kubectl output and humans.
	// +kubebuilder:validation:Enum=Pending;Reconciling;Ready;Invalid;Expired;Failed
	// +optional
	Phase string `json:"phase,omitempty"`

	// url is set only after the Tailscale Ingress reports a ready hostname.
	// +optional
	URL string `json:"url,omitempty"`

	// expiresAt is computed once from creation time and ttlSeconds.
	// +optional
	ExpiresAt *metav1.Time `json:"expiresAt,omitempty"`

	// observedGeneration is the latest metadata.generation reconciled into status.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// conditions represent the current state of the PreviewApp resource.
	// Known condition types include DeploymentReady, ServiceReady, IngressReady,
	// Ready, and Expiring.
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PreviewApp is the Schema for the previewapps API
type PreviewApp struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of PreviewApp
	// +required
	Spec PreviewAppSpec `json:"spec"`

	// status defines the observed state of PreviewApp
	// +optional
	Status PreviewAppStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// PreviewAppList contains a list of PreviewApp
type PreviewAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []PreviewApp `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &PreviewApp{}, &PreviewAppList{})
		return nil
	})
}
