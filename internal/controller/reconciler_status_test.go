package controller

import (
	"errors"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	asdbcev1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
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

func TestSetCondition(t *testing.T) {
	t.Run("new condition type is appended", func(t *testing.T) {
		cluster := &asdbcev1alpha1.AerospikeCECluster{}

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
		cluster := &asdbcev1alpha1.AerospikeCECluster{}

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
		cluster := &asdbcev1alpha1.AerospikeCECluster{}

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

	t.Run("existing condition with changed status IS updated", func(t *testing.T) {
		cluster := &asdbcev1alpha1.AerospikeCECluster{}

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
		cluster := &asdbcev1alpha1.AerospikeCECluster{}

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

func TestConditionsChanged(t *testing.T) {
	tests := []struct {
		name string
		prev map[string]metav1.ConditionStatus
		cur  []metav1.Condition
		want bool
	}{
		{
			name: "both empty",
			prev: map[string]metav1.ConditionStatus{},
			cur:  nil,
			want: false,
		},
		{
			name: "same single condition",
			prev: map[string]metav1.ConditionStatus{"Available": metav1.ConditionTrue},
			cur:  []metav1.Condition{{Type: "Available", Status: metav1.ConditionTrue}},
			want: false,
		},
		{
			name: "status changed",
			prev: map[string]metav1.ConditionStatus{"Available": metav1.ConditionTrue},
			cur:  []metav1.Condition{{Type: "Available", Status: metav1.ConditionFalse}},
			want: true,
		},
		{
			name: "condition added",
			prev: map[string]metav1.ConditionStatus{"Available": metav1.ConditionTrue},
			cur: []metav1.Condition{
				{Type: "Available", Status: metav1.ConditionTrue},
				{Type: "Ready", Status: metav1.ConditionFalse},
			},
			want: true,
		},
		{
			name: "condition removed",
			prev: map[string]metav1.ConditionStatus{
				"Available": metav1.ConditionTrue,
				"Ready":     metav1.ConditionFalse,
			},
			cur:  []metav1.Condition{{Type: "Available", Status: metav1.ConditionTrue}},
			want: true,
		},
		{
			name: "different condition type",
			prev: map[string]metav1.ConditionStatus{"Available": metav1.ConditionTrue},
			cur:  []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue}},
			want: true,
		},
		{
			name: "multiple conditions same",
			prev: map[string]metav1.ConditionStatus{
				"Available": metav1.ConditionTrue,
				"Ready":     metav1.ConditionTrue,
			},
			cur: []metav1.Condition{
				{Type: "Available", Status: metav1.ConditionTrue},
				{Type: "Ready", Status: metav1.ConditionTrue},
			},
			want: false,
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

func TestConditionsSnapshot(t *testing.T) {
	conds := []metav1.Condition{
		{Type: "Available", Status: metav1.ConditionTrue},
		{Type: "Ready", Status: metav1.ConditionFalse},
	}
	snap := conditionsSnapshot(conds)
	if len(snap) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(snap))
	}
	if snap["Available"] != metav1.ConditionTrue {
		t.Errorf("Available = %q, want True", snap["Available"])
	}
	if snap["Ready"] != metav1.ConditionFalse {
		t.Errorf("Ready = %q, want False", snap["Ready"])
	}

	// nil input
	snap2 := conditionsSnapshot(nil)
	if len(snap2) != 0 {
		t.Errorf("expected empty map for nil input, got %d entries", len(snap2))
	}
}

func TestSetFineGrainedConditions(t *testing.T) {
	findCondition := func(cluster *asdbcev1alpha1.AerospikeCECluster, condType string) *metav1.Condition {
		for i := range cluster.Status.Conditions {
			if cluster.Status.Conditions[i].Type == condType {
				return &cluster.Status.Conditions[i]
			}
		}
		return nil
	}

	t.Run("Paused=true sets ReconciliationPaused=True", func(t *testing.T) {
		cluster := &asdbcev1alpha1.AerospikeCECluster{}
		setFineGrainedConditions(cluster, StatusUpdateOpts{Paused: true})

		cond := findCondition(cluster, asdbcev1alpha1.ConditionReconciliationPaused)
		if cond == nil {
			t.Fatal("ReconciliationPaused condition not found")
		}
		if cond.Status != metav1.ConditionTrue {
			t.Errorf("ReconciliationPaused = %q, want True", cond.Status)
		}
	})

	t.Run("Paused=false sets ReconciliationPaused=False", func(t *testing.T) {
		cluster := &asdbcev1alpha1.AerospikeCECluster{}
		setFineGrainedConditions(cluster, StatusUpdateOpts{Paused: false})

		cond := findCondition(cluster, asdbcev1alpha1.ConditionReconciliationPaused)
		if cond == nil {
			t.Fatal("ReconciliationPaused condition not found")
		}
		if cond.Status != metav1.ConditionFalse {
			t.Errorf("ReconciliationPaused = %q, want False", cond.Status)
		}
	})

	t.Run("ACL spec nil: ACLSynced condition is not set", func(t *testing.T) {
		cluster := &asdbcev1alpha1.AerospikeCECluster{}
		// AerospikeAccessControl is nil by default
		setFineGrainedConditions(cluster, StatusUpdateOpts{})

		cond := findCondition(cluster, asdbcev1alpha1.ConditionACLSynced)
		if cond != nil {
			t.Errorf("ACLSynced should not be set when ACL spec is nil, got %q", cond.Status)
		}
	})

	t.Run("ACL spec set and ACLSynced true: ACLSynced=True", func(t *testing.T) {
		cluster := &asdbcev1alpha1.AerospikeCECluster{}
		cluster.Spec.AerospikeAccessControl = &asdbcev1alpha1.AerospikeAccessControlSpec{}
		setFineGrainedConditions(cluster, StatusUpdateOpts{ACLErr: nil, ACLSynced: true})

		cond := findCondition(cluster, asdbcev1alpha1.ConditionACLSynced)
		if cond == nil {
			t.Fatal("ACLSynced condition not found")
		}
		if cond.Status != metav1.ConditionTrue {
			t.Errorf("ACLSynced = %q, want True", cond.Status)
		}
	})

	t.Run("ACL spec set and ACL skipped (no ready pods): ACLSynced=False with pending reason", func(t *testing.T) {
		cluster := &asdbcev1alpha1.AerospikeCECluster{}
		cluster.Spec.AerospikeAccessControl = &asdbcev1alpha1.AerospikeAccessControlSpec{}
		setFineGrainedConditions(cluster, StatusUpdateOpts{ACLErr: nil, ACLSynced: false})

		cond := findCondition(cluster, asdbcev1alpha1.ConditionACLSynced)
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
		cluster := &asdbcev1alpha1.AerospikeCECluster{}
		cluster.Spec.AerospikeAccessControl = &asdbcev1alpha1.AerospikeAccessControlSpec{}
		setFineGrainedConditions(cluster, StatusUpdateOpts{ACLErr: errors.New("acl sync failed")})

		cond := findCondition(cluster, asdbcev1alpha1.ConditionACLSynced)
		if cond == nil {
			t.Fatal("ACLSynced condition not found")
		}
		if cond.Status != metav1.ConditionFalse {
			t.Errorf("ACLSynced = %q, want False", cond.Status)
		}
	})

	t.Run("RestartInProgress=true sets MigrationComplete=False", func(t *testing.T) {
		cluster := &asdbcev1alpha1.AerospikeCECluster{}
		setFineGrainedConditions(cluster, StatusUpdateOpts{RestartInProgress: true})

		cond := findCondition(cluster, asdbcev1alpha1.ConditionMigrationComplete)
		if cond == nil {
			t.Fatal("MigrationComplete condition not found")
		}
		if cond.Status != metav1.ConditionFalse {
			t.Errorf("MigrationComplete = %q, want False", cond.Status)
		}
	})

	t.Run("RestartInProgress=false sets MigrationComplete=True", func(t *testing.T) {
		cluster := &asdbcev1alpha1.AerospikeCECluster{}
		setFineGrainedConditions(cluster, StatusUpdateOpts{RestartInProgress: false})

		cond := findCondition(cluster, asdbcev1alpha1.ConditionMigrationComplete)
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
		want map[string]metav1.ConditionStatus
	}{
		{
			name: "nil conditions returns empty map",
			in:   nil,
			want: map[string]metav1.ConditionStatus{},
		},
		{
			name: "empty slice returns empty map",
			in:   []metav1.Condition{},
			want: map[string]metav1.ConditionStatus{},
		},
		{
			name: "single condition",
			in: []metav1.Condition{
				{Type: "Ready", Status: metav1.ConditionTrue},
			},
			want: map[string]metav1.ConditionStatus{"Ready": metav1.ConditionTrue},
		},
		{
			name: "multiple conditions",
			in: []metav1.Condition{
				{Type: "Ready", Status: metav1.ConditionTrue},
				{Type: "ACLSynced", Status: metav1.ConditionFalse},
				{Type: "MigrationComplete", Status: metav1.ConditionTrue},
			},
			want: map[string]metav1.ConditionStatus{
				"Ready":             metav1.ConditionTrue,
				"ACLSynced":         metav1.ConditionFalse,
				"MigrationComplete": metav1.ConditionTrue,
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
		prev map[string]metav1.ConditionStatus
		cur  []metav1.Condition
		want bool
	}{
		{
			name: "both empty — no change",
			prev: map[string]metav1.ConditionStatus{},
			cur:  []metav1.Condition{},
			want: false,
		},
		{
			name: "identical single condition — no change",
			prev: map[string]metav1.ConditionStatus{"Ready": metav1.ConditionTrue},
			cur:  []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue}},
			want: false,
		},
		{
			name: "status changed True→False",
			prev: map[string]metav1.ConditionStatus{"Ready": metav1.ConditionTrue},
			cur:  []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse}},
			want: true,
		},
		{
			name: "condition added",
			prev: map[string]metav1.ConditionStatus{"Ready": metav1.ConditionTrue},
			cur: []metav1.Condition{
				{Type: "Ready", Status: metav1.ConditionTrue},
				{Type: "ACLSynced", Status: metav1.ConditionTrue},
			},
			want: true,
		},
		{
			name: "condition removed",
			prev: map[string]metav1.ConditionStatus{
				"Ready":     metav1.ConditionTrue,
				"ACLSynced": metav1.ConditionTrue,
			},
			cur:  []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue}},
			want: true,
		},
		{
			name: "new condition type replaces old",
			prev: map[string]metav1.ConditionStatus{"OldType": metav1.ConditionTrue},
			cur:  []metav1.Condition{{Type: "NewType", Status: metav1.ConditionTrue}},
			want: true,
		},
		{
			name: "multiple conditions identical — no change",
			prev: map[string]metav1.ConditionStatus{
				"Ready":             metav1.ConditionTrue,
				"ACLSynced":         metav1.ConditionFalse,
				"MigrationComplete": metav1.ConditionTrue,
			},
			cur: []metav1.Condition{
				{Type: "Ready", Status: metav1.ConditionTrue},
				{Type: "ACLSynced", Status: metav1.ConditionFalse},
				{Type: "MigrationComplete", Status: metav1.ConditionTrue},
			},
			want: false,
		},
		{
			name: "prev empty, cur has condition — changed",
			prev: map[string]metav1.ConditionStatus{},
			cur:  []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue}},
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
