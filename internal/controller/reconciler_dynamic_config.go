package controller

import (
	"context"
	"fmt"
	"strings"

	aero "github.com/aerospike/aerospike-client-go/v8"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	ackov1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/configdiff"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/metrics"
)

// tryDynamicConfigUpdate attempts to apply config changes dynamically without
// restarting pods. Returns true if all changes were applied dynamically and
// no restart is needed.
func (r *AerospikeClusterReconciler) tryDynamicConfigUpdate(
	ctx context.Context,
	cluster *ackov1alpha1.AerospikeCluster,
	pod *corev1.Pod,
	oldConfig, newConfig map[string]any,
	aeroClient *aero.Client,
) bool {
	log := logf.FromContext(ctx)

	// Check if dynamic config update is enabled
	if cluster.Spec.EnableDynamicConfigUpdate == nil || !*cluster.Spec.EnableDynamicConfigUpdate {
		return false
	}

	// Diff the configs
	diff := configdiff.Diff(oldConfig, newConfig)
	if !diff.HasChanges() {
		return true // No changes at all
	}

	// If there are static changes, dynamic update alone is not sufficient
	if diff.HasStaticChanges() {
		log.Info("Config has static changes, dynamic update not sufficient",
			"pod", pod.Name, "staticChanges", len(diff.Static))
		return false
	}

	// Find the node corresponding to this pod
	node := findNodeForPod(aeroClient, pod)
	if node == nil {
		log.Info("Could not find Aerospike node for pod, skipping dynamic update", "pod", pod.Name)
		return false
	}

	// Apply each dynamic change
	for _, change := range diff.Dynamic {
		cmd, err := buildSetConfigCommand(change)
		if err != nil {
			log.Error(err, "Invalid dynamic config change", "pod", pod.Name, "change", change)
			return false
		}
		log.Info("Applying dynamic config", "pod", pod.Name, "command", cmd)

		result, err := asinfoCommandOnNode(node, cmd)
		if err != nil {
			log.Error(err, "Dynamic config command failed", "pod", pod.Name, "command", cmd)
			return false
		}
		if result != "ok" {
			log.Info("Dynamic config command returned non-ok", "pod", pod.Name, "command", cmd, "result", result)
			return false
		}
	}

	// All dynamic changes applied successfully — update the config hash annotation
	// on the pod so that the rolling restart logic doesn't delete it.
	desiredHash := configHash(&ackov1alpha1.AerospikeConfigSpec{Value: newConfig})

	if desiredHash != "" {
		if err := r.updatePodConfigHash(ctx, pod, desiredHash); err != nil {
			log.Error(err, "Failed to update pod config hash after dynamic update", "pod", pod.Name)
			// This is non-fatal; the pod may get restarted but config is already applied
		}
	}

	metrics.DynamicConfigUpdatesTotal.WithLabelValues(cluster.Namespace, cluster.Name).Inc()
	r.Recorder.Eventf(cluster, corev1.EventTypeNormal, EventDynamicConfigApplied,
		"Dynamic config applied to pod %s (%d changes)", pod.Name, len(diff.Dynamic))
	log.Info("Dynamic config update successful", "pod", pod.Name, "changes", len(diff.Dynamic))

	// Update pod status with dynamic config status
	r.updateDynamicConfigStatus(ctx, cluster, pod.Name, "Applied")

	return true
}

// buildSetConfigCommand builds the asinfo set-config command for a change.
// Returns an error if any field contains characters that could break the
// asinfo protocol (semicolons or colons used as delimiters).
func buildSetConfigCommand(change configdiff.Change) (string, error) {
	valueStr := fmt.Sprintf("%v", change.NewValue)
	for _, field := range []struct{ name, val string }{
		{"key", change.Key},
		{"context", change.Context},
		{"namespace", change.Namespace},
		{"value", valueStr},
	} {
		if strings.ContainsAny(field.val, ";:") {
			return "", fmt.Errorf("invalid character in %s %q: must not contain ';' or ':'", field.name, field.val)
		}
	}

	if change.Namespace != "" {
		// Namespace-scoped parameter
		return fmt.Sprintf("set-config:context=namespace;id=%s;%s=%v",
			change.Namespace, change.Key, change.NewValue), nil
	}

	return fmt.Sprintf("set-config:context=%s;%s=%v",
		change.Context, change.Key, change.NewValue), nil
}

// findNodeForPod finds the Aerospike node that corresponds to a given pod by
// matching the pod IP. Returns nil if no match is found (no single-node fallback
// to avoid applying config to the wrong node).
func findNodeForPod(aeroClient *aero.Client, pod *corev1.Pod) *aero.Node {
	podIP := pod.Status.PodIP
	if podIP == "" {
		return nil
	}

	for _, node := range aeroClient.GetNodes() {
		host := node.GetHost()
		if host != nil && host.Name == podIP {
			return node
		}
	}

	return nil
}

// updateDynamicConfigStatus updates the DynamicConfigStatus field in the pod's
// status within the cluster CR. Uses Patch (MergeFrom) for atomic updates to
// avoid race conditions with concurrent reconcile loops.
// Failures are non-fatal: they are logged and reported as warning Events since
// the caller cannot meaningfully retry.
func (r *AerospikeClusterReconciler) updateDynamicConfigStatus(
	ctx context.Context,
	cluster *ackov1alpha1.AerospikeCluster,
	podName, status string,
) {
	log := logf.FromContext(ctx)

	// Re-fetch the cluster to get the latest status
	latest := &ackov1alpha1.AerospikeCluster{}
	if err := r.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace}, latest); err != nil {
		log.Error(err, "Failed to re-fetch cluster for dynamic config status update", "pod", podName)
		r.Recorder.Eventf(cluster, corev1.EventTypeWarning, EventDynamicConfigStatusFailed,
			"Failed to update dynamic config status for pod %s: %v", podName, err)
		return
	}

	if latest.Status.Pods == nil {
		latest.Status.Pods = make(map[string]ackov1alpha1.AerospikePodStatus)
	}

	base := latest.DeepCopy()
	podStatus := latest.Status.Pods[podName]
	podStatus.DynamicConfigStatus = status
	latest.Status.Pods[podName] = podStatus
	if err := r.Status().Patch(ctx, latest, client.MergeFrom(base)); err != nil {
		log.Error(err, "Failed to patch dynamic config status", "pod", podName)
		r.Recorder.Eventf(cluster, corev1.EventTypeWarning, EventDynamicConfigStatusFailed,
			"Failed to update dynamic config status for pod %s: %v", podName, err)
	}
}
