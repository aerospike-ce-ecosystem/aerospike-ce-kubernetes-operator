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
	asdbcev1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
)

// MergeTemplateSpec merges base and override AerospikeCEClusterTemplateSpec.
// The override's non-nil/non-zero fields take precedence over the base.
// Returns a new spec; neither input is modified.
func MergeTemplateSpec(base, override *asdbcev1alpha1.AerospikeCEClusterTemplateSpec) *asdbcev1alpha1.AerospikeCEClusterTemplateSpec {
	if base == nil && override == nil {
		return nil
	}
	if base == nil {
		return override.DeepCopy()
	}
	if override == nil {
		return base.DeepCopy()
	}

	result := base.DeepCopy()

	// Merge AerospikeConfig.
	result.AerospikeConfig = mergeTemplateAerospikeConfig(base.AerospikeConfig, override.AerospikeConfig)

	// Merge Scheduling.
	result.Scheduling = mergeTemplateScheduling(base.Scheduling, override.Scheduling)

	// Merge Storage: override replaces entirely if set.
	if override.Storage != nil {
		result.Storage = override.Storage.DeepCopy()
	}

	// Merge Resources: override replaces entirely if set.
	if override.Resources != nil {
		result.Resources = override.Resources.DeepCopy()
	}

	// Merge RackConfig: override replaces entirely if set.
	if override.RackConfig != nil {
		result.RackConfig = override.RackConfig.DeepCopy()
	}

	// Merge Image: override takes precedence if non-empty.
	if override.Image != "" {
		result.Image = override.Image
	}

	// Merge Size: override takes precedence if non-nil.
	if override.Size != nil {
		sizeCopy := *override.Size
		result.Size = &sizeCopy
	}

	// Merge Monitoring: override replaces entirely if set.
	if override.Monitoring != nil {
		result.Monitoring = override.Monitoring.DeepCopy()
	}

	// Merge AerospikeNetworkPolicy: override replaces entirely if set.
	if override.AerospikeNetworkPolicy != nil {
		result.AerospikeNetworkPolicy = override.AerospikeNetworkPolicy.DeepCopy()
	}

	return result
}

// mergeTemplateAerospikeConfig merges two TemplateAerospikeConfig values.
func mergeTemplateAerospikeConfig(base, override *asdbcev1alpha1.TemplateAerospikeConfig) *asdbcev1alpha1.TemplateAerospikeConfig {
	if base == nil && override == nil {
		return nil
	}
	if base == nil {
		return override.DeepCopy()
	}
	if override == nil {
		return base.DeepCopy()
	}

	result := *base

	// Merge NamespaceDefaults: deep map merge.
	if override.NamespaceDefaults != nil && len(override.NamespaceDefaults.Value) > 0 {
		var baseMap map[string]any
		if base.NamespaceDefaults != nil {
			baseMap = base.NamespaceDefaults.Value
		}
		merged := deepMergeMapBaseFirst(baseMap, override.NamespaceDefaults.Value)
		result.NamespaceDefaults = &asdbcev1alpha1.AerospikeConfigSpec{Value: merged}
	}

	// Merge Service: deep map merge.
	if override.Service != nil && len(override.Service.Value) > 0 {
		var baseMap map[string]any
		if base.Service != nil {
			baseMap = base.Service.Value
		}
		merged := deepMergeMapBaseFirst(baseMap, override.Service.Value)
		result.Service = &asdbcev1alpha1.AerospikeConfigSpec{Value: merged}
	}

	// Merge Network.
	if override.Network != nil {
		result.Network = mergeTemplateNetworkConfig(base.Network, override.Network)
	}

	return &result
}

// mergeTemplateNetworkConfig merges two TemplateNetworkConfig values.
func mergeTemplateNetworkConfig(base, override *asdbcev1alpha1.TemplateNetworkConfig) *asdbcev1alpha1.TemplateNetworkConfig {
	if base == nil && override == nil {
		return nil
	}
	if base == nil {
		return override.DeepCopy()
	}
	if override == nil {
		return base.DeepCopy()
	}

	result := *base
	if override.Heartbeat != nil {
		hb := mergeTemplateHeartbeatConfig(base.Heartbeat, override.Heartbeat)
		result.Heartbeat = hb
	}
	return &result
}

// mergeTemplateHeartbeatConfig merges two TemplateHeartbeatConfig values.
func mergeTemplateHeartbeatConfig(base, override *asdbcev1alpha1.TemplateHeartbeatConfig) *asdbcev1alpha1.TemplateHeartbeatConfig {
	if base == nil && override == nil {
		return nil
	}
	if base == nil {
		return override.DeepCopy()
	}
	if override == nil {
		return base.DeepCopy()
	}

	result := *base
	if override.Mode != "" {
		result.Mode = override.Mode
	}
	if override.Interval != 0 {
		result.Interval = override.Interval
	}
	if override.Timeout != 0 {
		result.Timeout = override.Timeout
	}
	return &result
}

// mergeTemplateScheduling merges two TemplateScheduling values.
func mergeTemplateScheduling(base, override *asdbcev1alpha1.TemplateScheduling) *asdbcev1alpha1.TemplateScheduling {
	if base == nil && override == nil {
		return nil
	}
	if base == nil {
		return override.DeepCopy()
	}
	if override == nil {
		return base.DeepCopy()
	}

	result := *base

	if override.PodAntiAffinityLevel != "" {
		result.PodAntiAffinityLevel = override.PodAntiAffinityLevel
	}
	if override.NodeAffinity != nil {
		result.NodeAffinity = override.NodeAffinity
	}
	// Arrays: override replaces entirely.
	if len(override.Tolerations) > 0 {
		result.Tolerations = override.Tolerations
	}
	if len(override.TopologySpreadConstraints) > 0 {
		result.TopologySpreadConstraints = override.TopologySpreadConstraints
	}
	if override.PodManagementPolicy != "" {
		result.PodManagementPolicy = override.PodManagementPolicy
	}

	return &result
}
