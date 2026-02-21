package controller

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	// Check if any volumes have cascadeDelete
	if cluster.Spec.Storage != nil {
		for _, vol := range cluster.Spec.Storage.Volumes {
			if vol.CascadeDelete && vol.Source.PersistentVolume != nil {
				// Delete PVCs for all racks
				stsList := &appsv1.StatefulSetList{}
				if err := r.List(ctx, stsList,
					client.InNamespace(cluster.Namespace),
					client.MatchingLabels(utils.SelectorLabelsForCluster(cluster.Name)),
				); err != nil {
					return ctrl.Result{}, err
				}
				for _, sts := range stsList.Items {
					if err := storage.DeletePVCsForStatefulSet(ctx, r.Client, cluster.Namespace, sts.Name); err != nil {
						if !errors.IsNotFound(err) {
							return ctrl.Result{}, err
						}
					}
				}
				break
			}
		}
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(cluster, utils.StorageFinalizer)
	if err := r.Update(ctx, cluster); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Cluster deletion handled successfully")
	return ctrl.Result{}, nil
}
