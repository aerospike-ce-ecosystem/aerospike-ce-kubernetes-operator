package controller

import (
	"context"
	"maps"

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

	if errors.IsNotFound(err) {
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      svcName,
				Namespace: cluster.Namespace,
				Labels:    labels,
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
		return err
	}

	// Update if ports or labels changed.
	needsUpdate := !maps.Equal(existing.Labels, labels)
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
		existing.Spec.Ports = desiredPorts
		existing.Spec.Selector = selectorLabels
		log.Info("Updating headless service", "name", svcName)
		return r.Update(ctx, existing)
	}

	return nil
}
