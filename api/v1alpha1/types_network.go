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

// AerospikeNetworkType defines how Aerospike advertises its addresses.
// +kubebuilder:validation:Enum=pod;hostInternal;hostExternal;configuredIP
type AerospikeNetworkType string

const (
	// AerospikeNetworkTypePod uses the Pod IP (default, for in-cluster clients).
	AerospikeNetworkTypePod AerospikeNetworkType = "pod"
	// AerospikeNetworkTypeHostInternal uses the node's internal IP.
	AerospikeNetworkTypeHostInternal AerospikeNetworkType = "hostInternal"
	// AerospikeNetworkTypeHostExternal uses the node's external IP.
	AerospikeNetworkTypeHostExternal AerospikeNetworkType = "hostExternal"
	// AerospikeNetworkTypeConfiguredIP uses a user-provided IP from pod annotations.
	AerospikeNetworkTypeConfiguredIP AerospikeNetworkType = "configuredIP"
)

// AerospikeNetworkPolicy defines network access configuration.
type AerospikeNetworkPolicy struct {
	// AccessType determines how clients access the Aerospike service port.
	// +kubebuilder:default=pod
	// +optional
	AccessType AerospikeNetworkType `json:"accessType,omitempty"`

	// AlternateAccessType determines how clients from alternate networks
	// access the Aerospike service port.
	// +kubebuilder:default=pod
	// +optional
	AlternateAccessType AerospikeNetworkType `json:"alternateAccessType,omitempty"`

	// FabricType determines the network type for fabric (inter-node) communication.
	// +kubebuilder:default=pod
	// +optional
	FabricType AerospikeNetworkType `json:"fabricType,omitempty"`

	// CustomAccessNetworkNames specifies network names for configuredIP access type.
	// +optional
	CustomAccessNetworkNames []string `json:"customAccessNetworkNames,omitempty"`

	// CustomAlternateAccessNetworkNames specifies network names for
	// configuredIP alternate access type.
	// +optional
	CustomAlternateAccessNetworkNames []string `json:"customAlternateAccessNetworkNames,omitempty"`

	// CustomFabricNetworkNames specifies network names for
	// configuredIP fabric type.
	// +optional
	CustomFabricNetworkNames []string `json:"customFabricNetworkNames,omitempty"`
}

// SeedsFinderServices configures external seed discovery via LoadBalancer.
type SeedsFinderServices struct {
	// LoadBalancer creates a LoadBalancer service for seed discovery by external clients.
	// +optional
	LoadBalancer *LoadBalancerSpec `json:"loadBalancer,omitempty"`
}

// LoadBalancerSpec defines a LoadBalancer service configuration.
type LoadBalancerSpec struct {
	// Annotations to add to the LoadBalancer service.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels to add to the LoadBalancer service.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// ExternalTrafficPolicy defines the external traffic policy.
	// +kubebuilder:validation:Enum=Cluster;Local
	// +optional
	ExternalTrafficPolicy string `json:"externalTrafficPolicy,omitempty"`

	// Port is the external port number for the LoadBalancer.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default=3000
	// +optional
	Port int32 `json:"port,omitempty"`

	// TargetPort is the container port to forward to.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default=3000
	// +optional
	TargetPort int32 `json:"targetPort,omitempty"`

	// LoadBalancerSourceRanges restricts traffic to specified CIDRs.
	// +optional
	LoadBalancerSourceRanges []string `json:"loadBalancerSourceRanges,omitempty"`
}
