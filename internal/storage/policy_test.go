package storage

import (
	"testing"

	corev1 "k8s.io/api/core/v1"

	v1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
)

func boolPtr(b bool) *bool { return &b }

// --- ResolveInitMethod tests ---

func TestResolveInitMethod_PerVolumeOverridesPolicy(t *testing.T) {
	vol := &v1alpha1.VolumeSpec{
		InitMethod: v1alpha1.VolumeInitMethodDD,
		Source: v1alpha1.VolumeSource{
			PersistentVolume: &v1alpha1.PersistentVolumeSpec{Size: "10Gi"},
		},
	}
	spec := &v1alpha1.AerospikeStorageSpec{
		FilesystemVolumePolicy: &v1alpha1.AerospikeVolumePolicy{
			InitMethod: v1alpha1.VolumeInitMethodDeleteFiles,
		},
	}

	result := ResolveInitMethod(vol, spec)
	if result != v1alpha1.VolumeInitMethodDD {
		t.Errorf("expected %q, got %q", v1alpha1.VolumeInitMethodDD, result)
	}
}

func TestResolveInitMethod_FilesystemPolicyFallback(t *testing.T) {
	vol := &v1alpha1.VolumeSpec{
		Source: v1alpha1.VolumeSource{
			PersistentVolume: &v1alpha1.PersistentVolumeSpec{
				Size:       "10Gi",
				VolumeMode: corev1.PersistentVolumeFilesystem,
			},
		},
	}
	spec := &v1alpha1.AerospikeStorageSpec{
		FilesystemVolumePolicy: &v1alpha1.AerospikeVolumePolicy{
			InitMethod: v1alpha1.VolumeInitMethodDeleteFiles,
		},
	}

	result := ResolveInitMethod(vol, spec)
	if result != v1alpha1.VolumeInitMethodDeleteFiles {
		t.Errorf("expected %q, got %q", v1alpha1.VolumeInitMethodDeleteFiles, result)
	}
}

func TestResolveInitMethod_BlockPolicyFallback(t *testing.T) {
	vol := &v1alpha1.VolumeSpec{
		Source: v1alpha1.VolumeSource{
			PersistentVolume: &v1alpha1.PersistentVolumeSpec{
				Size:       "10Gi",
				VolumeMode: corev1.PersistentVolumeBlock,
			},
		},
	}
	spec := &v1alpha1.AerospikeStorageSpec{
		BlockVolumePolicy: &v1alpha1.AerospikeVolumePolicy{
			InitMethod: v1alpha1.VolumeInitMethodBlkdiscard,
		},
		FilesystemVolumePolicy: &v1alpha1.AerospikeVolumePolicy{
			InitMethod: v1alpha1.VolumeInitMethodDeleteFiles,
		},
	}

	result := ResolveInitMethod(vol, spec)
	if result != v1alpha1.VolumeInitMethodBlkdiscard {
		t.Errorf("expected %q, got %q", v1alpha1.VolumeInitMethodBlkdiscard, result)
	}
}

func TestResolveInitMethod_NoPolicyDefaultNone(t *testing.T) {
	vol := &v1alpha1.VolumeSpec{
		Source: v1alpha1.VolumeSource{
			PersistentVolume: &v1alpha1.PersistentVolumeSpec{Size: "10Gi"},
		},
	}
	spec := &v1alpha1.AerospikeStorageSpec{}

	result := ResolveInitMethod(vol, spec)
	if result != v1alpha1.VolumeInitMethodNone {
		t.Errorf("expected %q, got %q", v1alpha1.VolumeInitMethodNone, result)
	}
}

func TestResolveInitMethod_DefaultVolumeModeIsFilesystem(t *testing.T) {
	vol := &v1alpha1.VolumeSpec{
		Source: v1alpha1.VolumeSource{
			PersistentVolume: &v1alpha1.PersistentVolumeSpec{
				Size: "10Gi",
				// VolumeMode empty → defaults to Filesystem
			},
		},
	}
	spec := &v1alpha1.AerospikeStorageSpec{
		FilesystemVolumePolicy: &v1alpha1.AerospikeVolumePolicy{
			InitMethod: v1alpha1.VolumeInitMethodHeaderCleanup,
		},
	}

	result := ResolveInitMethod(vol, spec)
	if result != v1alpha1.VolumeInitMethodHeaderCleanup {
		t.Errorf("expected %q, got %q", v1alpha1.VolumeInitMethodHeaderCleanup, result)
	}
}

// --- ResolveWipeMethod tests ---

func TestResolveWipeMethod_PerVolumeOverridesPolicy(t *testing.T) {
	vol := &v1alpha1.VolumeSpec{
		WipeMethod: v1alpha1.VolumeWipeMethodDD,
		Source: v1alpha1.VolumeSource{
			PersistentVolume: &v1alpha1.PersistentVolumeSpec{Size: "10Gi"},
		},
	}
	spec := &v1alpha1.AerospikeStorageSpec{
		FilesystemVolumePolicy: &v1alpha1.AerospikeVolumePolicy{
			WipeMethod: v1alpha1.VolumeWipeMethodDeleteFiles,
		},
	}

	result := ResolveWipeMethod(vol, spec)
	if result != v1alpha1.VolumeWipeMethodDD {
		t.Errorf("expected %q, got %q", v1alpha1.VolumeWipeMethodDD, result)
	}
}

func TestResolveWipeMethod_PolicyFallback(t *testing.T) {
	vol := &v1alpha1.VolumeSpec{
		Source: v1alpha1.VolumeSource{
			PersistentVolume: &v1alpha1.PersistentVolumeSpec{Size: "10Gi"},
		},
	}
	spec := &v1alpha1.AerospikeStorageSpec{
		FilesystemVolumePolicy: &v1alpha1.AerospikeVolumePolicy{
			WipeMethod: v1alpha1.VolumeWipeMethodBlkdiscardWithHeaderCleanup,
		},
	}

	result := ResolveWipeMethod(vol, spec)
	if result != v1alpha1.VolumeWipeMethodBlkdiscardWithHeaderCleanup {
		t.Errorf("expected %q, got %q", v1alpha1.VolumeWipeMethodBlkdiscardWithHeaderCleanup, result)
	}
}

func TestResolveWipeMethod_NoPolicyDefaultNone(t *testing.T) {
	vol := &v1alpha1.VolumeSpec{
		Source: v1alpha1.VolumeSource{
			PersistentVolume: &v1alpha1.PersistentVolumeSpec{Size: "10Gi"},
		},
	}
	spec := &v1alpha1.AerospikeStorageSpec{}

	result := ResolveWipeMethod(vol, spec)
	if result != v1alpha1.VolumeWipeMethodNone {
		t.Errorf("expected %q, got %q", v1alpha1.VolumeWipeMethodNone, result)
	}
}

// --- ResolveCascadeDelete tests ---

func TestResolveCascadeDelete_PerVolumeTrueOverridesPolicy(t *testing.T) {
	vol := &v1alpha1.VolumeSpec{
		CascadeDelete: true,
		Source: v1alpha1.VolumeSource{
			PersistentVolume: &v1alpha1.PersistentVolumeSpec{Size: "10Gi"},
		},
	}
	spec := &v1alpha1.AerospikeStorageSpec{
		FilesystemVolumePolicy: &v1alpha1.AerospikeVolumePolicy{
			CascadeDelete: boolPtr(false),
		},
	}

	if !ResolveCascadeDelete(vol, spec) {
		t.Error("expected true when per-volume cascadeDelete is true")
	}
}

func TestResolveCascadeDelete_PolicyFallback(t *testing.T) {
	vol := &v1alpha1.VolumeSpec{
		CascadeDelete: false,
		Source: v1alpha1.VolumeSource{
			PersistentVolume: &v1alpha1.PersistentVolumeSpec{Size: "10Gi"},
		},
	}
	spec := &v1alpha1.AerospikeStorageSpec{
		FilesystemVolumePolicy: &v1alpha1.AerospikeVolumePolicy{
			CascadeDelete: boolPtr(true),
		},
	}

	if !ResolveCascadeDelete(vol, spec) {
		t.Error("expected true when policy cascadeDelete is true")
	}
}

func TestResolveCascadeDelete_NoPolicyDefaultFalse(t *testing.T) {
	vol := &v1alpha1.VolumeSpec{
		Source: v1alpha1.VolumeSource{
			PersistentVolume: &v1alpha1.PersistentVolumeSpec{Size: "10Gi"},
		},
	}
	spec := &v1alpha1.AerospikeStorageSpec{}

	if ResolveCascadeDelete(vol, spec) {
		t.Error("expected false when no policy and no per-volume cascadeDelete")
	}
}

// --- getVolumePolicy tests ---

func TestGetVolumePolicy_NonPersistent_ReturnsNil(t *testing.T) {
	vol := &v1alpha1.VolumeSpec{
		Source: v1alpha1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	spec := &v1alpha1.AerospikeStorageSpec{
		FilesystemVolumePolicy: &v1alpha1.AerospikeVolumePolicy{
			InitMethod: v1alpha1.VolumeInitMethodDeleteFiles,
		},
	}

	if getVolumePolicy(vol, spec) != nil {
		t.Error("expected nil for non-persistent volume")
	}
}

func TestGetVolumePolicy_NilStorageSpec(t *testing.T) {
	vol := &v1alpha1.VolumeSpec{
		Source: v1alpha1.VolumeSource{
			PersistentVolume: &v1alpha1.PersistentVolumeSpec{Size: "10Gi"},
		},
	}

	if getVolumePolicy(vol, nil) != nil {
		t.Error("expected nil for nil storageSpec")
	}
}

func TestGetVolumePolicy_BlockMode(t *testing.T) {
	blockPolicy := &v1alpha1.AerospikeVolumePolicy{
		InitMethod: v1alpha1.VolumeInitMethodBlkdiscard,
	}
	fsPolicy := &v1alpha1.AerospikeVolumePolicy{
		InitMethod: v1alpha1.VolumeInitMethodDeleteFiles,
	}

	vol := &v1alpha1.VolumeSpec{
		Source: v1alpha1.VolumeSource{
			PersistentVolume: &v1alpha1.PersistentVolumeSpec{
				Size:       "10Gi",
				VolumeMode: corev1.PersistentVolumeBlock,
			},
		},
	}
	spec := &v1alpha1.AerospikeStorageSpec{
		BlockVolumePolicy:      blockPolicy,
		FilesystemVolumePolicy: fsPolicy,
	}

	result := getVolumePolicy(vol, spec)
	if result != blockPolicy {
		t.Error("expected block policy for Block volume mode")
	}
}
