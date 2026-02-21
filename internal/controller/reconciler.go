package controller

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	asdbcev1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/utils"
)

// AerospikeCEClusterReconciler reconciles an AerospikeCECluster object.
type AerospikeCEClusterReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// RBAC markers
// +kubebuilder:rbac:groups=acko.io,resources=aerospikececlusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=acko.io,resources=aerospikececlusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=acko.io,resources=aerospikececlusters/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cilium.io,resources=ciliumnetworkpolicies,verbs=get;list;watch;create;update;patch;delete

func (r *AerospikeCEClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// 1. Fetch CR
	cluster := &asdbcev1alpha1.AerospikeCECluster{}
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// 2. Handle deletion
	if !cluster.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, cluster)
	}

	// 3. Add finalizer
	if !controllerutil.ContainsFinalizer(cluster, utils.StorageFinalizer) {
		controllerutil.AddFinalizer(cluster, utils.StorageFinalizer)
		if err := r.Update(ctx, cluster); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// 4. Check if paused
	if cluster.Spec.Paused != nil && *cluster.Spec.Paused {
		log.Info("Reconciliation paused")
		return ctrl.Result{}, nil
	}

	// 5. Set phase to InProgress
	if err := r.setPhase(ctx, cluster, asdbcev1alpha1.AerospikePhaseInProgress); err != nil {
		return ctrl.Result{}, err
	}

	// 6. Reconcile headless service
	if err := r.reconcileHeadlessService(ctx, cluster); err != nil {
		log.Error(err, "Failed to reconcile headless service")
		r.Recorder.Eventf(cluster, corev1.EventTypeWarning, "ReconcileError", "Headless service: %v", err)
		return ctrl.Result{}, err
	}

	// 7. Get rack list (default rack if not specified)
	racks := r.getRacks(cluster)

	// 8. Reconcile each rack
	for _, rack := range racks {
		// ConfigMap
		if err := r.reconcileConfigMap(ctx, cluster, &rack); err != nil {
			log.Error(err, "Failed to reconcile ConfigMap", "rack", rack.ID)
			return ctrl.Result{}, err
		}

		// StatefulSet
		if err := r.reconcileStatefulSet(ctx, cluster, &rack); err != nil {
			log.Error(err, "Failed to reconcile StatefulSet", "rack", rack.ID)
			return ctrl.Result{}, err
		}
	}

	// 9. Clean up removed racks
	if err := r.cleanupRemovedRacks(ctx, cluster, racks); err != nil {
		log.Error(err, "Failed to clean up removed racks")
		return ctrl.Result{}, err
	}

	// 10. Reconcile PDB
	if err := r.reconcilePDB(ctx, cluster); err != nil {
		log.Error(err, "Failed to reconcile PDB")
		return ctrl.Result{}, err
	}

	// 10.5. Reconcile Monitoring (metrics Service + ServiceMonitor)
	if err := r.reconcileMonitoring(ctx, cluster); err != nil {
		log.Error(err, "Failed to reconcile monitoring")
		return ctrl.Result{}, err
	}

	// 10.6. Reconcile NetworkPolicy
	if err := r.reconcileNetworkPolicy(ctx, cluster); err != nil {
		log.Error(err, "Failed to reconcile network policy")
		return ctrl.Result{}, err
	}

	// 11. Rolling restart if needed
	for _, rack := range racks {
		restarted, err := r.reconcileRollingRestart(ctx, cluster, &rack)
		if err != nil {
			log.Error(err, "Failed rolling restart", "rack", rack.ID)
			return ctrl.Result{}, err
		}
		if restarted {
			// Requeue to check again
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
	}

	// 12. Update status
	if err := r.updateStatus(ctx, cluster); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	// 13. Set phase to Completed
	if err := r.setPhase(ctx, cluster, asdbcev1alpha1.AerospikePhaseCompleted); err != nil {
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

// setPhase updates the cluster's phase.
func (r *AerospikeCEClusterReconciler) setPhase(ctx context.Context, cluster *asdbcev1alpha1.AerospikeCECluster, phase asdbcev1alpha1.AerospikePhase) error {
	if cluster.Status.Phase == phase {
		return nil
	}
	cluster.Status.Phase = phase
	return r.Status().Update(ctx, cluster)
}

// configHash computes a SHA256 hash of the aerospike config for change detection.
func configHash(config *asdbcev1alpha1.AerospikeConfigSpec) string {
	if config == nil {
		return ""
	}
	h := sha256.Sum256(fmt.Appendf(nil, "%v", *config))
	return fmt.Sprintf("%x", h[:8])
}

// SetupWithManager sets up the controller with the Manager.
func (r *AerospikeCEClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&asdbcev1alpha1.AerospikeCECluster{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&policyv1.PodDisruptionBudget{}).
		Named("aerospikececluster").
		Complete(r)
}
