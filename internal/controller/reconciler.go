package controller

import (
	"context"
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
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	asdbcev1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/metrics"
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
		return ctrl.Result{}, nil
	}

	// 5. Set phase to InProgress only when the spec has actually changed
	// (i.e., observedGeneration is behind the current generation).
	// This prevents a Completed->InProgress->Completed feedback loop
	// where each status update triggers a new reconcile.
	if cluster.Status.ObservedGeneration != cluster.Generation ||
		cluster.Status.Phase == "" {
		if err := r.setPhase(ctx, cluster, asdbcev1alpha1.AerospikePhaseInProgress); err != nil {
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
		return ctrl.Result{}, err
	}

	// 6b. Reconcile per-pod services
	if err := r.reconcilePodServices(ctx, cluster); err != nil {
		log.Error(err, "Failed to reconcile per-pod services")
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
	for i, rack := range racks {
		ec := r.getEffectiveConfig(cluster, &rack)
		rackInfos[i] = rackInfo{
			effectiveConfig: ec,
			hash:            configHash(ec),
			rackSize:        r.getRackSize(cluster, racks, i),
		}
	}

	// Reconcile each rack's ConfigMap + StatefulSet
	for i, rack := range racks {
		ri := rackInfos[i]
		if err := r.reconcileConfigMap(ctx, cluster, &rack, ri.effectiveConfig); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.reconcileStatefulSet(ctx, cluster, &rack, ri.effectiveConfig, ri.hash, ri.rackSize); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Clean up removed racks
	if err := r.cleanupRemovedRacks(ctx, cluster, racks); err != nil {
		return ctrl.Result{}, err
	}

	// Reconcile auxiliary resources: PDB, Monitoring, NetworkPolicy
	for _, fn := range []func(context.Context, *asdbcev1alpha1.AerospikeCECluster) error{
		r.reconcilePDB,
		r.reconcileMonitoring,
		r.reconcileNetworkPolicy,
	} {
		if err := fn(ctx, cluster); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Handle on-demand operations
	if inProgress, err := r.reconcileOperations(ctx, cluster); err != nil {
		return ctrl.Result{}, err
	} else if inProgress {
		return ctrl.Result{RequeueAfter: defaultReconcileRetryInterval}, nil
	}

	// Rolling restart if needed
	for _, rack := range racks {
		restarted, err := r.reconcileRollingRestart(ctx, cluster, &rack)
		if err != nil {
			log.Error(err, "Failed rolling restart", "rack", rack.ID)
			return ctrl.Result{}, err
		}
		if restarted {
			return ctrl.Result{RequeueAfter: defaultReconcileRetryInterval}, nil
		}
	}

	// Reconcile ACL (non-fatal)
	if err := r.reconcileACL(ctx, cluster); err != nil {
		log.Error(err, "Failed to reconcile ACL")
		r.Recorder.Eventf(cluster, corev1.EventTypeWarning, "ACLSyncError", "ACL sync failed: %v", err)
	}

	// Update status and set phase to Completed.
	if err := r.updateStatusAndPhase(ctx, namespacedName, asdbcev1alpha1.AerospikePhaseCompleted); err != nil {
		if errors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		}
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

// setPhase re-fetches the latest cluster object and updates its phase.
// It handles conflict errors by returning a requeue result (nil error)
// so the caller can decide to requeue without logging a spurious error.
func (r *AerospikeCEClusterReconciler) setPhase(ctx context.Context, cluster *asdbcev1alpha1.AerospikeCECluster, phase asdbcev1alpha1.AerospikePhase) error {
	log := logf.FromContext(ctx)

	// Re-fetch the latest version to avoid "object has been modified" conflicts.
	latest, err := r.refetchCluster(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace})
	if err != nil {
		return err
	}

	if latest.Status.Phase == phase {
		return nil
	}

	latest.Status.Phase = phase
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
			builder.WithPredicates(predicate.GenerationChangedPredicate{}),
		).
		// GenerationChangedPredicate on Owns() suppresses reconcile triggers from
		// status-only updates on owned resources (e.g., StatefulSet ready replicas).
		// The controller reads pod state directly via listClusterPods, not from
		// StatefulSet status, so these events are noise.
		Owns(&appsv1.StatefulSet{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&corev1.Service{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&corev1.ConfigMap{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&policyv1.PodDisruptionBudget{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Named("aerospikececluster").
		WithOptions(controller.Options{MaxConcurrentReconciles: 2}).
		Complete(r)
}
