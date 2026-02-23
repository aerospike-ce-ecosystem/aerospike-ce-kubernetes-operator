package v1alpha1

import (
	"context"
	"slices"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func boolPtr(b bool) *bool { return &b }

// --- Defaulter tests ---

func TestDefaultMonitoring_SetsDefaults(t *testing.T) {
	d := &AerospikeCEClusterDefaulter{}
	cluster := &AerospikeCECluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			Monitoring: &AerospikeMonitoringSpec{
				Enabled: true,
				ServiceMonitor: &ServiceMonitorSpec{
					Enabled: true,
				},
			},
		},
	}

	if err := d.Default(context.Background(), cluster); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := cluster.Spec.Monitoring
	if m.ExporterImage != defaultExporterImage {
		t.Errorf("ExporterImage = %q, want %q", m.ExporterImage, defaultExporterImage)
	}
	if m.Port != defaultExporterPort {
		t.Errorf("Port = %d, want %d", m.Port, defaultExporterPort)
	}
	if m.ServiceMonitor.Interval != defaultScrapeInterval {
		t.Errorf("Interval = %q, want %q", m.ServiceMonitor.Interval, defaultScrapeInterval)
	}
}

func TestDefaultMonitoring_NoopWhenDisabled(t *testing.T) {
	d := &AerospikeCEClusterDefaulter{}
	cluster := &AerospikeCECluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
		},
	}

	if err := d.Default(context.Background(), cluster); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cluster.Spec.Monitoring != nil {
		t.Error("monitoring should remain nil when not set")
	}
}

func TestDefaultMonitoring_PreservesCustomValues(t *testing.T) {
	d := &AerospikeCEClusterDefaulter{}
	cluster := &AerospikeCECluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			Monitoring: &AerospikeMonitoringSpec{
				Enabled:       true,
				ExporterImage: "custom-exporter:v2",
				Port:          9999,
			},
		},
	}

	if err := d.Default(context.Background(), cluster); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cluster.Spec.Monitoring.ExporterImage != "custom-exporter:v2" {
		t.Errorf("custom ExporterImage was overwritten: %q", cluster.Spec.Monitoring.ExporterImage)
	}
	if cluster.Spec.Monitoring.Port != 9999 {
		t.Errorf("custom Port was overwritten: %d", cluster.Spec.Monitoring.Port)
	}
}

// --- HostNetwork defaulting tests ---

func TestDefaultHostNetwork_SetsMultiPodPerHostFalse(t *testing.T) {
	d := &AerospikeCEClusterDefaulter{}
	cluster := &AerospikeCECluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			PodSpec: &AerospikeCEPodSpec{
				HostNetwork: true,
			},
		},
	}

	if err := d.Default(context.Background(), cluster); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cluster.Spec.PodSpec.MultiPodPerHost == nil {
		t.Fatal("multiPodPerHost should be set")
	}
	if *cluster.Spec.PodSpec.MultiPodPerHost {
		t.Error("multiPodPerHost should default to false for hostNetwork=true")
	}
}

func TestDefaultHostNetwork_SetsDNSPolicy(t *testing.T) {
	d := &AerospikeCEClusterDefaulter{}
	cluster := &AerospikeCECluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			PodSpec: &AerospikeCEPodSpec{
				HostNetwork: true,
			},
		},
	}

	if err := d.Default(context.Background(), cluster); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cluster.Spec.PodSpec.DNSPolicy != corev1.DNSClusterFirstWithHostNet {
		t.Errorf("dnsPolicy = %q, want %q", cluster.Spec.PodSpec.DNSPolicy, corev1.DNSClusterFirstWithHostNet)
	}
}

func TestDefaultHostNetwork_PreservesExplicitMultiPodPerHost(t *testing.T) {
	d := &AerospikeCEClusterDefaulter{}
	cluster := &AerospikeCECluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			PodSpec: &AerospikeCEPodSpec{
				HostNetwork:     true,
				MultiPodPerHost: boolPtr(true),
			},
		},
	}

	if err := d.Default(context.Background(), cluster); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !*cluster.Spec.PodSpec.MultiPodPerHost {
		t.Error("should not override explicitly set multiPodPerHost=true")
	}
}

func TestDefaultHostNetwork_NoopWhenHostNetworkFalse(t *testing.T) {
	d := &AerospikeCEClusterDefaulter{}
	cluster := &AerospikeCECluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			PodSpec: &AerospikeCEPodSpec{
				HostNetwork: false,
			},
		},
	}

	if err := d.Default(context.Background(), cluster); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cluster.Spec.PodSpec.MultiPodPerHost != nil {
		t.Error("multiPodPerHost should remain nil when hostNetwork=false")
	}
	if cluster.Spec.PodSpec.DNSPolicy != "" {
		t.Errorf("dnsPolicy should remain empty when hostNetwork=false, got %q", cluster.Spec.PodSpec.DNSPolicy)
	}
}

func TestDefaultHostNetwork_PreservesExplicitDNSPolicy(t *testing.T) {
	d := &AerospikeCEClusterDefaulter{}
	cluster := &AerospikeCECluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			PodSpec: &AerospikeCEPodSpec{
				HostNetwork: true,
				DNSPolicy:   corev1.DNSDefault,
			},
		},
	}

	if err := d.Default(context.Background(), cluster); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cluster.Spec.PodSpec.DNSPolicy != corev1.DNSDefault {
		t.Errorf("should not override explicitly set dnsPolicy, got %q", cluster.Spec.PodSpec.DNSPolicy)
	}
}

// --- Validator tests ---

func TestValidate_HostNetworkMultiPodPerHostWarning(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			PodSpec: &AerospikeCEPodSpec{
				HostNetwork:     true,
				MultiPodPerHost: boolPtr(true),
			},
		},
	}

	warnings, err := v.validate(cluster)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := slices.Contains(warnings, "hostNetwork=true with multiPodPerHost=true may cause port conflicts")
	if !found {
		t.Error("expected warning about hostNetwork+multiPodPerHost, got none")
	}
}

func TestValidate_HostNetworkDNSPolicyWarning(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			PodSpec: &AerospikeCEPodSpec{
				HostNetwork: true,
				DNSPolicy:   corev1.DNSDefault,
			},
		},
	}

	warnings, err := v.validate(cluster)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := slices.Contains(warnings, "hostNetwork=true with dnsPolicy other than ClusterFirstWithHostNet may cause DNS resolution issues")
	if !found {
		t.Errorf("expected DNS warning, got warnings: %v", warnings)
	}
}

func TestValidate_NoWarningsForValidHostNetwork(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			PodSpec: &AerospikeCEPodSpec{
				HostNetwork:     true,
				MultiPodPerHost: boolPtr(false),
				DNSPolicy:       corev1.DNSClusterFirstWithHostNet,
			},
		},
	}

	warnings, err := v.validate(cluster)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
}

func TestValidate_NoHostNetworkNoWarnings(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
		},
	}

	warnings, err := v.validate(cluster)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
}

// --- Existing validation tests (ensure no regressions) ---

func TestValidate_CEClusterSizeLimit(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  9,
			Image: "aerospike:ce-8.1.1.1",
		},
	}

	_, err := v.validate(cluster)
	if err == nil {
		t.Error("expected error for size > 8")
	}
}

func TestValidate_EnterpriseImageRejected(t *testing.T) {
	v := &AerospikeCEClusterValidator{}

	tests := []struct {
		image string
	}{
		{"aerospike:ee-8.0.0.1_1"},
		{"aerospike-enterprise:8.0.0"},
	}

	for _, tc := range tests {
		cluster := &AerospikeCECluster{
			Spec: AerospikeCEClusterSpec{
				Size:  3,
				Image: tc.image,
			},
		}
		_, err := v.validate(cluster)
		if err == nil {
			t.Errorf("expected error for enterprise image %q", tc.image)
		}
	}
}

func TestValidate_XDRRejected(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			AerospikeConfig: &AerospikeConfigSpec{
				Value: map[string]any{
					"xdr": map[string]any{},
				},
			},
		},
	}

	_, err := v.validate(cluster)
	if err == nil {
		t.Error("expected error for XDR section")
	}
}

func TestValidate_TLSRejected(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			AerospikeConfig: &AerospikeConfigSpec{
				Value: map[string]any{
					"tls": map[string]any{},
				},
			},
		},
	}

	_, err := v.validate(cluster)
	if err == nil {
		t.Error("expected error for TLS section")
	}
}

func TestValidate_MaxNamespaces(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			AerospikeConfig: &AerospikeConfigSpec{
				Value: map[string]any{
					"namespaces": []any{
						map[string]any{"name": "ns1"},
						map[string]any{"name": "ns2"},
						map[string]any{"name": "ns3"},
					},
				},
			},
		},
	}

	_, err := v.validate(cluster)
	if err == nil {
		t.Error("expected error for > 2 namespaces")
	}
}

func TestValidate_DuplicateRackIDs(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  4,
			Image: "aerospike:ce-8.1.1.1",
			RackConfig: &RackConfig{
				Racks: []Rack{
					{ID: 1},
					{ID: 1},
				},
			},
		},
	}

	_, err := v.validate(cluster)
	if err == nil {
		t.Error("expected error for duplicate rack IDs")
	}
}

// --- Enterprise-only namespace config validation tests ---

func TestValidate_EnterpriseOnlyNamespaceKeys(t *testing.T) {
	v := &AerospikeCEClusterValidator{}

	enterpriseKeys := []string{
		"compression", "compression-level", "durable-delete", "fast-restart",
		"index-type", "sindex-type", "rack-id", "strong-consistency",
		"tomb-raider-eligible-age", "tomb-raider-period",
	}

	for _, key := range enterpriseKeys {
		cluster := &AerospikeCECluster{
			Spec: AerospikeCEClusterSpec{
				Size:  3,
				Image: "aerospike:ce-8.1.1.1",
				AerospikeConfig: &AerospikeConfigSpec{
					Value: map[string]any{
						"namespaces": []any{
							map[string]any{
								"name": "test",
								key:    "some-value",
							},
						},
					},
				},
			},
		}

		_, err := v.validate(cluster)
		if err == nil {
			t.Errorf("expected error for enterprise-only namespace key %q", key)
		}
	}
}

func TestValidate_HeartbeatModeMulticastRejected(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			AerospikeConfig: &AerospikeConfigSpec{
				Value: map[string]any{
					"network": map[string]any{
						"heartbeat": map[string]any{
							"mode": "multicast",
							"port": 3002,
						},
					},
				},
			},
		},
	}

	_, err := v.validate(cluster)
	if err == nil {
		t.Error("expected error for heartbeat mode=multicast (CE only supports mesh)")
	}
}

func TestValidate_HeartbeatModeMeshAccepted(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			AerospikeConfig: &AerospikeConfigSpec{
				Value: map[string]any{
					"network": map[string]any{
						"heartbeat": map[string]any{
							"mode": "mesh",
							"port": 3002,
						},
					},
				},
			},
		},
	}

	_, err := v.validate(cluster)
	if err != nil {
		t.Errorf("unexpected error for heartbeat mode=mesh: %v", err)
	}
}

func TestValidate_DataInMemoryWarning(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			AerospikeConfig: &AerospikeConfigSpec{
				Value: map[string]any{
					"namespaces": []any{
						map[string]any{
							"name": "myns",
							"storage-engine": map[string]any{
								"type":           "device",
								"data-in-memory": true,
								"file":           "/data/myns.dat",
								"filesize":       "4G",
							},
						},
					},
				},
			},
		},
	}

	warnings, err := v.validate(cluster)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, w := range warnings {
		if strings.Contains(w, "data-in-memory=true") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about data-in-memory, got warnings: %v", warnings)
	}
}

func TestValidate_ReplicationFactorOutOfRange(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			AerospikeConfig: &AerospikeConfigSpec{
				Value: map[string]any{
					"namespaces": []any{
						map[string]any{
							"name":               "test",
							"replication-factor": 5,
						},
					},
				},
			},
		},
	}

	_, err := v.validate(cluster)
	if err == nil {
		t.Error("expected error for replication-factor > 4")
	}
}

func TestValidate_ValidNamespaceConfigAccepted(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			AerospikeConfig: &AerospikeConfigSpec{
				Value: map[string]any{
					"namespaces": []any{
						map[string]any{
							"name":               "test",
							"replication-factor": 2,
							"memory-size":        "4G",
							"storage-engine": map[string]any{
								"type": "memory",
							},
						},
					},
				},
			},
		},
	}

	_, err := v.validate(cluster)
	if err != nil {
		t.Errorf("unexpected error for valid namespace config: %v", err)
	}
}

// --- Security section validation tests ---

func TestValidate_SecuritySectionRejected(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			AerospikeConfig: &AerospikeConfigSpec{
				Value: map[string]any{
					"security": map[string]any{},
				},
			},
		},
	}

	_, err := v.validate(cluster)
	if err == nil {
		t.Error("expected error for security section in CE")
	}
	if !strings.Contains(err.Error(), "security") {
		t.Errorf("error should mention 'security', got: %v", err)
	}
}

// --- ACL validation tests ---

func TestValidate_ACLWithAdminUser(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			AerospikeAccessControl: &AerospikeAccessControlSpec{
				Users: []AerospikeUserSpec{
					{
						Name:       "admin",
						SecretName: "admin-secret",
						Roles:      []string{"sys-admin", "user-admin"},
					},
				},
			},
		},
	}

	_, err := v.validate(cluster)
	if err != nil {
		t.Errorf("unexpected error for valid ACL config: %v", err)
	}
}

func TestValidate_ACLWithoutAdminUser(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			AerospikeAccessControl: &AerospikeAccessControlSpec{
				Users: []AerospikeUserSpec{
					{
						Name:       "reader",
						SecretName: "reader-secret",
						Roles:      []string{"read"},
					},
				},
			},
		},
	}

	_, err := v.validate(cluster)
	if err == nil {
		t.Error("expected error for ACL without admin user")
	}
	if !strings.Contains(err.Error(), "sys-admin") {
		t.Errorf("error should mention 'sys-admin', got: %v", err)
	}
}

func TestValidate_ACLWithSplitAdminRoles(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			AerospikeAccessControl: &AerospikeAccessControlSpec{
				Users: []AerospikeUserSpec{
					{
						Name:       "sysadmin",
						SecretName: "sa-secret",
						Roles:      []string{"sys-admin"},
					},
					{
						Name:       "useradmin",
						SecretName: "ua-secret",
						Roles:      []string{"user-admin"},
					},
				},
			},
		},
	}

	_, err := v.validate(cluster)
	if err == nil {
		t.Error("expected error when sys-admin and user-admin are on different users")
	}
}

// --- RollingUpdateBatchSize validation tests ---

func int32Ptr(i int32) *int32 { return &i }

func TestValidate_RollingUpdateBatchSizeGreaterThanSize(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:                   3,
			Image:                  "aerospike:ce-8.1.1.1",
			RollingUpdateBatchSize: int32Ptr(5),
		},
	}

	warnings, err := v.validate(cluster)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, w := range warnings {
		if strings.Contains(w, "rollingUpdateBatchSize") && strings.Contains(w, "greater than cluster size") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about batchSize > clusterSize, got warnings: %v", warnings)
	}
}

func TestValidate_RollingUpdateBatchSizeEqualToSize(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:                   3,
			Image:                  "aerospike:ce-8.1.1.1",
			RollingUpdateBatchSize: int32Ptr(3),
		},
	}

	warnings, err := v.validate(cluster)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// batchSize == clusterSize should not produce a warning
	for _, w := range warnings {
		if strings.Contains(w, "rollingUpdateBatchSize") {
			t.Errorf("unexpected batchSize warning: %v", w)
		}
	}
}

func TestValidate_RollingUpdateBatchSizeLessThanSize(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:                   4,
			Image:                  "aerospike:ce-8.1.1.1",
			RollingUpdateBatchSize: int32Ptr(2),
		},
	}

	warnings, err := v.validate(cluster)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, w := range warnings {
		if strings.Contains(w, "rollingUpdateBatchSize") {
			t.Errorf("unexpected batchSize warning: %v", w)
		}
	}
}

func TestValidate_RollingUpdateBatchSizeNil(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
		},
	}

	warnings, err := v.validate(cluster)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, w := range warnings {
		if strings.Contains(w, "rollingUpdateBatchSize") {
			t.Errorf("unexpected batchSize warning: %v", w)
		}
	}
}

// --- Defaulter core behavior tests ---

func TestDefault_SetsClusterName(t *testing.T) {
	d := &AerospikeCEClusterDefaulter{}
	cluster := &AerospikeCECluster{
		ObjectMeta: metav1.ObjectMeta{Name: "my-cluster", Namespace: "default"},
		Spec: AerospikeCEClusterSpec{
			Size:  1,
			Image: "aerospike:ce-8.1.1.1",
		},
	}

	if err := d.Default(context.Background(), cluster); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svc := cluster.Spec.AerospikeConfig.Value["service"].(map[string]any)
	if svc["cluster-name"] != "my-cluster" {
		t.Errorf("cluster-name = %v, want 'my-cluster'", svc["cluster-name"])
	}
	if svc["proto-fd-max"] != defaultProtoFdMax {
		t.Errorf("proto-fd-max = %v, want %d", svc["proto-fd-max"], defaultProtoFdMax)
	}
}

func TestDefault_SetsNetworkPorts(t *testing.T) {
	d := &AerospikeCEClusterDefaulter{}
	cluster := &AerospikeCECluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: AerospikeCEClusterSpec{
			Size:  1,
			Image: "aerospike:ce-8.1.1.1",
		},
	}

	if err := d.Default(context.Background(), cluster); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	net := cluster.Spec.AerospikeConfig.Value["network"].(map[string]any)
	svcNet := net["service"].(map[string]any)
	hb := net["heartbeat"].(map[string]any)
	fabric := net["fabric"].(map[string]any)

	if svcNet["port"] != defaultServicePort {
		t.Errorf("service port = %v, want %d", svcNet["port"], defaultServicePort)
	}
	if hb["port"] != defaultHeartbeatPort {
		t.Errorf("heartbeat port = %v, want %d", hb["port"], defaultHeartbeatPort)
	}
	if hb["mode"] != defaultHeartbeatMode {
		t.Errorf("heartbeat mode = %v, want %q", hb["mode"], defaultHeartbeatMode)
	}
	if fabric["port"] != defaultFabricPort {
		t.Errorf("fabric port = %v, want %d", fabric["port"], defaultFabricPort)
	}
}

func TestDefault_PreservesExistingConfig(t *testing.T) {
	d := &AerospikeCEClusterDefaulter{}
	cluster := &AerospikeCECluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: AerospikeCEClusterSpec{
			Size:  1,
			Image: "aerospike:ce-8.1.1.1",
			AerospikeConfig: &AerospikeConfigSpec{
				Value: map[string]any{
					"service": map[string]any{
						"cluster-name": "custom-name",
						"proto-fd-max": 20000,
					},
					"network": map[string]any{
						"service": map[string]any{
							"port": 4000,
						},
					},
				},
			},
		},
	}

	if err := d.Default(context.Background(), cluster); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svc := cluster.Spec.AerospikeConfig.Value["service"].(map[string]any)
	if svc["cluster-name"] != "custom-name" {
		t.Errorf("should preserve custom cluster-name, got %v", svc["cluster-name"])
	}
	if svc["proto-fd-max"] != 20000 {
		t.Errorf("should preserve custom proto-fd-max, got %v", svc["proto-fd-max"])
	}

	net := cluster.Spec.AerospikeConfig.Value["network"].(map[string]any)
	svcNet := net["service"].(map[string]any)
	if svcNet["port"] != 4000 {
		t.Errorf("should preserve custom service port, got %v", svcNet["port"])
	}
}

// --- Storage validation tests ---

func TestValidate_Storage_NoSourceSpecified(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			Storage: &AerospikeStorageSpec{
				Volumes: []VolumeSpec{
					{Name: "data"},
				},
			},
		},
	}

	_, err := v.validate(cluster)
	if err == nil {
		t.Error("expected error when no volume source is specified")
	}
	if !strings.Contains(err.Error(), "exactly one volume source") {
		t.Errorf("error should mention 'exactly one volume source', got: %v", err)
	}
}

func TestValidate_Storage_MultipleSourcesRejected(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			Storage: &AerospikeStorageSpec{
				Volumes: []VolumeSpec{
					{
						Name: "data",
						Source: VolumeSource{
							PersistentVolume: &PersistentVolumeSpec{Size: "10Gi"},
							EmptyDir:         &corev1.EmptyDirVolumeSource{},
						},
					},
				},
			},
		},
	}

	_, err := v.validate(cluster)
	if err == nil {
		t.Error("expected error when multiple volume sources are specified")
	}
	if !strings.Contains(err.Error(), "only one volume source") {
		t.Errorf("error should mention 'only one volume source', got: %v", err)
	}
}

func TestValidate_Storage_HostPathWarning(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			Storage: &AerospikeStorageSpec{
				Volumes: []VolumeSpec{
					{
						Name: "host-data",
						Source: VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{Path: "/mnt/data"},
						},
					},
				},
			},
		},
	}

	warnings, err := v.validate(cluster)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, w := range warnings {
		if strings.Contains(w, "hostPath") && strings.Contains(w, "not recommended") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected hostPath warning, got warnings: %v", warnings)
	}
}

func TestValidate_Storage_SubPathAndSubPathExprMutuallyExclusive_Aerospike(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			Storage: &AerospikeStorageSpec{
				Volumes: []VolumeSpec{
					{
						Name: "data",
						Source: VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
						Aerospike: &AerospikeVolumeAttachment{
							Path:        "/data",
							SubPath:     "sub",
							SubPathExpr: "$(POD_NAME)",
						},
					},
				},
			},
		},
	}

	_, err := v.validate(cluster)
	if err == nil {
		t.Error("expected error when both subPath and subPathExpr are set")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("error should mention 'mutually exclusive', got: %v", err)
	}
}

func TestValidate_Storage_SubPathAndSubPathExprMutuallyExclusive_Sidecar(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			Storage: &AerospikeStorageSpec{
				Volumes: []VolumeSpec{
					{
						Name: "data",
						Source: VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
						Sidecars: []VolumeAttachment{
							{
								ContainerName: "exporter",
								Path:          "/data",
								SubPath:       "sub",
								SubPathExpr:   "$(POD_NAME)",
							},
						},
					},
				},
			},
		},
	}

	_, err := v.validate(cluster)
	if err == nil {
		t.Error("expected error when sidecar has both subPath and subPathExpr")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("error should mention 'mutually exclusive', got: %v", err)
	}
}

func TestValidate_Storage_SubPathAndSubPathExprMutuallyExclusive_InitContainer(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			Storage: &AerospikeStorageSpec{
				Volumes: []VolumeSpec{
					{
						Name: "data",
						Source: VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
						InitContainers: []VolumeAttachment{
							{
								ContainerName: "init",
								Path:          "/data",
								SubPath:       "sub",
								SubPathExpr:   "$(POD_NAME)",
							},
						},
					},
				},
			},
		},
	}

	_, err := v.validate(cluster)
	if err == nil {
		t.Error("expected error when init container has both subPath and subPathExpr")
	}
}

func TestValidate_Storage_DeleteLocalStorageWithoutClasses(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			Storage: &AerospikeStorageSpec{
				DeleteLocalStorageOnRestart: boolPtr(true),
			},
		},
	}

	_, err := v.validate(cluster)
	if err == nil {
		t.Error("expected error when deleteLocalStorageOnRestart is true but localStorageClasses is empty")
	}
	if !strings.Contains(err.Error(), "localStorageClasses") {
		t.Errorf("error should mention 'localStorageClasses', got: %v", err)
	}
}

func TestValidate_Storage_LocalStorageClassesWithoutDeleteFlag(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			Storage: &AerospikeStorageSpec{
				LocalStorageClasses: []string{"local-path"},
				Volumes: []VolumeSpec{
					{
						Name: "data",
						Source: VolumeSource{
							PersistentVolume: &PersistentVolumeSpec{Size: "10Gi"},
						},
					},
				},
			},
		},
	}

	warnings, err := v.validate(cluster)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, w := range warnings {
		if strings.Contains(w, "localStorageClasses") && strings.Contains(w, "deleteLocalStorageOnRestart") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about localStorageClasses without deleteLocalStorageOnRestart, got: %v", warnings)
	}
}

func TestValidate_Storage_ValidConfig(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			Storage: &AerospikeStorageSpec{
				LocalStorageClasses:         []string{"local-path"},
				DeleteLocalStorageOnRestart: boolPtr(true),
				Volumes: []VolumeSpec{
					{
						Name: "data",
						Source: VolumeSource{
							PersistentVolume: &PersistentVolumeSpec{
								Size:         "10Gi",
								StorageClass: "local-path",
							},
						},
						Aerospike:  &AerospikeVolumeAttachment{Path: "/data"},
						InitMethod: VolumeInitMethodDeleteFiles,
						WipeMethod: VolumeWipeMethodDeleteFiles,
					},
					{
						Name: "logs",
						Source: VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
						Aerospike: &AerospikeVolumeAttachment{Path: "/logs"},
					},
				},
			},
		},
	}

	_, err := v.validate(cluster)
	if err != nil {
		t.Errorf("unexpected error for valid storage config: %v", err)
	}
}

func TestValidate_Storage_DeleteLocalStorageFalse_NoError(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			Storage: &AerospikeStorageSpec{
				DeleteLocalStorageOnRestart: boolPtr(false),
			},
		},
	}

	_, err := v.validate(cluster)
	if err != nil {
		t.Errorf("unexpected error when deleteLocalStorageOnRestart is false: %v", err)
	}
}

func TestValidate_Storage_CascadeDeleteOnNonPersistent(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			Storage: &AerospikeStorageSpec{
				Volumes: []VolumeSpec{
					{
						Name: "data",
						Source: VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
						CascadeDelete: boolPtr(true),
					},
				},
			},
		},
	}

	// CascadeDelete on non-persistent should not cause error (just ignored)
	_, err := v.validate(cluster)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_Storage_GlobalPoliciesAccepted(t *testing.T) {
	v := &AerospikeCEClusterValidator{}
	cluster := &AerospikeCECluster{
		Spec: AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			Storage: &AerospikeStorageSpec{
				FilesystemVolumePolicy: &AerospikeVolumePolicy{
					InitMethod:    VolumeInitMethodDeleteFiles,
					WipeMethod:    VolumeWipeMethodDeleteFiles,
					CascadeDelete: boolPtr(true),
				},
				BlockVolumePolicy: &AerospikeVolumePolicy{
					InitMethod:    VolumeInitMethodBlkdiscard,
					WipeMethod:    VolumeWipeMethodBlkdiscard,
					CascadeDelete: boolPtr(false),
				},
				Volumes: []VolumeSpec{
					{
						Name: "data",
						Source: VolumeSource{
							PersistentVolume: &PersistentVolumeSpec{Size: "10Gi"},
						},
					},
				},
			},
		},
	}

	_, err := v.validate(cluster)
	if err != nil {
		t.Errorf("unexpected error for valid storage with global policies: %v", err)
	}
}

// --- isEnterpriseTag tests ---

func TestIsEnterpriseTag(t *testing.T) {
	tests := []struct {
		image    string
		expected bool
	}{
		{"aerospike:ce-8.1.1.1", false},
		{"aerospike:ee-8.0.0.1_1", true},
		{"aerospike:EE-8.0.0.1", true},
		{"aerospike:latest", false},
		{"aerospike", false},
		{"myrepo/aerospike:ce-8.1.1.1", false},
		{"myrepo/aerospike:ee-8.1.1.1", true},
	}

	for _, tc := range tests {
		got := isEnterpriseTag(tc.image)
		if got != tc.expected {
			t.Errorf("isEnterpriseTag(%q) = %v, want %v", tc.image, got, tc.expected)
		}
	}
}
