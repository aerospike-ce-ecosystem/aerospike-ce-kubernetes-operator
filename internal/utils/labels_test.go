package utils

import (
	"testing"
)

func TestLabelsForCluster(t *testing.T) {
	labels := LabelsForCluster("my-cluster")

	expected := map[string]string{
		AppLabel:       appName,
		InstanceLabel:  "my-cluster",
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
	labels := SelectorLabelsForCluster("my-cluster")

	if len(labels) != 2 {
		t.Fatalf("expected 2 selector labels, got %d", len(labels))
	}
	if labels[AppLabel] != appName {
		t.Errorf("AppLabel = %q, want %q", labels[AppLabel], appName)
	}
	if labels[InstanceLabel] != "my-cluster" {
		t.Errorf("InstanceLabel = %q, want %q", labels[InstanceLabel], "my-cluster")
	}
}

func TestLabelsForRack(t *testing.T) {
	labels := LabelsForRack("my-cluster", 5)

	if labels[RackLabel] != "5" {
		t.Errorf("RackLabel = %q, want %q", labels[RackLabel], "5")
	}
	// Should include all cluster labels plus the rack label
	if labels[AppLabel] != appName {
		t.Error("should include AppLabel from LabelsForCluster")
	}
	if labels[InstanceLabel] != "my-cluster" {
		t.Error("should include InstanceLabel from LabelsForCluster")
	}
}

func TestLabelsForCluster_ReturnsFreshMap(t *testing.T) {
	labels1 := LabelsForCluster("a")
	labels2 := LabelsForCluster("b")

	// Mutating one should not affect the other
	labels1["custom"] = "value"
	if _, ok := labels2["custom"]; ok {
		t.Error("LabelsForCluster should return a fresh map each time")
	}
}
