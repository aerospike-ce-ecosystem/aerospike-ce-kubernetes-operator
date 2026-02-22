package controller

import (
	"context"
	"fmt"

	aero "github.com/aerospike/aerospike-client-go/v8"
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
	replicas := int32(0)
	if sts.Spec.Replicas != nil {
		replicas = *sts.Spec.Replicas
	}
	var podsToRestart []*corev1.Pod
	for i := int(replicas) - 1; i >= 0; i-- {
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

	// Create Aerospike client once for all pods (lazy, only if dynamic config is attempted).
	var aeroClient *aero.Client
	defer func() {
		if aeroClient != nil {
			closeAerospikeClient(aeroClient)
		}
	}()

	// Restart up to batchSize pods
	restarted := int32(0)
	for _, pod := range podsToRestart {
		if restarted >= batchSize {
			break
		}

		// 1. Try dynamic config update first (no restart needed)
		if oldConfig != nil && newConfig != nil {
			// Lazily create the Aerospike client once for all pods.
			if aeroClient == nil {
				var clientErr error
				aeroClient, clientErr = r.getAerospikeClient(ctx, cluster)
				if clientErr != nil {
					log.V(1).Info("Could not create Aerospike client for dynamic config, will fall back to restart", "error", clientErr)
				}
			}
			if aeroClient != nil && r.tryDynamicConfigUpdate(ctx, cluster, pod, oldConfig, newConfig, aeroClient) {
				log.Info("Dynamic config update succeeded, no restart needed", "pod", pod.Name)
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
				// Update config hash annotation so next reconcile won't re-restart this pod.
				if err := r.updatePodConfigHash(ctx, pod, desiredHash); err != nil {
					log.Error(err, "Failed to update pod config hash after warm restart", "pod", pod.Name)
				}
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

// updatePodConfigHash updates the config hash annotation on a pod after a warm restart.
func (r *AerospikeCEClusterReconciler) updatePodConfigHash(ctx context.Context, pod *corev1.Pod, hash string) error {
	podCopy := pod.DeepCopy()
	if podCopy.Annotations == nil {
		podCopy.Annotations = make(map[string]string)
	}
	podCopy.Annotations[utils.ConfigHashAnnotation] = hash
	return r.Update(ctx, podCopy)
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
