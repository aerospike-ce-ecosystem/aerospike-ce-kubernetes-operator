package controller

import (
	"testing"

	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/utils"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestPDBNeedsUpdate(t *testing.T) {
	desiredLabels := utils.LabelsForCluster("demo")
	desiredSelector := utils.SelectorLabelsForCluster("demo")
	desiredMaxUnavailable := intstr.FromInt32(1)

	base := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Labels: desiredLabels,
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MaxUnavailable: &desiredMaxUnavailable,
			Selector: &metav1.LabelSelector{
				MatchLabels: desiredSelector,
			},
		},
	}

	tests := []struct {
		name     string
		mutate   func(*policyv1.PodDisruptionBudget)
		expected bool
	}{
		{
			name:     "unchanged",
			mutate:   func(_ *policyv1.PodDisruptionBudget) {},
			expected: false,
		},
		{
			name: "max unavailable drift",
			mutate: func(pdb *policyv1.PodDisruptionBudget) {
				value := intstr.FromInt32(2)
				pdb.Spec.MaxUnavailable = &value
			},
			expected: true,
		},
		{
			name: "selector drift",
			mutate: func(pdb *policyv1.PodDisruptionBudget) {
				pdb.Spec.Selector = &metav1.LabelSelector{
					MatchLabels: map[string]string{utils.InstanceLabel: "other"},
				}
			},
			expected: true,
		},
		{
			name: "label drift",
			mutate: func(pdb *policyv1.PodDisruptionBudget) {
				pdb.Labels["custom"] = "drifted"
			},
			expected: true,
		},
		{
			name: "min available drift",
			mutate: func(pdb *policyv1.PodDisruptionBudget) {
				value := intstr.FromInt32(1)
				pdb.Spec.MinAvailable = &value
				pdb.Spec.MaxUnavailable = nil
			},
			expected: true,
		},
		{
			name: "missing selector",
			mutate: func(pdb *policyv1.PodDisruptionBudget) {
				pdb.Spec.Selector = nil
			},
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pdb := base.DeepCopy()
			tc.mutate(pdb)
			if got := pdbNeedsUpdate(pdb, desiredLabels, desiredSelector, desiredMaxUnavailable); got != tc.expected {
				t.Fatalf("pdbNeedsUpdate() = %v, want %v", got, tc.expected)
			}
		})
	}
}
