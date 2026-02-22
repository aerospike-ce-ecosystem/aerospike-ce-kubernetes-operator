package controller

import (
	"context"
	"fmt"
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

// reconcilePodServices creates or updates individual Services for each pod
// when spec.podService is configured.
func (r *AerospikeCEClusterReconciler) reconcilePodServices(
	ctx context.Context,
	cluster *asdbcev1alpha1.AerospikeCECluster,
) error {
	if cluster.Spec.PodService == nil {
		return nil
	}

	log := logf.FromContext(ctx)

	pods, err := r.listClusterPods(ctx, cluster)
	if err != nil {
		return err
	}

	for i := range pods.Items {
		pod := &pods.Items[i]
		svcName := fmt.Sprintf("%s-pod", pod.Name)

		labels := utils.LabelsForCluster(cluster.Name)
		labels["acko.io/pod-service"] = pod.Name

		desiredPorts := []corev1.ServicePort{
			{Name: "service", Port: podutil.ServicePort, TargetPort: intstr.FromInt32(podutil.ServicePort), Protocol: corev1.ProtocolTCP},
		}

		// Pod-specific selector
		podSelector := map[string]string{
			"statefulset.kubernetes.io/pod-name": pod.Name,
		}

		existing := &corev1.Service{}
		err := r.Get(ctx, types.NamespacedName{Name: svcName, Namespace: cluster.Namespace}, existing)

		if errors.IsNotFound(err) {
			svc := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      svcName,
					Namespace: cluster.Namespace,
					Labels:    labels,
				},
				Spec: corev1.ServiceSpec{
					Type:     corev1.ServiceTypeClusterIP,
					Selector: podSelector,
					Ports:    desiredPorts,
				},
			}

			// Apply custom metadata
			if cluster.Spec.PodService.Metadata != nil {
				if cluster.Spec.PodService.Metadata.Annotations != nil {
					svc.Annotations = make(map[string]string)
					maps.Copy(svc.Annotations, cluster.Spec.PodService.Metadata.Annotations)
				}
				if cluster.Spec.PodService.Metadata.Labels != nil {
					maps.Copy(svc.Labels, cluster.Spec.PodService.Metadata.Labels)
				}
			}

			if err := r.setOwnerRef(cluster, svc); err != nil {
				return err
			}

			log.Info("Creating per-pod service", "name", svcName, "pod", pod.Name)
			if err := r.Create(ctx, svc); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		// If exists, check if update needed (skip for now - basic implementation)
	}

	return nil
}
