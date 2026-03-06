package controller

import (
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestToStringMap(t *testing.T) {
	tests := []struct {
		name string
		in   map[string]string
		want map[string]any
	}{
		{
			name: "nil map returns empty map (not nil)",
			in:   nil,
			want: map[string]any{},
		},
		{
			name: "empty map returns empty map",
			in:   map[string]string{},
			want: map[string]any{},
		},
		{
			name: "single entry is converted",
			in:   map[string]string{"app": "aerospike"},
			want: map[string]any{"app": "aerospike"},
		},
		{
			name: "multiple entries are all converted",
			in: map[string]string{
				"app":  "aerospike",
				"team": "platform",
				"env":  "prod",
			},
			want: map[string]any{
				"app":  "aerospike",
				"team": "platform",
				"env":  "prod",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := toStringMap(tc.in)
			if got == nil {
				t.Fatal("toStringMap() returned nil, want non-nil map")
			}
			if len(got) != len(tc.want) {
				t.Fatalf("toStringMap() returned %d entries, want %d", len(got), len(tc.want))
			}
			for k, wantVal := range tc.want {
				gotVal, ok := got[k]
				if !ok {
					t.Errorf("toStringMap() missing key %q", k)
					continue
				}
				if gotVal != wantVal {
					t.Errorf("toStringMap()[%q] = %v, want %v", k, gotVal, wantVal)
				}
			}
		})
	}
}

func TestDefaultAlertRules(t *testing.T) {
	rules := defaultAlertRules("my-cluster", "aerospike")
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule group, got %d", len(rules))
	}

	group, ok := rules[0].(map[string]any)
	if !ok {
		t.Fatal("rule group is not map[string]any")
	}

	groupName, ok := group["name"].(string)
	if !ok || groupName != "my-cluster.rules" {
		t.Errorf("group name = %q, want %q", groupName, "my-cluster.rules")
	}

	ruleList, ok := group["rules"].([]any)
	if !ok {
		t.Fatal("rules is not []any")
	}

	expectedAlerts := []string{
		"AerospikeNodeDown",
		"AerospikeNamespaceStopWrites",
		"AerospikeHighDiskUsage",
		"AerospikeHighMemoryUsage",
		"AerospikeReconcileStale",
		"AerospikeClusterSizeMismatch",
	}

	if len(ruleList) != len(expectedAlerts) {
		t.Fatalf("expected %d alert rules, got %d", len(expectedAlerts), len(ruleList))
	}

	for i, expected := range expectedAlerts {
		rule, ok := ruleList[i].(map[string]any)
		if !ok {
			t.Fatalf("rule[%d] is not map[string]any", i)
		}
		alertName, ok := rule["alert"].(string)
		if !ok || alertName != expected {
			t.Errorf("rule[%d].alert = %q, want %q", i, alertName, expected)
		}

		// Verify expressions reference the cluster/namespace context
		expr, ok := rule["expr"].(string)
		if !ok {
			t.Errorf("rule[%d].expr is not string", i)
			continue
		}
		if !strings.Contains(expr, "aerospike") {
			t.Errorf("rule[%d].expr = %q, expected to contain namespace reference", i, expr)
		}
		if !strings.Contains(expr, "my-cluster") {
			t.Errorf("rule[%d].expr = %q, expected to contain cluster name reference", i, expr)
		}
	}
}

func TestDefaultAlertRules_LabelSeverity(t *testing.T) {
	rules := defaultAlertRules("test", "ns")
	group := rules[0].(map[string]any)
	ruleList := group["rules"].([]any)

	criticalCount := 0
	warningCount := 0
	for _, r := range ruleList {
		rule := r.(map[string]any)
		labels := rule["labels"].(map[string]any)
		switch labels["severity"].(string) {
		case "critical":
			criticalCount++
		case "warning":
			warningCount++
		}
	}

	if criticalCount != 2 {
		t.Errorf("expected 2 critical alerts, got %d", criticalCount)
	}
	if warningCount != 4 {
		t.Errorf("expected 4 warning alerts, got %d", warningCount)
	}
}

func TestUnstructuredResourceChanged(t *testing.T) {
	t.Run("returns false when spec and labels are unchanged", func(t *testing.T) {
		existing := &unstructured.Unstructured{
			Object: map[string]any{
				"spec": map[string]any{
					"groups": []any{map[string]any{"name": "test.rules"}},
				},
			},
		}
		existing.SetLabels(map[string]string{"app": "aerospike", "instance": "test"})

		changed := unstructuredResourceChanged(
			existing,
			map[string]any{
				"groups": []any{map[string]any{"name": "test.rules"}},
			},
			map[string]string{"app": "aerospike", "instance": "test"},
		)
		if changed {
			t.Fatal("unstructuredResourceChanged() = true, want false")
		}
	})

	t.Run("returns true when labels change", func(t *testing.T) {
		existing := &unstructured.Unstructured{
			Object: map[string]any{
				"spec": map[string]any{
					"selector": map[string]any{"matchLabels": map[string]any{"app": "aerospike"}},
				},
			},
		}
		existing.SetLabels(map[string]string{"app": "aerospike", "instance": "test"})

		changed := unstructuredResourceChanged(
			existing,
			map[string]any{
				"selector": map[string]any{"matchLabels": map[string]any{"app": "aerospike"}},
			},
			map[string]string{"app": "aerospike", "instance": "test2"},
		)
		if !changed {
			t.Fatal("unstructuredResourceChanged() = false, want true when labels differ")
		}
	})

	t.Run("returns true when spec changes", func(t *testing.T) {
		existing := &unstructured.Unstructured{
			Object: map[string]any{
				"spec": map[string]any{
					"groups": []any{map[string]any{"name": "test.rules"}},
				},
			},
		}
		existing.SetLabels(map[string]string{"app": "aerospike"})

		changed := unstructuredResourceChanged(
			existing,
			map[string]any{
				"groups": []any{map[string]any{"name": "test-v2.rules"}},
			},
			map[string]string{"app": "aerospike"},
		)
		if !changed {
			t.Fatal("unstructuredResourceChanged() = false, want true when spec differs")
		}
	})

	t.Run("returns true when existing spec is missing", func(t *testing.T) {
		existing := &unstructured.Unstructured{Object: map[string]any{}}
		existing.SetLabels(map[string]string{"app": "aerospike"})

		changed := unstructuredResourceChanged(
			existing,
			map[string]any{"groups": []any{}},
			map[string]string{"app": "aerospike"},
		)
		if !changed {
			t.Fatal("unstructuredResourceChanged() = false, want true when existing spec is missing")
		}
	})
}
