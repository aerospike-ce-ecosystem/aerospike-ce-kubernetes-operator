package configgen

import (
	"strings"
	"testing"
)

func TestGenerateConfig_BasicServiceNetworkNamespace(t *testing.T) {
	config := map[string]any{
		"service": map[string]any{
			"cluster-name": "myCluster",
			"proto-fd-max": 15000,
		},
		"network": map[string]any{
			"service": map[string]any{
				"address": "any",
				"port":    3000,
			},
			"heartbeat": map[string]any{
				"mode": "mesh",
				"port": 3002,
			},
			"fabric": map[string]any{
				"address": "any",
				"port":    3001,
			},
		},
		"namespaces": []any{
			map[string]any{
				"name":               "test",
				"replication-factor": 2,
				"storage-engine": map[string]any{
					"type": "memory",
				},
			},
		},
	}

	result, err := GenerateConfig(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify key sections exist.
	assertContains(t, result, "service {")
	assertContains(t, result, "cluster-name myCluster")
	assertContains(t, result, "proto-fd-max 15000")
	assertContains(t, result, "network {")
	assertContains(t, result, "namespace test {")
	assertContains(t, result, "replication-factor 2")
	assertContains(t, result, "storage-engine memory")
}

func TestGenerateConfForPod_MeshSeeds(t *testing.T) {
	config := map[string]any{
		"service": map[string]any{
			"cluster-name": "testCluster",
		},
		"network": map[string]any{
			"service": map[string]any{
				"address": "any",
				"port":    3000,
			},
			"heartbeat": map[string]any{
				"mode": "mesh",
				"port": 3002,
			},
			"fabric": map[string]any{
				"address": "any",
				"port":    3001,
			},
		},
	}

	podNames := []string{"asc-0", "asc-1", "asc-2"}

	result, err := GenerateConfForPod(config, "asc-0", "asc-headless", "default", podNames, 3002)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify mesh seeds are injected for all pods.
	assertContains(t, result, "mesh-seed-address-port asc-0.asc-headless.default.svc.cluster.local 3002")
	assertContains(t, result, "mesh-seed-address-port asc-1.asc-headless.default.svc.cluster.local 3002")
	assertContains(t, result, "mesh-seed-address-port asc-2.asc-headless.default.svc.cluster.local 3002")

	// Verify heartbeat config is still present.
	assertContains(t, result, "mode mesh")
	assertContains(t, result, "port 3002")
}

func TestGenerateNamespaces_WithStorageEngine(t *testing.T) {
	config := map[string]any{
		"namespaces": []any{
			map[string]any{
				"name":               "ns1",
				"replication-factor": 2,
				"storage-engine": map[string]any{
					"type":     "device",
					"file":     "/opt/aerospike/data/ns1.dat",
					"filesize": "4G",
				},
			},
			map[string]any{
				"name":               "ns2",
				"replication-factor": 1,
				"storage-engine": map[string]any{
					"type": "memory",
				},
			},
		},
	}

	result, err := GenerateConfig(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, result, "namespace ns1 {")
	assertContains(t, result, "storage-engine device {")
	assertContains(t, result, "file /opt/aerospike/data/ns1.dat")
	assertContains(t, result, "filesize 4G")
	assertContains(t, result, "namespace ns2 {")
	assertContains(t, result, "storage-engine memory")
}

func TestGenerateConfig_SecuritySection(t *testing.T) {
	config := map[string]any{
		"service": map[string]any{
			"cluster-name": "secure-cluster",
		},
		"security": map[string]any{},
	}

	result, err := GenerateConfig(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, result, "security {")
	assertContains(t, result, "}")
}

func TestGenerateConfig_LoggingSection(t *testing.T) {
	config := map[string]any{
		"logging": []any{
			map[string]any{
				"name":    "/var/log/aerospike/aerospike.log",
				"context": "any info",
			},
		},
	}

	result, err := GenerateConfig(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, result, "logging {")
	assertContains(t, result, "file /var/log/aerospike/aerospike.log {")
	assertContains(t, result, "context any info")
}

func TestGenerateConfig_BoolValues(t *testing.T) {
	config := map[string]any{
		"service": map[string]any{
			"feature-key-file":   "/etc/aerospike/features.conf",
			"migrate-fill-delay": 0,
		},
		"namespaces": []any{
			map[string]any{
				"name":                   "test",
				"allow-ttl-without-nsup": true,
				"disallow-null-setname":  false,
			},
		},
	}

	result, err := GenerateConfig(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, result, "allow-ttl-without-nsup true")
	assertContains(t, result, "disallow-null-setname false")
}

func TestGenerateConfig_NilConfig(t *testing.T) {
	_, err := GenerateConfig(nil)
	if err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		input    any
		expected string
	}{
		{true, "true"},
		{false, "false"},
		{42, "42"},
		{int64(100), "100"},
		{float64(3.14), "3.14"},
		{float64(100), "100"},
		{"hello", "hello"},
	}
	for _, tc := range tests {
		got := formatValue(tc.input)
		if got != tc.expected {
			t.Errorf("formatValue(%v) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestGenerateConfig_EmptyNamespaceList(t *testing.T) {
	config := map[string]any{
		"namespaces": []any{},
	}

	result, err := GenerateConfig(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should produce no namespace blocks.
	if strings.Contains(result, "namespace") {
		t.Error("expected no namespace block for empty list")
	}
}

func TestGenerateConfig_NamespaceMissingName(t *testing.T) {
	config := map[string]any{
		"namespaces": []any{
			map[string]any{
				"replication-factor": 2,
			},
		},
	}

	_, err := GenerateConfig(config)
	if err == nil {
		t.Fatal("expected error for namespace without name")
	}
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected output to contain %q, got:\n%s", substr, s)
	}
}
