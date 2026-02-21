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
}

// VolumeSpec defines a single volume attachment.
type VolumeSpec struct {
	// Name is the volume name, used in Aerospike config references.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Source defines the volume source: PVC, emptyDir, secret, or configMap.
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
	// +kubebuilder:validation:Enum=none;deleteFiles;dd;blkdiscard;headerCleanup
	// +kubebuilder:default=none
	// +optional
	InitMethod VolumeInitMethod `json:"initMethod,omitempty"`

	// CascadeDelete determines if PVCs should be deleted when the CR is deleted.
	// Only applicable to persistent volumes.
	// +kubebuilder:default=false
	// +optional
	CascadeDelete bool `json:"cascadeDelete,omitempty"`
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
}

// AerospikeVolumeAttachment defines how a volume is mounted in the Aerospike container.
type AerospikeVolumeAttachment struct {
	// Path is the mount path in the Aerospike container.
	// +kubebuilder:validation:Required
	Path string `json:"path"`
}

// VolumeAttachment defines a volume mount for sidecar or init containers.
type VolumeAttachment struct {
	// ContainerName is the name of the container to mount to.
	// +kubebuilder:validation:Required
	ContainerName string `json:"containerName"`

	// Path is the mount path in the container.
	// +kubebuilder:validation:Required
	Path string `json:"path"`
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
