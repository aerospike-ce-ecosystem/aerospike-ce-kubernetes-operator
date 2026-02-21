package utils

import "fmt"

// StatefulSetName returns the name for a rack's StatefulSet.
func StatefulSetName(clusterName string, rackID int) string {
	return fmt.Sprintf("%s-%d", clusterName, rackID)
}

// HeadlessServiceName returns the headless service name for a cluster.
func HeadlessServiceName(clusterName string) string {
	return clusterName
}

// PodServiceName returns the service name for a specific pod.
func PodServiceName(clusterName string, podIndex int) string {
	return fmt.Sprintf("%s-%d", clusterName, podIndex)
}

// ConfigMapName returns the ConfigMap name for a rack.
func ConfigMapName(clusterName string, rackID int) string {
	return fmt.Sprintf("%s-%d-config", clusterName, rackID)
}

// PDBName returns the PodDisruptionBudget name for a cluster.
func PDBName(clusterName string) string {
	return fmt.Sprintf("%s-pdb", clusterName)
}

// PodDNSName returns the fully qualified DNS name for a pod.
func PodDNSName(podName, serviceName, namespace string) string {
	return fmt.Sprintf("%s.%s.%s.svc.cluster.local", podName, serviceName, namespace)
}
