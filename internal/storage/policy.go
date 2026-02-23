package storage

import (
	corev1 "k8s.io/api/core/v1"

	v1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
)

// ResolveInitMethod returns the effective init method for a volume.
// Precedence: per-volume > global policy > "none".
func ResolveInitMethod(vol *v1alpha1.VolumeSpec, storageSpec *v1alpha1.AerospikeStorageSpec) v1alpha1.VolumeInitMethod {
	// Per-volume override
	if vol.InitMethod != "" && vol.InitMethod != v1alpha1.VolumeInitMethodNone {
		return vol.InitMethod
	}

	// Global policy fallback
	if policy := getVolumePolicy(vol, storageSpec); policy != nil {
		if policy.InitMethod != "" && policy.InitMethod != v1alpha1.VolumeInitMethodNone {
			return policy.InitMethod
		}
	}

	return v1alpha1.VolumeInitMethodNone
}

// ResolveWipeMethod returns the effective wipe method for a volume.
// Precedence: per-volume > global policy > "none".
func ResolveWipeMethod(vol *v1alpha1.VolumeSpec, storageSpec *v1alpha1.AerospikeStorageSpec) v1alpha1.VolumeWipeMethod {
	// Per-volume override
	if vol.WipeMethod != "" && vol.WipeMethod != v1alpha1.VolumeWipeMethodNone {
		return vol.WipeMethod
	}

	// Global policy fallback
	if policy := getVolumePolicy(vol, storageSpec); policy != nil {
		if policy.WipeMethod != "" && policy.WipeMethod != v1alpha1.VolumeWipeMethodNone {
			return policy.WipeMethod
		}
	}

	return v1alpha1.VolumeWipeMethodNone
}

// ResolveCascadeDelete returns the effective cascade delete setting for a volume.
// Precedence: per-volume true > global policy > false.
func ResolveCascadeDelete(vol *v1alpha1.VolumeSpec, storageSpec *v1alpha1.AerospikeStorageSpec) bool {
	// Per-volume override
	if vol.CascadeDelete {
		return true
	}

	// Global policy fallback
	if policy := getVolumePolicy(vol, storageSpec); policy != nil {
		if policy.CascadeDelete != nil {
			return *policy.CascadeDelete
		}
	}

	return false
}

// getVolumePolicy returns the applicable global volume policy for the given volume.
// Returns nil for non-persistent volumes or when no policy is defined.
func getVolumePolicy(vol *v1alpha1.VolumeSpec, storageSpec *v1alpha1.AerospikeStorageSpec) *v1alpha1.AerospikeVolumePolicy {
	if storageSpec == nil || vol.Source.PersistentVolume == nil {
		return nil
	}

	pv := vol.Source.PersistentVolume

	volumeMode := pv.VolumeMode
	if volumeMode == "" {
		volumeMode = corev1.PersistentVolumeFilesystem
	}

	if volumeMode == corev1.PersistentVolumeBlock {
		return storageSpec.BlockVolumePolicy
	}

	return storageSpec.FilesystemVolumePolicy
}
