package controller

import (
	"context"
	"fmt"

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
			if err := r.Delete(ctx, existing); err != nil && !errors.IsNotFound(err) {
				return fmt.Errorf("deleting PDB %s: %w", pdbName, err)
			}
		} else if !errors.IsNotFound(err) {
			return fmt.Errorf("getting PDB %s for deletion: %w", pdbName, err)
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
		if err := r.Create(ctx, pdb); err != nil {
			return fmt.Errorf("creating PDB %s: %w", pdbName, err)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("getting PDB %s: %w", pdbName, err)
	}

	// Update only if MaxUnavailable changed
	if existing.Spec.MaxUnavailable != nil && intOrStringEqual(*existing.Spec.MaxUnavailable, maxUnavailable) {
		return nil
	}
	existing.Spec.MaxUnavailable = &maxUnavailable
	log.Info("Updating PDB", "name", pdbName)
	if err := r.Update(ctx, existing); err != nil {
		return fmt.Errorf("updating PDB %s: %w", pdbName, err)
	}
	return nil
}

// intOrStringEqual compares two IntOrString values by type and value,
// avoiding the ambiguity of String()-based comparison where int(1) and
// string("1") would appear equal.
func intOrStringEqual(a, b intstr.IntOrString) bool {
	return a.Type == b.Type && a.IntVal == b.IntVal && a.StrVal == b.StrVal
}
