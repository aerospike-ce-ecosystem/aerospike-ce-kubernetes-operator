package utils

import (
	"testing"
)

const (
	testClusterName = "my-cluster"
	testValue       = "value"
)

func TestLabelsForCluster(t *testing.T) {
	labels := LabelsForCluster(testClusterName)

	expected := map[string]string{
		AppLabel:       appName,
		InstanceLabel:  testClusterName,
		ComponentLabel: "database",
		ManagedByLabel: managerName,
	}

	if len(labels) != len(expected) {
		t.Fatalf("expected %d labels, got %d", len(expected), len(labels))
	}
	for k, v := range expected {
		if labels[k] != v {
			t.Errorf("label %q = %q, want %q", k, labels[k], v)
		}
	}
}

func TestSelectorLabelsForCluster(t *testing.T) {
	labels := SelectorLabelsForCluster(testClusterName)

	if len(labels) != 2 {
		t.Fatalf("expected 2 selector labels, got %d", len(labels))
	}
	if labels[AppLabel] != appName {
		t.Errorf("AppLabel = %q, want %q", labels[AppLabel], appName)
	}
	if labels[InstanceLabel] != testClusterName {
		t.Errorf("InstanceLabel = %q, want %q", labels[InstanceLabel], testClusterName)
	}
}

func TestLabelsForRack(t *testing.T) {
	labels := LabelsForRack(testClusterName, 5)

	if labels[RackLabel] != "5" {
		t.Errorf("RackLabel = %q, want %q", labels[RackLabel], "5")
	}
	// Should include all cluster labels plus the rack label
	if labels[AppLabel] != appName {
		t.Error("should include AppLabel from LabelsForCluster")
	}
	if labels[InstanceLabel] != testClusterName {
		t.Error("should include InstanceLabel from LabelsForCluster")
	}
}

func TestLabelsForCluster_ReturnsFreshMap(t *testing.T) {
	labels1 := LabelsForCluster("a")
	labels2 := LabelsForCluster("b")

	// Mutating one should not affect the other
	labels1["custom"] = testValue
	if _, ok := labels2["custom"]; ok {
		t.Error("LabelsForCluster should return a fresh map each time")
	}
}
