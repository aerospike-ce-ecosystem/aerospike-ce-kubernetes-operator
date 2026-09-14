package controller

import (
	"context"
	"testing"

	aero "github.com/aerospike/aerospike-client-go/v8"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	ackov1alpha1 "github.com/aerospike-ce-ecosystem/aerospike-ce-kubernetes-operator/api/v1alpha1"
	"github.com/aerospike-ce-ecosystem/aerospike-ce-kubernetes-operator/internal/podutil"
	"github.com/aerospike-ce-ecosystem/aerospike-ce-kubernetes-operator/internal/utils"
)

// rollingRestartScheme builds a scheme with both the acko CRD types and the
// core/apps built-ins needed for StatefulSets and Pods.
func rollingRestartScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(s); err != nil {
		t.Fatalf("AddToScheme(client-go) error = %v", err)
	}
	if err := ackov1alpha1.AddToScheme(s); err != nil {
		t.Fatalf("AddToScheme(acko) error = %v", err)
	}
	return s
}

// newRackPod builds pod "demo-0" labelled for rack 0 of the given cluster with
// the supplied config-hash and pod-spec-hash annotations.
func newRackPod(clusterName, configHash, podSpecHash string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-0",
			Namespace: "default",
			Labels:    utils.LabelsForRack(clusterName, 0),
			Annotations: map[string]string{
				utils.ConfigHashAnnotation:  configHash,
				utils.PodSpecHashAnnotation: podSpecHash,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: podutil.AerospikeContainerName, Image: "aerospike:ce-8.1.1.1"},
			},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}
}

// TestSelectPodsToRestart is the focused Gap A trigger-logic test. It locks in
// that a pod is selected for restart when EITHER its config-hash OR its
// pod-spec-hash differs from the StatefulSet template, and that configChanged
// reflects only config-hash mismatches (the Gap B gate signal).
func TestSelectPodsToRestart(t *testing.T) {
	const (
		desiredHash        = "cfg-NEW"
		desiredPodSpecHash = "podspec-NEW"
	)

	tests := []struct {
		name              string
		configHash        string
		podSpecHash       string
		wantSelected      bool
		wantConfigChanged bool
	}{
		{
			name:              "both hashes match → not selected",
			configHash:        desiredHash,
			podSpecHash:       desiredPodSpecHash,
			wantSelected:      false,
			wantConfigChanged: false,
		},
		{
			name:              "config-hash differs → selected, configChanged",
			configHash:        "cfg-OLD",
			podSpecHash:       desiredPodSpecHash,
			wantSelected:      true,
			wantConfigChanged: true,
		},
		{
			name:              "pod-spec-hash differs (config matches) → selected, NOT configChanged",
			configHash:        desiredHash,
			podSpecHash:       "podspec-OLD",
			wantSelected:      true,
			wantConfigChanged: false,
		},
		{
			name:              "both hashes differ → selected, configChanged",
			configHash:        "cfg-OLD",
			podSpecHash:       "podspec-OLD",
			wantSelected:      true,
			wantConfigChanged: true,
		},
	}

	scheme := rollingRestartScheme(t)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cluster := &ackov1alpha1.AerospikeCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
			}
			pod := *newRackPod(cluster.Name, tc.configHash, tc.podSpecHash)

			reconciler := &AerospikeClusterReconciler{
				Client:   fake.NewClientBuilder().WithScheme(scheme).Build(),
				Scheme:   scheme,
				Recorder: record.NewFakeRecorder(8),
			}

			selected, configChanged := reconciler.selectPodsToRestart(
				context.Background(), cluster, []corev1.Pod{pod},
				desiredHash, desiredPodSpecHash, "", 0)

			if got := len(selected) == 1; got != tc.wantSelected {
				t.Errorf("selected = %v, want %v (selected=%d)", got, tc.wantSelected, len(selected))
			}
			if configChanged != tc.wantConfigChanged {
				t.Errorf("configChanged = %v, want %v", configChanged, tc.wantConfigChanged)
			}
		})
	}
}

// TestSelectPodsToRestart_EmptyDesiredPodSpecHash verifies that when the
// StatefulSet template has no pod-spec-hash annotation (desiredPodSpecHash ==
// ""), a pod is never selected solely on a pod-spec-hash difference — the
// pod-spec check is gated on a non-empty desired hash.
func TestSelectPodsToRestart_EmptyDesiredPodSpecHash(t *testing.T) {
	scheme := rollingRestartScheme(t)
	cluster := &ackov1alpha1.AerospikeCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
	}
	pod := *newRackPod(cluster.Name, "cfg-SAME", "podspec-anything")

	reconciler := &AerospikeClusterReconciler{
		Client:   fake.NewClientBuilder().WithScheme(scheme).Build(),
		Scheme:   scheme,
		Recorder: record.NewFakeRecorder(8),
	}

	selected, configChanged := reconciler.selectPodsToRestart(
		context.Background(), cluster, []corev1.Pod{pod}, "cfg-SAME", "", "", 0)

	if len(selected) != 0 {
		t.Errorf("expected no pod selected when desiredPodSpecHash is empty, got %d", len(selected))
	}
	if configChanged {
		t.Error("configChanged should be false when config hash matches")
	}
}

// TestReconcileRollingRestart_TriggersOnPodSpecHashChange is the Gap A test:
// a pod whose PodSpecHashAnnotation differs from the StatefulSet template MUST
// be selected for restart even when its ConfigHashAnnotation already matches.
// Against the pre-fix code (which only compared ConfigHashAnnotation) the pod
// is never added to podsToRestart, reconcileRollingRestart returns (false, nil)
// and the pod is never deleted.
func TestReconcileRollingRestart_TriggersOnPodSpecHashChange(t *testing.T) {
	scheme := rollingRestartScheme(t)

	// Config is unchanged: status config == spec config. Only the pod-spec
	// hash drifts, which also keeps the dynamic-config path out of the picture
	// (configChanged == false), so this is a pure cold restart.
	// ReadinessGateEnabled routes isBatchBlocked through the in-memory gate
	// check instead of a (slow, network-bound) migration probe; the test pod
	// predates the gate so the batch is not blocked.
	readinessGate := true
	cfg := &ackov1alpha1.AerospikeConfigSpec{Value: map[string]any{"service": map[string]any{}}}
	cluster := &ackov1alpha1.AerospikeCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: ackov1alpha1.AerospikeClusterSpec{
			Image:           "aerospike:ce-8.1.1.1",
			AerospikeConfig: cfg,
			PodSpec:         &ackov1alpha1.AerospikePodSpec{ReadinessGateEnabled: &readinessGate},
		},
		Status: ackov1alpha1.AerospikeClusterStatus{AerospikeConfig: cfg},
	}

	replicas := int32(1)
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.StatefulSetName(cluster.Name, 0),
			Namespace: cluster.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						utils.ConfigHashAnnotation:  "cfg-hash",
						utils.PodSpecHashAnnotation: "podspec-NEW",
					},
				},
			},
		},
	}

	// Pod's config hash matches the template, but its pod-spec hash is stale.
	pod := newRackPod(cluster.Name, "cfg-hash", "podspec-OLD")

	reconciler := &AerospikeClusterReconciler{
		Client: fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&ackov1alpha1.AerospikeCluster{}).
			WithObjects(cluster, sts, pod).
			Build(),
		Scheme:   scheme,
		Recorder: record.NewFakeRecorder(16),
	}

	rack := &ackov1alpha1.Rack{ID: 0}
	triggered, err := reconciler.reconcileRollingRestart(context.Background(), cluster, rack)
	if err != nil {
		t.Fatalf("reconcileRollingRestart() error = %v", err)
	}
	if !triggered {
		t.Fatal("expected reconcileRollingRestart to trigger a restart for a stale pod-spec-hash pod, got false")
	}

	// Cold restart deletes the pod. Confirm it is gone.
	got := &corev1.Pod{}
	err = reconciler.Get(context.Background(), types.NamespacedName{Name: "demo-0", Namespace: "default"}, got)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected pod demo-0 to be deleted (cold restart), Get err = %v", err)
	}
}

// TestReconcileRollingRestart_PodSpecChangeNotShortCircuited is the Gap B test.
// With spec.enableDynamicConfigUpdate=true and a pure pod-spec change (config
// genuinely unchanged), the restart MUST NOT short-circuit through the dynamic
// 2PC path. tryDynamicConfigUpdateBatch would see no config diff and return
// allOk=true, causing restartPodBatch to claim every pod restarted while
// nothing happened. The fix passes nil configs when no config-hash changed, so
// the pod is genuinely cold-restarted (deleted) here.
func TestReconcileRollingRestart_PodSpecChangeNotShortCircuited(t *testing.T) {
	scheme := rollingRestartScheme(t)

	enable := true
	readinessGate := true
	cfg := &ackov1alpha1.AerospikeConfigSpec{Value: map[string]any{"service": map[string]any{}}}
	cluster := &ackov1alpha1.AerospikeCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: ackov1alpha1.AerospikeClusterSpec{
			Image:                     "aerospike:ce-8.1.1.1",
			AerospikeConfig:           cfg,
			EnableDynamicConfigUpdate: &enable,
			// ReadinessGateEnabled keeps isBatchBlocked off the network path;
			// see the Gap A test for the rationale.
			PodSpec: &ackov1alpha1.AerospikePodSpec{ReadinessGateEnabled: &readinessGate},
		},
		Status: ackov1alpha1.AerospikeClusterStatus{AerospikeConfig: cfg},
	}

	replicas := int32(1)
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.StatefulSetName(cluster.Name, 0),
			Namespace: cluster.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						utils.ConfigHashAnnotation:  "cfg-hash",
						utils.PodSpecHashAnnotation: "podspec-NEW",
					},
				},
			},
		},
	}
	pod := newRackPod(cluster.Name, "cfg-hash", "podspec-OLD")

	reconciler := &AerospikeClusterReconciler{
		Client: fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&ackov1alpha1.AerospikeCluster{}).
			WithObjects(cluster, sts, pod).
			Build(),
		Scheme:   scheme,
		Recorder: record.NewFakeRecorder(16),
	}

	rack := &ackov1alpha1.Rack{ID: 0}
	triggered, err := reconciler.reconcileRollingRestart(context.Background(), cluster, rack)
	if err != nil {
		t.Fatalf("reconcileRollingRestart() error = %v", err)
	}
	if !triggered {
		t.Fatal("expected a restart to be triggered for the stale pod-spec-hash pod")
	}

	// The pod must actually be deleted. If the dynamic short-circuit had run,
	// restartPodBatch would report success without deleting anything.
	got := &corev1.Pod{}
	err = reconciler.Get(context.Background(), types.NamespacedName{Name: "demo-0", Namespace: "default"}, got)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("Gap B: pod demo-0 was NOT cold-restarted — dynamic-config path falsely reported success; Get err = %v", err)
	}
}

// TestRestartPodBatch_DynamicShortCircuitFalseSuccess documents the underlying
// trap behind Gap B: when restartPodBatch is given non-nil identical configs
// and a live Aerospike client, tryDynamicConfigUpdateBatch sees no config diff
// and returns allOk=true, so restartPodBatch reports every pod restarted
// WITHOUT deleting any pod. This is exactly the false success the Gap B fix
// avoids by passing nil configs for pure pod-spec changes. If a future change
// makes restartPodBatch stop trusting an empty diff, update this test.
func TestRestartPodBatch_DynamicShortCircuitFalseSuccess(t *testing.T) {
	scheme := rollingRestartScheme(t)

	enable := true
	cfg := map[string]any{"service": map[string]any{}}
	cluster := &ackov1alpha1.AerospikeCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: ackov1alpha1.AerospikeClusterSpec{
			Image:                     "aerospike:ce-8.1.1.1",
			EnableDynamicConfigUpdate: &enable,
		},
	}

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.StatefulSetName(cluster.Name, 0),
			Namespace: cluster.Namespace,
		},
	}
	pod := newRackPod(cluster.Name, "cfg-hash", "podspec-OLD")

	reconciler := &AerospikeClusterReconciler{
		Client: fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&ackov1alpha1.AerospikeCluster{}).
			WithObjects(cluster, sts, pod).
			Build(),
		Scheme:   scheme,
		Recorder: record.NewFakeRecorder(16),
	}

	// A zero-value client is enough: tryDynamicConfigUpdateBatch returns at the
	// empty-diff check before ever touching the network.
	preset := &aero.Client{}
	aeroClient := preset

	// Identical configs (oldConfig == newConfig content) => empty diff =>
	// dynamic path returns allOk=true => restartPodBatch reports success.
	restarted, failed, batch := reconciler.restartPodBatch(
		context.Background(), cluster, []*corev1.Pod{pod}, sts, "cfg-hash",
		1, cfg, cfg, &aeroClient)

	if restarted != 1 || len(failed) != 0 || len(batch) != 1 {
		t.Fatalf("expected dynamic short-circuit to report (1, [], 1 batch); got (%d, %v, %d)",
			restarted, failed, len(batch))
	}

	// Prove it was a FALSE success: the pod is still present, never deleted.
	got := &corev1.Pod{}
	if err := reconciler.Get(context.Background(),
		types.NamespacedName{Name: "demo-0", Namespace: "default"}, got); err != nil {
		t.Fatalf("pod demo-0 should still exist after the (false) dynamic success, Get err = %v", err)
	}
}

// --- the next batch must wait for the previous batch's replacement ---
//
// With spec.podSpec.readinessGateEnabled unset (the documented default) the
// only gate between rolling-restart batches was the cluster-wide migration
// probe. A replacement pod that is still Pending or cold-starting is not in
// client.GetNodes() at all, so the surviving nodes answer
// migrate_partitions_remaining=0 within seconds and the next stale pod is
// deleted while the first replacement is still down — two nodes down at once,
// which is a total outage for a size-2 cluster and partition unavailability at
// replication-factor 2. docs/content/operations/manage-cluster.md has always
// promised the operator "waits for replacements to become ready before
// continuing"; only the readiness-gate branch actually did.

const (
	restartGateConfigHash  = "cfg-NEW"
	restartGatePodSpecHash = "podspec-NEW"
)

// restartGatePod builds a rack-0 pod for the restart-gate tests. Every pod
// carries the desired config hash and drifts (or not) only on the pod-spec
// hash, so configChanged stays false and the dynamic-config 2PC path — which
// would build a real Aerospike client and sit on a connect timeout — is never
// entered. Staleness is therefore decided entirely by podSpecHash.
func restartGatePod(
	clusterName, name, podSpecHash string,
	phase corev1.PodPhase,
	ready bool,
) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			Labels:    utils.LabelsForRack(clusterName, 0),
			Annotations: map[string]string{
				utils.ConfigHashAnnotation:  restartGateConfigHash,
				utils.PodSpecHashAnnotation: podSpecHash,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: podutil.AerospikeContainerName, Image: "aerospike:ce-8.1.1.1"},
			},
		},
		Status: corev1.PodStatus{Phase: phase},
	}
	readyStatus := corev1.ConditionFalse
	if ready {
		readyStatus = corev1.ConditionTrue
	}
	pod.Status.Conditions = []corev1.PodCondition{{Type: corev1.PodReady, Status: readyStatus}}
	return pod
}

// restartGateSTS builds the rack-0 StatefulSet whose template carries the
// desired hashes.
func restartGateSTS(clusterName string, replicas int32) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.StatefulSetName(clusterName, 0),
			Namespace: "default",
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						utils.ConfigHashAnnotation:  restartGateConfigHash,
						utils.PodSpecHashAnnotation: restartGatePodSpecHash,
					},
				},
			},
		},
	}
}

// restartGateCluster builds a cluster with readiness gates left at their
// default (disabled) — the branch that had no readiness wait at all.
func restartGateCluster() *ackov1alpha1.AerospikeCluster {
	cfg := &ackov1alpha1.AerospikeConfigSpec{Value: map[string]any{"service": map[string]any{}}}
	return &ackov1alpha1.AerospikeCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default", UID: "restart-gate-uid"},
		Spec: ackov1alpha1.AerospikeClusterSpec{
			Size:            2,
			Image:           "aerospike:ce-8.1.1.1",
			AerospikeConfig: cfg,
		},
		Status: ackov1alpha1.AerospikeClusterStatus{AerospikeConfig: cfg},
	}
}

// restartGateReconciler wires a fake client plus a migration seam that reports
// "not migrating" without touching the network — exactly the answer the lone
// survivor of a two-node cluster gives while its peer is still starting, and
// the answer that used to wave the next batch through.
func restartGateReconciler(t *testing.T, objs ...client.Object) *AerospikeClusterReconciler {
	t.Helper()
	scheme := rollingRestartScheme(t)
	return &AerospikeClusterReconciler{
		Client: fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&ackov1alpha1.AerospikeCluster{}).
			WithObjects(objs...).
			Build(),
		Scheme:   scheme,
		Recorder: record.NewFakeRecorder(32),
		migrationCheck: func(context.Context, *ackov1alpha1.AerospikeCluster) (bool, error) {
			return false, nil
		},
	}
}

// TestReconcileRollingRestart_BlocksWhileReplacementNotReady is the regression
// test. demo-1 already carries the desired hashes (it is the replacement from
// the previous batch) but is not Ready; demo-0 is still stale. The stale pod
// must NOT be deleted until the replacement is Ready.
func TestReconcileRollingRestart_BlocksWhileReplacementNotReady(t *testing.T) {
	tests := []struct {
		name  string
		phase corev1.PodPhase
	}{
		{name: "replacement is Running but not Ready", phase: corev1.PodRunning},
		// Pending is the image-pull window of an upgrade. The readiness-gate
		// branch skips non-Running pods, so this case has to block here too.
		{name: "replacement is still Pending", phase: corev1.PodPending},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cluster := restartGateCluster()
			replacement := restartGatePod(cluster.Name, "demo-1", restartGatePodSpecHash, tc.phase, false)
			stale := restartGatePod(cluster.Name, "demo-0", "podspec-OLD", corev1.PodRunning, true)

			r := restartGateReconciler(t, cluster, restartGateSTS(cluster.Name, 2), replacement, stale)

			blocked, err := r.reconcileRollingRestart(context.Background(), cluster, &ackov1alpha1.Rack{ID: 0})
			if err != nil {
				t.Fatalf("reconcileRollingRestart() error = %v", err)
			}
			if !blocked {
				t.Error("reconcileRollingRestart() = false, want true (batch held pending a requeue)")
			}

			got := &corev1.Pod{}
			if err := r.Get(context.Background(),
				types.NamespacedName{Name: "demo-0", Namespace: "default"}, got); err != nil {
				t.Fatalf("stale pod demo-0 was deleted while replacement demo-1 was %s and not Ready — "+
					"two nodes down at once; Get err = %v", tc.phase, err)
			}
		})
	}
}

// TestReconcileRollingRestart_BlocksWhileReplacementMissing covers the window
// between the delete completing and the StatefulSet recreating the pod. The
// pod object is gone entirely, so there is nothing to find not-Ready — the
// reconcile that the pod-delete event itself triggers lands right here.
func TestReconcileRollingRestart_BlocksWhileReplacementMissing(t *testing.T) {
	cluster := restartGateCluster()
	stale := restartGatePod(cluster.Name, "demo-0", "podspec-OLD", corev1.PodRunning, true)

	// sts wants 2 replicas; only demo-0 exists (demo-1 was just deleted).
	r := restartGateReconciler(t, cluster, restartGateSTS(cluster.Name, 2), stale)

	blocked, err := r.reconcileRollingRestart(context.Background(), cluster, &ackov1alpha1.Rack{ID: 0})
	if err != nil {
		t.Fatalf("reconcileRollingRestart() error = %v", err)
	}
	if !blocked {
		t.Error("reconcileRollingRestart() = false, want true (batch held pending a requeue)")
	}

	got := &corev1.Pod{}
	if err := r.Get(context.Background(),
		types.NamespacedName{Name: "demo-0", Namespace: "default"}, got); err != nil {
		t.Fatalf("stale pod demo-0 was deleted while the previous replacement had not been "+
			"recreated yet; Get err = %v", err)
	}
}

// TestReconcileRollingRestart_StaleCrashLoopingPodStillRestarts is the
// anti-deadlock half. A cluster whose pods are crash-looping on the OLD config
// is exactly the case the migration escape hatch exists for: those pods are not
// Ready, but they are stale, so the restart that would fix them must still run.
// A "block on any not-Ready pod" gate would wedge this cluster forever.
func TestReconcileRollingRestart_StaleCrashLoopingPodStillRestarts(t *testing.T) {
	cluster := restartGateCluster()
	cluster.Spec.Size = 1
	crashing := restartGatePod(cluster.Name, "demo-0", "podspec-OLD", corev1.PodRunning, false)

	r := restartGateReconciler(t, cluster, restartGateSTS(cluster.Name, 1), crashing)

	triggered, err := r.reconcileRollingRestart(context.Background(), cluster, &ackov1alpha1.Rack{ID: 0})
	if err != nil {
		t.Fatalf("reconcileRollingRestart() error = %v", err)
	}
	if !triggered {
		t.Fatal("reconcileRollingRestart() = false, want true (the stale crash-looping pod must be restarted)")
	}

	got := &corev1.Pod{}
	err = r.Get(context.Background(), types.NamespacedName{Name: "demo-0", Namespace: "default"}, got)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("stale crash-looping pod demo-0 was not restarted — the fixing restart is deadlocked; Get err = %v", err)
	}
}

// TestIsBatchBlocked_ReadinessGateEnabled_PendingReplacementBlocks covers the
// narrower hole in the gate branch: anyPodGateUnsatisfied only inspects Running
// pods, so a replacement still pulling its new image (Pending, no gate
// condition yet) sailed straight through it.
func TestIsBatchBlocked_ReadinessGateEnabled_PendingReplacementBlocks(t *testing.T) {
	gateEnabled := true
	cluster := restartGateCluster()
	cluster.Spec.PodSpec = &ackov1alpha1.AerospikePodSpec{ReadinessGateEnabled: &gateEnabled}

	replacement := restartGatePod(cluster.Name, "demo-1", restartGatePodSpecHash, corev1.PodPending, false)
	ready := restartGatePod(cluster.Name, "demo-0", "podspec-OLD", corev1.PodRunning, true)

	r := restartGateReconciler(t, cluster)
	target := rackRestartTarget{
		configHash:  restartGateConfigHash,
		podSpecHash: restartGatePodSpecHash,
		replicas:    2,
	}

	if !r.isBatchBlocked(context.Background(), cluster, 0, []corev1.Pod{*replacement, *ready}, target) {
		t.Error("isBatchBlocked() = false while the replacement pod was still Pending, want true")
	}
}

// TestIsBatchBlocked_TerminatingPodBlocks pins the terminating rule. A pod in
// its preStop window is a restart that has not finished; taking the next node
// down on top of it is the same double outage.
func TestIsBatchBlocked_TerminatingPodBlocks(t *testing.T) {
	cluster := restartGateCluster()
	r := restartGateReconciler(t, cluster)

	terminating := restartGatePod(cluster.Name, "demo-1", restartGatePodSpecHash, corev1.PodRunning, true)
	now := metav1.Now()
	terminating.DeletionTimestamp = &now
	stale := restartGatePod(cluster.Name, "demo-0", "podspec-OLD", corev1.PodRunning, true)

	target := rackRestartTarget{
		configHash:  restartGateConfigHash,
		podSpecHash: restartGatePodSpecHash,
		replicas:    2,
	}
	if !r.isBatchBlocked(context.Background(), cluster, 0, []corev1.Pod{*terminating, *stale}, target) {
		t.Error("isBatchBlocked() = false while a pod was terminating, want true")
	}

	// The same rule has to survive the zero target, which is what the on-demand
	// operations path passes.
	if !r.isBatchBlocked(context.Background(), cluster, onDemandOperationRackID,
		[]corev1.Pod{*terminating}, rackRestartTarget{}) {
		t.Error("isBatchBlocked() = false for a terminating pod under the zero target, want true")
	}
}
