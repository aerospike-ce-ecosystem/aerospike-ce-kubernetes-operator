package controller

import (
	"context"

	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	asdbcev1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/utils"
)

func (r *AerospikeCEClusterReconciler) reconcilePDB(
	ctx context.Context,
	cluster *asdbcev1alpha1.AerospikeCECluster,
) error {
	log := logf.FromContext(ctx)

	// Check if PDB is disabled
	if cluster.Spec.DisablePDB != nil && *cluster.Spec.DisablePDB {
		// Delete PDB if it exists
		pdbName := utils.PDBName(cluster.Name)
		existing := &policyv1.PodDisruptionBudget{}
		if err := r.Get(ctx, types.NamespacedName{Name: pdbName, Namespace: cluster.Namespace}, existing); err == nil {
			return r.Delete(ctx, existing)
		}
		return nil
	}

	pdbName := utils.PDBName(cluster.Name)
	labels := utils.LabelsForCluster(cluster.Name)
	selectorLabels := utils.SelectorLabelsForCluster(cluster.Name)

	maxUnavailable := intstr.FromInt32(1)
	if cluster.Spec.MaxUnavailable != nil {
		maxUnavailable = *cluster.Spec.MaxUnavailable
	}

	existing := &policyv1.PodDisruptionBudget{}
	err := r.Get(ctx, types.NamespacedName{Name: pdbName, Namespace: cluster.Namespace}, existing)

	if errors.IsNotFound(err) {
		pdb := &policyv1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pdbName,
				Namespace: cluster.Namespace,
				Labels:    labels,
			},
			Spec: policyv1.PodDisruptionBudgetSpec{
				MaxUnavailable: &maxUnavailable,
				Selector: &metav1.LabelSelector{
					MatchLabels: selectorLabels,
				},
			},
		}
		if err := r.setOwnerRef(cluster, pdb); err != nil {
			return err
		}
		log.Info("Creating PDB", "name", pdbName)
		return r.Create(ctx, pdb)
	} else if err != nil {
		return err
	}

	// Update
	existing.Spec.MaxUnavailable = &maxUnavailable
	return r.Update(ctx, existing)
}
