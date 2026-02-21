package controller

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	asdbcev1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/podutil"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/storage"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/utils"
)

func (r *AerospikeCEClusterReconciler) reconcileStatefulSet(
	ctx context.Context,
	cluster *asdbcev1alpha1.AerospikeCECluster,
	rack *asdbcev1alpha1.Rack,
) error {
	log := logf.FromContext(ctx)

	stsName := utils.StatefulSetName(cluster.Name, rack.ID)
	configMapName := utils.ConfigMapName(cluster.Name, rack.ID)

	racks := r.getRacks(cluster)
	rackIndex := 0
	for i, rk := range racks {
		if rk.ID == rack.ID {
			rackIndex = i
			break
		}
	}
	rackSize := r.getRackSize(cluster, racks, rackIndex)

	// Determine effective config for this rack
	effectiveConfig := cluster.Spec.AerospikeConfig
	if rack.AerospikeConfig != nil && cluster.Spec.AerospikeConfig != nil {
		merged := utils.DeepMerge(cluster.Spec.AerospikeConfig.Value, rack.AerospikeConfig.Value)
		effectiveConfig = &asdbcev1alpha1.AerospikeConfigSpec{Value: merged}
	}

	hash := configHash(effectiveConfig)

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
		return r.Create(ctx, sts)
	} else if err != nil {
		return fmt.Errorf("getting StatefulSet %s: %w", stsName, err)
	}

	// Update only if replicas or config hash changed
	oldReplicas := int32(0)
	if existing.Spec.Replicas != nil {
		oldReplicas = *existing.Spec.Replicas
	}
	needsUpdate := oldReplicas != rackSize
	existingHash := existing.Spec.Template.Annotations[utils.ConfigHashAnnotation]
	if existingHash != hash {
		needsUpdate = true
	}
	existingPodSpecHash := existing.Spec.Template.Annotations[utils.PodSpecHashAnnotation]
	if existingPodSpecHash != podSpecHash {
		needsUpdate = true
	}

	if !needsUpdate {
		return nil
	}

	// Cleanup orphaned PVCs on scale-down
	if rackSize < oldReplicas {
		log.Info("Scale-down detected, cleaning up orphaned PVCs", "name", stsName, "old", oldReplicas, "new", rackSize)
		if err := storage.DeleteOrphanedPVCs(ctx, r.Client, cluster.Namespace, stsName, rackSize); err != nil {
			log.Error(err, "Failed to delete orphaned PVCs", "statefulset", stsName)
			// Non-fatal: continue with the update
		}
	}

	existing.Spec.Replicas = &rackSize
	existing.Spec.Template = podTemplate
	log.Info("Updating StatefulSet", "name", stsName)
	return r.Update(ctx, existing)
}

func (r *AerospikeCEClusterReconciler) buildStatefulSet(
	cluster *asdbcev1alpha1.AerospikeCECluster,
	name string,
	replicas int32,
	podTemplate corev1.PodTemplateSpec,
	pvcTemplates []corev1.PersistentVolumeClaim,
) *appsv1.StatefulSet {
	labels := utils.LabelsForCluster(cluster.Name)
	selectorLabels := utils.SelectorLabelsForCluster(cluster.Name)
	serviceName := utils.HeadlessServiceName(cluster.Name)

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName:         serviceName,
			Replicas:            &replicas,
			PodManagementPolicy: appsv1.ParallelPodManagement,
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
func (r *AerospikeCEClusterReconciler) cleanupRemovedRacks(
	ctx context.Context,
	cluster *asdbcev1alpha1.AerospikeCECluster,
	currentRacks []asdbcev1alpha1.Rack,
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

	for i := range stsList.Items {
		sts := &stsList.Items[i]
		if !currentRackNames[sts.Name] {
			log.Info("Deleting removed rack StatefulSet", "name", sts.Name)
			// Delete PVCs for removed rack before deleting the StatefulSet
			if err := storage.DeletePVCsForStatefulSet(ctx, r.Client, cluster.Namespace, sts.Name); err != nil {
				log.Error(err, "Failed to delete PVCs for removed rack", "statefulset", sts.Name)
			}
			if err := r.Delete(ctx, sts); err != nil && !errors.IsNotFound(err) {
				return err
			}
		}
	}

	return nil
}

// computePodSpecHash returns a short SHA256 hash derived from the cluster image
// and pod-level spec settings so that changes to the pod template (aside from
// config) are captured.
func computePodSpecHash(cluster *asdbcev1alpha1.AerospikeCECluster, rack *asdbcev1alpha1.Rack) string {
	input := struct {
		Image      string                                  `json:"image"`
		PodSpec    *asdbcev1alpha1.AerospikeCEPodSpec      `json:"podSpec,omitempty"`
		Monitoring *asdbcev1alpha1.AerospikeMonitoringSpec `json:"monitoring,omitempty"`
		RackID     int                                     `json:"rackID"`
	}{
		Image:      cluster.Spec.Image,
		PodSpec:    cluster.Spec.PodSpec,
		Monitoring: cluster.Spec.Monitoring,
		RackID:     rack.ID,
	}
	data, _ := json.Marshal(input)
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:8])
}
