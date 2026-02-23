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
		return fmt.Errorf("listing cluster pods for pod services: %w", err)
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

		// Build desired annotations from custom metadata.
		var desiredAnnotations map[string]string
		if cluster.Spec.PodService.Metadata != nil {
			if cluster.Spec.PodService.Metadata.Annotations != nil {
				desiredAnnotations = make(map[string]string)
				maps.Copy(desiredAnnotations, cluster.Spec.PodService.Metadata.Annotations)
			}
			if cluster.Spec.PodService.Metadata.Labels != nil {
				maps.Copy(labels, cluster.Spec.PodService.Metadata.Labels)
			}
		}

		existing := &corev1.Service{}
		err := r.Get(ctx, types.NamespacedName{Name: svcName, Namespace: cluster.Namespace}, existing)

		if errors.IsNotFound(err) {
			svc := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:        svcName,
					Namespace:   cluster.Namespace,
					Labels:      labels,
					Annotations: desiredAnnotations,
				},
				Spec: corev1.ServiceSpec{
					Type:     corev1.ServiceTypeClusterIP,
					Selector: podSelector,
					Ports:    desiredPorts,
				},
			}

			if err := r.setOwnerRef(cluster, svc); err != nil {
				return err
			}

			log.Info("Creating per-pod service", "name", svcName, "pod", pod.Name)
			if err := r.Create(ctx, svc); err != nil {
				return err
			}

			continue
		} else if err != nil {
			return fmt.Errorf("getting pod service %s: %w", svcName, err)
		}

		// Compare and update if needed.
		needsUpdate := !equalAnnotations(existing.Annotations, desiredAnnotations)
		if !needsUpdate {
			needsUpdate = !maps.Equal(existing.Labels, labels)
		}
		if !needsUpdate && len(existing.Spec.Ports) == len(desiredPorts) {
			for i, p := range existing.Spec.Ports {
				if p.Name != desiredPorts[i].Name || p.Port != desiredPorts[i].Port {
					needsUpdate = true
					break
				}
			}
		} else if !needsUpdate {
			needsUpdate = true
		}

		if needsUpdate {
			existing.Labels = labels
			if desiredAnnotations != nil {
				if existing.Annotations == nil {
					existing.Annotations = make(map[string]string)
				}
				maps.Copy(existing.Annotations, desiredAnnotations)
			}
			existing.Spec.Ports = desiredPorts
			log.Info("Updating per-pod service", "name", svcName)
			if err := r.Update(ctx, existing); err != nil {
				return fmt.Errorf("updating pod service %s: %w", svcName, err)
			}
		} else {
			log.V(1).Info("Pod service up-to-date, no update needed", "name", svcName)
		}
	}

	return nil
}
