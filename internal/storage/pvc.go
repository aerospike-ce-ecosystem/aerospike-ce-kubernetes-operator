package storage

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetPVCsForStatefulSet lists all PVCs that belong to the given StatefulSet
// by matching the standard controller-revision label convention (the PVC name
// prefix matches "<stsName>-<ordinal>").
func GetPVCsForStatefulSet(ctx context.Context, c client.Client, namespace, stsName string) ([]corev1.PersistentVolumeClaim, error) {
	pvcList := &corev1.PersistentVolumeClaimList{}
	if err := c.List(ctx, pvcList, client.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("listing PVCs in namespace %s: %w", namespace, err)
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
				return fmt.Errorf("deleting orphaned PVC %s: %w", pvc.Name, err)
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

	// Parse ordinal.
	var ordinal int32
	for _, c := range pvcName[idx+1:] {
		ordinal = ordinal*10 + (c - '0')
	}

	return ordinal, true
}
