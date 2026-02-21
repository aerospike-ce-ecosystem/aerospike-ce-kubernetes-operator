package utils

import "fmt"

const (
	AppLabel       = "app.kubernetes.io/name"
	InstanceLabel  = "app.kubernetes.io/instance"
	ComponentLabel = "app.kubernetes.io/component"
	ManagedByLabel = "app.kubernetes.io/managed-by"
	RackLabel      = "acko.io/rack"

	ConfigHashAnnotation = "acko.io/config-hash"
	StorageFinalizer     = "acko.io/storage-finalizer"

	appName     = "aerospike-cluster"
	managerName = "aerospike-ce-operator"
)

// LabelsForCluster returns the common labels for resources belonging to a cluster.
func LabelsForCluster(clusterName string) map[string]string {
	return map[string]string{
		AppLabel:       appName,
		InstanceLabel:  clusterName,
		ComponentLabel: "database",
		ManagedByLabel: managerName,
	}
}

// SelectorLabelsForCluster returns the minimal label set used for label selectors.
func SelectorLabelsForCluster(clusterName string) map[string]string {
	return map[string]string{
		AppLabel:      appName,
		InstanceLabel: clusterName,
	}
}

// LabelsForRack returns labels for a specific rack, including the rack ID.
func LabelsForRack(clusterName string, rackID int) map[string]string {
	labels := LabelsForCluster(clusterName)
	labels[RackLabel] = fmt.Sprintf("%d", rackID)
	return labels
}
