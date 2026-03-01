/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package template

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	asdbcev1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
)

func TestMergeTemplateSpec_NilInputs(t *testing.T) {
	result := MergeTemplateSpec(nil, nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestMergeTemplateSpec_NilBase(t *testing.T) {
	override := &asdbcev1alpha1.AerospikeCEClusterTemplateSpec{
		RackConfig: &asdbcev1alpha1.TemplateRackConfig{MaxRacksPerNode: 2},
	}
	result := MergeTemplateSpec(nil, override)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.RackConfig == nil || result.RackConfig.MaxRacksPerNode != 2 {
		t.Errorf("expected MaxRacksPerNode=2, got %+v", result.RackConfig)
	}
}

func TestMergeTemplateSpec_NilOverride(t *testing.T) {
	base := &asdbcev1alpha1.AerospikeCEClusterTemplateSpec{
		RackConfig: &asdbcev1alpha1.TemplateRackConfig{MaxRacksPerNode: 1},
	}
	result := MergeTemplateSpec(base, nil)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.RackConfig == nil || result.RackConfig.MaxRacksPerNode != 1 {
		t.Errorf("expected MaxRacksPerNode=1, got %+v", result.RackConfig)
	}
}

func TestMergeTemplateSpec_OverrideTakesPrecedence(t *testing.T) {
	base := &asdbcev1alpha1.AerospikeCEClusterTemplateSpec{
		RackConfig: &asdbcev1alpha1.TemplateRackConfig{MaxRacksPerNode: 1},
		Scheduling: &asdbcev1alpha1.TemplateScheduling{
			PodAntiAffinityLevel: asdbcev1alpha1.PodAntiAffinityPreferred,
		},
	}
	override := &asdbcev1alpha1.AerospikeCEClusterTemplateSpec{
		Scheduling: &asdbcev1alpha1.TemplateScheduling{
			PodAntiAffinityLevel: asdbcev1alpha1.PodAntiAffinityRequired,
		},
	}
	result := MergeTemplateSpec(base, override)

	if result.RackConfig == nil || result.RackConfig.MaxRacksPerNode != 1 {
		t.Errorf("expected base RackConfig to be preserved")
	}
	if result.Scheduling == nil || result.Scheduling.PodAntiAffinityLevel != asdbcev1alpha1.PodAntiAffinityRequired {
		t.Errorf("expected override scheduling to take precedence")
	}
}

func TestMergeTemplateSpec_AerospikeConfigDeepMerge(t *testing.T) {
	base := &asdbcev1alpha1.AerospikeCEClusterTemplateSpec{
		AerospikeConfig: &asdbcev1alpha1.TemplateAerospikeConfig{
			Service: &asdbcev1alpha1.AerospikeConfigSpec{
				Value: map[string]any{
					"proto-fd-max": 15000,
					"base-key":     "base-value",
				},
			},
		},
	}
	override := &asdbcev1alpha1.AerospikeCEClusterTemplateSpec{
		AerospikeConfig: &asdbcev1alpha1.TemplateAerospikeConfig{
			Service: &asdbcev1alpha1.AerospikeConfigSpec{
				Value: map[string]any{
					"proto-fd-max": 20000,
					"override-key": "override-value",
				},
			},
		},
	}
	result := MergeTemplateSpec(base, override)

	if result.AerospikeConfig == nil || result.AerospikeConfig.Service == nil {
		t.Fatal("expected AerospikeConfig.Service to be set")
	}
	// override-key should be present
	if result.AerospikeConfig.Service.Value["override-key"] != "override-value" {
		t.Errorf("override-key missing from merged service")
	}
	// base-key should be present (from base, not overridden)
	if result.AerospikeConfig.Service.Value["base-key"] != "base-value" {
		t.Errorf("base-key should be preserved in merged service")
	}
	// override value should win
	if result.AerospikeConfig.Service.Value["proto-fd-max"] != 20000 {
		t.Errorf("expected proto-fd-max=20000, got %v", result.AerospikeConfig.Service.Value["proto-fd-max"])
	}
}

func TestApplyAerospikeConfig_NamespaceDefaults(t *testing.T) {
	cluster := &asdbcev1alpha1.AerospikeCECluster{}
	cluster.Spec.AerospikeConfig = &asdbcev1alpha1.AerospikeConfigSpec{
		Value: map[string]any{
			"namespaces": []any{
				map[string]any{"name": "test", "replication-factor": 1},
			},
		},
	}

	tmplConfig := &asdbcev1alpha1.TemplateAerospikeConfig{
		NamespaceDefaults: &asdbcev1alpha1.AerospikeConfigSpec{
			Value: map[string]any{
				"memory-size":        int64(1073741824),
				"replication-factor": 2, // should be overridden by namespace's own value
			},
		},
	}

	applyAerospikeConfig(tmplConfig, cluster)

	nsList, ok := cluster.Spec.AerospikeConfig.Value["namespaces"].([]any)
	if !ok || len(nsList) == 0 {
		t.Fatal("expected namespaces to be set")
	}
	nsMap, ok := nsList[0].(map[string]any)
	if !ok {
		t.Fatal("expected namespace to be a map")
	}

	// memory-size from defaults should be applied
	if nsMap["memory-size"] != int64(1073741824) {
		t.Errorf("expected memory-size to be applied from defaults, got %v", nsMap["memory-size"])
	}
	// replication-factor should keep the namespace's own value (1), not the default (2)
	if nsMap["replication-factor"] != 1 {
		t.Errorf("expected replication-factor=1 (from namespace), got %v", nsMap["replication-factor"])
	}
}

func TestNeedsResync(t *testing.T) {
	tests := []struct {
		name    string
		cluster *asdbcev1alpha1.AerospikeCECluster
		want    bool
	}{
		{
			name:    "no templateRef",
			cluster: &asdbcev1alpha1.AerospikeCECluster{},
			want:    false,
		},
		{
			name: "templateRef set, no snapshot",
			cluster: &asdbcev1alpha1.AerospikeCECluster{
				Spec: asdbcev1alpha1.AerospikeCEClusterSpec{
					TemplateRef: &asdbcev1alpha1.TemplateRef{Name: "prod"},
				},
			},
			want: true,
		},
		{
			name: "templateRef set, snapshot exists, no annotation",
			cluster: &asdbcev1alpha1.AerospikeCECluster{
				Spec: asdbcev1alpha1.AerospikeCEClusterSpec{
					TemplateRef: &asdbcev1alpha1.TemplateRef{Name: "prod"},
				},
				Status: asdbcev1alpha1.AerospikeCEClusterStatus{
					TemplateSnapshot: &asdbcev1alpha1.TemplateSnapshotStatus{Name: "prod"},
				},
			},
			want: false,
		},
		{
			name: "templateRef set, snapshot exists, resync annotation",
			cluster: &asdbcev1alpha1.AerospikeCECluster{
				Spec: asdbcev1alpha1.AerospikeCEClusterSpec{
					TemplateRef: &asdbcev1alpha1.TemplateRef{Name: "prod"},
				},
				Status: asdbcev1alpha1.AerospikeCEClusterStatus{
					TemplateSnapshot: &asdbcev1alpha1.TemplateSnapshotStatus{Name: "prod"},
				},
			},
			want: true,
		},
	}

	// Add annotation to the last test case
	tests[3].cluster.Annotations = map[string]string{AnnotationResyncTemplate: "true"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NeedsResync(tt.cluster)
			if got != tt.want {
				t.Errorf("NeedsResync() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplyTemplate_Resources(t *testing.T) {
	cluster := &asdbcev1alpha1.AerospikeCECluster{
		Spec: asdbcev1alpha1.AerospikeCEClusterSpec{
			Size:  1,
			Image: "aerospike:ce-8.1.1.1",
		},
	}

	// Resources should be applied when not set in cluster spec.
	templateSpec := &asdbcev1alpha1.AerospikeCEClusterTemplateSpec{}
	ApplyTemplate(templateSpec, cluster)

	// With nil resources in template, nothing should be set.
	if cluster.Spec.PodSpec != nil && cluster.Spec.PodSpec.AerospikeContainerSpec != nil &&
		cluster.Spec.PodSpec.AerospikeContainerSpec.Resources != nil {
		t.Errorf("expected no resources to be set when template has none")
	}
}

// ---- applyStorage tests ----

func TestApplyStorage_CreatesVolumeFromTemplate(t *testing.T) {
	cluster := &asdbcev1alpha1.AerospikeCECluster{}
	qty := resource.MustParse("50Gi")
	tmplStorage := &asdbcev1alpha1.TemplateStorage{
		StorageClassName: "local-path",
		Resources: corev1.VolumeResourceRequirements{
			Requests: corev1.ResourceList{corev1.ResourceStorage: qty},
		},
	}

	applyStorage(tmplStorage, cluster)

	if cluster.Spec.Storage == nil || len(cluster.Spec.Storage.Volumes) == 0 {
		t.Fatal("expected storage volume to be created")
	}
	vol := cluster.Spec.Storage.Volumes[0]
	if vol.Name != defaultDataVolumeName {
		t.Errorf("expected volume name %q, got %q", defaultDataVolumeName, vol.Name)
	}
	if vol.Source.PersistentVolume == nil {
		t.Fatal("expected PersistentVolume to be set")
	}
	if vol.Source.PersistentVolume.StorageClass != "local-path" {
		t.Errorf("expected storageClass=local-path, got %q", vol.Source.PersistentVolume.StorageClass)
	}
	if vol.Source.PersistentVolume.Size != "50Gi" {
		t.Errorf("expected size=50Gi, got %q", vol.Source.PersistentVolume.Size)
	}
}

func TestApplyStorage_SkipsIfVolumeAlreadySet(t *testing.T) {
	cluster := &asdbcev1alpha1.AerospikeCECluster{
		Spec: asdbcev1alpha1.AerospikeCEClusterSpec{
			Storage: &asdbcev1alpha1.AerospikeStorageSpec{
				Volumes: []asdbcev1alpha1.VolumeSpec{
					{Name: "existing"},
				},
			},
		},
	}
	tmplStorage := &asdbcev1alpha1.TemplateStorage{StorageClassName: "other"}

	applyStorage(tmplStorage, cluster)

	if len(cluster.Spec.Storage.Volumes) != 1 || cluster.Spec.Storage.Volumes[0].Name != "existing" {
		t.Error("expected existing volume to be preserved when cluster already has volumes")
	}
}

func TestApplyStorage_DefaultsApplied(t *testing.T) {
	cluster := &asdbcev1alpha1.AerospikeCECluster{}
	tmplStorage := &asdbcev1alpha1.TemplateStorage{StorageClassName: "standard"}

	applyStorage(tmplStorage, cluster)

	if cluster.Spec.Storage == nil || len(cluster.Spec.Storage.Volumes) == 0 {
		t.Fatal("expected volume to be created")
	}
	pv := cluster.Spec.Storage.Volumes[0].Source.PersistentVolume
	if pv.Size != "1Gi" {
		t.Errorf("expected default size=1Gi, got %q", pv.Size)
	}
	if pv.VolumeMode != corev1.PersistentVolumeFilesystem {
		t.Errorf("expected default volumeMode=Filesystem, got %v", pv.VolumeMode)
	}
	if len(pv.AccessModes) != 1 || pv.AccessModes[0] != corev1.ReadWriteOnce {
		t.Errorf("expected default accessModes=[ReadWriteOnce], got %v", pv.AccessModes)
	}
}

// ---- applyScheduling tests ----

func TestApplyScheduling_Tolerations(t *testing.T) {
	cluster := &asdbcev1alpha1.AerospikeCECluster{}
	scheduling := &asdbcev1alpha1.TemplateScheduling{
		Tolerations: []corev1.Toleration{
			{Key: "aerospike", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule},
		},
	}

	applyScheduling(scheduling, cluster)

	if cluster.Spec.PodSpec == nil || len(cluster.Spec.PodSpec.Tolerations) == 0 {
		t.Fatal("expected tolerations to be applied")
	}
	if cluster.Spec.PodSpec.Tolerations[0].Key != "aerospike" {
		t.Errorf("expected toleration key=aerospike, got %q", cluster.Spec.PodSpec.Tolerations[0].Key)
	}
}

func TestApplyScheduling_TolerationsNotOverriddenIfAlreadySet(t *testing.T) {
	cluster := &asdbcev1alpha1.AerospikeCECluster{
		Spec: asdbcev1alpha1.AerospikeCEClusterSpec{
			PodSpec: &asdbcev1alpha1.AerospikeCEPodSpec{
				Tolerations: []corev1.Toleration{{Key: "existing"}},
			},
		},
	}
	scheduling := &asdbcev1alpha1.TemplateScheduling{
		Tolerations: []corev1.Toleration{{Key: "from-template"}},
	}

	applyScheduling(scheduling, cluster)

	if cluster.Spec.PodSpec.Tolerations[0].Key != "existing" {
		t.Error("existing tolerations should not be overridden by template")
	}
}

func TestApplyScheduling_TopologySpreadConstraints(t *testing.T) {
	cluster := &asdbcev1alpha1.AerospikeCECluster{}
	scheduling := &asdbcev1alpha1.TemplateScheduling{
		TopologySpreadConstraints: []corev1.TopologySpreadConstraint{
			{MaxSkew: 1, TopologyKey: "zone", WhenUnsatisfiable: corev1.DoNotSchedule},
		},
	}

	applyScheduling(scheduling, cluster)

	if cluster.Spec.PodSpec == nil || len(cluster.Spec.PodSpec.TopologySpreadConstraints) == 0 {
		t.Fatal("expected TopologySpreadConstraints to be applied")
	}
	if cluster.Spec.PodSpec.TopologySpreadConstraints[0].TopologyKey != "zone" {
		t.Errorf("expected topologyKey=zone, got %q", cluster.Spec.PodSpec.TopologySpreadConstraints[0].TopologyKey)
	}
}

func TestApplyScheduling_PodManagementPolicy(t *testing.T) {
	cluster := &asdbcev1alpha1.AerospikeCECluster{}
	scheduling := &asdbcev1alpha1.TemplateScheduling{
		PodManagementPolicy: appsv1.OrderedReadyPodManagement,
	}

	applyScheduling(scheduling, cluster)

	if cluster.Spec.PodSpec == nil || cluster.Spec.PodSpec.PodManagementPolicy != appsv1.OrderedReadyPodManagement {
		t.Errorf("expected PodManagementPolicy=OrderedReady, got %v", cluster.Spec.PodSpec.PodManagementPolicy)
	}
}

func TestDeepMergeMapBaseFirst(t *testing.T) {
	base := map[string]any{
		"key1": "base-value",
		"key2": "base-only",
		"nested": map[string]any{
			"a": 1,
			"b": "base",
		},
	}
	override := map[string]any{
		"key1": "override-value",
		"key3": "override-only",
		"nested": map[string]any{
			"a": 99,
			"c": "override",
		},
	}

	result := deepMergeMapBaseFirst(base, override)

	if result["key1"] != "override-value" {
		t.Errorf("override should win for key1")
	}
	if result["key2"] != "base-only" {
		t.Errorf("base-only key should be preserved")
	}
	if result["key3"] != "override-only" {
		t.Errorf("override-only key should be present")
	}

	nested, ok := result["nested"].(map[string]any)
	if !ok {
		t.Fatal("nested should be a map")
	}
	if nested["a"] != 99 {
		t.Errorf("nested.a should be overridden to 99")
	}
	if nested["b"] != "base" {
		t.Errorf("nested.b should be preserved from base")
	}
	if nested["c"] != "override" {
		t.Errorf("nested.c should be present from override")
	}
}
