package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	asdbcev1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/metrics"
	aerotmpl "github.com/ksr/aerospike-ce-kubernetes-operator/internal/template"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/utils"
)

const defaultReconcileRetryInterval = 5 * time.Second

// AerospikeCEClusterReconciler reconciles an AerospikeCECluster object.
type AerospikeCEClusterReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Recorder   record.EventRecorder
	RestConfig *rest.Config
	// KubeClientset is a cached kubernetes.Clientset for pod exec operations.
	KubeClientset kubernetes.Interface
	kubeClientMu  sync.Mutex
}

// RBAC markers
// +kubebuilder:rbac:groups=acko.io,resources=aerospikececlusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=acko.io,resources=aerospikececlusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=acko.io,resources=aerospikececlusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=acko.io,resources=aerospikececlustertemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=acko.io,resources=aerospikececlustertemplates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;delete
// +kubebuilder:rbac:groups="",resources=pods/exec,verbs=create
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheusrules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cilium.io,resources=ciliumnetworkpolicies,verbs=get;list;watch;create;update;patch;delete

func (r *AerospikeCEClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	reconcileStart := time.Now()

	// 1. Fetch CR
	cluster := &asdbcev1alpha1.AerospikeCECluster{}
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		if errors.IsNotFound(err) {
			// Cluster deleted — clean up metrics
			metrics.CleanupClusterMetrics(req.Namespace, req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Record reconcile duration on exit
	defer func() {
		metrics.ReconcileDuration.WithLabelValues(cluster.Namespace, cluster.Name).
			Observe(time.Since(reconcileStart).Seconds())
	}()

	// 2. Handle deletion
	if !cluster.DeletionTimestamp.IsZero() {
		result, err := r.handleDeletion(ctx, cluster)
		if err == nil {
			metrics.CleanupClusterMetrics(cluster.Namespace, cluster.Name)
		}
		return result, err
	}

	// 3. Add finalizer
	if !controllerutil.ContainsFinalizer(cluster, utils.StorageFinalizer) {
		controllerutil.AddFinalizer(cluster, utils.StorageFinalizer)
		if err := r.Update(ctx, cluster); err != nil {
			if errors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// 4. Check if paused
	if cluster.Spec.Paused != nil && *cluster.Spec.Paused {
		log.Info("Reconciliation paused")
		if err := r.setPhase(ctx, cluster, asdbcev1alpha1.AerospikePhasePaused, "Reconciliation paused by user"); err != nil {
			if errors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// 4.5 Template resolution: fetch/snapshot template and apply to in-memory spec.
	if cluster.Spec.TemplateRef != nil {
		resolveResult, err := aerotmpl.Resolve(ctx, r.Client, cluster)
		if err != nil {
			log.Error(err, "Failed to resolve template", "template", cluster.Spec.TemplateRef.Name)
			r.Recorder.Eventf(cluster, corev1.EventTypeWarning, "TemplateResolutionError",
				"Failed to resolve template %q: %v", cluster.Spec.TemplateRef.Name, err)
			return ctrl.Result{}, err
		}
		if resolveResult.SnapshotUpdated {
			// Persist the new snapshot to the API server immediately so that
			// subsequent setPhase/updateStatusAndPhase calls (which re-fetch
			// the object) do not overwrite it with the stale version.
			// Status must be persisted BEFORE the Annotation Patch: the Patch
			// response refreshes the full cluster object (incl. Status) from the
			// server, which would nil-out the in-memory snapshot if Status has
			// not been saved yet.
			snapshotRV := cluster.Status.TemplateSnapshot.ResourceVersion
			if err := r.Status().Update(ctx, cluster); err != nil {
				if errors.IsConflict(err) {
					return ctrl.Result{Requeue: true}, nil
				}
				return ctrl.Result{}, err
			}
			r.Recorder.Eventf(cluster, corev1.EventTypeNormal, "TemplateApplied",
				"Applied template %q (rv: %s)",
				cluster.Spec.TemplateRef.Name,
				snapshotRV)
		}
		// Remove the resync annotation from the API server now that the snapshot is persisted.
		// This Patch runs after Status.Update so it does not overwrite the snapshot.
		if resolveResult.AnnotationNeedsCleanup {
			patch := client.MergeFrom(cluster.DeepCopy())
			delete(cluster.Annotations, aerotmpl.AnnotationResyncTemplate)
			if err := r.Patch(ctx, cluster, patch); err != nil {
				if errors.IsConflict(err) {
					return ctrl.Result{Requeue: true}, nil
				}
				return ctrl.Result{}, err
			}
		}
		for _, w := range resolveResult.Warnings {
			r.Recorder.Eventf(cluster, corev1.EventTypeWarning, "ValidationWarning", "%s", w)
		}
	}

	// 5. Set phase to InProgress only when the spec has actually changed
	// (i.e., observedGeneration is behind the current generation).
	// This prevents a Completed->InProgress->Completed feedback loop
	// where each status update triggers a new reconcile.
	if cluster.Status.ObservedGeneration != cluster.Generation ||
		cluster.Status.Phase == "" {
		if err := r.setPhase(ctx, cluster, asdbcev1alpha1.AerospikePhaseInProgress, "Reconciliation started"); err != nil {
			if errors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, err
		}
	}

	// 6. Reconcile headless service
	if err := r.reconcileHeadlessService(ctx, cluster); err != nil {
		log.Error(err, "Failed to reconcile headless service")
		r.Recorder.Eventf(cluster, corev1.EventTypeWarning, "ReconcileError", "Headless service: %v", err)
		metrics.ReconcileErrorsTotal.WithLabelValues(cluster.Namespace, cluster.Name, metrics.ReasonService).Inc()
		return ctrl.Result{}, err
	}

	// 6b. Reconcile per-pod services
	if err := r.reconcilePodServices(ctx, cluster); err != nil {
		log.Error(err, "Failed to reconcile per-pod services")
		metrics.ReconcileErrorsTotal.WithLabelValues(cluster.Namespace, cluster.Name, metrics.ReasonService).Inc()
		return ctrl.Result{}, err
	}

	// 7-17. Reconcile cluster resources
	return r.reconcileCluster(ctx, req.NamespacedName, cluster)
}

// reconcileCluster reconciles all cluster resources (racks, services, operations, ACL, status).
func (r *AerospikeCEClusterReconciler) reconcileCluster(
	ctx context.Context,
	namespacedName types.NamespacedName,
	cluster *asdbcev1alpha1.AerospikeCECluster,
) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	racks := r.getRacks(cluster)

	// Pre-compute effective config and hash per rack.
	type rackInfo struct {
		effectiveConfig *asdbcev1alpha1.AerospikeConfigSpec
		hash            string
		rackSize        int32
	}
	rackInfos := make([]rackInfo, len(racks))
	rackSizes := make([]int32, len(racks))
	for i, rack := range racks {
		ec := r.getEffectiveConfig(cluster, &rack)
		rackSizes[i] = r.getRackSize(cluster, racks, i)
		rackInfos[i] = rackInfo{
			effectiveConfig: ec,
			hash:            configHash(ec),
			rackSize:        rackSizes[i],
		}
	}

	// Detect scaling and update phase accordingly.
	scalingUp, scalingDown, err := r.detectScaling(ctx, cluster, racks, rackSizes)
	if err != nil {
		return ctrl.Result{}, err
	}
	if scalingUp {
		if err := r.setPhase(ctx, cluster, asdbcev1alpha1.AerospikePhaseScalingUp, "Scaling up cluster"); err != nil && !errors.IsConflict(err) {
			return ctrl.Result{}, err
		}
	} else if scalingDown {
		if err := r.setPhase(ctx, cluster, asdbcev1alpha1.AerospikePhaseScalingDown, "Scaling down cluster"); err != nil && !errors.IsConflict(err) {
			return ctrl.Result{}, err
		}
	}

	// Reconcile each rack's ConfigMap + StatefulSet
	for i, rack := range racks {
		ri := rackInfos[i]
		if err := r.reconcileConfigMap(ctx, cluster, &rack, ri.effectiveConfig); err != nil {
			metrics.ReconcileErrorsTotal.WithLabelValues(cluster.Namespace, cluster.Name, metrics.ReasonConfigMap).Inc()
			return ctrl.Result{}, err
		}
		if err := r.reconcileStatefulSet(ctx, cluster, &rack, ri.effectiveConfig, ri.hash, ri.rackSize); err != nil {
			metrics.ReconcileErrorsTotal.WithLabelValues(cluster.Namespace, cluster.Name, metrics.ReasonStatefulSet).Inc()
			return ctrl.Result{}, err
		}
	}

	// Clean up removed racks
	if err := r.cleanupRemovedRacks(ctx, cluster, racks); err != nil {
		metrics.ReconcileErrorsTotal.WithLabelValues(cluster.Namespace, cluster.Name, metrics.ReasonStatefulSet).Inc()
		return ctrl.Result{}, err
	}

	// Reconcile auxiliary resources: PDB, Monitoring, NetworkPolicy
	auxReasons := []string{metrics.ReasonPDB, metrics.ReasonMonitoring, metrics.ReasonNetPolicy}
	for idx, fn := range []func(context.Context, *asdbcev1alpha1.AerospikeCECluster) error{
		r.reconcilePDB,
		r.reconcileMonitoring,
		r.reconcileNetworkPolicy,
	} {
		if err := fn(ctx, cluster); err != nil {
			metrics.ReconcileErrorsTotal.WithLabelValues(cluster.Namespace, cluster.Name, auxReasons[idx]).Inc()
			return ctrl.Result{}, err
		}
	}

	// Handle on-demand operations
	if inProgress, err := r.reconcileOperations(ctx, cluster); err != nil {
		metrics.ReconcileErrorsTotal.WithLabelValues(cluster.Namespace, cluster.Name, metrics.ReasonOperations).Inc()
		return ctrl.Result{}, err
	} else if inProgress {
		return ctrl.Result{RequeueAfter: defaultReconcileRetryInterval}, nil
	}

	// Rolling restart if needed
	for _, rack := range racks {
		restarted, err := r.reconcileRollingRestart(ctx, cluster, &rack)
		if err != nil {
			log.Error(err, "Failed rolling restart", "rack", rack.ID)
			metrics.ReconcileErrorsTotal.WithLabelValues(cluster.Namespace, cluster.Name, metrics.ReasonRestart).Inc()
			return ctrl.Result{}, err
		}
		if restarted {
			if err := r.setPhase(ctx, cluster, asdbcev1alpha1.AerospikePhaseRollingRestart,
				fmt.Sprintf("Rolling restart in progress for rack %d", rack.ID)); err != nil && !errors.IsConflict(err) {
				return ctrl.Result{}, err
			}
			return ctrl.Result{RequeueAfter: defaultReconcileRetryInterval}, nil
		}
	}

	// Reconcile ACL (non-fatal); capture error for ACLSynced condition.
	var aclErr error
	if cluster.Spec.AerospikeAccessControl != nil {
		if err := r.setPhase(ctx, cluster, asdbcev1alpha1.AerospikePhaseACLSync, "Synchronizing ACL roles and users"); err != nil && !errors.IsConflict(err) {
			return ctrl.Result{}, err
		}
	}
	if err := r.reconcileACL(ctx, cluster); err != nil {
		log.Error(err, "Failed to reconcile ACL")
		r.Recorder.Eventf(cluster, corev1.EventTypeWarning, "ACLSyncError", "ACL sync failed: %v", err)
		metrics.ReconcileErrorsTotal.WithLabelValues(cluster.Namespace, cluster.Name, metrics.ReasonACL).Inc()
		aclErr = err
	}

	// Update status and set phase to Completed.
	statusOpts := StatusUpdateOpts{ACLErr: aclErr}
	if err := r.updateStatusAndPhase(ctx, namespacedName, asdbcev1alpha1.AerospikePhaseCompleted, "Cluster is healthy and stable", statusOpts); err != nil {
		if errors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		}
		metrics.ReconcileErrorsTotal.WithLabelValues(cluster.Namespace, cluster.Name, metrics.ReasonStatus).Inc()
		return ctrl.Result{}, err
	}

	log.Info("Reconciliation completed successfully")
	return ctrl.Result{}, nil
}

// getRacks returns the list of racks. If no rack config, returns a default rack.
func (r *AerospikeCEClusterReconciler) getRacks(cluster *asdbcev1alpha1.AerospikeCECluster) []asdbcev1alpha1.Rack {
	if cluster.Spec.RackConfig != nil && len(cluster.Spec.RackConfig.Racks) > 0 {
		return cluster.Spec.RackConfig.Racks
	}
	return []asdbcev1alpha1.Rack{{ID: 0}}
}

// getRackSize returns the number of pods for a given rack.
func (r *AerospikeCEClusterReconciler) getRackSize(cluster *asdbcev1alpha1.AerospikeCECluster, racks []asdbcev1alpha1.Rack, rackIndex int) int32 {
	totalSize := cluster.Spec.Size
	numRacks := int32(len(racks))
	baseSize := totalSize / numRacks
	remainder := totalSize % numRacks

	if int32(rackIndex) < remainder {
		return baseSize + 1
	}
	return baseSize
}

// setPhase re-fetches the latest cluster object and updates its phase and reason.
// It handles conflict errors by returning a requeue result (nil error)
// so the caller can decide to requeue without logging a spurious error.
func (r *AerospikeCEClusterReconciler) setPhase(ctx context.Context, cluster *asdbcev1alpha1.AerospikeCECluster, phase asdbcev1alpha1.AerospikePhase, reason string) error {
	log := logf.FromContext(ctx)

	// Re-fetch the latest version to avoid "object has been modified" conflicts.
	latest, err := r.refetchCluster(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace})
	if err != nil {
		return err
	}

	if latest.Status.Phase == phase && latest.Status.PhaseReason == reason {
		return nil
	}

	latest.Status.Phase = phase
	latest.Status.PhaseReason = reason
	if err := r.Status().Update(ctx, latest); err != nil {
		if errors.IsConflict(err) {
			log.V(1).Info("Conflict updating phase, will requeue", "phase", phase)
			return err
		}
		return err
	}

	// Propagate the updated resource version back to the caller's object
	// so subsequent operations in the same reconcile loop use fresh data.
	cluster.ResourceVersion = latest.ResourceVersion
	cluster.Status.Phase = phase
	cluster.Status.PhaseReason = reason
	return nil
}

// configHash computes a deterministic SHA256 hash of the aerospike config for
// change detection. Uses json.Marshal which sorts map keys, unlike fmt.Sprintf
// which iterates maps in non-deterministic order.
func configHash(config *asdbcev1alpha1.AerospikeConfigSpec) string {
	if config == nil {
		return ""
	}
	return utils.ShortSHA256(config.Value)
}

// SetupWithManager sets up the controller with the Manager.
func (r *AerospikeCEClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&asdbcev1alpha1.AerospikeCECluster{},
			// AnnotationChangedPredicate allows the resync annotation
			// (acko.io/resync-template=true) to trigger reconciliation,
			// since annotation-only changes do not increment generation.
			builder.WithPredicates(predicate.Or(
				predicate.GenerationChangedPredicate{},
				predicate.AnnotationChangedPredicate{},
			)),
		).
		// GenerationChangedPredicate on Owns() suppresses reconcile triggers from
		// status-only updates on owned resources (e.g., StatefulSet ready replicas).
		// The controller reads pod state directly via listClusterPods, not from
		// StatefulSet status, so these events are noise.
		Owns(&appsv1.StatefulSet{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&corev1.Service{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&corev1.ConfigMap{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&policyv1.PodDisruptionBudget{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		// Watch AerospikeCEClusterTemplate changes and mark referencing clusters as out-of-sync.
		Watches(
			&asdbcev1alpha1.AerospikeCEClusterTemplate{},
			handler.EnqueueRequestsFromMapFunc(r.mapTemplateToCluster),
			builder.WithPredicates(predicate.GenerationChangedPredicate{}),
		).
		Named("aerospikececluster").
		WithOptions(controller.Options{MaxConcurrentReconciles: 2}).
		Complete(r)
}

// mapTemplateToCluster maps an AerospikeCEClusterTemplate change to all clusters
// that reference it, so the controller can mark them as out-of-sync.
func (r *AerospikeCEClusterReconciler) mapTemplateToCluster(ctx context.Context, obj client.Object) []reconcile.Request {
	log := logf.FromContext(ctx)

	// List all AerospikeCEClusters in the template's namespace.
	clusterList := &asdbcev1alpha1.AerospikeCEClusterList{}
	if err := r.List(ctx, clusterList, client.InNamespace(obj.GetNamespace())); err != nil {
		log.Error(err, "Failed to list clusters for template watch", "template", obj.GetName())
		return nil
	}

	var requests []reconcile.Request
	for i := range clusterList.Items {
		cl := &clusterList.Items[i]
		if cl.Spec.TemplateRef == nil || cl.Spec.TemplateRef.Name != obj.GetName() {
			continue
		}

		// Mark the cluster as out-of-sync by updating its snapshot status.
		if cl.Status.TemplateSnapshot != nil && cl.Status.TemplateSnapshot.Synced {
			latest := cl.DeepCopy()
			latest.Status.TemplateSnapshot.Synced = false
			if err := r.Status().Update(ctx, latest); err != nil {
				log.Error(err, "Failed to mark cluster template as drifted", "cluster", cl.Name)
			} else {
				r.Recorder.Eventf(cl, corev1.EventTypeWarning, "TemplateDrifted",
					"Template %q changed (rv: %s → %s); cluster using snapshot. Set annotation acko.io/resync-template=true to resync.",
					obj.GetName(),
					cl.Status.TemplateSnapshot.ResourceVersion,
					obj.GetResourceVersion(),
				)
			}
		}

		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: cl.Namespace,
				Name:      cl.Name,
			},
		})
	}
	return requests
}
