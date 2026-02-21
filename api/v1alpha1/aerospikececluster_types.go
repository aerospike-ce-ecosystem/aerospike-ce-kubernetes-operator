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
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// AerospikeConfigSpec holds the raw Aerospike configuration as an unstructured JSON object.
// The data is converted to aerospike.conf format by the operator.
//
// +kubebuilder:object:generate=false
// +kubebuilder:pruning:PreserveUnknownFields
// +kubebuilder:validation:Type=object
type AerospikeConfigSpec struct {
	Value map[string]any `json:"-"`
}

// MarshalJSON implements json.Marshaler.
func (a AerospikeConfigSpec) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.Value)
}

// UnmarshalJSON implements json.Unmarshaler.
func (a *AerospikeConfigSpec) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &a.Value)
}

// DeepCopy returns a deep copy of the AerospikeConfigSpec.
func (a *AerospikeConfigSpec) DeepCopy() *AerospikeConfigSpec {
	if a == nil {
		return nil
	}
	out := &AerospikeConfigSpec{
		Value: deepCopyMap(a.Value),
	}
	return out
}

// DeepCopyInto writes a deep copy of AerospikeConfigSpec into out.
func (a *AerospikeConfigSpec) DeepCopyInto(out *AerospikeConfigSpec) {
	out.Value = deepCopyMap(a.Value)
}

func deepCopyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for key, val := range m {
		out[key] = deepCopyValue(val)
	}
	return out
}

func deepCopyValue(val any) any {
	switch v := val.(type) {
	case map[string]any:
		return deepCopyMap(v)
	case []any:
		out := make([]any, len(v))
		for i, inner := range v {
			out[i] = deepCopyValue(inner)
		}
		return out
	default:
		return v
	}
}

// AerospikeCEClusterSpec defines the desired state of an Aerospike CE cluster.
type AerospikeCEClusterSpec struct {
	// Size is the number of Aerospike nodes (pods) in the cluster.
	// CE limits this to a maximum of 8.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=8
	// +kubebuilder:validation:Required
	Size int32 `json:"size"`

	// Image is the Aerospike CE server container image.
	// Must be a community edition image (e.g., aerospike:ce-8.1.1.1).
	// +kubebuilder:validation:Required
	Image string `json:"image"`

	// AerospikeConfig is the raw Aerospike configuration map.
	// This is converted to aerospike.conf by the operator.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	AerospikeConfig *AerospikeConfigSpec `json:"aerospikeConfig,omitempty"`

	// Storage defines volumes and volume mounts for Aerospike pods.
	// +optional
	Storage *AerospikeStorageSpec `json:"storage,omitempty"`

	// RackConfig defines rack-aware deployment topology.
	// +optional
	RackConfig *RackConfig `json:"rackConfig,omitempty"`

	// AerospikeNetworkPolicy defines how clients access the Aerospike cluster.
	// +optional
	AerospikeNetworkPolicy *AerospikeNetworkPolicy `json:"aerospikeNetworkPolicy,omitempty"`

	// PodSpec defines pod-level configuration (sidecars, resources, scheduling).
	// +optional
	PodSpec *AerospikeCEPodSpec `json:"podSpec,omitempty"`

	// AerospikeAccessControl defines ACL roles and users for CE 7.x+.
	// +optional
	AerospikeAccessControl *AerospikeAccessControlSpec `json:"aerospikeAccessControl,omitempty"`

	// Monitoring configures Prometheus monitoring via an exporter sidecar.
	// +optional
	Monitoring *AerospikeMonitoringSpec `json:"monitoring,omitempty"`

	// NetworkPolicyConfig configures automatic NetworkPolicy creation for the cluster.
	// +optional
	NetworkPolicyConfig *NetworkPolicyConfig `json:"networkPolicyConfig,omitempty"`

	// BandwidthConfig defines bandwidth annotations for CNI traffic shaping.
	// +optional
	BandwidthConfig *BandwidthConfig `json:"bandwidthConfig,omitempty"`

	// EnableDynamicConfigUpdate enables runtime configuration changes
	// without pod restart using Aerospike's set-config command.
	// +optional
	EnableDynamicConfigUpdate *bool `json:"enableDynamicConfigUpdate,omitempty"`

	// RollingUpdateBatchSize is the number of pods to restart in parallel
	// during a rolling restart. Defaults to 1 (sequential).
	// +kubebuilder:validation:Minimum=1
	// +optional
	RollingUpdateBatchSize *int32 `json:"rollingUpdateBatchSize,omitempty"`

	// DisablePDB disables PodDisruptionBudget creation for the cluster.
	// +optional
	DisablePDB *bool `json:"disablePDB,omitempty"`

	// MaxUnavailable is the maximum number of pods that can be unavailable during disruption.
	// Used for PodDisruptionBudget. Defaults to 1.
	// +optional
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`

	// Paused stops reconciliation when set to true.
	// +optional
	Paused *bool `json:"paused,omitempty"`

	// SeedsFinderServices configures LoadBalancer services for seed discovery.
	// +optional
	SeedsFinderServices *SeedsFinderServices `json:"seedsFinderServices,omitempty"`

	// K8sNodeBlockList contains Kubernetes node names that should not run Aerospike pods.
	// +optional
	K8sNodeBlockList []string `json:"k8sNodeBlockList,omitempty"`
}

// AerospikePhase represents the current phase of the cluster.
// +kubebuilder:validation:Enum=InProgress;Completed;Error
type AerospikePhase string

const (
	AerospikePhaseInProgress AerospikePhase = "InProgress"
	AerospikePhaseCompleted  AerospikePhase = "Completed"
	AerospikePhaseError      AerospikePhase = "Error"
)

// AerospikeCEClusterStatus defines the observed state of the Aerospike CE cluster.
type AerospikeCEClusterStatus struct {
	// Phase indicates the overall cluster phase.
	// +optional
	Phase AerospikePhase `json:"phase,omitempty"`

	// Size is the current cluster size.
	// +optional
	Size int32 `json:"size,omitempty"`

	// Conditions represent the latest observations of the cluster state.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Pods contains status information for each pod in the cluster.
	// +optional
	Pods map[string]AerospikePodStatus `json:"pods,omitempty"`

	// ObservedGeneration is the most recent generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Selector is the label selector for HPA compatibility.
	// +optional
	Selector string `json:"selector,omitempty"`

	// AerospikeConfig is the last applied Aerospike configuration.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	AerospikeConfig *AerospikeConfigSpec `json:"aerospikeConfig,omitempty"`
}

// AerospikePodStatus holds per-pod status information.
type AerospikePodStatus struct {
	// PodIP is the IP address of the pod.
	PodIP string `json:"podIP,omitempty"`
	// HostIP is the IP address of the host node.
	HostIP string `json:"hostIP,omitempty"`
	// Image is the container image running on the pod.
	Image string `json:"image,omitempty"`
	// PodPort is the Aerospike service port on the pod.
	PodPort int32 `json:"podPort,omitempty"`
	// ServicePort is the Aerospike service port exposed via node/LB.
	ServicePort int32 `json:"servicePort,omitempty"`
	// Rack is the rack ID assigned to this pod.
	Rack int `json:"rack,omitempty"`
	// InitializedVolumes lists volumes that have been initialized.
	InitializedVolumes []string `json:"initializedVolumes,omitempty"`
	// IsRunningAndReady indicates whether the pod is running and ready.
	IsRunningAndReady bool `json:"isRunningAndReady,omitempty"`
	// ConfigHash is the SHA256 hash of the Aerospike configuration applied to this pod.
	ConfigHash string `json:"configHash,omitempty"`
	// PodSpecHash is the hash of the pod template spec applied to this pod.
	PodSpecHash string `json:"podSpecHash,omitempty"`
	// DynamicConfigStatus indicates the last dynamic config update result.
	// Possible values: "", "Applied", "Failed", "Pending".
	DynamicConfigStatus string `json:"dynamicConfigStatus,omitempty"`
	// DirtyVolumes lists volumes that need initialization or cleanup.
	DirtyVolumes []string `json:"dirtyVolumes,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.size,statuspath=.status.size,selectorpath=.status.selector
// +kubebuilder:resource:shortName=asce;ascecluster
// +kubebuilder:printcolumn:name="Size",type=integer,JSONPath=`.spec.size`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.spec.image`,priority=1

// AerospikeCECluster is the Schema for the aerospikececlusters API.
// It manages the lifecycle of an Aerospike Community Edition cluster.
type AerospikeCECluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AerospikeCEClusterSpec   `json:"spec"`
	Status AerospikeCEClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AerospikeCEClusterList contains a list of AerospikeCECluster.
type AerospikeCEClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AerospikeCECluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AerospikeCECluster{}, &AerospikeCEClusterList{})
}
