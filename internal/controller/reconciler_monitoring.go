package controller

import (
	"context"
	"fmt"
	"maps"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	asdbcev1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/utils"
)

var serviceMonitorGVK = schema.GroupVersionKind{
	Group:   "monitoring.coreos.com",
	Version: "v1",
	Kind:    "ServiceMonitor",
}

func (r *AerospikeCEClusterReconciler) reconcileMonitoring(
	ctx context.Context,
	cluster *asdbcev1alpha1.AerospikeCECluster,
) error {
	log := logf.FromContext(ctx)

	monitoringEnabled := cluster.Spec.Monitoring != nil && cluster.Spec.Monitoring.Enabled

	// Reconcile metrics Service
	if err := r.reconcileMetricsService(ctx, cluster, monitoringEnabled); err != nil {
		return err
	}

	// Reconcile ServiceMonitor
	smEnabled := monitoringEnabled &&
		cluster.Spec.Monitoring.ServiceMonitor != nil &&
		cluster.Spec.Monitoring.ServiceMonitor.Enabled

	if err := r.reconcileServiceMonitor(ctx, cluster, smEnabled); err != nil {
		log.Info("ServiceMonitor reconciliation skipped (CRD may not be installed)", "error", err)
	}

	return nil
}

func (r *AerospikeCEClusterReconciler) reconcileMetricsService(
	ctx context.Context,
	cluster *asdbcev1alpha1.AerospikeCECluster,
	enabled bool,
) error {
	log := logf.FromContext(ctx)
	svcName := utils.MetricsServiceName(cluster.Name)

	existing := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: svcName, Namespace: cluster.Namespace}, existing)

	if !enabled {
		if err == nil {
			log.Info("Deleting metrics Service", "name", svcName)
			return r.Delete(ctx, existing)
		}
		return nil
	}

	port := cluster.Spec.Monitoring.Port
	labels := utils.LabelsForCluster(cluster.Name)
	selectorLabels := utils.SelectorLabelsForCluster(cluster.Name)

	desired := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: selectorLabels,
			Ports: []corev1.ServicePort{
				{
					Name:       "metrics",
					Port:       port,
					TargetPort: intstr.FromInt32(port),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}

	if errors.IsNotFound(err) {
		if err := ctrl.SetControllerReference(cluster, desired, r.Scheme); err != nil {
			return fmt.Errorf("setting controller reference for metrics service: %w", err)
		}
		log.Info("Creating metrics Service", "name", svcName)
		return r.Create(ctx, desired)
	} else if err != nil {
		return fmt.Errorf("getting metrics service %s: %w", svcName, err)
	}

	// Update existing
	existing.Spec.Ports = desired.Spec.Ports
	existing.Spec.Selector = desired.Spec.Selector
	existing.Labels = labels
	log.Info("Updating metrics Service", "name", svcName)
	return r.Update(ctx, existing)
}

func (r *AerospikeCEClusterReconciler) reconcileServiceMonitor(
	ctx context.Context,
	cluster *asdbcev1alpha1.AerospikeCECluster,
	enabled bool,
) error {
	log := logf.FromContext(ctx)
	smName := utils.ServiceMonitorName(cluster.Name)

	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(serviceMonitorGVK)

	err := r.Get(ctx, types.NamespacedName{Name: smName, Namespace: cluster.Namespace}, existing)

	if !enabled {
		if err == nil {
			log.Info("Deleting ServiceMonitor", "name", smName)
			return r.Delete(ctx, existing)
		}
		return nil
	}

	// CRD not installed — graceful skip
	if err != nil && meta.IsNoMatchError(err) {
		log.Info("ServiceMonitor CRD not installed, skipping")
		return nil
	}

	monitoring := cluster.Spec.Monitoring
	interval := monitoring.ServiceMonitor.Interval

	labels := utils.LabelsForCluster(cluster.Name)
	maps.Copy(labels, monitoring.ServiceMonitor.Labels)

	selectorLabels := utils.SelectorLabelsForCluster(cluster.Name)

	smSpec := map[string]any{
		"selector": map[string]any{
			"matchLabels": toStringMap(selectorLabels),
		},
		"endpoints": []any{
			map[string]any{
				"port":     "metrics",
				"interval": interval,
				"path":     "/metrics",
			},
		},
		"namespaceSelector": map[string]any{
			"matchNames": []any{cluster.Namespace},
		},
	}

	if errors.IsNotFound(err) {
		sm := &unstructured.Unstructured{}
		sm.SetGroupVersionKind(serviceMonitorGVK)
		sm.SetName(smName)
		sm.SetNamespace(cluster.Namespace)
		sm.SetLabels(labels)
		sm.Object["spec"] = smSpec

		if setErr := ctrl.SetControllerReference(cluster, sm, r.Scheme); setErr != nil {
			return fmt.Errorf("setting controller reference for ServiceMonitor: %w", setErr)
		}
		log.Info("Creating ServiceMonitor", "name", smName)
		return r.Create(ctx, sm)
	} else if err != nil {
		return fmt.Errorf("getting ServiceMonitor %s: %w", smName, err)
	}

	// Update existing
	existing.Object["spec"] = smSpec
	existing.SetLabels(labels)
	log.Info("Updating ServiceMonitor", "name", smName)
	return r.Update(ctx, existing)
}

func toStringMap(m map[string]string) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
