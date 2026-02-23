package controller

import (
	"context"
	"fmt"
	"maps"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	asdbcev1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/podutil"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/utils"
)

// reconcileHeadlessService creates or updates the headless service for the cluster.
func (r *AerospikeCEClusterReconciler) reconcileHeadlessService(
	ctx context.Context,
	cluster *asdbcev1alpha1.AerospikeCECluster,
) error {
	log := logf.FromContext(ctx)
	svcName := utils.HeadlessServiceName(cluster.Name)
	labels := utils.LabelsForCluster(cluster.Name)
	selectorLabels := utils.SelectorLabelsForCluster(cluster.Name)

	existing := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: svcName, Namespace: cluster.Namespace}, existing)

	desiredPorts := []corev1.ServicePort{
		{Name: "service", Port: podutil.ServicePort, TargetPort: intstr.FromInt32(podutil.ServicePort), Protocol: corev1.ProtocolTCP},
		{Name: "fabric", Port: podutil.FabricPort, TargetPort: intstr.FromInt32(podutil.FabricPort), Protocol: corev1.ProtocolTCP},
		{Name: "heartbeat", Port: podutil.HeartbeatPort, TargetPort: intstr.FromInt32(podutil.HeartbeatPort), Protocol: corev1.ProtocolTCP},
		{Name: "info", Port: podutil.InfoPort, TargetPort: intstr.FromInt32(podutil.InfoPort), Protocol: corev1.ProtocolTCP},
	}

	// Build desired annotations from custom metadata.
	var desiredAnnotations map[string]string
	if cluster.Spec.HeadlessService != nil && cluster.Spec.HeadlessService.Metadata != nil {
		if cluster.Spec.HeadlessService.Metadata.Annotations != nil {
			desiredAnnotations = make(map[string]string)
			maps.Copy(desiredAnnotations, cluster.Spec.HeadlessService.Metadata.Annotations)
		}
		if cluster.Spec.HeadlessService.Metadata.Labels != nil {
			maps.Copy(labels, cluster.Spec.HeadlessService.Metadata.Labels)
		}
	}

	if errors.IsNotFound(err) {
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:        svcName,
				Namespace:   cluster.Namespace,
				Labels:      labels,
				Annotations: desiredAnnotations,
			},
			Spec: corev1.ServiceSpec{
				ClusterIP:                corev1.ClusterIPNone,
				Selector:                 selectorLabels,
				PublishNotReadyAddresses: true,
				Ports:                    desiredPorts,
			},
		}
		if err := r.setOwnerRef(cluster, svc); err != nil {
			return err
		}
		log.Info("Creating headless service", "name", svcName)
		return r.Create(ctx, svc)
	} else if err != nil {
		return fmt.Errorf("getting headless service %s: %w", svcName, err)
	}

	// Update if ports, labels, or annotations changed.
	needsUpdate := !maps.Equal(existing.Labels, labels)

	if !needsUpdate {
		needsUpdate = !equalAnnotations(existing.Annotations, desiredAnnotations)
	}

	if !needsUpdate && len(existing.Spec.Ports) == len(desiredPorts) {
		for i, p := range existing.Spec.Ports {
			if p.Name != desiredPorts[i].Name || p.Port != desiredPorts[i].Port {
				needsUpdate = true
				break
			}
		}
	} else if len(existing.Spec.Ports) != len(desiredPorts) {
		needsUpdate = true
	}

	if needsUpdate {
		existing.Labels = labels
		existing.Annotations = reconcileAnnotations(existing.Annotations, desiredAnnotations)
		existing.Spec.Ports = desiredPorts
		existing.Spec.Selector = selectorLabels
		log.Info("Updating headless service", "name", svcName)
		return r.Update(ctx, existing)
	}

	return nil
}

// equalAnnotations checks whether the existing annotations already match the
// desired state after reconciliation. It builds the expected annotation map
// (preserving system annotations, applying desired) and compares it to actual.
// This correctly detects additions, updates, and removals of operator-managed annotations.
func equalAnnotations(actual, desired map[string]string) bool {
	reconciled := reconcileAnnotations(actual, desired)
	return maps.Equal(actual, reconciled)
}

// reconcileAnnotations builds the target annotations map by preserving
// Kubernetes system annotations from existing and overlaying operator-managed
// annotations from desired. Operator-managed annotations not present in desired
// are removed, allowing annotation cleanup when users remove entries from the CR.
//
// System annotations (containing "kubernetes.io/" or "k8s.io/" in the key) are
// preserved to avoid conflicts with Kubernetes and admission controllers.
func reconcileAnnotations(existing, desired map[string]string) map[string]string {
	if existing == nil && desired == nil {
		return nil
	}
	result := make(map[string]string)
	// Preserve system annotations from existing.
	for k, v := range existing {
		if isSystemAnnotation(k) {
			result[k] = v
		}
	}
	// Overlay desired operator annotations.
	maps.Copy(result, desired)
	if len(result) == 0 {
		return nil
	}
	return result
}

// isSystemAnnotation returns true for annotations managed by Kubernetes itself
// or by admission controllers (e.g., kubectl.kubernetes.io/last-applied-configuration).
func isSystemAnnotation(key string) bool {
	return strings.Contains(key, "kubernetes.io/") || strings.Contains(key, "k8s.io/")
}
