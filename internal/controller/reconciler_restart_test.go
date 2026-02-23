package controller

import (
	"testing"

	asdbcev1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
)

func TestGetDirtyVolumes_NilStorage(t *testing.T) {
	result := getDirtyVolumes(nil)
	if result != nil {
		t.Errorf("expected nil for nil storage, got %v", result)
	}
}

func TestGetDirtyVolumes_NoWipeMethod(t *testing.T) {
	storage := &asdbcev1alpha1.AerospikeStorageSpec{
		Volumes: []asdbcev1alpha1.VolumeSpec{
			{Name: "data"},
			{Name: "logs"},
		},
	}
	result := getDirtyVolumes(storage)
	if len(result) != 0 {
		t.Errorf("expected empty for volumes without wipe method, got %v", result)
	}
}

func TestGetDirtyVolumes_WithWipeMethod(t *testing.T) {
	storage := &asdbcev1alpha1.AerospikeStorageSpec{
		Volumes: []asdbcev1alpha1.VolumeSpec{
			{Name: "data", WipeMethod: asdbcev1alpha1.VolumeWipeMethodDeleteFiles},
			{Name: "logs"},
			{Name: "index", WipeMethod: asdbcev1alpha1.VolumeWipeMethodBlkdiscard},
		},
	}
	result := getDirtyVolumes(storage)
	if len(result) != 2 {
		t.Fatalf("expected 2 dirty volumes, got %d: %v", len(result), result)
	}
	if result[0] != "data" || result[1] != "index" {
		t.Errorf("expected [data, index], got %v", result)
	}
}

func TestGetDirtyVolumes_WipeMethodNone(t *testing.T) {
	storage := &asdbcev1alpha1.AerospikeStorageSpec{
		Volumes: []asdbcev1alpha1.VolumeSpec{
			{Name: "data", WipeMethod: asdbcev1alpha1.VolumeWipeMethodNone},
		},
	}
	result := getDirtyVolumes(storage)
	if len(result) != 0 {
		t.Errorf("expected empty for wipe method 'none', got %v", result)
	}
}

func TestGetDirtyVolumes_GlobalPolicy(t *testing.T) {
	storage := &asdbcev1alpha1.AerospikeStorageSpec{
		FilesystemVolumePolicy: &asdbcev1alpha1.AerospikeVolumePolicy{
			WipeMethod: asdbcev1alpha1.VolumeWipeMethodDeleteFiles,
		},
		Volumes: []asdbcev1alpha1.VolumeSpec{
			{
				Name: "data",
				Source: asdbcev1alpha1.VolumeSource{
					PersistentVolume: &asdbcev1alpha1.PersistentVolumeSpec{Size: "10Gi"},
				},
			}, // should inherit global filesystem policy
		},
	}
	result := getDirtyVolumes(storage)
	if len(result) != 1 {
		t.Fatalf("expected 1 dirty volume from global policy, got %d: %v", len(result), result)
	}
	if result[0] != "data" {
		t.Errorf("expected [data], got %v", result)
	}
}
