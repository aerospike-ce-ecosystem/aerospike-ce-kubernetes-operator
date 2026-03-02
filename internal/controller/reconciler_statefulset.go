package controller

import (
	"context"
	"fmt"
	"maps"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	ackov1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/podutil"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/storage"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/utils"
)

func (r *AerospikeClusterReconciler) reconcileStatefulSet(
	ctx context.Context,
	cluster *ackov1alpha1.AerospikeCluster,
	rack *ackov1alpha1.Rack,
	_ *ackov1alpha1.AerospikeConfigSpec, // effectiveConfig (pre-computed, hash passed separately)
	hash string,
	rackSize int32,
) error {
	log := logf.FromContext(ctx)

	stsName := utils.StatefulSetName(cluster.Name, rack.ID)
	configMapName := utils.ConfigMapName(cluster.Name, rack.ID)

	// Build pod template
	podTemplate := podutil.BuildPodTemplateSpec(cluster, rack, rack.ID, configMapName, hash)

	// Compute and add PodSpec hash for change detection
	podSpecHash := computePodSpecHash(cluster, rack)
	if podTemplate.Annotations == nil {
		podTemplate.Annotations = make(map[string]string)
	}
	podTemplate.Annotations[utils.PodSpecHashAnnotation] = podSpecHash

	// Build storage
	storageSpec := cluster.Spec.Storage
	if rack.Storage != nil {
		storageSpec = rack.Storage
	}
	pvcTemplates := storage.BuildVolumeClaimTemplates(storageSpec)
	// Add cluster labels to PVC templates so PVCs can be efficiently queried by label.
	pvcLabels := utils.LabelsForCluster(cluster.Name)
	for i := range pvcTemplates {
		if pvcTemplates[i].Labels == nil {
			pvcTemplates[i].Labels = make(map[string]string)
		}
		maps.Copy(pvcTemplates[i].Labels, pvcLabels)
	}

	// Check if StatefulSet exists
	existing := &appsv1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{Name: stsName, Namespace: cluster.Namespace}, existing)

	if errors.IsNotFound(err) {
		// Create new StatefulSet
		sts := r.buildStatefulSet(cluster, stsName, rackSize, podTemplate, pvcTemplates)
		if err := r.setOwnerRef(cluster, sts); err != nil {
			return err
		}
		log.Info("Creating StatefulSet", "name", stsName, "replicas", rackSize)
		if err := r.Create(ctx, sts); err != nil {
			return fmt.Errorf("creating StatefulSet %s: %w", stsName, err)
		}
		r.Recorder.Eventf(cluster, corev1.EventTypeNormal, EventStatefulSetCreated,
			"StatefulSet %s created: replicas=%d", stsName, rackSize)
		return nil
	} else if err != nil {
		return fmt.Errorf("getting StatefulSet %s: %w", stsName, err)
	}

	// Update only if replicas or config hash changed
	oldReplicas := int32(0)
	if existing.Spec.Replicas != nil {
		oldReplicas = *existing.Spec.Replicas
	}
	needsUpdate := oldReplicas != rackSize
	var existingHash, existingPodSpecHash string
	if existing.Spec.Template.Annotations != nil {
		existingHash = existing.Spec.Template.Annotations[utils.ConfigHashAnnotation]
		existingPodSpecHash = existing.Spec.Template.Annotations[utils.PodSpecHashAnnotation]
	}
	if existingHash != hash {
		needsUpdate = true
	}
	if existingPodSpecHash != podSpecHash {
		needsUpdate = true
	}

	if !needsUpdate {
		return nil
	}

	scaleDown := rackSize < oldReplicas

	targetReplicas := rackSize
	if scaleDown {
		// Apply scale-down batch size: only scale down a batch at a time.
		batchSize := r.getScaleDownBatchSize(cluster, oldReplicas-rackSize)
		targetReplicas = max(oldReplicas-batchSize, rackSize)
	}

	existing.Spec.Replicas = &targetReplicas
	existing.Spec.Template = podTemplate
	log.Info("Updating StatefulSet", "name", stsName, "targetReplicas", targetReplicas)
	if err := r.Update(ctx, existing); err != nil {
		return fmt.Errorf("updating StatefulSet %s: %w", stsName, err)
	}
	r.Recorder.Eventf(cluster, corev1.EventTypeNormal, EventStatefulSetUpdated,
		"StatefulSet %s updated: replicas=%d", stsName, targetReplicas)
	if oldReplicas != targetReplicas {
		r.Recorder.Eventf(cluster, corev1.EventTypeNormal, EventRackScaled,
			"Rack %d scaled from %d to %d replicas", rack.ID, oldReplicas, targetReplicas)
	}

	// Cleanup orphaned PVCs after StatefulSet update so pods terminate first.
	// Only delete PVCs for volumes with cascadeDelete enabled; preserve all others.
	if scaleDown {
		storageSpec := cluster.Spec.Storage
		if rack.Storage != nil {
			storageSpec = rack.Storage
		}
		log.Info("Scale-down detected, cleaning up orphaned cascade-delete PVCs",
			"name", stsName, "old", oldReplicas, "new", targetReplicas)
		deleted, err := storage.DeleteOrphanedCascadeDeletePVCs(
			ctx, r.Client, cluster.Namespace, stsName, targetReplicas, storageSpec)
		if err != nil {
			log.Error(err, "Failed to delete orphaned cascade PVCs", "statefulset", stsName)
			r.Recorder.Eventf(cluster, corev1.EventTypeWarning, EventPVCCleanupFailed,
				"Failed to delete orphaned cascade PVCs for %s: %v", stsName, err)
			// Non-fatal: PVCs will be cleaned up on next reconcile
		} else if deleted > 0 {
			log.Info("Deleted orphaned cascade-delete PVCs",
				"statefulset", stsName, "count", deleted)
			r.Recorder.Eventf(cluster, corev1.EventTypeNormal, EventPVCCleanedUp,
				"Deleted %d orphaned PVC(s) for %s after scale-down", deleted, stsName)
		}
	}

	return nil
}

func (r *AerospikeClusterReconciler) buildStatefulSet(
	cluster *ackov1alpha1.AerospikeCluster,
	name string,
	replicas int32,
	podTemplate corev1.PodTemplateSpec,
	pvcTemplates []corev1.PersistentVolumeClaim,
) *appsv1.StatefulSet {
	labels := utils.LabelsForCluster(cluster.Name)
	selectorLabels := utils.SelectorLabelsForCluster(cluster.Name)
	serviceName := utils.HeadlessServiceName(cluster.Name)

	podManagementPolicy := appsv1.ParallelPodManagement
	if cluster.Spec.PodSpec != nil && cluster.Spec.PodSpec.PodManagementPolicy != "" {
		podManagementPolicy = cluster.Spec.PodSpec.PodManagementPolicy
	}

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName:         serviceName,
			Replicas:            &replicas,
			PodManagementPolicy: podManagementPolicy,
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.OnDeleteStatefulSetStrategyType,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			Template:             podTemplate,
			VolumeClaimTemplates: pvcTemplates,
		},
	}

	return sts
}

// cleanupRemovedRacks deletes StatefulSets for racks that no longer exist in the spec.
func (r *AerospikeClusterReconciler) cleanupRemovedRacks(
	ctx context.Context,
	cluster *ackov1alpha1.AerospikeCluster,
	currentRacks []ackov1alpha1.Rack,
) error {
	log := logf.FromContext(ctx)

	stsList, err := r.listClusterStatefulSets(ctx, cluster)
	if err != nil {
		return err
	}

	currentRackNames := make(map[string]bool)
	for _, rack := range currentRacks {
		currentRackNames[utils.StatefulSetName(cluster.Name, rack.ID)] = true
	}

	// Resolve effective storage spec: per-rack storage overrides cluster-level.
	for i := range stsList.Items {
		sts := &stsList.Items[i]
		if !currentRackNames[sts.Name] {
			log.Info("Deleting removed rack StatefulSet", "name", sts.Name)
			// Delete PVCs for removed rack before deleting the StatefulSet,
			// but only for volumes that have cascadeDelete enabled.
			storageSpec := cluster.Spec.Storage
			if err := storage.DeleteCascadeDeletePVCs(ctx, r.Client, cluster.Namespace, sts.Name, storageSpec); err != nil {
				log.Error(err, "Failed to delete cascade PVCs for removed rack", "statefulset", sts.Name)
				r.Recorder.Eventf(cluster, corev1.EventTypeWarning, EventPVCCleanupFailed,
					"Failed to delete cascade PVCs for removed rack %s: %v", sts.Name, err)
			}
			if err := r.Delete(ctx, sts); err != nil && !errors.IsNotFound(err) {
				return err
			}
		}
	}

	return nil
}

// getScaleDownBatchSize returns the effective scale-down batch size.
func (r *AerospikeClusterReconciler) getScaleDownBatchSize(cluster *ackov1alpha1.AerospikeCluster, totalToScaleDown int32) int32 {
	if cluster.Spec.RackConfig != nil && cluster.Spec.RackConfig.ScaleDownBatchSize != nil {
		return resolveIntOrPercent(cluster.Spec.RackConfig.ScaleDownBatchSize, totalToScaleDown)
	}
	return totalToScaleDown // default: scale down all at once
}

// resolveIntOrPercent resolves an IntOrString to an absolute int32 value.
func resolveIntOrPercent(val *intstr.IntOrString, total int32) int32 {
	if val == nil {
		return 1
	}
	if val.Type == intstr.Int {
		v := val.IntVal
		if v < 1 {
			return 1
		}
		return v
	}
	// Percentage
	pct, err := intstr.GetScaledValueFromIntOrPercent(val, int(total), true)
	if err != nil || pct < 1 {
		return 1
	}
	return int32(pct)
}

// detectScaling checks each rack's current StatefulSet replicas against the
// desired rack size and returns whether a scale-up or scale-down is needed.
// Returns (scalingUp, scalingDown, error). Both can be false if no scaling is needed.
// If racks are simultaneously scaling in opposite directions (one rack up, another
// down), both flags can be true. The caller uses else-if so ScalingUp takes
// precedence in the phase display when both are true.
func (r *AerospikeClusterReconciler) detectScaling(
	ctx context.Context,
	cluster *ackov1alpha1.AerospikeCluster,
	racks []ackov1alpha1.Rack,
	rackSizes []int32,
) (scalingUp bool, scalingDown bool, err error) {
	for i, rack := range racks {
		stsName := utils.StatefulSetName(cluster.Name, rack.ID)
		existing := &appsv1.StatefulSet{}
		if err := r.Get(ctx, types.NamespacedName{Name: stsName, Namespace: cluster.Namespace}, existing); err != nil {
			if errors.IsNotFound(err) {
				// New rack — treated as scale-up (StatefulSet will be created).
				scalingUp = true
				continue
			}
			return false, false, err
		}
		oldReplicas := int32(0)
		if existing.Spec.Replicas != nil {
			oldReplicas = *existing.Spec.Replicas
		}
		desired := rackSizes[i]
		if desired > oldReplicas {
			scalingUp = true
		} else if desired < oldReplicas {
			scalingDown = true
		}
	}
	return scalingUp, scalingDown, nil
}

// computePodSpecHash returns a short SHA256 hash derived from the cluster image
// and pod-level spec settings so that changes to the pod template (aside from
// config) are captured.
func computePodSpecHash(cluster *ackov1alpha1.AerospikeCluster, rack *ackov1alpha1.Rack) string {
	input := struct {
		Image      string                                `json:"image"`
		PodSpec    *ackov1alpha1.AerospikeCEPodSpec      `json:"podSpec,omitempty"`
		Monitoring *ackov1alpha1.AerospikeMonitoringSpec `json:"monitoring,omitempty"`
		RackID     int                                   `json:"rackID"`
	}{
		Image:      cluster.Spec.Image,
		PodSpec:    cluster.Spec.PodSpec,
		Monitoring: cluster.Spec.Monitoring,
		RackID:     rack.ID,
	}
	return utils.ShortSHA256(input)
}
