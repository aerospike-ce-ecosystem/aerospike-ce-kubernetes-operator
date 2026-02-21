package controller

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	asdbcev1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/metrics"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/utils"
)

// updateStatusAndPhase re-fetches the latest cluster object from the API server,
// populates status fields, sets the desired phase, and performs a status update.
// This pattern avoids "object has been modified" conflict errors that occur when
// updating status on a stale object.
// If the status already matches the desired state, the update is skipped to avoid
// triggering unnecessary reconciliation loops.
func (r *AerospikeCEClusterReconciler) updateStatusAndPhase(
	ctx context.Context,
	namespacedName types.NamespacedName,
	phase asdbcev1alpha1.AerospikePhase,
) error {
	log := logf.FromContext(ctx)

	// Re-fetch the latest version from the API server.
	latest, err := r.refetchCluster(ctx, namespacedName)
	if err != nil {
		return err
	}

	// Capture the previous state for comparison.
	prevPhase := latest.Status.Phase
	prevSize := latest.Status.Size
	prevGeneration := latest.Status.ObservedGeneration

	readyCount := r.populateStatus(ctx, latest)
	latest.Status.Phase = phase

	// Skip the update if nothing meaningful changed to avoid
	// triggering a reconciliation feedback loop via the watch.
	if prevPhase == phase &&
		prevSize == readyCount &&
		prevGeneration == latest.Generation {
		log.V(1).Info("Status unchanged, skipping update",
			"readyPods", readyCount, "desiredSize", latest.Spec.Size, "phase", phase)
		return nil
	}

	log.Info("Updating status", "readyPods", readyCount, "desiredSize", latest.Spec.Size, "phase", phase)

	// Update Prometheus metrics
	metrics.ClusterPhase.WithLabelValues(latest.Namespace, latest.Name).Set(metrics.PhaseToFloat(string(phase)))
	metrics.ClusterReadyPods.WithLabelValues(latest.Namespace, latest.Name).Set(float64(readyCount))

	return r.Status().Update(ctx, latest)
}

// populateStatus fills in the cluster's status fields and returns the ready pod count.
func (r *AerospikeCEClusterReconciler) populateStatus(
	ctx context.Context,
	cluster *asdbcev1alpha1.AerospikeCECluster,
) int32 {
	// List all pods for this cluster
	podList, err := r.listClusterPods(ctx, cluster)
	if err != nil {
		return 0
	}

	podStatuses := make(map[string]asdbcev1alpha1.AerospikePodStatus)
	readyCount := int32(0)

	for i := range podList.Items {
		pod := &podList.Items[i]

		rackID := 0
		if rackStr, ok := pod.Labels[utils.RackLabel]; ok {
			_, _ = fmt.Sscanf(rackStr, "%d", &rackID)
		}

		isReady := isPodReady(pod)
		if isReady {
			readyCount++
		}

		// Read hashes from pod annotations
		configHash := ""
		podSpecHash := ""
		if pod.Annotations != nil {
			configHash = pod.Annotations[utils.ConfigHashAnnotation]
			podSpecHash = pod.Annotations[utils.PodSpecHashAnnotation]
		}

		podStatuses[pod.Name] = asdbcev1alpha1.AerospikePodStatus{
			PodIP:             pod.Status.PodIP,
			HostIP:            pod.Status.HostIP,
			Image:             cluster.Spec.Image,
			PodPort:           3000,
			Rack:              rackID,
			IsRunningAndReady: isReady,
			ConfigHash:        configHash,
			PodSpecHash:       podSpecHash,
		}
	}

	cluster.Status.Pods = podStatuses
	cluster.Status.Size = readyCount
	cluster.Status.ObservedGeneration = cluster.Generation
	cluster.Status.AerospikeConfig = cluster.Spec.AerospikeConfig

	// Build selector string for HPA
	selectorLabels := utils.SelectorLabelsForCluster(cluster.Name)
	selectorParts := make([]string, 0, len(selectorLabels))
	for k, v := range selectorLabels {
		selectorParts = append(selectorParts, fmt.Sprintf("%s=%s", k, v))
	}
	cluster.Status.Selector = strings.Join(selectorParts, ",")

	// Update conditions
	setCondition(cluster, "Available", readyCount > 0, "ClusterAvailable", "At least one pod is ready")
	setCondition(cluster, "Ready", readyCount == cluster.Spec.Size, "AllPodsReady", fmt.Sprintf("%d/%d pods ready", readyCount, cluster.Spec.Size))

	return readyCount
}

func isPodReady(pod *corev1.Pod) bool {
	if pod.Status.Phase != corev1.PodRunning {
		return false
	}
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func setCondition(cluster *asdbcev1alpha1.AerospikeCECluster, condType string, status bool, reason, message string) {
	condStatus := metav1.ConditionFalse
	if status {
		condStatus = metav1.ConditionTrue
	}

	newCond := metav1.Condition{
		Type:               condType,
		Status:             condStatus,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	for i, existing := range cluster.Status.Conditions {
		if existing.Type == condType {
			if existing.Status != condStatus {
				cluster.Status.Conditions[i] = newCond
			}
			return
		}
	}

	cluster.Status.Conditions = append(cluster.Status.Conditions, newCond)
}
