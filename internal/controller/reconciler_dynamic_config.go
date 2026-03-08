package controller

import (
	"context"
	"fmt"
	"strings"

	aero "github.com/aerospike/aerospike-client-go/v8"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	ackov1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/configdiff"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/metrics"
)

// appliedChange tracks a successfully applied dynamic config change along with
// the command needed to roll it back.
type appliedChange struct {
	change   configdiff.Change
	rollback string // the set-config command to revert this change; empty if rollback not possible
}

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
	log := logf.FromContext(ctx).WithValues("pod", pod.Name, "cluster", cluster.Name)

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
			"staticChanges", len(diff.Static))
		return false
	}

	// Find the node corresponding to this pod
	node := findNodeForPod(aeroClient, pod)
	if node == nil {
		log.Info("Could not find Aerospike node for pod, skipping dynamic update")
		return false
	}

	// Pre-flight: validate all changes before applying any
	if err := validateDynamicChanges(diff.Dynamic); err != nil {
		log.Error(err, "Pre-flight validation failed for dynamic config changes")
		return false
	}

	// Apply each dynamic change, tracking successfully applied changes for rollback.
	var applied []appliedChange

	for i, change := range diff.Dynamic {
		cmd, err := buildSetConfigCommand(change)
		if err != nil {
			log.Error(err, "Invalid dynamic config change", "change", change)
			r.rollbackDynamicChanges(log, node, applied)
			log.Info("Rolled back partial dynamic config changes, falling back to cold restart",
				"appliedBeforeFailure", len(applied))
			return false
		}
		log.Info("Applying dynamic config", "command", cmd, "index", i, "total", len(diff.Dynamic))

		result, err := asinfoCommandOnNode(node, cmd)
		if err != nil {
			log.Error(err, "Dynamic config command failed", "command", cmd)
			logAppliedChanges(log, applied, i, len(diff.Dynamic))
			r.rollbackDynamicChanges(log, node, applied)
			log.Info("Rolled back partial dynamic config changes, falling back to cold restart",
				"appliedBeforeFailure", len(applied))
			return false
		}
		if result != "ok" {
			log.Info("Dynamic config command returned non-ok", "command", cmd, "result", result)
			logAppliedChanges(log, applied, i, len(diff.Dynamic))
			r.rollbackDynamicChanges(log, node, applied)
			log.Info("Rolled back partial dynamic config changes, falling back to cold restart",
				"appliedBeforeFailure", len(applied))
			return false
		}

		// Build rollback command using the old value
		rollbackChange := configdiff.Change{
			Path:      change.Path,
			Context:   change.Context,
			Key:       change.Key,
			NewValue:  change.OldValue,
			Namespace: change.Namespace,
		}
		rollbackCmd, rbErr := buildSetConfigCommand(rollbackChange)
		if rbErr != nil {
			// Cannot build rollback command — log but still track as applied
			log.V(1).Info("Cannot build rollback command for applied change",
				"change", change.Path, "oldValue", change.OldValue, "error", rbErr)
			rollbackCmd = "" // empty means rollback not possible
		}
		applied = append(applied, appliedChange{change: change, rollback: rollbackCmd})
	}

	// All dynamic changes applied successfully — update the config hash annotation
	// on the pod so that the rolling restart logic doesn't delete it.
	desiredHash := configHash(&ackov1alpha1.AerospikeConfigSpec{Value: newConfig})

	if desiredHash != "" {
		if err := r.updatePodConfigHash(ctx, pod, desiredHash); err != nil {
			log.Error(err, "Failed to update pod config hash after dynamic update")
			// This is non-fatal; the pod may get restarted but config is already applied
		}
	}

	metrics.DynamicConfigUpdatesTotal.WithLabelValues(cluster.Namespace, cluster.Name).Inc()
	r.Recorder.Eventf(cluster, corev1.EventTypeNormal, EventDynamicConfigApplied,
		"Dynamic config applied to pod %s (%d changes)", pod.Name, len(diff.Dynamic))
	log.Info("Dynamic config update successful", "changes", len(diff.Dynamic))

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

// validateDynamicChanges performs pre-flight validation on all dynamic config
// changes to catch obvious errors before applying any of them. This prevents
// partial config state where some changes succeed and others fail.
func validateDynamicChanges(changes []configdiff.Change) error {
	var errs []string
	for _, change := range changes {
		if _, err := buildSetConfigCommand(change); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("pre-flight validation failed for %d change(s): %s",
			len(errs), strings.Join(errs, "; "))
	}
	return nil
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

// rollbackDynamicChanges attempts to revert previously applied dynamic config changes.
// This is best-effort: if rollback fails, it is logged but does not cause an error.
// The caller will fall back to a cold restart which applies the correct config.
func (r *AerospikeClusterReconciler) rollbackDynamicChanges(
	log logr.Logger,
	node *aero.Node,
	applied []appliedChange,
) {
	if len(applied) == 0 {
		return
	}

	log.Info("Attempting rollback of applied dynamic config changes", "count", len(applied))

	rollbackFailed := 0
	for i := len(applied) - 1; i >= 0; i-- {
		ac := applied[i]
		if ac.rollback == "" {
			log.Info("No rollback command available for change, skipping",
				"change", ac.change.Path, "appliedValue", ac.change.NewValue)
			rollbackFailed++
			continue
		}

		log.Info("Rolling back dynamic config change", "command", ac.rollback, "change", ac.change.Path)
		result, err := asinfoCommandOnNode(node, ac.rollback)
		if err != nil {
			log.Error(err, "Rollback command failed", "command", ac.rollback, "change", ac.change.Path)
			rollbackFailed++
			continue
		}
		if result != "ok" {
			log.Info("Rollback command returned non-ok", "command", ac.rollback, "result", result, "change", ac.change.Path)
			rollbackFailed++
			continue
		}
		log.Info("Successfully rolled back dynamic config change", "change", ac.change.Path)
	}

	if rollbackFailed > 0 {
		log.Info("Some rollback commands failed, cold restart will apply correct config",
			"failedRollbacks", rollbackFailed, "totalApplied", len(applied))
	} else {
		log.Info("All applied dynamic config changes rolled back successfully")
	}
}

// logAppliedChanges logs which changes were successfully applied before a failure,
// so operators can investigate partial config state if needed.
func logAppliedChanges(log logr.Logger, applied []appliedChange, failedIdx, total int) {
	if len(applied) == 0 {
		return
	}
	paths := make([]string, 0, len(applied))
	for _, ac := range applied {
		paths = append(paths, fmt.Sprintf("%s=%v", ac.change.Path, ac.change.NewValue))
	}
	log.Info("Dynamic config partially applied before failure",
		"appliedCount", len(applied), "failedAtIndex", failedIdx, "totalChanges", total,
		"appliedChanges", strings.Join(paths, ", "))
}
