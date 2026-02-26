package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	asdbcev1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
)

// reconcileOperations handles on-demand operations (WarmRestart, PodRestart).
// Returns true if an operation is in progress and caller should requeue.
func (r *AerospikeCEClusterReconciler) reconcileOperations(
	ctx context.Context,
	cluster *asdbcev1alpha1.AerospikeCECluster,
) (bool, error) {
	// If no operations specified, clear status and return
	if len(cluster.Spec.Operations) == 0 {
		return false, nil
	}

	op := cluster.Spec.Operations[0]

	// Check if this operation was already completed
	if cluster.Status.OperationStatus != nil &&
		cluster.Status.OperationStatus.ID == op.ID &&
		cluster.Status.OperationStatus.Phase == asdbcev1alpha1.AerospikePhaseCompleted {
		return false, nil
	}

	// Check if this operation already errored out
	if cluster.Status.OperationStatus != nil &&
		cluster.Status.OperationStatus.ID == op.ID &&
		cluster.Status.OperationStatus.Phase == asdbcev1alpha1.AerospikePhaseError {
		return false, nil
	}

	log := logf.FromContext(ctx)
	log.Info("Processing on-demand operation", "kind", op.Kind, "id", op.ID)

	// Get target pods
	pods, err := r.getOperationTargetPods(ctx, cluster, op.PodList)
	if err != nil {
		return false, err
	}

	// Initialize or update operation status
	opStatus := &asdbcev1alpha1.OperationStatus{
		ID:    op.ID,
		Kind:  op.Kind,
		Phase: asdbcev1alpha1.AerospikePhaseInProgress,
	}

	// Get batch size
	batchSize := int32(1)
	if cluster.Spec.RollingUpdateBatchSize != nil && *cluster.Spec.RollingUpdateBatchSize > 0 {
		batchSize = *cluster.Spec.RollingUpdateBatchSize
	}

	// Track completed pods from previous status
	completedSet := make(map[string]bool)
	if cluster.Status.OperationStatus != nil && cluster.Status.OperationStatus.ID == op.ID {
		for _, p := range cluster.Status.OperationStatus.CompletedPods {
			completedSet[p] = true
		}
		opStatus.CompletedPods = cluster.Status.OperationStatus.CompletedPods
		opStatus.FailedPods = cluster.Status.OperationStatus.FailedPods
	}

	processed := int32(0)
	allDone := true

	for _, pod := range pods {
		if completedSet[pod.Name] {
			continue
		}
		allDone = false

		if processed >= batchSize {
			break
		}

		var opErr error
		switch op.Kind {
		case asdbcev1alpha1.OperationWarmRestart:
			opErr = r.warmRestartPod(ctx, pod)
		case asdbcev1alpha1.OperationPodRestart:
			opErr = r.coldRestartPod(ctx, cluster, pod)
		}

		if opErr != nil {
			log.Error(opErr, "Operation failed on pod", "pod", pod.Name, "kind", op.Kind)
			opStatus.FailedPods = append(opStatus.FailedPods, pod.Name)
		} else {
			opStatus.CompletedPods = append(opStatus.CompletedPods, pod.Name)
			completedSet[pod.Name] = true
		}
		processed++
	}

	if allDone {
		opStatus.Phase = asdbcev1alpha1.AerospikePhaseCompleted
		if len(opStatus.FailedPods) > 0 {
			opStatus.Phase = asdbcev1alpha1.AerospikePhaseError
		}
	}

	// Update operation status
	// Re-fetch cluster to avoid conflicts
	latest, err := r.refetchCluster(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace})
	if err != nil {
		return !allDone, err
	}
	latest.Status.OperationStatus = opStatus
	if err := r.Status().Update(ctx, latest); err != nil {
		return !allDone, err
	}

	r.Recorder.Eventf(cluster, corev1.EventTypeNormal, "Operation",
		"Operation %s (%s): %d/%d pods processed", op.ID, op.Kind, len(opStatus.CompletedPods), len(pods))

	return !allDone, nil
}

// getOperationTargetPods returns the pods targeted by an operation.
func (r *AerospikeCEClusterReconciler) getOperationTargetPods(
	ctx context.Context,
	cluster *asdbcev1alpha1.AerospikeCECluster,
	podList []string,
) ([]*corev1.Pod, error) {
	allPods, err := r.listClusterPods(ctx, cluster)
	if err != nil {
		return nil, err
	}
	return filterPodsByNames(allPods.Items, podList), nil
}

// filterPodsByNames returns pointers to the pods matching the given names.
// If names is empty, all pods are returned.
func filterPodsByNames(allPods []corev1.Pod, names []string) []*corev1.Pod {
	if len(names) == 0 {
		result := make([]*corev1.Pod, len(allPods))
		for i := range allPods {
			result[i] = &allPods[i]
		}
		return result
	}

	podMap := make(map[string]*corev1.Pod, len(allPods))
	for i := range allPods {
		podMap[allPods[i].Name] = &allPods[i]
	}

	var result []*corev1.Pod
	for _, name := range names {
		if pod, ok := podMap[name]; ok {
			result = append(result, pod)
		}
	}
	return result
}
