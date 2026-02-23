package storage

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
)

// GetPVCsForStatefulSet lists PVCs belonging to the given StatefulSet.
// It first attempts to filter by the app instance label for efficiency, then
// falls back to listing all PVCs and name-matching if no labels are present
// (e.g., for PVCs created before labels were added to VolumeClaimTemplates).
func GetPVCsForStatefulSet(ctx context.Context, c client.Client, namespace, stsName string) ([]corev1.PersistentVolumeClaim, error) {
	pvcList := &corev1.PersistentVolumeClaimList{}
	if err := c.List(ctx, pvcList,
		client.InNamespace(namespace),
		client.MatchingLabels{"app.kubernetes.io/name": "aerospike-cluster"},
	); err != nil {
		return nil, fmt.Errorf("listing PVCs in namespace %s: %w", namespace, err)
	}

	// Fallback: if label-based query returned no results, re-list without labels
	// to find PVCs created before labels were added to VolumeClaimTemplates.
	if len(pvcList.Items) == 0 {
		if err := c.List(ctx, pvcList, client.InNamespace(namespace)); err != nil {
			return nil, fmt.Errorf("listing all PVCs in namespace %s: %w", namespace, err)
		}
	}

	var matched []corev1.PersistentVolumeClaim
	for i := range pvcList.Items {
		pvc := &pvcList.Items[i]
		// StatefulSet PVC names follow the pattern: <volumeName>-<stsName>-<ordinal>
		if isOwnedByStatefulSet(pvc, stsName) {
			matched = append(matched, *pvc)
		}
	}

	return matched, nil
}

// DeleteOrphanedPVCs removes PVCs that belong to pod ordinals >= desiredReplicas.
// This is useful after a scale-down to clean up storage for removed pods.
func DeleteOrphanedPVCs(ctx context.Context, c client.Client, namespace, stsName string, desiredReplicas int32) error {
	pvcs, err := GetPVCsForStatefulSet(ctx, c, namespace, stsName)
	if err != nil {
		return err
	}

	for i := range pvcs {
		pvc := &pvcs[i]
		ordinal, ok := extractOrdinal(pvc.Name, stsName)
		if !ok {
			continue
		}

		if ordinal >= desiredReplicas {
			if err := c.Delete(ctx, pvc); err != nil {
				if !errors.IsNotFound(err) {
					return fmt.Errorf("deleting orphaned PVC %s: %w", pvc.Name, err)
				}
			}
		}
	}

	return nil
}

// DeletePVCsForStatefulSet deletes all PVCs associated with the given StatefulSet.
// Used when the cluster CR is deleted with cascadeDelete.
func DeletePVCsForStatefulSet(ctx context.Context, c client.Client, namespace, stsName string) error {
	pvcs, err := GetPVCsForStatefulSet(ctx, c, namespace, stsName)
	if err != nil {
		return err
	}

	for i := range pvcs {
		if err := c.Delete(ctx, &pvcs[i]); err != nil {
			return fmt.Errorf("deleting PVC %s: %w", pvcs[i].Name, err)
		}
	}

	return nil
}

// isOwnedByStatefulSet checks if a PVC name contains the StatefulSet name as
// part of the standard naming convention: <volumeName>-<stsName>-<ordinal>.
func isOwnedByStatefulSet(pvc *corev1.PersistentVolumeClaim, stsName string) bool {
	_, ok := extractOrdinal(pvc.Name, stsName)
	return ok
}

// extractOrdinal parses the ordinal index from a PVC name that follows the
// StatefulSet naming pattern: <volumeName>-<stsName>-<ordinal>.
func extractOrdinal(pvcName, stsName string) (int32, bool) {
	// PVC names follow: <volumeClaimName>-<stsName>-<ordinal>
	// We search for "-<stsName>-" and then parse the trailing ordinal.
	pattern := "-" + stsName + "-"
	idx := len(pvcName) - 1

	// Find the last digit sequence (the ordinal).
	for idx >= 0 && pvcName[idx] >= '0' && pvcName[idx] <= '9' {
		idx--
	}

	if idx < 0 || idx == len(pvcName)-1 {
		return 0, false
	}

	// Check that the text before the ordinal ends with "-<stsName>-"
	prefix := pvcName[:idx+1]
	if len(prefix) < len(pattern) {
		return 0, false
	}

	if prefix[len(prefix)-len(pattern):] != pattern {
		return 0, false
	}

	// Parse ordinal using strconv for proper overflow/error handling.
	ordinal, err := strconv.ParseInt(pvcName[idx+1:], 10, 32)
	if err != nil {
		return 0, false
	}

	return int32(ordinal), true
}

// extractVolumeName extracts the volume claim template name from a PVC name
// that follows the StatefulSet naming pattern: <volumeName>-<stsName>-<ordinal>.
func extractVolumeName(pvcName, stsName string) (string, bool) {
	pattern := "-" + stsName + "-"
	idx := strings.LastIndex(pvcName, pattern)
	if idx <= 0 {
		return "", false
	}

	// Verify the suffix after the pattern is a valid ordinal
	suffix := pvcName[idx+len(pattern):]
	if suffix == "" {
		return "", false
	}
	if _, err := strconv.ParseInt(suffix, 10, 32); err != nil {
		return "", false
	}

	return pvcName[:idx], true
}

// DeleteCascadeDeletePVCs deletes only PVCs for volumes that have cascadeDelete=true.
// This ensures non-cascade volumes are preserved when the CR is deleted.
func DeleteCascadeDeletePVCs(
	ctx context.Context,
	c client.Client,
	namespace, stsName string,
	storageSpec *v1alpha1.AerospikeStorageSpec,
) error {
	if storageSpec == nil {
		return nil
	}

	// Build a set of volume names that have cascadeDelete enabled
	cascadeVolumes := make(map[string]bool)
	for i := range storageSpec.Volumes {
		vol := &storageSpec.Volumes[i]
		if vol.Source.PersistentVolume != nil && ResolveCascadeDelete(vol, storageSpec) {
			cascadeVolumes[vol.Name] = true
		}
	}

	if len(cascadeVolumes) == 0 {
		return nil
	}

	pvcs, err := GetPVCsForStatefulSet(ctx, c, namespace, stsName)
	if err != nil {
		return err
	}

	for i := range pvcs {
		pvc := &pvcs[i]
		volName, ok := extractVolumeName(pvc.Name, stsName)
		if !ok {
			continue
		}

		if !cascadeVolumes[volName] {
			continue
		}

		if err := c.Delete(ctx, pvc); err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return fmt.Errorf("deleting cascade PVC %s: %w", pvc.Name, err)
		}
	}

	return nil
}
