package configgen

import (
	"strings"
	"testing"
)

func TestGenerateNamespaceSections_SingleMemoryNamespace(t *testing.T) {
	namespaces := []any{
		map[string]any{
			"name":               "test-ns",
			"replication-factor": 2,
			"storage-engine": map[string]any{
				"type": "memory",
			},
		},
	}

	result, err := generateNamespaceSections(namespaces)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, result, "namespace test-ns {")
	assertContains(t, result, "replication-factor 2")
	assertContains(t, result, "storage-engine memory")
}

func TestGenerateNamespaceSections_DeviceStorageWithFile(t *testing.T) {
	namespaces := []any{
		map[string]any{
			"name": "data-ns",
			"storage-engine": map[string]any{
				"type":     "device",
				"file":     "/opt/aerospike/data/test.dat",
				"filesize": 4294967296,
			},
		},
	}

	result, err := generateNamespaceSections(namespaces)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, result, "namespace data-ns {")
	assertContains(t, result, "storage-engine device {")
	assertContains(t, result, "file /opt/aerospike/data/test.dat")
	assertContains(t, result, "filesize 4294967296")
}

func TestGenerateNamespaceSections_MissingName(t *testing.T) {
	namespaces := []any{
		map[string]any{
			"replication-factor": 2,
		},
	}

	_, err := generateNamespaceSections(namespaces)
	if err == nil {
		t.Fatal("expected error for missing name, got nil")
	}
	if !strings.Contains(err.Error(), "missing 'name' key") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestGenerateNamespaceSections_NonMapEntry(t *testing.T) {
	namespaces := []any{"not-a-map"}

	_, err := generateNamespaceSections(namespaces)
	if err == nil {
		t.Fatal("expected error for non-map entry, got nil")
	}
	if !strings.Contains(err.Error(), "is not a map") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestGenerateNamespaceSections_EmptyNameString(t *testing.T) {
	namespaces := []any{
		map[string]any{
			"name": "",
		},
	}

	_, err := generateNamespaceSections(namespaces)
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
}

func TestGenerateNamespaceSections_MultipleNamespaces(t *testing.T) {
	namespaces := []any{
		map[string]any{
			"name":               "ns1",
			"replication-factor": 1,
			"storage-engine":     "memory",
		},
		map[string]any{
			"name":               "ns2",
			"replication-factor": 2,
			"storage-engine":     "memory",
		},
	}

	result, err := generateNamespaceSections(namespaces)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, result, "namespace ns1 {")
	assertContains(t, result, "namespace ns2 {")
}

func TestGenerateNamespaceSections_EmptySlice(t *testing.T) {
	result, err := generateNamespaceSections([]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Fatalf("expected empty string for no namespaces, got %q", result)
	}
}

func TestGenerateNamespaceSections_NestedSubContext(t *testing.T) {
	namespaces := []any{
		map[string]any{
			"name": "test-ns",
			"index-type": map[string]any{
				"type": "shmem",
			},
			"storage-engine": "memory",
		},
	}

	result, err := generateNamespaceSections(namespaces)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, result, "index-type {")
	assertContains(t, result, "type shmem")
}

func TestInferStorageEngineType(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected string
	}{
		{"explicit device type", map[string]any{"type": "device"}, "device"},
		{"infer from file key", map[string]any{"file": "/data/test.dat", "filesize": 1024}, "device"},
		{"infer from device key", map[string]any{"device": "/dev/sda"}, "device"},
		{"default memory", map[string]any{"data-size": 4294967296}, "memory"},
		{"empty map", map[string]any{}, "memory"},
		{"explicit memory type", map[string]any{"type": "memory"}, "memory"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := inferStorageEngineType(tc.input); got != tc.expected {
				t.Errorf("inferStorageEngineType() = %s, want %s", got, tc.expected)
			}
		})
	}
}

func TestWriteStorageEngine_SimpleStringValue(t *testing.T) {
	var b strings.Builder
	writeStorageEngine(&b, "memory", 1)
	result := b.String()
	if !strings.Contains(result, "\tstorage-engine memory") {
		t.Errorf("unexpected result: %q", result)
	}
}

func TestWriteStorageEngine_MapNoRemainingKeys(t *testing.T) {
	var b strings.Builder
	writeStorageEngine(&b, map[string]any{"type": "memory"}, 1)
	result := b.String()
	if !strings.Contains(result, "\tstorage-engine memory\n") {
		t.Errorf("expected simple format, got: %q", result)
	}
	if strings.Contains(result, "{") {
		t.Errorf("should not have braces for type-only storage-engine, got: %q", result)
	}
}
