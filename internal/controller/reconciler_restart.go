package controller

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	asdbcev1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/metrics"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/utils"
)

// reconcileRollingRestart checks if pods need restart due to config changes.
// Returns true if a restart was triggered (caller should requeue).
// Supports batch restart via spec.rollingUpdateBatchSize.
//
// Precedence: dynamic config update > warm restart (SIGUSR1) > cold restart (pod delete).
func (r *AerospikeCEClusterReconciler) reconcileRollingRestart(
	ctx context.Context,
	cluster *asdbcev1alpha1.AerospikeCECluster,
	rack *asdbcev1alpha1.Rack,
) (bool, error) {
	log := logf.FromContext(ctx)

	stsName := utils.StatefulSetName(cluster.Name, rack.ID)

	sts := &appsv1.StatefulSet{}
	if err := r.Get(ctx, types.NamespacedName{Name: stsName, Namespace: cluster.Namespace}, sts); err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	// Get desired config hash from the StatefulSet template
	desiredHash := ""
	if sts.Spec.Template.Annotations != nil {
		desiredHash = sts.Spec.Template.Annotations[utils.ConfigHashAnnotation]
	}

	if desiredHash == "" {
		return false, nil
	}

	batchSize := int32(1)
	if cluster.Spec.RollingUpdateBatchSize != nil && *cluster.Spec.RollingUpdateBatchSize > 0 {
		batchSize = *cluster.Spec.RollingUpdateBatchSize
	}

	// Compute the old and new config for dynamic config comparison.
	// Old config comes from the CR's last-applied status; new config from the spec.
	var oldConfig, newConfig map[string]any
	if cluster.Status.AerospikeConfig != nil {
		oldConfig = cluster.Status.AerospikeConfig.Value
	}
	if cluster.Spec.AerospikeConfig != nil {
		newConfig = cluster.Spec.AerospikeConfig.Value
	}

	// Collect pods that need restart (reverse order = highest ordinal first)
	var podsToRestart []*corev1.Pod
	for i := int(*sts.Spec.Replicas) - 1; i >= 0; i-- {
		podName := fmt.Sprintf("%s-%d", stsName, i)

		pod := &corev1.Pod{}
		if err := r.Get(ctx, types.NamespacedName{Name: podName, Namespace: cluster.Namespace}, pod); err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return false, err
		}

		currentHash := ""
		if pod.Annotations != nil {
			currentHash = pod.Annotations[utils.ConfigHashAnnotation]
		}

		if currentHash != desiredHash {
			podsToRestart = append(podsToRestart, pod)
		}
	}

	if len(podsToRestart) == 0 {
		return false, nil
	}

	// Restart up to batchSize pods
	restarted := int32(0)
	for _, pod := range podsToRestart {
		if restarted >= batchSize {
			break
		}

		// 1. Try dynamic config update first (no restart needed)
		if oldConfig != nil && newConfig != nil {
			if r.tryDynamicConfigUpdate(ctx, cluster, pod, oldConfig, newConfig) {
				log.Info("Dynamic config update succeeded, no restart needed", "pod", pod.Name)
				// Pod's config hash is updated by tryDynamicConfigUpdate
				continue
			}
		}

		// 2. Try warm restart if only config changed (same image, same podspec)
		if r.shouldWarmRestart(cluster, pod, sts) {
			log.Info("Attempting warm restart (SIGUSR1)", "pod", pod.Name)
			if err := r.warmRestartPod(ctx, pod); err != nil {
				log.Info("Warm restart failed, falling back to cold restart", "pod", pod.Name, "error", err)
				if err := r.coldRestartPod(ctx, cluster, pod); err != nil {
					return false, err
				}
			} else {
				metrics.WarmRestartsTotal.WithLabelValues(cluster.Namespace, cluster.Name).Inc()
				restarted++
				continue
			}
		} else {
			// 3. Cold restart (pod delete)
			log.Info("Pod config/spec hash mismatch, deleting for restart", "pod", pod.Name)
			if err := r.coldRestartPod(ctx, cluster, pod); err != nil {
				return false, err
			}
		}

		restarted++
	}

	return true, nil
}

// coldRestartPod deletes the pod to trigger a cold restart via StatefulSet.
func (r *AerospikeCEClusterReconciler) coldRestartPod(
	ctx context.Context,
	cluster *asdbcev1alpha1.AerospikeCECluster,
	pod *corev1.Pod,
) error {
	if err := r.Delete(ctx, pod); err != nil && !errors.IsNotFound(err) {
		return err
	}
	metrics.ColdRestartsTotal.WithLabelValues(cluster.Namespace, cluster.Name).Inc()
	return nil
}
