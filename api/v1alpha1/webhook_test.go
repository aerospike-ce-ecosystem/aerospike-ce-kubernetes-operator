package v1alpha1

import (
	"context"
	"slices"
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
