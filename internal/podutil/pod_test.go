package podutil

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/utils"
)

const (
	hostnameTopologyKey   = "kubernetes.io/hostname"
	exporterContainerName = "aerospike-prometheus-exporter"
)

func boolPtr(b bool) *bool { return &b }

// --- shouldInjectAntiAffinity tests ---

func TestShouldInjectAntiAffinity_NilPodSpec(t *testing.T) {
	cluster := &v1alpha1.AerospikeCECluster{
		Spec: v1alpha1.AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
		},
	}
	if shouldInjectAntiAffinity(cluster) {
		t.Error("should not inject anti-affinity when podSpec is nil")
	}
}

func TestShouldInjectAntiAffinity_MultiPodPerHostNil(t *testing.T) {
	cluster := &v1alpha1.AerospikeCECluster{
		Spec: v1alpha1.AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			PodSpec: &v1alpha1.AerospikeCEPodSpec{
				MultiPodPerHost: nil, // nil = not set
			},
		},
	}
	if shouldInjectAntiAffinity(cluster) {
		t.Error("should not inject anti-affinity when multiPodPerHost is nil")
	}
}

func TestShouldInjectAntiAffinity_MultiPodPerHostFalse(t *testing.T) {
	cluster := &v1alpha1.AerospikeCECluster{
		Spec: v1alpha1.AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			PodSpec: &v1alpha1.AerospikeCEPodSpec{
				MultiPodPerHost: boolPtr(false),
			},
		},
	}
	if !shouldInjectAntiAffinity(cluster) {
		t.Error("should inject anti-affinity when multiPodPerHost=false")
	}
}

func TestShouldInjectAntiAffinity_MultiPodPerHostTrue(t *testing.T) {
	cluster := &v1alpha1.AerospikeCECluster{
		Spec: v1alpha1.AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			PodSpec: &v1alpha1.AerospikeCEPodSpec{
				MultiPodPerHost: boolPtr(true),
			},
		},
	}
	if shouldInjectAntiAffinity(cluster) {
		t.Error("should not inject anti-affinity when multiPodPerHost=true")
	}
}

// --- injectPodAntiAffinity tests ---

func TestInjectPodAntiAffinity_CreatesAffinityFromNil(t *testing.T) {
	podSpec := &corev1.PodSpec{}
	injectPodAntiAffinity(podSpec, "my-cluster")

	if podSpec.Affinity == nil || podSpec.Affinity.PodAntiAffinity == nil {
		t.Fatal("expected affinity to be set")
	}

	required := podSpec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	if len(required) != 1 {
		t.Fatalf("expected 1 anti-affinity term, got %d", len(required))
	}

	term := required[0]
	if term.TopologyKey != hostnameTopologyKey {
		t.Errorf("topologyKey = %q, want %q", term.TopologyKey, hostnameTopologyKey)
	}

	expectedLabels := utils.SelectorLabelsForCluster("my-cluster")
	if term.LabelSelector == nil {
		t.Fatal("expected labelSelector to be set")
	}
	for k, v := range expectedLabels {
		if term.LabelSelector.MatchLabels[k] != v {
			t.Errorf("label %q = %q, want %q", k, term.LabelSelector.MatchLabels[k], v)
		}
	}
}

func TestInjectPodAntiAffinity_PreservesExistingNodeAffinity(t *testing.T) {
	existingNodeAffinity := &corev1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
			NodeSelectorTerms: []corev1.NodeSelectorTerm{
				{
					MatchExpressions: []corev1.NodeSelectorRequirement{
						{Key: "topology.kubernetes.io/zone", Operator: corev1.NodeSelectorOpIn, Values: []string{"us-east-1a"}},
					},
				},
			},
		},
	}

	podSpec := &corev1.PodSpec{
		Affinity: &corev1.Affinity{
			NodeAffinity: existingNodeAffinity,
		},
	}

	injectPodAntiAffinity(podSpec, "my-cluster")

	// Node affinity should be preserved
	if podSpec.Affinity.NodeAffinity != existingNodeAffinity {
		t.Error("existing node affinity was overwritten")
	}

	// Pod anti-affinity should be added
	if podSpec.Affinity.PodAntiAffinity == nil {
		t.Fatal("expected pod anti-affinity to be set")
	}
	if len(podSpec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution) != 1 {
		t.Error("expected 1 anti-affinity term")
	}
}

func TestInjectPodAntiAffinity_AppendsToExistingAntiAffinity(t *testing.T) {
	podSpec := &corev1.PodSpec{
		Affinity: &corev1.Affinity{
			PodAntiAffinity: &corev1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
					{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"existing": "rule"},
						},
						TopologyKey: "topology.kubernetes.io/zone",
					},
				},
			},
		},
	}

	injectPodAntiAffinity(podSpec, "my-cluster")

	required := podSpec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	if len(required) != 2 {
		t.Fatalf("expected 2 anti-affinity terms, got %d", len(required))
	}

	// First should be the existing one
	if required[0].TopologyKey != "topology.kubernetes.io/zone" {
		t.Error("existing anti-affinity term was not preserved")
	}

	// Second should be the injected one
	if required[1].TopologyKey != hostnameTopologyKey {
		t.Error("injected anti-affinity term not found")
	}
}

// --- buildExporterSidecar tests ---

func TestBuildExporterSidecar_Basic(t *testing.T) {
	monitoring := &v1alpha1.AerospikeMonitoringSpec{
		Enabled:       true,
		ExporterImage: "aerospike/aerospike-prometheus-exporter:latest",
		Port:          9145,
	}

	c := buildExporterSidecar(monitoring)

	if c.Name != exporterContainerName {
		t.Errorf("name = %q, want %q", c.Name, exporterContainerName)
	}
	if c.Image != monitoring.ExporterImage {
		t.Errorf("image = %q, want %q", c.Image, monitoring.ExporterImage)
	}

	if len(c.Ports) != 1 || c.Ports[0].ContainerPort != 9145 {
		t.Errorf("expected port 9145, got %v", c.Ports)
	}

	envMap := make(map[string]string)
	for _, e := range c.Env {
		envMap[e.Name] = e.Value
	}
	if envMap["AS_HOST"] != "localhost" {
		t.Errorf("AS_HOST = %q, want %q", envMap["AS_HOST"], "localhost")
	}
	if envMap["AS_PORT"] != "3000" {
		t.Errorf("AS_PORT = %q, want %q", envMap["AS_PORT"], "3000")
	}
}

func TestBuildExporterSidecar_WithResources(t *testing.T) {
	monitoring := &v1alpha1.AerospikeMonitoringSpec{
		Enabled:       true,
		ExporterImage: "exporter:v1",
		Port:          9145,
		Resources: &corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("50m"),
			},
		},
	}

	c := buildExporterSidecar(monitoring)

	if c.Resources.Requests.Cpu().String() != "50m" {
		t.Errorf("expected CPU request 50m, got %s", c.Resources.Requests.Cpu().String())
	}
}

func TestBuildExporterSidecar_CustomPort(t *testing.T) {
	monitoring := &v1alpha1.AerospikeMonitoringSpec{
		Enabled:       true,
		ExporterImage: "exporter:v1",
		Port:          9999,
	}

	c := buildExporterSidecar(monitoring)

	if c.Ports[0].ContainerPort != 9999 {
		t.Errorf("expected port 9999, got %d", c.Ports[0].ContainerPort)
	}
}

// --- BuildPodTemplateSpec integration tests ---

func TestBuildPodTemplateSpec_MonitoringSidecarInjected(t *testing.T) {
	cluster := &v1alpha1.AerospikeCECluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"},
		Spec: v1alpha1.AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			Monitoring: &v1alpha1.AerospikeMonitoringSpec{
				Enabled:       true,
				ExporterImage: "exporter:v1",
				Port:          9145,
			},
		},
	}

	pt := BuildPodTemplateSpec(cluster, nil, 0, "test-config", "abc123")

	found := false
	for _, c := range pt.Spec.Containers {
		if c.Name == exporterContainerName {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected exporter sidecar container to be injected")
	}
}

func TestBuildPodTemplateSpec_NoMonitoringSidecarWhenDisabled(t *testing.T) {
	cluster := &v1alpha1.AerospikeCECluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"},
		Spec: v1alpha1.AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
		},
	}

	pt := BuildPodTemplateSpec(cluster, nil, 0, "test-config", "abc123")

	for _, c := range pt.Spec.Containers {
		if c.Name == exporterContainerName {
			t.Error("exporter sidecar should not be injected when monitoring is not set")
		}
	}
}

func TestBuildPodTemplateSpec_BandwidthAnnotations(t *testing.T) {
	cluster := &v1alpha1.AerospikeCECluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"},
		Spec: v1alpha1.AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			BandwidthConfig: &v1alpha1.BandwidthConfig{
				Ingress: "1Gbps",
				Egress:  "500Mbps",
			},
		},
	}

	pt := BuildPodTemplateSpec(cluster, nil, 0, "test-config", "abc123")

	annotations := pt.Annotations
	if annotations["kubernetes.io/ingress-bandwidth"] != "1Gbps" {
		t.Errorf("ingress-bandwidth = %q, want %q", annotations["kubernetes.io/ingress-bandwidth"], "1Gbps")
	}
	if annotations["kubernetes.io/egress-bandwidth"] != "500Mbps" {
		t.Errorf("egress-bandwidth = %q, want %q", annotations["kubernetes.io/egress-bandwidth"], "500Mbps")
	}
}

func TestBuildPodTemplateSpec_NoBandwidthAnnotationsWhenNotSet(t *testing.T) {
	cluster := &v1alpha1.AerospikeCECluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"},
		Spec: v1alpha1.AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
		},
	}

	pt := BuildPodTemplateSpec(cluster, nil, 0, "test-config", "abc123")

	if _, ok := pt.Annotations["kubernetes.io/ingress-bandwidth"]; ok {
		t.Error("ingress-bandwidth annotation should not be present")
	}
	if _, ok := pt.Annotations["kubernetes.io/egress-bandwidth"]; ok {
		t.Error("egress-bandwidth annotation should not be present")
	}
}

func TestBuildPodTemplateSpec_AntiAffinityInjectedWhenMultiPodPerHostFalse(t *testing.T) {
	cluster := &v1alpha1.AerospikeCECluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"},
		Spec: v1alpha1.AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			PodSpec: &v1alpha1.AerospikeCEPodSpec{
				MultiPodPerHost: boolPtr(false),
			},
		},
	}

	pt := BuildPodTemplateSpec(cluster, nil, 0, "test-config", "abc123")

	affinity := pt.Spec.Affinity
	if affinity == nil || affinity.PodAntiAffinity == nil {
		t.Fatal("expected anti-affinity to be set")
	}
	required := affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	if len(required) == 0 {
		t.Fatal("expected at least 1 required anti-affinity term")
	}
	if required[0].TopologyKey != hostnameTopologyKey {
		t.Error("expected topologyKey kubernetes.io/hostname")
	}
}

func TestBuildPodTemplateSpec_NoAntiAffinityWhenMultiPodPerHostTrue(t *testing.T) {
	cluster := &v1alpha1.AerospikeCECluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"},
		Spec: v1alpha1.AerospikeCEClusterSpec{
			Size:  3,
			Image: "aerospike:ce-8.1.1.1",
			PodSpec: &v1alpha1.AerospikeCEPodSpec{
				MultiPodPerHost: boolPtr(true),
			},
		},
	}

	pt := BuildPodTemplateSpec(cluster, nil, 0, "test-config", "abc123")

	if pt.Spec.Affinity != nil && pt.Spec.Affinity.PodAntiAffinity != nil {
		required := pt.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution
		for _, r := range required {
			if r.TopologyKey == hostnameTopologyKey {
				t.Error("should not inject anti-affinity when multiPodPerHost=true")
			}
		}
	}
}

func TestBuildPodTemplateSpec_InitContainerUsesClusterImage(t *testing.T) {
	cluster := &v1alpha1.AerospikeCECluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"},
		Spec: v1alpha1.AerospikeCEClusterSpec{
			Size:  1,
			Image: "aerospike:ce-7.2.0.6",
		},
	}

	pt := BuildPodTemplateSpec(cluster, nil, 0, "test-config", "abc123")

	if len(pt.Spec.InitContainers) == 0 {
		t.Fatal("expected at least 1 init container")
	}
	initC := pt.Spec.InitContainers[0]
	if initC.Image != "aerospike:ce-7.2.0.6" {
		t.Errorf("init container image = %q, want %q", initC.Image, "aerospike:ce-7.2.0.6")
	}
	if initC.Name != InitContainerName {
		t.Errorf("init container name = %q, want %q", initC.Name, InitContainerName)
	}
}
