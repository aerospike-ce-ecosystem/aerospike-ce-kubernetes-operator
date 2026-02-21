package controller

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
		if err := ctrl.SetControllerReference(cluster, sts, r.Scheme); err != nil {
			return fmt.Errorf("setting controller reference: %w", err)
		}
		log.Info("Creating StatefulSet", "name", stsName, "replicas", rackSize)
		return r.Create(ctx, sts)
	} else if err != nil {
		return fmt.Errorf("getting StatefulSet %s: %w", stsName, err)
	}

	// Update existing StatefulSet
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

	stsList := &appsv1.StatefulSetList{}
	if err := r.List(ctx, stsList,
		client.InNamespace(cluster.Namespace),
		client.MatchingLabels(utils.SelectorLabelsForCluster(cluster.Name)),
	); err != nil {
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
			if err := r.Delete(ctx, sts); err != nil && !errors.IsNotFound(err) {
				return err
			}
		}
	}

	return nil
}
