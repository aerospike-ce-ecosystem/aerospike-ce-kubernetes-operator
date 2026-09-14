package controller

import (
	"context"
	"fmt"
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
// 2PC path. The fix passes nil configs when no config-hash changed, so the pod
// is genuinely cold-restarted (deleted) here.
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

// newRackPodFor builds pod "<cluster>-<rack>-0" labelled for the given rack with
// the supplied config-hash and pod-spec-hash annotations.
func newRackPodFor(clusterName string, rackID int, configHash, podSpecHash string) *corev1.Pod {
	pod := newRackPod(clusterName, configHash, podSpecHash)
	pod.Name = fmt.Sprintf("%s-%d-0", clusterName, rackID)
	pod.Labels = utils.LabelsForRack(clusterName, rackID)
	return pod
}

// TestTryDynamicConfigUpdateBatch_EmptyDiffFallsThrough locks in that an empty
// config diff is NOT reported as a successful batch update. The pods handed to
// tryDynamicConfigUpdateBatch were selected precisely because their config hash
// does not match the StatefulSet template, so "nothing to apply" must route them
// to the per-pod restart path instead of claiming they are already done.
func TestTryDynamicConfigUpdateBatch_EmptyDiffFallsThrough(t *testing.T) {
	scheme := rollingRestartScheme(t)

	enable := true
	cfg := map[string]any{"service": map[string]any{"proto-fd-max": 15000}}
	cluster := &ackov1alpha1.AerospikeCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: ackov1alpha1.AerospikeClusterSpec{
			EnableDynamicConfigUpdate: &enable,
		},
	}
	pod := newRackPod(cluster.Name, "cfg-OLD", "podspec-SAME")

	reconciler := &AerospikeClusterReconciler{
		Client:   fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster, pod).Build(),
		Scheme:   scheme,
		Recorder: record.NewFakeRecorder(16),
	}

	// Identical old/new configs => empty diff. A zero-value client is enough:
	// the function must return before touching the network.
	allOk, updates, rbResult := reconciler.tryDynamicConfigUpdateBatch(
		context.Background(), cluster, []*corev1.Pod{pod}, cfg, cfg, &aero.Client{}, "cfg-NEW")

	if allOk {
		t.Error("tryDynamicConfigUpdateBatch reported success for an empty diff; " +
			"the batch must fall through to the per-pod restart path")
	}
	if updates != nil {
		t.Errorf("expected nil updates for an empty diff, got %v", updates)
	}
	if rbResult != nil {
		t.Errorf("expected nil rollback result for an empty diff, got %+v", rbResult)
	}
}

// TestRestartPodBatch_StaleConfigHashIsRestarted is the inverted successor of
// TestRestartPodBatch_DynamicShortCircuitFalseSuccess, which asserted the old
// (buggy) behaviour and whose comment asked for exactly this update.
//
// Scenario: a pod was skipped during an earlier rollout (pending/failed within
// maxIgnorablePods, or non-ready past maxPodUnstableDuration) and recovers only
// after populateStatus already stamped Status.AerospikeConfig = Spec.Aerospike
// Config. Its config hash is stale, so it is selected for restart, but the
// cluster-level diff handed to the 2PC path is empty. The empty diff must NOT
// be reported as "every pod updated" — the pod has to be genuinely restarted,
// otherwise the cluster loops in RollingRestart forever without ever applying
// the config.
func TestRestartPodBatch_StaleConfigHashIsRestarted(t *testing.T) {
	scheme := rollingRestartScheme(t)

	enable := true
	cfg := map[string]any{"service": map[string]any{"proto-fd-max": 20000}}
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
	pod := newRackPod(cluster.Name, "cfg-OLD", "podspec-SAME")

	reconciler := &AerospikeClusterReconciler{
		Client: fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&ackov1alpha1.AerospikeCluster{}).
			WithObjects(cluster, sts, pod).
			Build(),
		Scheme:   scheme,
		Recorder: record.NewFakeRecorder(16),
	}

	// Preset a non-nil client so restartPodBatch takes the dynamic path without
	// dialling anything. A zero-value client is enough: tryDynamicConfigUpdate
	// Batch must return at the empty-diff check before touching the network.
	aeroClient := &aero.Client{}

	// oldConfig == newConfig (Status already mirrors Spec) => empty diff.
	restarted, failed, batch := reconciler.restartPodBatch(
		context.Background(), cluster, []*corev1.Pod{pod}, sts, "cfg-NEW",
		1, cfg, cfg, &aeroClient)

	if restarted != 1 || len(failed) != 0 || len(batch) != 1 {
		t.Fatalf("expected the empty-diff fallthrough to cold-restart the pod (1, [], 1 batch); got (%d, %v, %d)",
			restarted, failed, len(batch))
	}

	got := &corev1.Pod{}
	err := reconciler.Get(context.Background(),
		types.NamespacedName{Name: "demo-0", Namespace: "default"}, got)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("stale-hash pod demo-0 must be deleted (cold restart) instead of being falsely "+
			"reported as dynamically updated; Get err = %v", err)
	}
}

// TestRollingRestart_RackOverrideConverges covers a rack-scoped config edit with
// enableDynamicConfigUpdate=true. The StatefulSet template carries the per-rack
// EFFECTIVE hash (cluster config DeepMerged with the rack override), but the
// dynamic-config path only ever diffs the cluster-level config, which is
// unchanged here. Before the fix the empty diff was reported as "all pods
// restarted", no pod was touched and the cluster requeued in RollingRestart
// forever. After the fix the pod is cold-restarted and, once the StatefulSet
// recreates it with the template hash, the next reconcile finds nothing to do.
func TestRollingRestart_RackOverrideConverges(t *testing.T) {
	scheme := rollingRestartScheme(t)

	const rackID = 1
	enable := true
	readinessGate := true
	clusterCfg := &ackov1alpha1.AerospikeConfigSpec{Value: map[string]any{
		"service": map[string]any{"proto-fd-max": 15000},
	}}
	// Only the rack override changed; the cluster-level config is identical in
	// Spec and Status, so the cluster-level diff is empty.
	rack := ackov1alpha1.Rack{
		ID: rackID,
		AerospikeConfig: &ackov1alpha1.AerospikeConfigSpec{Value: map[string]any{
			"service": map[string]any{"proto-fd-max": 30000},
		}},
	}
	cluster := &ackov1alpha1.AerospikeCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: ackov1alpha1.AerospikeClusterSpec{
			Image:                     "aerospike:ce-8.1.1.1",
			AerospikeConfig:           clusterCfg,
			EnableDynamicConfigUpdate: &enable,
			RackConfig:                &ackov1alpha1.RackConfig{Racks: []ackov1alpha1.Rack{rack}},
			// ReadinessGateEnabled keeps isBatchBlocked off the network path;
			// see the Gap A test for the rationale.
			PodSpec: &ackov1alpha1.AerospikePodSpec{ReadinessGateEnabled: &readinessGate},
		},
		Status: ackov1alpha1.AerospikeClusterStatus{AerospikeConfig: clusterCfg},
	}

	reconciler := &AerospikeClusterReconciler{
		Scheme:   scheme,
		Recorder: record.NewFakeRecorder(32),
	}
	// The StatefulSet template hash is the rack's EFFECTIVE hash, which no
	// cluster-level diff can ever reproduce.
	effectiveHash := configHash(reconciler.getEffectiveConfig(cluster, &rack))

	replicas := int32(1)
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.StatefulSetName(cluster.Name, rackID),
			Namespace: cluster.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						utils.ConfigHashAnnotation:  effectiveHash,
						utils.PodSpecHashAnnotation: "podspec-SAME",
					},
				},
			},
		},
	}
	pod := newRackPodFor(cluster.Name, rackID, "cfg-OLD-EFFECTIVE", "podspec-SAME")

	reconciler.Client = fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&ackov1alpha1.AerospikeCluster{}).
		WithObjects(cluster, sts, pod).
		Build()

	// Drive the batch directly with a preset client so the dynamic path is taken
	// without dialling anything (reconcileRollingRestart would otherwise spend a
	// full connect timeout failing to reach the headless service).
	aeroClient := &aero.Client{}
	restarted, failed, batch := reconciler.restartPodBatch(
		context.Background(), cluster, []*corev1.Pod{pod}, sts, effectiveHash,
		1, clusterCfg.Value, clusterCfg.Value, &aeroClient)

	if restarted != 1 || len(failed) != 0 || len(batch) != 1 {
		t.Fatalf("expected the rack-override pod to be restarted (1, [], 1 batch); got (%d, %v, %d)",
			restarted, failed, len(batch))
	}

	got := &corev1.Pod{}
	err := reconciler.Get(context.Background(),
		types.NamespacedName{Name: pod.Name, Namespace: "default"}, got)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("rack-override pod %s was NOT restarted — the empty cluster-level diff was reported "+
			"as a successful dynamic update, which loops in RollingRestart forever; Get err = %v",
			pod.Name, err)
	}

	// The StatefulSet recreates the pod from the template, so it now carries the
	// effective hash. The next reconcile must converge: nothing left to restart.
	recreated := newRackPodFor(cluster.Name, rackID, effectiveHash, "podspec-SAME")
	if err := reconciler.Create(context.Background(), recreated); err != nil {
		t.Fatalf("recreating pod: %v", err)
	}

	triggered, err := reconciler.reconcileRollingRestart(context.Background(), cluster, &rack)
	if err != nil {
		t.Fatalf("reconcileRollingRestart() error = %v", err)
	}
	if triggered {
		t.Fatal("expected the second pass to converge (no restart), got triggered=true")
	}
}
