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
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/utils"
)

// reconcileRollingRestart checks if pods need restart due to config changes.
// Returns true if a restart was triggered (caller should requeue).
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

	// Process pods in reverse order (highest ordinal first)
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
			log.Info("Pod config hash mismatch, deleting for restart", "pod", podName, "current", currentHash, "desired", desiredHash)
			if err := r.Delete(ctx, pod); err != nil && !errors.IsNotFound(err) {
				return false, err
			}
			return true, nil
		}
	}

	return false, nil
}
