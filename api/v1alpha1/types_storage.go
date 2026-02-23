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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AerospikeStorageSpec defines storage volumes for the Aerospike pods.
type AerospikeStorageSpec struct {
	// Volumes defines a list of volumes to attach to Aerospike pods.
	// +optional
	Volumes []VolumeSpec `json:"volumes,omitempty"`

	// CleanupThreads is the max number of threads used during volume cleanup/init.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	// +optional
	CleanupThreads int32 `json:"cleanupThreads,omitempty"`

	// FilesystemVolumePolicy defines the default policy for filesystem-mode persistent volumes.
	// Per-volume settings override this policy.
	// +optional
	FilesystemVolumePolicy *AerospikeVolumePolicy `json:"filesystemVolumePolicy,omitempty"`

	// BlockVolumePolicy defines the default policy for block-mode persistent volumes.
	// Per-volume settings override this policy.
	// +optional
	BlockVolumePolicy *AerospikeVolumePolicy `json:"blockVolumePolicy,omitempty"`

	// LocalStorageClasses lists StorageClass names that use local storage (e.g., local-path, openebs-hostpath).
	// Volumes using these classes require special handling on pod restart.
	// +optional
	LocalStorageClasses []string `json:"localStorageClasses,omitempty"`

	// DeleteLocalStorageOnRestart controls whether PVCs backed by local storage classes
	// are deleted before pod restart, forcing re-provisioning on the new node.
	// +optional
	DeleteLocalStorageOnRestart *bool `json:"deleteLocalStorageOnRestart,omitempty"`
}

// VolumeSpec defines a single volume attachment.
type VolumeSpec struct {
	// Name is the volume name, used in Aerospike config references.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Source defines the volume source: PVC, emptyDir, secret, configMap, or hostPath.
	// +kubebuilder:validation:Required
	Source VolumeSource `json:"source"`

	// Aerospike defines how this volume maps into Aerospike config.
	// +optional
	Aerospike *AerospikeVolumeAttachment `json:"aerospike,omitempty"`

	// Sidecars defines volume mounts for sidecar containers.
	// +optional
	Sidecars []VolumeAttachment `json:"sidecars,omitempty"`

	// InitContainers defines volume mounts for init containers.
	// +optional
	InitContainers []VolumeAttachment `json:"initContainers,omitempty"`

	// InitMethod defines how this volume should be initialized.
	// When empty, falls back to the global volume policy. Set to "none" to explicitly disable initialization.
	// +kubebuilder:validation:Enum=none;deleteFiles;dd;blkdiscard;headerCleanup
	// +optional
	InitMethod VolumeInitMethod `json:"initMethod,omitempty"`

	// WipeMethod defines how this volume should be wiped when marked dirty.
	// Wipe runs on volumes listed in the pod's DirtyVolumes status.
	// +kubebuilder:validation:Enum=none;deleteFiles;dd;blkdiscard;headerCleanup;blkdiscardWithHeaderCleanup
	// +optional
	WipeMethod VolumeWipeMethod `json:"wipeMethod,omitempty"`

	// CascadeDelete determines if PVCs should be deleted when the CR is deleted.
	// Only applicable to persistent volumes. When nil, falls back to global volume policy.
	// +optional
	CascadeDelete *bool `json:"cascadeDelete,omitempty"`
}

// VolumeSource describes the volume data source.
type VolumeSource struct {
	// PersistentVolume creates a PVC for this volume.
	// +optional
	PersistentVolume *PersistentVolumeSpec `json:"persistentVolume,omitempty"`

	// EmptyDir uses an emptyDir volume.
	// +optional
	EmptyDir *corev1.EmptyDirVolumeSource `json:"emptyDir,omitempty"`

	// Secret uses a Kubernetes Secret as the volume source.
	// +optional
	Secret *corev1.SecretVolumeSource `json:"secret,omitempty"`

	// ConfigMap uses a Kubernetes ConfigMap as the volume source.
	// +optional
	ConfigMap *corev1.ConfigMapVolumeSource `json:"configMap,omitempty"`

	// HostPath uses a path on the host node as the volume source.
	// +optional
	HostPath *corev1.HostPathVolumeSource `json:"hostPath,omitempty"`
}

// PersistentVolumeSpec defines a persistent volume claim template.
type PersistentVolumeSpec struct {
	// StorageClass is the name of the StorageClass to use.
	// +optional
	StorageClass string `json:"storageClass,omitempty"`

	// VolumeMode defines whether the volume is filesystem or block.
	// +kubebuilder:validation:Enum=Filesystem;Block
	// +kubebuilder:default=Filesystem
	// +optional
	VolumeMode corev1.PersistentVolumeMode `json:"volumeMode,omitempty"`

	// Size is the storage size request (e.g., "10Gi").
	// +kubebuilder:validation:Required
	Size string `json:"size"`

	// AccessModes defines how the volume can be accessed.
	// +optional
	AccessModes []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty"`

	// Selector is a label selector for binding to a specific PV.
	// +optional
	Selector *metav1.LabelSelector `json:"selector,omitempty"`

	// Metadata defines custom labels and annotations for the PVC.
	// +optional
	Metadata *AerospikeObjectMeta `json:"metadata,omitempty"`
}

// AerospikeVolumeAttachment defines how a volume is mounted in the Aerospike container.
type AerospikeVolumeAttachment struct {
	// Path is the mount path in the Aerospike container.
	// +kubebuilder:validation:Required
	Path string `json:"path"`

	// ReadOnly mounts the volume as read-only.
	// +optional
	ReadOnly bool `json:"readOnly,omitempty"`

	// SubPath mounts only a sub-path of the volume.
	// +optional
	SubPath string `json:"subPath,omitempty"`

	// SubPathExpr is an expanded path using environment variables.
	// Mutually exclusive with SubPath.
	// +optional
	SubPathExpr string `json:"subPathExpr,omitempty"`

	// MountPropagation determines how mounts are propagated.
	// +optional
	MountPropagation *corev1.MountPropagationMode `json:"mountPropagation,omitempty"`
}

// VolumeAttachment defines a volume mount for sidecar or init containers.
type VolumeAttachment struct {
	// ContainerName is the name of the container to mount to.
	// +kubebuilder:validation:Required
	ContainerName string `json:"containerName"`

	// Path is the mount path in the container.
	// +kubebuilder:validation:Required
	Path string `json:"path"`

	// ReadOnly mounts the volume as read-only.
	// +optional
	ReadOnly bool `json:"readOnly,omitempty"`

	// SubPath mounts only a sub-path of the volume.
	// +optional
	SubPath string `json:"subPath,omitempty"`

	// SubPathExpr is an expanded path using environment variables.
	// Mutually exclusive with SubPath.
	// +optional
	SubPathExpr string `json:"subPathExpr,omitempty"`

	// MountPropagation determines how mounts are propagated.
	// +optional
	MountPropagation *corev1.MountPropagationMode `json:"mountPropagation,omitempty"`
}

// AerospikeVolumePolicy defines default policies for a category of persistent volumes.
type AerospikeVolumePolicy struct {
	// InitMethod is the default initialization method for this volume category.
	// +optional
	InitMethod VolumeInitMethod `json:"initMethod,omitempty"`

	// WipeMethod is the default wipe method for this volume category.
	// Wipe runs on volumes that are marked dirty (e.g., after unclean shutdown).
	// +optional
	WipeMethod VolumeWipeMethod `json:"wipeMethod,omitempty"`

	// CascadeDelete controls whether PVCs are deleted when the CR is deleted.
	// +optional
	CascadeDelete *bool `json:"cascadeDelete,omitempty"`
}

// VolumeInitMethod defines how a volume is initialized on first use.
// +kubebuilder:validation:Enum=none;deleteFiles;dd;blkdiscard;headerCleanup
type VolumeInitMethod string

const (
	VolumeInitMethodNone          VolumeInitMethod = "none"
	VolumeInitMethodDeleteFiles   VolumeInitMethod = "deleteFiles"
	VolumeInitMethodDD            VolumeInitMethod = "dd"
	VolumeInitMethodBlkdiscard    VolumeInitMethod = "blkdiscard"
	VolumeInitMethodHeaderCleanup VolumeInitMethod = "headerCleanup"
)

// VolumeWipeMethod defines how a volume is wiped when marked dirty.
// +kubebuilder:validation:Enum=none;deleteFiles;dd;blkdiscard;headerCleanup;blkdiscardWithHeaderCleanup
type VolumeWipeMethod string

const (
	VolumeWipeMethodNone                        VolumeWipeMethod = "none"
	VolumeWipeMethodDeleteFiles                 VolumeWipeMethod = "deleteFiles"
	VolumeWipeMethodDD                          VolumeWipeMethod = "dd"
	VolumeWipeMethodBlkdiscard                  VolumeWipeMethod = "blkdiscard"
	VolumeWipeMethodHeaderCleanup               VolumeWipeMethod = "headerCleanup"
	VolumeWipeMethodBlkdiscardWithHeaderCleanup VolumeWipeMethod = "blkdiscardWithHeaderCleanup"
)
