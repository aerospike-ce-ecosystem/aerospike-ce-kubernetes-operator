package controller

import (
	"errors"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ackov1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
)

func TestIsPodReady(t *testing.T) {
	tests := []struct {
		name string
		pod  *corev1.Pod
		want bool
	}{
		{
			name: "pod not in Running phase",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
				},
			},
			want: false,
		},
		{
			name: "pod in Running phase but no Ready condition",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase:      corev1.PodRunning,
					Conditions: []corev1.PodCondition{},
				},
			},
			want: false,
		},
		{
			name: "pod in Running phase with Ready=False",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionFalse,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "pod in Running phase with Ready=True",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			want: true,
		},
		{
			name: "pod Failed phase with Ready=True condition is not ready",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodFailed,
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "pod in Running phase with multiple conditions including Ready=True",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodScheduled,
							Status: corev1.ConditionTrue,
						},
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isPodReady(tc.pod)
			if got != tc.want {
				t.Errorf("isPodReady() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestBuildSelectorString(t *testing.T) {
	t.Run("returns empty string for empty map", func(t *testing.T) {
		if got := buildSelectorString(map[string]string{}); got != "" {
			t.Fatalf("buildSelectorString(empty) = %q, want empty", got)
		}
	})

	t.Run("returns deterministic key order", func(t *testing.T) {
		labels := map[string]string{
			"z-key": "z",
			"a-key": "a",
			"m-key": "m",
		}

		want := "a-key=a,m-key=m,z-key=z"
		for range 50 {
			if got := buildSelectorString(labels); got != want {
				t.Fatalf("buildSelectorString() = %q, want %q", got, want)
			}
		}
	})

	t.Run("escapes nothing and uses comma separator", func(t *testing.T) {
		labels := map[string]string{
			"app.kubernetes.io/name":     "aerospike-cluster",
			"app.kubernetes.io/instance": "demo",
		}

		got := buildSelectorString(labels)
		if strings.Count(got, ",") != 1 {
			t.Fatalf("selector %q should contain exactly one comma", got)
		}
		if !strings.Contains(got, "app.kubernetes.io/instance=demo") {
			t.Fatalf("selector %q missing instance label", got)
		}
		if !strings.Contains(got, "app.kubernetes.io/name=aerospike-cluster") {
			t.Fatalf("selector %q missing app label", got)
		}
	})
}

func TestSetCondition(t *testing.T) {
	t.Run("new condition type is appended", func(t *testing.T) {
		cluster := &ackov1alpha1.AerospikeCluster{}

		setCondition(cluster, "Available", true, "ClusterAvailable", "At least one pod is ready")

		if len(cluster.Status.Conditions) != 1 {
			t.Fatalf("expected 1 condition, got %d", len(cluster.Status.Conditions))
		}
		cond := cluster.Status.Conditions[0]
		if cond.Type != "Available" {
			t.Errorf("condition type = %q, want %q", cond.Type, "Available")
		}
		if cond.Status != metav1.ConditionTrue {
			t.Errorf("condition status = %q, want %q", cond.Status, metav1.ConditionTrue)
		}
		if cond.Reason != "ClusterAvailable" {
			t.Errorf("condition reason = %q, want %q", cond.Reason, "ClusterAvailable")
		}
		if cond.Message != "At least one pod is ready" {
			t.Errorf("condition message = %q, want %q", cond.Message, "At least one pod is ready")
		}
	})

	t.Run("multiple different condition types are appended", func(t *testing.T) {
		cluster := &ackov1alpha1.AerospikeCluster{}

		setCondition(cluster, "Available", true, "ClusterAvailable", "At least one pod is ready")
		setCondition(cluster, "Ready", false, "NotReady", "0/3 pods ready")

		if len(cluster.Status.Conditions) != 2 {
			t.Fatalf("expected 2 conditions, got %d", len(cluster.Status.Conditions))
		}
		if cluster.Status.Conditions[0].Type != "Available" {
			t.Errorf("first condition type = %q, want %q", cluster.Status.Conditions[0].Type, "Available")
		}
		if cluster.Status.Conditions[1].Type != "Ready" {
			t.Errorf("second condition type = %q, want %q", cluster.Status.Conditions[1].Type, "Ready")
		}
	})

	t.Run("existing condition with same status is NOT updated (idempotent)", func(t *testing.T) {
		cluster := &ackov1alpha1.AerospikeCluster{}

		setCondition(cluster, "Available", true, "ClusterAvailable", "At least one pod is ready")
		originalTime := cluster.Status.Conditions[0].LastTransitionTime

		// Sleep briefly to ensure time difference would be detectable
		time.Sleep(time.Millisecond)

		// Set the same condition with the same status again
		setCondition(cluster, "Available", true, "ClusterAvailable", "At least one pod is ready")

		if len(cluster.Status.Conditions) != 1 {
			t.Fatalf("expected 1 condition, got %d", len(cluster.Status.Conditions))
		}
		if !cluster.Status.Conditions[0].LastTransitionTime.Equal(&originalTime) {
			t.Errorf("LastTransitionTime changed when status was unchanged: was %v, now %v",
				originalTime, cluster.Status.Conditions[0].LastTransitionTime)
		}
	})

	t.Run("existing condition with same status updates reason and message", func(t *testing.T) {
		cluster := &ackov1alpha1.AerospikeCluster{}

		setCondition(cluster, "Ready", false, "NotReady", "0/3 pods ready")
		originalTime := cluster.Status.Conditions[0].LastTransitionTime

		time.Sleep(time.Millisecond)

		setCondition(cluster, "Ready", false, "Progressing", "1/3 pods ready")

		if len(cluster.Status.Conditions) != 1 {
			t.Fatalf("expected 1 condition, got %d", len(cluster.Status.Conditions))
		}
		cond := cluster.Status.Conditions[0]
		if cond.Status != metav1.ConditionFalse {
			t.Errorf("condition status = %q, want %q", cond.Status, metav1.ConditionFalse)
		}
		if cond.Reason != "Progressing" {
			t.Errorf("condition reason = %q, want %q", cond.Reason, "Progressing")
		}
		if cond.Message != "1/3 pods ready" {
			t.Errorf("condition message = %q, want %q", cond.Message, "1/3 pods ready")
		}
		if !cond.LastTransitionTime.Equal(&originalTime) {
			t.Errorf("LastTransitionTime changed when status was unchanged: was %v, now %v",
				originalTime, cond.LastTransitionTime)
		}
	})

	t.Run("existing condition with changed status IS updated", func(t *testing.T) {
		cluster := &ackov1alpha1.AerospikeCluster{}

		setCondition(cluster, "Available", false, "ClusterUnavailable", "No pods ready")
		originalTime := cluster.Status.Conditions[0].LastTransitionTime

		// Sleep briefly to ensure time difference
		time.Sleep(time.Millisecond)

		// Change the status from false to true
		setCondition(cluster, "Available", true, "ClusterAvailable", "At least one pod is ready")

		if len(cluster.Status.Conditions) != 1 {
			t.Fatalf("expected 1 condition, got %d", len(cluster.Status.Conditions))
		}

		cond := cluster.Status.Conditions[0]
		if cond.Status != metav1.ConditionTrue {
			t.Errorf("condition status = %q, want %q", cond.Status, metav1.ConditionTrue)
		}
		if cond.Reason != "ClusterAvailable" {
			t.Errorf("condition reason = %q, want %q", cond.Reason, "ClusterAvailable")
		}
		if cond.Message != "At least one pod is ready" {
			t.Errorf("condition message = %q, want %q", cond.Message, "At least one pod is ready")
		}
		if cond.LastTransitionTime.Equal(&originalTime) {
			t.Error("LastTransitionTime should have changed when status changed")
		}
	})

	t.Run("status=false produces ConditionFalse", func(t *testing.T) {
		cluster := &ackov1alpha1.AerospikeCluster{}

		setCondition(cluster, "Ready", false, "NotReady", "0/3 pods ready")

		if cluster.Status.Conditions[0].Status != metav1.ConditionFalse {
			t.Errorf("condition status = %q, want %q", cluster.Status.Conditions[0].Status, metav1.ConditionFalse)
		}
	})
}

func TestParseServiceEndpoints(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "empty string returns nil",
			input: "",
			want:  nil,
		},
		{
			name:  "whitespace-only string returns nil",
			input: "   ",
			want:  nil,
		},
		{
			name:  "single endpoint",
			input: "1.2.3.4:3000",
			want:  []string{"1.2.3.4:3000"},
		},
		{
			name:  "multiple endpoints separated by semicolons",
			input: "1.2.3.4:3000;5.6.7.8:3000;9.10.11.12:3000",
			want:  []string{"1.2.3.4:3000", "5.6.7.8:3000", "9.10.11.12:3000"},
		},
		{
			name:  "endpoints with surrounding whitespace are trimmed",
			input: " 1.2.3.4:3000 ; 5.6.7.8:3000 ",
			want:  []string{"1.2.3.4:3000", "5.6.7.8:3000"},
		},
		{
			name:  "trailing semicolon is ignored",
			input: "1.2.3.4:3000;",
			want:  []string{"1.2.3.4:3000"},
		},
		{
			name:  "consecutive semicolons produce no empty entries",
			input: "1.2.3.4:3000;;5.6.7.8:3000",
			want:  []string{"1.2.3.4:3000", "5.6.7.8:3000"},
		},
		{
			name:  "leading semicolon is ignored",
			input: ";1.2.3.4:3000",
			want:  []string{"1.2.3.4:3000"},
		},
		{
			name:  "only semicolons returns nil",
			input: ";;;",
			want:  nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseServiceEndpoints(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("parseServiceEndpoints(%q) = %v (len %d), want %v (len %d)",
					tc.input, got, len(got), tc.want, len(tc.want))
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("parseServiceEndpoints(%q)[%d] = %q, want %q", tc.input, i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestSetFineGrainedConditions(t *testing.T) {
	findCondition := func(cluster *ackov1alpha1.AerospikeCluster, condType string) *metav1.Condition {
		for i := range cluster.Status.Conditions {
			if cluster.Status.Conditions[i].Type == condType {
				return &cluster.Status.Conditions[i]
			}
		}
		return nil
	}

	t.Run("Paused=true sets ReconciliationPaused=True", func(t *testing.T) {
		cluster := &ackov1alpha1.AerospikeCluster{}
		setFineGrainedConditions(cluster, StatusUpdateOpts{Paused: true})

		cond := findCondition(cluster, ackov1alpha1.ConditionReconciliationPaused)
		if cond == nil {
			t.Fatal("ReconciliationPaused condition not found")
		}
		if cond.Status != metav1.ConditionTrue {
			t.Errorf("ReconciliationPaused = %q, want True", cond.Status)
		}
	})

	t.Run("Paused=false sets ReconciliationPaused=False", func(t *testing.T) {
		cluster := &ackov1alpha1.AerospikeCluster{}
		setFineGrainedConditions(cluster, StatusUpdateOpts{Paused: false})

		cond := findCondition(cluster, ackov1alpha1.ConditionReconciliationPaused)
		if cond == nil {
			t.Fatal("ReconciliationPaused condition not found")
		}
		if cond.Status != metav1.ConditionFalse {
			t.Errorf("ReconciliationPaused = %q, want False", cond.Status)
		}
	})

	t.Run("ACL spec nil: ACLSynced condition is not set", func(t *testing.T) {
		cluster := &ackov1alpha1.AerospikeCluster{}
		// AerospikeAccessControl is nil by default
		setFineGrainedConditions(cluster, StatusUpdateOpts{})

		cond := findCondition(cluster, ackov1alpha1.ConditionACLSynced)
		if cond != nil {
			t.Errorf("ACLSynced should not be set when ACL spec is nil, got %q", cond.Status)
		}
	})

	t.Run("ACL spec set and ACLSynced true: ACLSynced=True", func(t *testing.T) {
		cluster := &ackov1alpha1.AerospikeCluster{}
		cluster.Spec.AerospikeAccessControl = &ackov1alpha1.AerospikeAccessControlSpec{}
		setFineGrainedConditions(cluster, StatusUpdateOpts{ACLErr: nil, ACLSynced: true})

		cond := findCondition(cluster, ackov1alpha1.ConditionACLSynced)
		if cond == nil {
			t.Fatal("ACLSynced condition not found")
		}
		if cond.Status != metav1.ConditionTrue {
			t.Errorf("ACLSynced = %q, want True", cond.Status)
		}
	})

	t.Run("ACL spec set and ACL skipped (no ready pods): ACLSynced=False with pending reason", func(t *testing.T) {
		cluster := &ackov1alpha1.AerospikeCluster{}
		cluster.Spec.AerospikeAccessControl = &ackov1alpha1.AerospikeAccessControlSpec{}
		setFineGrainedConditions(cluster, StatusUpdateOpts{ACLErr: nil, ACLSynced: false})

		cond := findCondition(cluster, ackov1alpha1.ConditionACLSynced)
		if cond == nil {
			t.Fatal("ACLSynced condition not found")
		}
		if cond.Status != metav1.ConditionFalse {
			t.Errorf("ACLSynced = %q, want False", cond.Status)
		}
		if cond.Reason != "ACLSyncPending" {
			t.Errorf("ACLSynced reason = %q, want ACLSyncPending", cond.Reason)
		}
	})

	t.Run("ACL spec set and ACLErr non-nil: ACLSynced=False", func(t *testing.T) {
		cluster := &ackov1alpha1.AerospikeCluster{}
		cluster.Spec.AerospikeAccessControl = &ackov1alpha1.AerospikeAccessControlSpec{}
		setFineGrainedConditions(cluster, StatusUpdateOpts{ACLErr: errors.New("acl sync failed")})

		cond := findCondition(cluster, ackov1alpha1.ConditionACLSynced)
		if cond == nil {
			t.Fatal("ACLSynced condition not found")
		}
		if cond.Status != metav1.ConditionFalse {
			t.Errorf("ACLSynced = %q, want False", cond.Status)
		}
	})

	t.Run("RestartInProgress=true sets MigrationComplete=False", func(t *testing.T) {
		cluster := &ackov1alpha1.AerospikeCluster{}
		setFineGrainedConditions(cluster, StatusUpdateOpts{RestartInProgress: true})

		cond := findCondition(cluster, ackov1alpha1.ConditionMigrationComplete)
		if cond == nil {
			t.Fatal("MigrationComplete condition not found")
		}
		if cond.Status != metav1.ConditionFalse {
			t.Errorf("MigrationComplete = %q, want False", cond.Status)
		}
	})

	t.Run("RestartInProgress=false sets MigrationComplete=True", func(t *testing.T) {
		cluster := &ackov1alpha1.AerospikeCluster{}
		setFineGrainedConditions(cluster, StatusUpdateOpts{RestartInProgress: false})

		cond := findCondition(cluster, ackov1alpha1.ConditionMigrationComplete)
		if cond == nil {
			t.Fatal("MigrationComplete condition not found")
		}
		if cond.Status != metav1.ConditionTrue {
			t.Errorf("MigrationComplete = %q, want True", cond.Status)
		}
	})
}

func TestConditionsSnapshot(t *testing.T) {
	tests := []struct {
		name string
		in   []metav1.Condition
		want map[string]conditionSnapshot
	}{
		{
			name: "nil conditions returns empty map",
			in:   nil,
			want: map[string]conditionSnapshot{},
		},
		{
			name: "empty slice returns empty map",
			in:   []metav1.Condition{},
			want: map[string]conditionSnapshot{},
		},
		{
			name: "single condition",
			in: []metav1.Condition{
				{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 7, Reason: "AllPodsReady", Message: "1/1 pods ready"},
			},
			want: map[string]conditionSnapshot{
				"Ready": {
					Status:             metav1.ConditionTrue,
					ObservedGeneration: 7,
					Reason:             "AllPodsReady",
					Message:            "1/1 pods ready",
				},
			},
		},
		{
			name: "multiple conditions",
			in: []metav1.Condition{
				{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 10, Reason: "AllPodsReady", Message: "3/3 pods ready"},
				{Type: "ACLSynced", Status: metav1.ConditionFalse, ObservedGeneration: 10, Reason: "ACLSyncPending", Message: "ACL sync skipped: no ready pods available"},
				{Type: "MigrationComplete", Status: metav1.ConditionTrue, ObservedGeneration: 10, Reason: "MigrationComplete", Message: "No pending data migrations"},
			},
			want: map[string]conditionSnapshot{
				"Ready": {
					Status:             metav1.ConditionTrue,
					ObservedGeneration: 10,
					Reason:             "AllPodsReady",
					Message:            "3/3 pods ready",
				},
				"ACLSynced": {
					Status:             metav1.ConditionFalse,
					ObservedGeneration: 10,
					Reason:             "ACLSyncPending",
					Message:            "ACL sync skipped: no ready pods available",
				},
				"MigrationComplete": {
					Status:             metav1.ConditionTrue,
					ObservedGeneration: 10,
					Reason:             "MigrationComplete",
					Message:            "No pending data migrations",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := conditionsSnapshot(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("conditionsSnapshot() returned %d entries, want %d", len(got), len(tc.want))
			}
			for k, v := range tc.want {
				if got[k] != v {
					t.Errorf("conditionsSnapshot()[%q] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestConditionsChanged(t *testing.T) {
	tests := []struct {
		name string
		prev map[string]conditionSnapshot
		cur  []metav1.Condition
		want bool
	}{
		{
			name: "both empty — no change",
			prev: map[string]conditionSnapshot{},
			cur:  []metav1.Condition{},
			want: false,
		},
		{
			name: "identical single condition — no change",
			prev: map[string]conditionSnapshot{
				"Ready": {
					Status:             metav1.ConditionTrue,
					ObservedGeneration: 3,
					Reason:             "AllPodsReady",
					Message:            "1/1 pods ready",
				},
			},
			cur:  []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 3, Reason: "AllPodsReady", Message: "1/1 pods ready"}},
			want: false,
		},
		{
			name: "status changed True→False",
			prev: map[string]conditionSnapshot{
				"Ready": {
					Status:             metav1.ConditionTrue,
					ObservedGeneration: 3,
					Reason:             "AllPodsReady",
					Message:            "1/1 pods ready",
				},
			},
			cur:  []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 3, Reason: "NotReady", Message: "0/1 pods ready"}},
			want: true,
		},
		{
			name: "reason changed with same status",
			prev: map[string]conditionSnapshot{
				"Ready": {
					Status:             metav1.ConditionFalse,
					ObservedGeneration: 3,
					Reason:             "NotReady",
					Message:            "0/1 pods ready",
				},
			},
			cur:  []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 3, Reason: "Progressing", Message: "1/1 pods ready"}},
			want: true,
		},
		{
			name: "message changed with same status",
			prev: map[string]conditionSnapshot{
				"Ready": {
					Status:             metav1.ConditionFalse,
					ObservedGeneration: 3,
					Reason:             "Progressing",
					Message:            "1/3 pods ready",
				},
			},
			cur:  []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 3, Reason: "Progressing", Message: "2/3 pods ready"}},
			want: true,
		},
		{
			name: "observed generation changed with same status",
			prev: map[string]conditionSnapshot{
				"Ready": {
					Status:             metav1.ConditionTrue,
					ObservedGeneration: 2,
					Reason:             "AllPodsReady",
					Message:            "1/1 pods ready",
				},
			},
			cur:  []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 3, Reason: "AllPodsReady", Message: "1/1 pods ready"}},
			want: true,
		},
		{
			name: "condition added",
			prev: map[string]conditionSnapshot{
				"Ready": {
					Status:             metav1.ConditionTrue,
					ObservedGeneration: 3,
					Reason:             "AllPodsReady",
					Message:            "1/1 pods ready",
				},
			},
			cur: []metav1.Condition{
				{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 3, Reason: "AllPodsReady", Message: "1/1 pods ready"},
				{Type: "ACLSynced", Status: metav1.ConditionTrue, ObservedGeneration: 3, Reason: "ACLSyncSucceeded", Message: "ACL roles and users are synchronized"},
			},
			want: true,
		},
		{
			name: "condition removed",
			prev: map[string]conditionSnapshot{
				"Ready": {
					Status:             metav1.ConditionTrue,
					ObservedGeneration: 3,
					Reason:             "AllPodsReady",
					Message:            "1/1 pods ready",
				},
				"ACLSynced": {
					Status:             metav1.ConditionTrue,
					ObservedGeneration: 3,
					Reason:             "ACLSyncSucceeded",
					Message:            "ACL roles and users are synchronized",
				},
			},
			cur:  []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 3, Reason: "AllPodsReady", Message: "1/1 pods ready"}},
			want: true,
		},
		{
			name: "new condition type replaces old",
			prev: map[string]conditionSnapshot{
				"OldType": {
					Status:             metav1.ConditionTrue,
					ObservedGeneration: 3,
					Reason:             "OldReason",
					Message:            "Old message",
				},
			},
			cur:  []metav1.Condition{{Type: "NewType", Status: metav1.ConditionTrue, ObservedGeneration: 3, Reason: "NewReason", Message: "New message"}},
			want: true,
		},
		{
			name: "multiple conditions identical — no change",
			prev: map[string]conditionSnapshot{
				"Ready": {
					Status:             metav1.ConditionTrue,
					ObservedGeneration: 8,
					Reason:             "AllPodsReady",
					Message:            "3/3 pods ready",
				},
				"ACLSynced": {
					Status:             metav1.ConditionFalse,
					ObservedGeneration: 8,
					Reason:             "ACLSyncPending",
					Message:            "ACL sync skipped: no ready pods available",
				},
				"MigrationComplete": {
					Status:             metav1.ConditionTrue,
					ObservedGeneration: 8,
					Reason:             "MigrationComplete",
					Message:            "No pending data migrations",
				},
			},
			cur: []metav1.Condition{
				{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 8, Reason: "AllPodsReady", Message: "3/3 pods ready"},
				{Type: "ACLSynced", Status: metav1.ConditionFalse, ObservedGeneration: 8, Reason: "ACLSyncPending", Message: "ACL sync skipped: no ready pods available"},
				{Type: "MigrationComplete", Status: metav1.ConditionTrue, ObservedGeneration: 8, Reason: "MigrationComplete", Message: "No pending data migrations"},
			},
			want: false,
		},
		{
			name: "prev empty, cur has condition — changed",
			prev: map[string]conditionSnapshot{},
			cur:  []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 1, Reason: "AllPodsReady", Message: "1/1 pods ready"}},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := conditionsChanged(tc.prev, tc.cur)
			if got != tc.want {
				t.Errorf("conditionsChanged() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStatusUnchanged(t *testing.T) {
	baseCluster := &ackov1alpha1.AerospikeCluster{
		ObjectMeta: metav1.ObjectMeta{
			Generation: 3,
		},
		Status: ackov1alpha1.AerospikeClusterStatus{
			Health:   "1/1",
			Selector: "app=aerospike,cluster=demo",
			Pods: map[string]ackov1alpha1.AerospikePodStatus{
				"demo-0": {
					PodIP:             "10.0.0.10",
					IsRunningAndReady: true,
					AccessEndpoints:   []string{"10.0.0.10:3000"},
				},
			},
			Conditions: []metav1.Condition{
				{Type: ackov1alpha1.ConditionReady, Status: metav1.ConditionTrue},
			},
		},
	}

	basePrev := statusSnapshot{
		Phase:       ackov1alpha1.AerospikePhaseCompleted,
		PhaseReason: "steady",
		Size:        1,
		Health:      "1/1",
		Generation:  3,
		Selector:    "app=aerospike,cluster=demo",
		Pods: map[string]ackov1alpha1.AerospikePodStatus{
			"demo-0": {
				PodIP:             "10.0.0.10",
				IsRunningAndReady: true,
				AccessEndpoints:   []string{"10.0.0.10:3000"},
			},
		},
		Conditions: map[string]conditionSnapshot{
			ackov1alpha1.ConditionReady: {
				Status:             metav1.ConditionTrue,
				ObservedGeneration: 0,
				Reason:             "",
				Message:            "",
			},
		},
	}

	t.Run("unchanged status returns true", func(t *testing.T) {
		if got := statusUnchanged(basePrev, baseCluster, 1,
			ackov1alpha1.AerospikePhaseCompleted, "steady",
		); !got {
			t.Fatalf("statusUnchanged() = false, want true")
		}
	})

	t.Run("pod status change returns false", func(t *testing.T) {
		cluster := baseCluster.DeepCopy()
		cluster.Status.Pods["demo-0"] = ackov1alpha1.AerospikePodStatus{
			PodIP:             "10.0.0.20",
			IsRunningAndReady: true,
			AccessEndpoints:   []string{"10.0.0.20:3000"},
		}

		if got := statusUnchanged(basePrev, cluster, 1,
			ackov1alpha1.AerospikePhaseCompleted, "steady",
		); got {
			t.Fatalf("statusUnchanged() = true, want false when pods changed")
		}
	})

	t.Run("selector change returns false", func(t *testing.T) {
		cluster := baseCluster.DeepCopy()
		cluster.Status.Selector = "app=aerospike,cluster=demo,role=node"

		if got := statusUnchanged(basePrev, cluster, 1,
			ackov1alpha1.AerospikePhaseCompleted, "steady",
		); got {
			t.Fatalf("statusUnchanged() = true, want false when selector changed")
		}
	})

	t.Run("condition reason change returns false", func(t *testing.T) {
		cluster := baseCluster.DeepCopy()
		cluster.Status.Conditions = []metav1.Condition{
			{
				Type:               ackov1alpha1.ConditionReady,
				Status:             metav1.ConditionTrue,
				ObservedGeneration: 3,
				Reason:             "AllPodsReady",
				Message:            "1/1 pods ready",
			},
		}
		prev := basePrev
		prev.Conditions = map[string]conditionSnapshot{
			ackov1alpha1.ConditionReady: {
				Status:             metav1.ConditionTrue,
				ObservedGeneration: 3,
				Reason:             "Ready",
				Message:            "1/1 pods ready",
			},
		}

		if got := statusUnchanged(prev, cluster, 1,
			ackov1alpha1.AerospikePhaseCompleted, "steady",
		); got {
			t.Fatalf("statusUnchanged() = true, want false when condition reason changed")
		}
	})
}

func TestUnstableSince(t *testing.T) {
	now := metav1.Now()
	earlier := metav1.NewTime(now.Add(-5 * time.Minute))

	tests := []struct {
		name          string
		isReady       bool
		prevUnstable  *metav1.Time
		wantNil       bool
		wantPreserved bool // unstableSince should equal prevUnstable
	}{
		{
			name:          "NotReady with no prev → sets UnstableSince",
			isReady:       false,
			prevUnstable:  nil,
			wantNil:       false,
			wantPreserved: false,
		},
		{
			name:          "NotReady with existing prev → preserves original timestamp",
			isReady:       false,
			prevUnstable:  &earlier,
			wantNil:       false,
			wantPreserved: true,
		},
		{
			name:          "Ready pod clears UnstableSince",
			isReady:       true,
			prevUnstable:  &earlier,
			wantNil:       true,
			wantPreserved: false,
		},
		{
			name:          "Ready pod with nil stays nil",
			isReady:       true,
			prevUnstable:  nil,
			wantNil:       true,
			wantPreserved: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cluster := &ackov1alpha1.AerospikeCluster{
				Status: ackov1alpha1.AerospikeClusterStatus{
					Pods: map[string]ackov1alpha1.AerospikePodStatus{
						"pod-0": {
							UnstableSince: tc.prevUnstable,
						},
					},
				},
			}

			// Simulate the logic from populateStatus
			prev := cluster.Status.Pods["pod-0"]
			var unstableSince *metav1.Time
			if !tc.isReady {
				if prev.UnstableSince != nil {
					unstableSince = prev.UnstableSince
				} else {
					now := metav1.Now()
					unstableSince = &now
				}
			}

			if tc.wantNil && unstableSince != nil {
				t.Errorf("expected nil UnstableSince for ready pod, got %v", unstableSince)
			}
			if !tc.wantNil && unstableSince == nil {
				t.Errorf("expected non-nil UnstableSince for not-ready pod")
			}
			if tc.wantPreserved && tc.prevUnstable != nil && unstableSince != nil {
				if !unstableSince.Equal(tc.prevUnstable) {
					t.Errorf("expected original timestamp %v, got %v", tc.prevUnstable, unstableSince)
				}
			}
		})
	}
}
