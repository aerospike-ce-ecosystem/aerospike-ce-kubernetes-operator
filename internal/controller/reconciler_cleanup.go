package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	asdbcev1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/storage"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/utils"
)

func (r *AerospikeCEClusterReconciler) handleDeletion(
	ctx context.Context,
	cluster *asdbcev1alpha1.AerospikeCECluster,
) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(cluster, utils.StorageFinalizer) {
		return ctrl.Result{}, nil
	}

	log.Info("Handling cluster deletion")

	// Selectively delete PVCs for volumes that have cascadeDelete enabled
	if cluster.Spec.Storage != nil {
		stsList, err := r.listClusterStatefulSets(ctx, cluster)
		if err != nil {
			return ctrl.Result{}, err
		}
		for _, sts := range stsList.Items {
			if err := storage.DeleteCascadeDeletePVCs(ctx, r.Client, cluster.Namespace, sts.Name, cluster.Spec.Storage); err != nil {
				if !errors.IsNotFound(err) {
					return ctrl.Result{}, err
				}
			}
		}
	}

	// Re-fetch before removing finalizer to avoid conflict on stale object.
	latest, err := r.refetchCluster(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace})
	if err != nil {
		return ctrl.Result{}, err
	}

	controllerutil.RemoveFinalizer(latest, utils.StorageFinalizer)
	if err := r.Update(ctx, latest); err != nil {
		if errors.IsConflict(err) {
			log.V(1).Info("Conflict removing finalizer, will requeue")
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	log.Info("Cluster deletion handled successfully")
	return ctrl.Result{}, nil
}
