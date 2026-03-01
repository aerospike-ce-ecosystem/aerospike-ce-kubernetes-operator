package controller

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	asdbcev1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/metrics"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/podutil"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/utils"
)

// StatusUpdateOpts carries optional annotations for updateStatusAndPhase.
type StatusUpdateOpts struct {
	// ACLErr, if non-nil, sets the ACLSynced condition to False.
	// If nil and ACL spec is present, ACLSynced is set to True.
	ACLErr error
	// ACLSynced indicates whether ACL was actually applied (not skipped).
	// When false and ACLErr is nil, ACL sync was skipped (e.g., no ready pods).
	ACLSynced bool
	// RestartInProgress indicates a rolling restart is active (MigrationComplete=False).
	RestartInProgress bool
	// Paused indicates reconciliation is paused (ReconciliationPaused=True).
	Paused bool
}

// updateStatusAndPhase re-fetches the latest cluster object from the API server,
// populates status fields, sets the desired phase and reason, and performs a status update.
// This pattern avoids "object has been modified" conflict errors that occur when
// updating status on a stale object.
// If the status already matches the desired state, the update is skipped to avoid
// triggering unnecessary reconciliation loops.
func (r *AerospikeCEClusterReconciler) updateStatusAndPhase(
	ctx context.Context,
	namespacedName types.NamespacedName,
	phase asdbcev1alpha1.AerospikePhase,
	phaseReason string,
	opts StatusUpdateOpts,
) error {
	log := logf.FromContext(ctx)

	// Re-fetch the latest version from the API server.
	latest, err := r.refetchCluster(ctx, namespacedName)
	if err != nil {
		return err
	}

	// Capture the previous state for comparison (before populateStatus modifies it).
	prevPhase := latest.Status.Phase
	prevPhaseReason := latest.Status.PhaseReason
	prevSize := latest.Status.Size
	prevGeneration := latest.Status.ObservedGeneration
	prevConditions := conditionsSnapshot(latest.Status.Conditions)

	readyCount, err := r.populateStatus(ctx, latest)
	if err != nil {
		return err
	}
	latest.Status.Phase = phase
	latest.Status.PhaseReason = phaseReason

	// Apply fine-grained conditions.
	setFineGrainedConditions(latest, opts)

	// Skip the update if nothing meaningful changed to avoid
	// triggering a reconciliation feedback loop via the watch.
	if prevPhase == phase &&
		prevPhaseReason == phaseReason &&
		prevSize == readyCount &&
		prevGeneration == latest.Generation &&
		!conditionsChanged(prevConditions, latest.Status.Conditions) {
		log.V(1).Info("Status unchanged, skipping update",
			"readyPods", readyCount, "desiredSize", latest.Spec.Size, "phase", phase)
		return nil
	}

	log.Info("Updating status", "readyPods", readyCount, "desiredSize", latest.Spec.Size, "phase", phase, "phaseReason", phaseReason)

	// On successful completion: record the full applied spec and refresh per-node info.
	if phase == asdbcev1alpha1.AerospikePhaseCompleted {
		// AppliedSpec records the last successfully reconciled spec for drift detection.
		latest.Status.AppliedSpec = latest.Spec.DeepCopy()

		// Enrich pod status with per-node Aerospike info (NodeID, ClusterName, endpoints).
		if aeroInfoMap := r.collectAerospikeInfo(ctx, latest); aeroInfoMap != nil {
			for podName, info := range aeroInfoMap {
				if ps, ok := latest.Status.Pods[podName]; ok {
					ps.NodeID = info.NodeID
					ps.ClusterName = info.ClusterName
					ps.AccessEndpoints = info.AccessEndpoints
					latest.Status.Pods[podName] = ps
				}
			}
		}
	}

	// Update Prometheus metrics
	metrics.ClusterPhase.WithLabelValues(latest.Namespace, latest.Name).Set(metrics.PhaseToFloat(string(phase)))
	metrics.ClusterReadyPods.WithLabelValues(latest.Namespace, latest.Name).Set(float64(readyCount))

	return r.Status().Update(ctx, latest)
}

// populateStatus fills in the cluster's status fields and returns the ready pod count.
func (r *AerospikeCEClusterReconciler) populateStatus(
	ctx context.Context,
	cluster *asdbcev1alpha1.AerospikeCECluster,
) (int32, error) {
	log := logf.FromContext(ctx)

	// List all pods for this cluster
	podList, err := r.listClusterPods(ctx, cluster)
	if err != nil {
		return 0, err
	}

	podStatuses := make(map[string]asdbcev1alpha1.AerospikePodStatus, len(podList.Items))
	readyCount := int32(0)

	for i := range podList.Items {
		pod := &podList.Items[i]

		rackID := 0
		if rackStr, ok := pod.Labels[utils.RackLabel]; ok {
			if id, err := strconv.Atoi(rackStr); err != nil {
				log.V(1).Info("Failed to parse rack ID label", "pod", pod.Name, "label", rackStr, "error", err)
			} else {
				rackID = id
			}
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

		// Use the actual running image from the pod, not the desired spec image.
		// During rolling updates the pod may still run the old image.
		podImage := cluster.Spec.Image
		for _, c := range pod.Spec.Containers {
			if c.Name == podutil.AerospikeContainerName {
				podImage = c.Image
				break
			}
		}

		// Preserve dirty volumes from previous status; clear them once the pod is ready
		// (meaning the init container has already wiped them during restart).
		var dirtyVolumes []string
		if prev, exists := cluster.Status.Pods[pod.Name]; exists && len(prev.DirtyVolumes) > 0 {
			if !isReady {
				dirtyVolumes = prev.DirtyVolumes
			}
			// else: pod is ready → init container completed wipe → clear dirty volumes
		}

		// Preserve Aerospike node info from previous status.
		// These fields are refreshed via collectAerospikeInfo only when phase == Completed.
		var nodeID, clusterName string
		var accessEndpoints []string
		if prev, exists := cluster.Status.Pods[pod.Name]; exists {
			nodeID = prev.NodeID
			clusterName = prev.ClusterName
			accessEndpoints = prev.AccessEndpoints
		}

		podStatuses[pod.Name] = asdbcev1alpha1.AerospikePodStatus{
			PodIP:             pod.Status.PodIP,
			HostIP:            pod.Status.HostIP,
			Image:             podImage,
			PodPort:           int32(getServicePort(cluster)),
			ServicePort:       int32(getServicePort(cluster)),
			Rack:              rackID,
			IsRunningAndReady: isReady,
			ConfigHash:        configHash,
			PodSpecHash:       podSpecHash,
			DirtyVolumes:      dirtyVolumes,
			NodeID:            nodeID,
			ClusterName:       clusterName,
			AccessEndpoints:   accessEndpoints,
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

	// Update base conditions (Available, Ready).
	setCondition(cluster, asdbcev1alpha1.ConditionAvailable, readyCount > 0, "ClusterAvailable", "At least one pod is ready")
	setCondition(cluster, asdbcev1alpha1.ConditionReady, readyCount == cluster.Spec.Size, "AllPodsReady", fmt.Sprintf("%d/%d pods ready", readyCount, cluster.Spec.Size))

	return readyCount, nil
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
		ObservedGeneration: cluster.Generation,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	for i, existing := range cluster.Status.Conditions {
		if existing.Type == condType {
			if existing.Status != condStatus || existing.ObservedGeneration != cluster.Generation {
				cluster.Status.Conditions[i] = newCond
			}
			return
		}
	}

	cluster.Status.Conditions = append(cluster.Status.Conditions, newCond)
}

// setFineGrainedConditions sets all fine-grained status conditions:
// ConfigApplied, ReconciliationPaused, ACLSynced, MigrationComplete.
// Called from updateStatusAndPhase after populateStatus.
func setFineGrainedConditions(cluster *asdbcev1alpha1.AerospikeCECluster, o StatusUpdateOpts) {
	// ConfigApplied: true when all pods carry the same config hash as the desired config.
	desiredHash := configHash(cluster.Spec.AerospikeConfig)
	allConfigApplied := len(cluster.Status.Pods) > 0
	for _, ps := range cluster.Status.Pods {
		if ps.ConfigHash != desiredHash {
			allConfigApplied = false
			break
		}
	}
	if allConfigApplied {
		setCondition(cluster, asdbcev1alpha1.ConditionConfigApplied, true,
			"ConfigApplied", "All pods have the desired Aerospike configuration")
	} else {
		setCondition(cluster, asdbcev1alpha1.ConditionConfigApplied, false,
			"ConfigPending", "One or more pods do not yet have the desired configuration")
	}

	// ReconciliationPaused
	setCondition(cluster, asdbcev1alpha1.ConditionReconciliationPaused, o.Paused,
		"ReconciliationPaused", "Reconciliation is paused by user (spec.paused=true)")

	// ACLSynced — only set if ACL is configured
	if cluster.Spec.AerospikeAccessControl != nil {
		if o.ACLErr != nil {
			setCondition(cluster, asdbcev1alpha1.ConditionACLSynced, false,
				"ACLSyncFailed", o.ACLErr.Error())
		} else if o.ACLSynced {
			setCondition(cluster, asdbcev1alpha1.ConditionACLSynced, true,
				"ACLSyncSucceeded", "ACL roles and users are synchronized")
		} else {
			setCondition(cluster, asdbcev1alpha1.ConditionACLSynced, false,
				"ACLSyncPending", "ACL sync skipped: no ready pods available")
		}
	}

	// MigrationComplete — False while rolling restart is in progress
	setCondition(cluster, asdbcev1alpha1.ConditionMigrationComplete, !o.RestartInProgress,
		"MigrationComplete", "No pending data migrations")
}

// conditionsSnapshot returns a map of condition Type → Status for skip-check comparison.
func conditionsSnapshot(conds []metav1.Condition) map[string]metav1.ConditionStatus {
	m := make(map[string]metav1.ConditionStatus, len(conds))
	for _, c := range conds {
		m[c.Type] = c.Status
	}
	return m
}

// conditionsChanged returns true if any condition type or status differs between
// the snapshot taken before populateStatus and the current slice after all updates.
func conditionsChanged(prev map[string]metav1.ConditionStatus, cur []metav1.Condition) bool {
	if len(prev) != len(cur) {
		return true
	}
	for _, c := range cur {
		if s, ok := prev[c.Type]; !ok || s != c.Status {
			return true
		}
	}
	return false
}

// aeroPodInfo holds per-node Aerospike information collected via asinfo commands.
type aeroPodInfo struct {
	NodeID          string
	ClusterName     string
	AccessEndpoints []string
}

// collectAerospikeInfo connects to the Aerospike cluster and collects per-node
// information (NodeID, ClusterName, AccessEndpoints) keyed by pod name.
// Errors are logged at V(1) and the function returns nil rather than failing
// so that status updates are never blocked by an unreachable cluster.
func (r *AerospikeCEClusterReconciler) collectAerospikeInfo(
	ctx context.Context,
	cluster *asdbcev1alpha1.AerospikeCECluster,
) map[string]aeroPodInfo {
	log := logf.FromContext(ctx)

	aeroClient, err := r.getAerospikeClient(ctx, cluster)
	if err != nil {
		log.V(1).Info("Skipping Aerospike info collection: could not connect", "error", err)
		return nil
	}
	defer closeAerospikeClient(aeroClient)

	// Build a pod-IP → pod-name lookup from the current status pods.
	podIPToPodName := make(map[string]string, len(cluster.Status.Pods))
	for podName, ps := range cluster.Status.Pods {
		if ps.PodIP != "" {
			podIPToPodName[ps.PodIP] = podName
		}
	}

	result := make(map[string]aeroPodInfo)

	for _, node := range aeroClient.GetNodes() {
		nodeHost := node.GetHost()
		if nodeHost == nil {
			log.V(1).Info("Skipping Aerospike node with nil host info")
			continue
		}
		podName, ok := podIPToPodName[nodeHost.Name]
		if !ok {
			log.V(1).Info("Aerospike node IP not matched to any pod", "nodeIP", nodeHost.Name)
			continue
		}

		info := aeroPodInfo{}

		if nodeID, err := asinfoCommandOnNode(node, "node"); err == nil {
			info.NodeID = strings.TrimSpace(nodeID)
		} else {
			log.V(1).Info("Failed to get nodeID", "pod", podName, "error", err)
		}

		if clusterName, err := asinfoCommandOnNode(node, "cluster-name"); err == nil {
			info.ClusterName = strings.TrimSpace(clusterName)
		} else {
			log.V(1).Info("Failed to get cluster-name", "pod", podName, "error", err)
		}

		if serviceStr, err := asinfoCommandOnNode(node, "service"); err == nil {
			info.AccessEndpoints = parseServiceEndpoints(serviceStr)
		} else {
			log.V(1).Info("Failed to get service endpoints", "pod", podName, "error", err)
		}

		result[podName] = info
	}

	return result
}

// parseServiceEndpoints splits the asinfo "service" response (semicolon-separated
// "host:port" entries) into a string slice.
func parseServiceEndpoints(serviceStr string) []string {
	serviceStr = strings.TrimSpace(serviceStr)
	if serviceStr == "" {
		return nil
	}
	parts := strings.Split(serviceStr, ";")
	endpoints := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			endpoints = append(endpoints, p)
		}
	}
	return endpoints
}
