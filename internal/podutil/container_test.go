package podutil

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"

	v1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
)

const testImage = "aerospike:ce-8.1.1.1"

func newTestCluster() *v1alpha1.AerospikeCECluster {
	return &v1alpha1.AerospikeCECluster{
		Spec: v1alpha1.AerospikeCEClusterSpec{
			Size:  3,
			Image: testImage,
		},
	}
}

func TestBuildInitContainer_UsesClusterImage(t *testing.T) {
	cluster := newTestCluster()

	c := BuildInitContainer(cluster, "my-config", nil, nil, nil)

	if c.Image != "aerospike:ce-8.1.1.1" {
		t.Errorf("expected image %q, got %q", "aerospike:ce-8.1.1.1", c.Image)
	}
}

func TestBuildInitContainer_RunsInitScript(t *testing.T) {
	cluster := newTestCluster()

	c := BuildInitContainer(cluster, "my-config", nil, nil, nil)

	if len(c.Command) != 2 || c.Command[0] != "bash" {
		t.Fatalf("expected command [bash /configmap/aerospike-init.sh], got %v", c.Command)
	}
	if !strings.Contains(c.Command[1], "aerospike-init.sh") {
		t.Errorf("expected command to reference aerospike-init.sh, got %v", c.Command)
	}
}

func TestBuildInitContainer_HasDownwardAPIEnvVars(t *testing.T) {
	cluster := newTestCluster()

	c := BuildInitContainer(cluster, "my-config", nil, nil, nil)

	envMap := make(map[string]*corev1.EnvVar)
	for i := range c.Env {
		envMap[c.Env[i].Name] = &c.Env[i]
	}

	// POD_IP
	if e, ok := envMap["POD_IP"]; !ok {
		t.Error("missing POD_IP env var")
	} else if e.ValueFrom == nil || e.ValueFrom.FieldRef == nil || e.ValueFrom.FieldRef.FieldPath != "status.podIP" {
		t.Errorf("POD_IP should use fieldRef status.podIP, got %+v", e.ValueFrom)
	}

	// POD_NAME
	if e, ok := envMap["POD_NAME"]; !ok {
		t.Error("missing POD_NAME env var")
	} else if e.ValueFrom == nil || e.ValueFrom.FieldRef == nil || e.ValueFrom.FieldRef.FieldPath != "metadata.name" {
		t.Errorf("POD_NAME should use fieldRef metadata.name, got %+v", e.ValueFrom)
	}

	// NODE_IP
	if e, ok := envMap["NODE_IP"]; !ok {
		t.Error("missing NODE_IP env var")
	} else if e.ValueFrom == nil || e.ValueFrom.FieldRef == nil || e.ValueFrom.FieldRef.FieldPath != "status.hostIP" {
		t.Errorf("NODE_IP should use fieldRef status.hostIP, got %+v", e.ValueFrom)
	}
}

func TestBuildInitContainer_HasConfigMapAndConfigMounts(t *testing.T) {
	cluster := newTestCluster()

	c := BuildInitContainer(cluster, "my-config", nil, nil, nil)

	mountNames := make(map[string]string)
	for _, m := range c.VolumeMounts {
		mountNames[m.Name] = m.MountPath
	}

	if path, ok := mountNames[configMapVolumeName]; !ok {
		t.Error("missing configmap volume mount")
	} else if path != configMapVolumeMountPath {
		t.Errorf("configmap mount path = %q, want %q", path, configMapVolumeMountPath)
	}

	if path, ok := mountNames[aerospikeConfigVolumeName]; !ok {
		t.Error("missing aerospike config volume mount")
	} else if path != aerospikeConfigPath {
		t.Errorf("config mount path = %q, want %q", path, aerospikeConfigPath)
	}
}

func TestBuildInitContainer_AppendsExtraVolumeMounts(t *testing.T) {
	cluster := newTestCluster()

	extraMounts := []corev1.VolumeMount{
		{Name: "data-vol", MountPath: "/opt/aerospike/data"},
	}

	c := BuildInitContainer(cluster, "my-config", nil, extraMounts, nil)

	found := false
	for _, m := range c.VolumeMounts {
		if m.Name == "data-vol" && m.MountPath == "/opt/aerospike/data" {
			found = true
			break
		}
	}
	if !found {
		t.Error("extra volume mount not found in init container")
	}
}

func TestBuildInitVolumesEnv_NilStorage(t *testing.T) {
	result := buildInitVolumesEnv(nil)
	if result != "" {
		t.Errorf("expected empty string for nil storage, got %q", result)
	}
}

func TestBuildInitVolumesEnv_NoInitMethods(t *testing.T) {
	storage := &v1alpha1.AerospikeStorageSpec{
		Volumes: []v1alpha1.VolumeSpec{
			{
				Name:       "data",
				InitMethod: v1alpha1.VolumeInitMethodNone,
				Aerospike:  &v1alpha1.AerospikeVolumeAttachment{Path: "/data"},
			},
		},
	}
	result := buildInitVolumesEnv(storage)
	if result != "" {
		t.Errorf("expected empty string for none init method, got %q", result)
	}
}

func TestBuildInitVolumesEnv_WithInitMethods(t *testing.T) {
	storage := &v1alpha1.AerospikeStorageSpec{
		Volumes: []v1alpha1.VolumeSpec{
			{
				Name:       "data",
				InitMethod: v1alpha1.VolumeInitMethodDeleteFiles,
				Aerospike:  &v1alpha1.AerospikeVolumeAttachment{Path: "/opt/aerospike/data"},
			},
			{
				Name:       "index",
				InitMethod: v1alpha1.VolumeInitMethodDD,
				Aerospike:  &v1alpha1.AerospikeVolumeAttachment{Path: "/opt/aerospike/index"},
			},
			{
				Name:       "no-init",
				InitMethod: v1alpha1.VolumeInitMethodNone,
				Aerospike:  &v1alpha1.AerospikeVolumeAttachment{Path: "/no-init"},
			},
		},
	}
	result := buildInitVolumesEnv(storage)
	expected := "deleteFiles:/opt/aerospike/data,dd:/opt/aerospike/index"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestBuildInitVolumesEnv_SkipsVolumesWithoutAerospikePath(t *testing.T) {
	storage := &v1alpha1.AerospikeStorageSpec{
		Volumes: []v1alpha1.VolumeSpec{
			{
				Name:       "sidecar-only",
				InitMethod: v1alpha1.VolumeInitMethodDeleteFiles,
				// No Aerospike attachment
			},
		},
	}
	result := buildInitVolumesEnv(storage)
	if result != "" {
		t.Errorf("expected empty string for volume without aerospike path, got %q", result)
	}
}

func TestBuildInitContainer_INIT_VOLUMES_EnvVar(t *testing.T) {
	cluster := newTestCluster()
	storage := &v1alpha1.AerospikeStorageSpec{
		Volumes: []v1alpha1.VolumeSpec{
			{
				Name:       "data",
				InitMethod: v1alpha1.VolumeInitMethodDeleteFiles,
				Aerospike:  &v1alpha1.AerospikeVolumeAttachment{Path: "/data"},
			},
		},
	}

	c := BuildInitContainer(cluster, "my-config", storage, nil, nil)

	var initVolumes string
	for _, env := range c.Env {
		if env.Name == "INIT_VOLUMES" {
			initVolumes = env.Value
			break
		}
	}
	if initVolumes != "deleteFiles:/data" {
		t.Errorf("INIT_VOLUMES = %q, want %q", initVolumes, "deleteFiles:/data")
	}
}

func TestBuildInitContainer_NoINIT_VOLUMES_WhenEmpty(t *testing.T) {
	cluster := newTestCluster()

	c := BuildInitContainer(cluster, "my-config", nil, nil, nil)

	for _, env := range c.Env {
		if env.Name == "INIT_VOLUMES" {
			t.Error("INIT_VOLUMES should not be present when no volumes need init")
		}
	}
}

// --- Probe tests ---

func TestBuildAerospikeContainer_HasLivenessProbe(t *testing.T) {
	cluster := newTestCluster()
	c := BuildAerospikeContainer(cluster, nil)

	if c.LivenessProbe == nil {
		t.Fatal("expected liveness probe to be set")
	}
	if c.LivenessProbe.TCPSocket == nil {
		t.Error("liveness probe should use TCPSocket handler")
	}
	if c.LivenessProbe.TCPSocket.Port.IntVal != ServicePort {
		t.Errorf("liveness probe port = %d, want %d", c.LivenessProbe.TCPSocket.Port.IntVal, ServicePort)
	}
	// Liveness probe should have more generous timing than readiness
	if c.LivenessProbe.InitialDelaySeconds < c.ReadinessProbe.InitialDelaySeconds {
		t.Error("liveness probe InitialDelaySeconds should be >= readiness probe's")
	}
}

func TestBuildAerospikeContainer_HasReadinessProbe(t *testing.T) {
	cluster := newTestCluster()
	c := BuildAerospikeContainer(cluster, nil)

	if c.ReadinessProbe == nil {
		t.Fatal("expected readiness probe to be set")
	}
	if c.ReadinessProbe.TCPSocket == nil {
		t.Error("readiness probe should use TCPSocket handler")
	}
	if c.ReadinessProbe.TCPSocket.Port.IntVal != ServicePort {
		t.Errorf("readiness probe port = %d, want %d", c.ReadinessProbe.TCPSocket.Port.IntVal, ServicePort)
	}
}

func TestBuildAerospikeContainer_DefaultPorts(t *testing.T) {
	cluster := newTestCluster()
	c := BuildAerospikeContainer(cluster, nil)

	portMap := make(map[string]int32)
	for _, p := range c.Ports {
		portMap[p.Name] = p.ContainerPort
	}

	expectedPorts := map[string]int32{
		"service":   ServicePort,
		"fabric":    FabricPort,
		"heartbeat": HeartbeatPort,
		"info":      InfoPort,
	}

	for name, expected := range expectedPorts {
		if got, ok := portMap[name]; !ok {
			t.Errorf("missing port %q", name)
		} else if got != expected {
			t.Errorf("port %q = %d, want %d", name, got, expected)
		}
	}
}
