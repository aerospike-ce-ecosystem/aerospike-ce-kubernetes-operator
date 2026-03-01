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
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	asdbcev1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
)

const (
	// AnnotationResyncTemplate triggers a manual template resync when set to "true".
	AnnotationResyncTemplate = "acko.io/resync-template"
)

// ResolveResult holds the outcome of template resolution.
type ResolveResult struct {
	// SnapshotUpdated is true if the template snapshot was created or refreshed.
	SnapshotUpdated bool
	// AnnotationNeedsCleanup is true when the resync annotation was consumed and must
	// be removed from the cluster object by the caller (via a Patch call).
	AnnotationNeedsCleanup bool
	// Warnings contains non-fatal messages from validation.
	Warnings []string
}

// FetchAndSnapshot fetches the referenced template and stores it as a snapshot
// in the cluster's status. Returns the fetched template spec.
func FetchAndSnapshot(
	ctx context.Context,
	reader client.Reader,
	cluster *asdbcev1alpha1.AerospikeCECluster,
) (*asdbcev1alpha1.AerospikeCEClusterTemplateSpec, bool, error) {
	if cluster.Spec.TemplateRef == nil {
		return nil, false, nil
	}

	tmpl := &asdbcev1alpha1.AerospikeCEClusterTemplate{}
	if err := reader.Get(ctx, types.NamespacedName{
		Name:      cluster.Spec.TemplateRef.Name,
		Namespace: cluster.Namespace,
	}, tmpl); err != nil {
		return nil, false, fmt.Errorf("fetching template %q: %w", cluster.Spec.TemplateRef.Name, err)
	}

	specCopy := tmpl.Spec.DeepCopy()
	snapshot := &asdbcev1alpha1.TemplateSnapshotStatus{
		Name:              tmpl.Name,
		ResourceVersion:   tmpl.ResourceVersion,
		SnapshotTimestamp: metav1.NewTime(time.Now()),
		Synced:            true,
		Spec:              specCopy,
	}
	cluster.Status.TemplateSnapshot = snapshot

	return specCopy, true, nil
}

// NeedsResync returns true if the template snapshot should be refreshed.
// This happens when:
// 1. No snapshot exists (first creation).
// 2. The "acko.io/resync-template: true" annotation is present.
func NeedsResync(cluster *asdbcev1alpha1.AerospikeCECluster) bool {
	if cluster.Spec.TemplateRef == nil {
		return false
	}
	if cluster.Status.TemplateSnapshot == nil {
		return true
	}
	if cluster.Annotations != nil && cluster.Annotations[AnnotationResyncTemplate] == "true" {
		return true
	}
	return false
}

// ApplyTemplate applies the resolved template spec (after merge with overrides)
// to the cluster's spec in-memory. The API server object is not modified.
// Only fields not already explicitly set in the cluster spec are applied.
func ApplyTemplate(resolvedTemplateSpec *asdbcev1alpha1.AerospikeCEClusterTemplateSpec, cluster *asdbcev1alpha1.AerospikeCECluster) {
	if resolvedTemplateSpec == nil {
		return
	}

	// Apply scheduling (pod anti-affinity, tolerations, node affinity).
	applyScheduling(resolvedTemplateSpec.Scheduling, cluster)

	// Apply storage defaults.
	applyStorage(resolvedTemplateSpec.Storage, cluster)

	// Apply Aerospike config defaults.
	applyAerospikeConfig(resolvedTemplateSpec.AerospikeConfig, cluster)

	// Apply resource defaults.
	if resolvedTemplateSpec.Resources != nil {
		if cluster.Spec.PodSpec == nil {
			cluster.Spec.PodSpec = &asdbcev1alpha1.AerospikeCEPodSpec{}
		}
		if cluster.Spec.PodSpec.AerospikeContainerSpec == nil {
			cluster.Spec.PodSpec.AerospikeContainerSpec = &asdbcev1alpha1.AerospikeContainerSpec{}
		}
		if cluster.Spec.PodSpec.AerospikeContainerSpec.Resources == nil {
			resourcesCopy := *resolvedTemplateSpec.Resources
			cluster.Spec.PodSpec.AerospikeContainerSpec.Resources = &resourcesCopy
		}
	}
}

// Resolve is the main entry point for template resolution in the reconciler.
// It:
//  1. Checks if a resync is needed and fetches+snapshots the template if so.
//  2. Merges the snapshot spec with any cluster-level overrides.
//  3. Applies the merged spec to the cluster's in-memory spec.
//
// Returns ResolveResult and an error if the template fetch fails.
func Resolve(
	ctx context.Context,
	reader client.Reader,
	cluster *asdbcev1alpha1.AerospikeCECluster,
) (ResolveResult, error) {
	result := ResolveResult{}

	if cluster.Spec.TemplateRef == nil {
		return result, nil
	}

	// Determine if we need to (re)fetch the template.
	if NeedsResync(cluster) {
		annotationTriggered := cluster.Annotations != nil && cluster.Annotations[AnnotationResyncTemplate] == "true"

		_, updated, err := FetchAndSnapshot(ctx, reader, cluster)
		if err != nil {
			return result, err
		}
		result.SnapshotUpdated = updated

		// Signal that the annotation must be deleted from the API server by the caller.
		// We do NOT delete it in-memory here to avoid a stale resourceVersion when the
		// caller subsequently patches the object.
		if annotationTriggered && updated {
			result.AnnotationNeedsCleanup = true
		}
	}

	// Build effective template spec: snapshot base + overrides.
	if cluster.Status.TemplateSnapshot == nil || cluster.Status.TemplateSnapshot.Spec == nil {
		return result, fmt.Errorf("template snapshot is missing or has no spec; cannot resolve template %q", cluster.Spec.TemplateRef.Name)
	}
	snapshotSpec := cluster.Status.TemplateSnapshot.Spec
	effectiveSpec := MergeTemplateSpec(snapshotSpec, cluster.Spec.Overrides)

	// Validate the effective spec.
	result.Warnings = ValidateResolvedSpec(&cluster.Spec, effectiveSpec)

	// Validate LocalPV StorageClass binding mode when localPVRequired is set.
	if effectiveSpec.Storage != nil && effectiveSpec.Storage.LocalPVRequired {
		if err := ValidateLocalPV(ctx, reader, effectiveSpec.Storage.StorageClassName); err != nil {
			result.Warnings = append(result.Warnings, "localPVRequired: "+err.Error())
		}
	}

	// Apply the effective template spec to the in-memory cluster spec.
	ApplyTemplate(effectiveSpec, cluster)

	return result, nil
}
