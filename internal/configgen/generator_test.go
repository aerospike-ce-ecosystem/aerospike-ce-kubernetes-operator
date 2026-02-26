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

	// CE does not support the security stanza, so it must be omitted.
	if strings.Contains(result, "security {") {
		t.Errorf("expected security section to be omitted for CE, got:\n%s", result)
	}
	assertContains(t, result, "service {")
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
		name     string
		input    any
		expected string
	}{
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"int", 42, "42"},
		{"int32", int32(256), "256"},
		{"int64", int64(100), "100"},
		{"float64 fractional", float64(3.14), "3.14"},
		{"float64 whole", float64(100), "100"},
		{"float32 fractional", float32(2.5), "2.5"},
		{"float32 whole", float32(42), "42"},
		{"string", "hello", "hello"},
		{"nil", nil, ""},
		{"negative int", -1, "-1"},
		{"zero", 0, "0"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatValue(tc.input)
			if got != tc.expected {
				t.Errorf("formatValue(%v) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestSortedKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected []string
	}{
		{"nil map", nil, nil},
		{"empty map", map[string]any{}, []string{}},
		{"single key", map[string]any{"a": 1}, []string{"a"}},
		{"multiple keys sorted", map[string]any{
			"zebra":  1,
			"alpha":  2,
			"middle": 3,
		}, []string{"alpha", "middle", "zebra"}},
		{"keys with hyphens", map[string]any{
			"mesh-seed-address-port": 1,
			"cluster-name":           2,
			"address":                3,
		}, []string{"address", "cluster-name", "mesh-seed-address-port"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sortedKeys(tc.input)
			if tc.expected == nil {
				if len(got) != 0 {
					t.Errorf("sortedKeys(%v) = %v, want empty", tc.input, got)
				}
				return
			}
			if len(got) != len(tc.expected) {
				t.Fatalf("sortedKeys(%v) = %v, want %v", tc.input, got, tc.expected)
			}
			for i := range got {
				if got[i] != tc.expected[i] {
					t.Errorf("sortedKeys(%v)[%d] = %q, want %q", tc.input, i, got[i], tc.expected[i])
				}
			}
		})
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

func TestGenerateConfig_ConsoleLogging(t *testing.T) {
	config := map[string]any{
		"logging": []any{
			map[string]any{
				"name":    "console",
				"context": "any info",
			},
		},
	}

	result, err := GenerateConfig(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, result, "logging {")
	assertContains(t, result, "console {")
	assertContains(t, result, "context any info")

	// Must NOT contain "file console"
	if strings.Contains(result, "file console") {
		t.Errorf("console logging should not generate 'file console', got:\n%s", result)
	}
}

func TestGenerateConfig_StderrLogging(t *testing.T) {
	config := map[string]any{
		"logging": []any{
			map[string]any{
				"name":    "stderr",
				"context": "any info",
			},
		},
	}

	result, err := GenerateConfig(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, result, "console {")
	if strings.Contains(result, "file stderr") {
		t.Errorf("stderr logging should generate 'console' block, not 'file stderr', got:\n%s", result)
	}
}

func TestGenerateConfig_SyslogLogging(t *testing.T) {
	config := map[string]any{
		"logging": []any{
			map[string]any{
				"name":    "syslog",
				"context": "any info",
			},
		},
	}

	result, err := GenerateConfig(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, result, "syslog {")
	if strings.Contains(result, "file syslog") {
		t.Errorf("syslog logging should generate 'syslog' block, not 'file syslog', got:\n%s", result)
	}
}

func TestGenerateConfig_MixedLogging(t *testing.T) {
	config := map[string]any{
		"logging": []any{
			map[string]any{
				"name":    "console",
				"context": "any info",
			},
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

	assertContains(t, result, "console {")
	assertContains(t, result, "file /var/log/aerospike/aerospike.log {")
}

func TestGenerateConfForPod_SecuritySectionSkipped(t *testing.T) {
	config := map[string]any{
		"service": map[string]any{
			"cluster-name": "test",
		},
		"security": map[string]any{},
		"network": map[string]any{
			"heartbeat": map[string]any{
				"mode": "mesh",
				"port": 3002,
			},
		},
	}

	result, err := GenerateConfForPod(config, "pod-0", "svc", "ns", []string{"pod-0"}, 3002)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(result, "security") {
		t.Errorf("security section should be skipped for CE, got:\n%s", result)
	}
}

func TestGenerateConfForPod_CustomHeartbeatPort(t *testing.T) {
	config := map[string]any{
		"network": map[string]any{
			"heartbeat": map[string]any{
				"mode": "mesh",
				"port": 4002,
			},
		},
	}

	podNames := []string{"pod-0", "pod-1"}
	result, err := GenerateConfForPod(config, "pod-0", "svc", "default", podNames, 4002)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, result, "mesh-seed-address-port pod-0.svc.default.svc.cluster.local 4002")
	assertContains(t, result, "mesh-seed-address-port pod-1.svc.default.svc.cluster.local 4002")
}

func TestGenerateConfig_StorageEngineInference(t *testing.T) {
	// Test that storage-engine type is inferred from "file" key presence
	config := map[string]any{
		"namespaces": []any{
			map[string]any{
				"name": "test",
				"storage-engine": map[string]any{
					"file":     "/data/test.dat",
					"filesize": "4G",
				},
			},
		},
	}

	result, err := GenerateConfig(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, result, "storage-engine device {")
	assertContains(t, result, "file /data/test.dat")
}

func TestGenerateConfig_EmptyConfigMap(t *testing.T) {
	config := map[string]any{}

	result, err := GenerateConfig(config)
	if err != nil {
		t.Fatalf("unexpected error for empty config map: %v", err)
	}

	// An empty config map should produce an empty (or whitespace-only) output.
	if strings.TrimSpace(result) != "" {
		t.Errorf("expected empty output for empty config map, got:\n%s", result)
	}
}

func TestGenerateConfig_NilValuesInMap(t *testing.T) {
	config := map[string]any{
		"service": map[string]any{
			"cluster-name": "test",
			"proto-fd-max": nil,
		},
	}

	result, err := GenerateConfig(config)
	if err != nil {
		t.Fatalf("unexpected error for config with nil value: %v", err)
	}

	// The service section and cluster-name should still be generated.
	assertContains(t, result, "service {")
	assertContains(t, result, "cluster-name test")
	// nil values should be silently skipped (not rendered).
	if strings.Contains(result, "proto-fd-max") {
		t.Errorf("expected nil value 'proto-fd-max' to be skipped, got:\n%s", result)
	}
}

func TestGenerateConfig_TopLevelNilValue(t *testing.T) {
	config := map[string]any{
		"some-key": nil,
	}

	// Should not panic; nil values are silently skipped.
	result, err := GenerateConfig(config)
	if err != nil {
		t.Fatalf("unexpected error for top-level nil value: %v", err)
	}

	if strings.Contains(result, "some-key") {
		t.Errorf("expected top-level nil value 'some-key' to be skipped, got:\n%s", result)
	}
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected output to contain %q, got:\n%s", substr, s)
	}
}
