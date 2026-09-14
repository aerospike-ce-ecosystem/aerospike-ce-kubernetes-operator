package controller

import (
	"context"
	"testing"

	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ackov1alpha1 "github.com/aerospike-ce-ecosystem/aerospike-ce-kubernetes-operator/api/v1alpha1"
	"github.com/aerospike-ce-ecosystem/aerospike-ce-kubernetes-operator/internal/utils"
)

// --- one cluster-wide PodDisruptionBudget by default ---
//
// Per-rack PDBs (#370) gave every rack its own budget of replication-factor - 1.
// Kubernetes evaluates PDBs with disjoint selectors independently, so the
// cluster-wide concurrent-eviction bound became numRacks x (RF - 1): a 6-node,
// 3-rack, RF=2 cluster allowed THREE simultaneous evictions where the pre-#370
// cluster-wide PDB allowed one.
//
// A rack is not a data-placement unit on this operator — `rack-id` in a
// namespace is rejected as Enterprise-only and internal/configgen never emits
// one — so both copies of a partition can sit on any two nodes regardless of
// rack. Racks are a scheduling topology, and the honest budget is cluster-wide.

// listClusterPDBs returns every PDB the operator owns for the test cluster.
func listClusterPDBs(t *testing.T, r *AerospikeClusterReconciler) []policyv1.PodDisruptionBudget {
	t.Helper()
	list := &policyv1.PodDisruptionBudgetList{}
	if err := r.List(context.Background(), list,
		client.InNamespace(pdbTestNS),
		client.MatchingLabels(utils.LabelsForCluster(pdbTestCluster)),
	); err != nil {
		t.Fatalf("List PDBs: %v", err)
	}
	return list.Items
}

// TestReconcilePDB_MultiRackDefaultsToOneClusterWidePDB is the regression test:
// with per-rack PDBs three budgets of 1 each existed, so a node-pool upgrade or
// an autoscaler consolidation could evict three of six pods at once on an RF=2
// cluster and take both copies of a partition offline.
func TestReconcilePDB_MultiRackDefaultsToOneClusterWidePDB(t *testing.T) {
	cluster := pdbTestCR(6, 1, 2, 3)
	r := pdbTestReconciler(t, cluster)

	if err := r.reconcilePDB(context.Background(), cluster); err != nil {
		t.Fatalf("reconcilePDB() error = %v", err)
	}

	pdbs := listClusterPDBs(t, r)
	if len(pdbs) != 1 {
		names := make([]string, 0, len(pdbs))
		for i := range pdbs {
			names = append(names, pdbs[i].Name)
		}
		t.Fatalf("got %d PDBs %v, want exactly one cluster-wide PDB; independent per-rack "+
			"budgets multiply the cluster's concurrent-eviction bound by the rack count", len(pdbs), names)
	}

	pdb := &pdbs[0]
	if pdb.Name != utils.PDBName(pdbTestCluster) {
		t.Errorf("PDB name = %q, want %q", pdb.Name, utils.PDBName(pdbTestCluster))
	}
	if got := pdb.Spec.Selector.MatchLabels[utils.RackLabel]; got != "" {
		t.Errorf("selector has rack label %q; the budget would only constrain one rack", got)
	}
	if pdb.Spec.MaxUnavailable == nil || !intOrStringEqual(*pdb.Spec.MaxUnavailable, intstr.FromInt32(1)) {
		t.Errorf("maxUnavailable = %v, want 1 (replication-factor 2 - 1)", pdb.Spec.MaxUnavailable)
	}
}

// TestReconcilePDB_AggregateBudgetNeverMultipliesWithRacks enumerates every
// CE-legal layout and asserts the SUM of maxUnavailable across every PDB the
// operator creates stays at max(1, minRF - 1). That aggregate is what the
// Eviction API actually allows, and per-rack budgets made it numRacks times too
// large.
func TestReconcilePDB_AggregateBudgetNeverMultipliesWithRacks(t *testing.T) {
	for rf := 1; rf <= 3; rf++ {
		for size := int32(1); size <= 8; size++ {
			for numRacks := 1; int32(numRacks) <= size; numRacks++ {
				cluster := pdbTestCR(size)
				racks := make([]ackov1alpha1.Rack, numRacks)
				for i := range racks {
					racks[i] = ackov1alpha1.Rack{ID: i + 1}
				}
				cluster.Spec.RackConfig = &ackov1alpha1.RackConfig{Racks: racks}
				cluster.Spec.AerospikeConfig = &ackov1alpha1.AerospikeConfigSpec{Value: map[string]any{
					"namespaces": []any{map[string]any{"name": "test", "replication-factor": rf}},
				}}

				r := pdbTestReconciler(t, cluster)
				if err := r.reconcilePDB(context.Background(), cluster); err != nil {
					t.Fatalf("rf=%d size=%d racks=%d: reconcilePDB() error = %v", rf, size, numRacks, err)
				}

				want := int32(1)
				if rf-1 > 1 {
					want = int32(rf - 1)
				}
				total := int32(0)
				for _, pdb := range listClusterPDBs(t, r) {
					if pdb.Spec.MaxUnavailable != nil {
						total += pdb.Spec.MaxUnavailable.IntVal
					}
				}
				if total != want {
					t.Errorf("rf=%d size=%d racks=%d: aggregate maxUnavailable = %d, want %d — "+
						"Kubernetes evaluates disjoint PDBs independently, so the sum is the "+
						"cluster's real concurrent-eviction bound", rf, size, numRacks, total, want)
				}
			}
		}
	}
}

// TestReconcilePDB_UpgradeReplacesPerRackPDBs covers the v1.11.x upgrade path:
// the per-rack PDBs the previous operator created must be deleted and the single
// cluster-wide PDB recreated, without the user touching the CR.
func TestReconcilePDB_UpgradeReplacesPerRackPDBs(t *testing.T) {
	cluster := pdbTestCR(6, 1, 2, 3)
	r := pdbTestReconciler(t, cluster)

	// What v1.11.x left behind.
	for _, rackID := range []int{1, 2, 3} {
		mu := intstr.FromInt32(1)
		legacy := &policyv1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{
				Name:      utils.RackPDBName(pdbTestCluster, rackID),
				Namespace: pdbTestNS,
				Labels:    utils.LabelsForCluster(pdbTestCluster),
			},
			Spec: policyv1.PodDisruptionBudgetSpec{
				MaxUnavailable: &mu,
				Selector:       &metav1.LabelSelector{MatchLabels: utils.LabelsForRack(pdbTestCluster, rackID)},
			},
		}
		if err := r.Create(context.Background(), legacy); err != nil {
			t.Fatalf("setup: creating legacy PDB: %v", err)
		}
	}

	if err := r.reconcilePDB(context.Background(), cluster); err != nil {
		t.Fatalf("reconcilePDB() error = %v", err)
	}

	for _, rackID := range []int{1, 2, 3} {
		if _, ok := getPDB(t, r, utils.RackPDBName(pdbTestCluster, rackID)); ok {
			t.Errorf("legacy per-rack PDB for rack %d survived the upgrade; it keeps handing out "+
				"an extra eviction slot", rackID)
		}
	}
	if _, ok := getPDB(t, r, utils.PDBName(pdbTestCluster)); !ok {
		t.Error("cluster-wide PDB was not created on upgrade")
	}

	// Idempotent: a second pass changes nothing.
	if err := r.reconcilePDB(context.Background(), cluster); err != nil {
		t.Fatalf("second reconcilePDB() error = %v", err)
	}
	if got := len(listClusterPDBs(t, r)); got != 1 {
		t.Errorf("after a second reconcile there are %d PDBs, want 1", got)
	}
}

// TestReconcilePDB_PerRackOptIn pins the escape hatch: a rack that sets
// maxUnavailable explicitly still gets the per-rack shape, and the operator says
// out loud that the cluster-wide bound is now the sum across racks.
func TestReconcilePDB_PerRackOptIn(t *testing.T) {
	cluster := pdbTestCR(6, 1, 2, 3)
	optIn := intstr.FromInt32(1)
	cluster.Spec.RackConfig.Racks[0].MaxUnavailable = &optIn
	r := pdbTestReconciler(t, cluster)

	if err := r.reconcilePDB(context.Background(), cluster); err != nil {
		t.Fatalf("reconcilePDB() error = %v", err)
	}

	for _, rackID := range []int{1, 2, 3} {
		name := utils.RackPDBName(pdbTestCluster, rackID)
		pdb, ok := getPDB(t, r, name)
		if !ok {
			t.Fatalf("PDB %s was not created even though a rack opted in", name)
		}
		if pdb.Spec.Selector.MatchLabels[utils.RackLabel] == "" {
			t.Errorf("PDB %s selector has no rack label", name)
		}
	}
	if _, ok := getPDB(t, r, utils.PDBName(pdbTestCluster)); ok {
		t.Error("cluster-wide PDB still present alongside opt-in per-rack PDBs; a pod matched by " +
			"two PDBs makes the Eviction API refuse")
	}

	recorder, ok := r.Recorder.(*record.FakeRecorder)
	if !ok {
		t.Fatalf("recorder is %T, want *record.FakeRecorder", r.Recorder)
	}
	events := drainRecorderEvents(recorder)
	if !containsEvent(events, EventPDBPerRackBudget) {
		t.Errorf("expected a %s warning event explaining that the cluster-wide bound is now the "+
			"sum across racks, got %v", EventPDBPerRackBudget, events)
	}
}

// TestReconcilePDB_SingleRackHonoursRackMaxUnavailable pins that the one rack of
// a single-rack cluster can still set maxUnavailable and have it govern the
// cluster-wide PDB, as it did before per-rack budgets became opt-in.
func TestReconcilePDB_SingleRackHonoursRackMaxUnavailable(t *testing.T) {
	cluster := pdbTestCR(4, 1)
	mu := intstr.FromInt32(2)
	cluster.Spec.RackConfig.Racks[0].MaxUnavailable = &mu
	r := pdbTestReconciler(t, cluster)

	if err := r.reconcilePDB(context.Background(), cluster); err != nil {
		t.Fatalf("reconcilePDB() error = %v", err)
	}

	pdb, ok := getPDB(t, r, utils.PDBName(pdbTestCluster))
	if !ok {
		t.Fatal("cluster-wide PDB missing")
	}
	if pdb.Spec.MaxUnavailable == nil || !intOrStringEqual(*pdb.Spec.MaxUnavailable, intstr.FromInt32(2)) {
		t.Errorf("maxUnavailable = %v, want 2", pdb.Spec.MaxUnavailable)
	}
}

// TestClusterPDBPolicy_UsesTheLeastReplicatedRack pins that a rack overriding
// replication-factor downwards lowers the cluster-wide budget: the budget has to
// hold for the least-replicated namespace anywhere in the cluster.
func TestClusterPDBPolicy_UsesTheLeastReplicatedRack(t *testing.T) {
	cluster := pdbTestCR(6, 1, 2)
	cluster.Spec.AerospikeConfig = &ackov1alpha1.AerospikeConfigSpec{Value: map[string]any{
		"namespaces": []any{map[string]any{"name": "test", "replication-factor": 3}},
	}}
	cluster.Spec.RackConfig.Racks[1].AerospikeConfig = &ackov1alpha1.AerospikeConfigSpec{Value: map[string]any{
		"namespaces": []any{map[string]any{"name": "test", "replication-factor": 2}},
	}}

	r := pdbTestReconciler(t, cluster)
	policy := r.clusterPDBPolicy(cluster, r.getRacks(cluster))
	if policy.MaxUnavailable == nil || !intOrStringEqual(*policy.MaxUnavailable, intstr.FromInt32(1)) {
		t.Errorf("maxUnavailable = %v, want 1 (the rack running replication-factor 2 is the "+
			"binding constraint)", policy.MaxUnavailable)
	}
}
