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

// NetworkPolicyType defines the type of network policy to create.
// +kubebuilder:validation:Enum=kubernetes;cilium
type NetworkPolicyType string

const (
	NetworkPolicyTypeKubernetes NetworkPolicyType = "kubernetes"
	NetworkPolicyTypeCilium     NetworkPolicyType = "cilium"
)

// NetworkPolicyConfig defines automatic NetworkPolicy creation for the cluster.
type NetworkPolicyConfig struct {
	// Enabled enables automatic NetworkPolicy creation.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Type specifies the NetworkPolicy type: "kubernetes" for standard
	// K8s NetworkPolicy or "cilium" for CiliumNetworkPolicy.
	// Defaults to "kubernetes".
	// +kubebuilder:default=kubernetes
	// +optional
	Type NetworkPolicyType `json:"type,omitempty"`
}

// BandwidthConfig defines bandwidth annotations for traffic shaping.
// These annotations are recognized by CNI plugins such as Cilium bandwidth manager.
type BandwidthConfig struct {
	// Ingress is the maximum ingress bandwidth (e.g., "1Gbps", "500Mbps").
	// +optional
	Ingress string `json:"ingress,omitempty"`

	// Egress is the maximum egress bandwidth (e.g., "1Gbps", "500Mbps").
	// +optional
	Egress string `json:"egress,omitempty"`
}
