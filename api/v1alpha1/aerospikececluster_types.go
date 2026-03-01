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

// TemplateRef is a reference to an AerospikeCEClusterTemplate in the same namespace.
type TemplateRef struct {
	// Name is the name of the AerospikeCEClusterTemplate resource.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// TemplateSnapshotStatus records which template version was resolved and when.
type TemplateSnapshotStatus struct {
	// Name is the name of the referenced AerospikeCEClusterTemplate.
	Name string `json:"name"`

	// ResourceVersion is the resourceVersion of the template at snapshot time.
	// +optional
	ResourceVersion string `json:"resourceVersion,omitempty"`

	// SnapshotTimestamp is when the snapshot was taken.
	// +optional
	SnapshotTimestamp metav1.Time `json:"snapshotTimestamp,omitempty"`

	// Synced indicates whether the cluster is using the latest version of the template.
	// Set to false when the template changes after the snapshot was taken.
	Synced bool `json:"synced"`

	// Spec is the resolved template spec at snapshot time.
	// +optional
	Spec *AerospikeCEClusterTemplateSpec `json:"spec,omitempty"`
}

// OperationKind defines the type of on-demand operation.
// +kubebuilder:validation:Enum=WarmRestart;PodRestart
type OperationKind string

const (
	// OperationWarmRestart sends SIGUSR1 to the asd process for a graceful warm restart
	// without data loss.
	OperationWarmRestart OperationKind = "WarmRestart"
	// OperationPodRestart deletes and recreates the pod for a full cold restart.
	OperationPodRestart OperationKind = "PodRestart"
)

// OperationSpec defines an on-demand operation to trigger.
type OperationSpec struct {
	// Kind is the type of operation.
	// WarmRestart sends SIGUSR1 to the Aerospike process.
	// PodRestart deletes and recreates the pod.
	// +kubebuilder:validation:Required
	Kind OperationKind `json:"kind"`

	// ID is a unique identifier for tracking the operation (1-20 chars).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=20
	ID string `json:"id"`

	// PodList is an optional list of specific pod names to target.
	// If empty, the operation applies to all pods.
	// +optional
	PodList []string `json:"podList,omitempty"`
}

// OperationStatus tracks the status of an on-demand operation.
type OperationStatus struct {
	// ID is the operation identifier.
	ID string `json:"id,omitempty"`

	// Kind is the operation type.
	Kind OperationKind `json:"kind,omitempty"`

	// Phase is the operation phase: InProgress, Completed, Error.
	Phase AerospikePhase `json:"phase,omitempty"`

	// CompletedPods lists pods that have completed the operation.
	// +optional
	CompletedPods []string `json:"completedPods,omitempty"`

	// FailedPods lists pods where the operation failed.
	// +optional
	FailedPods []string `json:"failedPods,omitempty"`
}

// ValidationPolicySpec controls validation behavior.
type ValidationPolicySpec struct {
	// SkipWorkDirValidate skips validation that the Aerospike work directory
	// is mounted on persistent storage.
	// +optional
	SkipWorkDirValidate bool `json:"skipWorkDirValidate,omitempty"`
}

// AerospikeObjectMeta defines custom metadata for Kubernetes objects.
type AerospikeObjectMeta struct {
	// Annotations is a map of custom annotations.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels is a map of custom labels.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

// AerospikeServiceSpec defines custom metadata for a Kubernetes Service.
type AerospikeServiceSpec struct {
	// Metadata defines custom annotations and labels for the service.
	// +optional
	Metadata *AerospikeObjectMeta `json:"metadata,omitempty"`
}

// AerospikeCEClusterSpec defines the desired state of an Aerospike CE cluster.
type AerospikeCEClusterSpec struct {
	// Size is the number of Aerospike nodes (pods) in the cluster.
	// CE limits this to a maximum of 8.
	// When spec.templateRef is set, size may be omitted and the template's default will be used.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=8
	// +optional
	Size int32 `json:"size,omitempty"`

	// Image is the Aerospike CE server container image.
	// Must be a community edition image (e.g., aerospike:ce-8.1.1.1).
	// When spec.templateRef is set, image may be omitted and the template's default will be used.
	// +optional
	Image string `json:"image,omitempty"`

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

	// Operations defines on-demand operations (e.g., WarmRestart, PodRestart).
	// Only one operation can be active at a time.
	// +kubebuilder:validation:MaxItems=1
	// +optional
	Operations []OperationSpec `json:"operations,omitempty"`

	// ValidationPolicy controls validation behavior.
	// +optional
	ValidationPolicy *ValidationPolicySpec `json:"validationPolicy,omitempty"`

	// HeadlessService defines custom metadata for the headless service.
	// +optional
	HeadlessService *AerospikeServiceSpec `json:"headlessService,omitempty"`

	// PodService defines custom metadata for per-pod services.
	// When set, the operator creates an individual Service for each pod.
	// +optional
	PodService *AerospikeServiceSpec `json:"podService,omitempty"`

	// EnableRackIDOverride enables dynamic rack ID assignment via pod annotations.
	// +optional
	EnableRackIDOverride *bool `json:"enableRackIDOverride,omitempty"`

	// TemplateRef references an AerospikeCEClusterTemplate to use as a configuration base.
	// When set, the template's spec is resolved at creation time and stored as a snapshot
	// in status.templateSnapshot. Template changes are not automatically propagated;
	// use the annotation "acko.io/resync-template: true" to trigger a manual resync.
	// +optional
	TemplateRef *TemplateRef `json:"templateRef,omitempty"`

	// Overrides contains fields that override the referenced template's spec.
	// Merge priority: overrides > template.spec > operator defaults.
	// +optional
	Overrides *AerospikeCEClusterTemplateSpec `json:"overrides,omitempty"`
}

// Condition type constants for AerospikeCECluster status conditions.
const (
	// ConditionAvailable indicates at least one pod is ready to serve requests.
	ConditionAvailable = "Available"
	// ConditionReady indicates all desired pods are running and ready.
	ConditionReady = "Ready"
	// ConditionConfigApplied indicates all pods have the desired Aerospike configuration.
	ConditionConfigApplied = "ConfigApplied"
	// ConditionACLSynced indicates ACL roles and users are synchronized with the cluster.
	ConditionACLSynced = "ACLSynced"
	// ConditionMigrationComplete indicates no data migrations are pending.
	ConditionMigrationComplete = "MigrationComplete"
	// ConditionReconciliationPaused indicates reconciliation is paused by the user.
	ConditionReconciliationPaused = "ReconciliationPaused"
)

// AerospikePhase represents the current phase of the cluster.
// +kubebuilder:validation:Enum=InProgress;Completed;Error;ScalingUp;ScalingDown;RollingRestart;ACLSync;Paused;Deleting
type AerospikePhase string

const (
	// AerospikePhaseInProgress indicates reconciliation is actively in progress (generic).
	AerospikePhaseInProgress AerospikePhase = "InProgress"
	// AerospikePhaseCompleted indicates the cluster has reached the desired state.
	AerospikePhaseCompleted AerospikePhase = "Completed"
	// AerospikePhaseError indicates an unrecoverable error during reconciliation.
	AerospikePhaseError AerospikePhase = "Error"
	// AerospikePhaseScalingUp indicates the cluster is scaling up (adding pods).
	AerospikePhaseScalingUp AerospikePhase = "ScalingUp"
	// AerospikePhaseScalingDown indicates the cluster is scaling down (removing pods).
	AerospikePhaseScalingDown AerospikePhase = "ScalingDown"
	// AerospikePhaseRollingRestart indicates a rolling restart is in progress.
	AerospikePhaseRollingRestart AerospikePhase = "RollingRestart"
	// AerospikePhaseACLSync indicates ACL roles and users are being synchronized.
	AerospikePhaseACLSync AerospikePhase = "ACLSync"
	// AerospikePhasePaused indicates reconciliation is paused by the user.
	AerospikePhasePaused AerospikePhase = "Paused"
	// AerospikePhaseDeleting indicates the cluster is being deleted.
	AerospikePhaseDeleting AerospikePhase = "Deleting"
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

	// OperationStatus tracks the current on-demand operation status.
	// +optional
	OperationStatus *OperationStatus `json:"operationStatus,omitempty"`

	// PhaseReason provides a human-readable explanation of the current phase.
	// Examples: "Rolling restart in progress for rack 1", "Scaling up rack 2 from 2 to 3 pods".
	// +optional
	PhaseReason string `json:"phaseReason,omitempty"`

	// AppliedSpec is a copy of the last successfully reconciled spec.
	// Use this to detect configuration drift or compare against the current spec.
	// +optional
	AppliedSpec *AerospikeCEClusterSpec `json:"appliedSpec,omitempty"`

	// TemplateSnapshot holds the resolved template spec at the time of last sync.
	// This is the basis for template-derived configuration; changes to the template
	// are not propagated until a manual resync is triggered.
	// +optional
	TemplateSnapshot *TemplateSnapshotStatus `json:"templateSnapshot,omitempty"`
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

	// NodeID is the Aerospike-assigned node identifier (e.g. "BB9020012AC4202").
	// Populated by querying the node via asinfo; empty if the node is unreachable.
	// +optional
	NodeID string `json:"nodeID,omitempty"`

	// ClusterName is the Aerospike cluster name as reported by the node.
	// +optional
	ClusterName string `json:"clusterName,omitempty"`

	// AccessEndpoints are the network endpoints (host:port) for direct client access.
	// Populated via the asinfo "service" command.
	// +optional
	AccessEndpoints []string `json:"accessEndpoints,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.size,statuspath=.status.size,selectorpath=.status.selector
// +kubebuilder:resource:shortName=asce;ascecluster
// +kubebuilder:printcolumn:name="Size",type=integer,JSONPath=`.spec.size`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.size`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.spec.image`,priority=1
// +kubebuilder:printcolumn:name="ObservedGen",type=integer,JSONPath=`.status.observedGeneration`,priority=1
// +kubebuilder:printcolumn:name="PhaseReason",type=string,JSONPath=`.status.phaseReason`,priority=1

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
