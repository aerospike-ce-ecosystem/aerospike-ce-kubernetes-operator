//go:build integration

package controller

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"

	asdbcev1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/podutil"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/utils"
)

var _ = Describe("reconcilePodServices", func() {
	var (
		reconciler *AerospikeCEClusterReconciler
		ns         *corev1.Namespace
	)

	BeforeEach(func() {
		reconciler = &AerospikeCEClusterReconciler{
			Client: k8sClient,
			Scheme: scheme.Scheme,
		}

		// Create a unique namespace for each test to avoid collisions.
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "pod-svc-test-",
			},
		}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(ctx, ns)).To(Succeed())
	})

	newCluster := func(namespace, name string, podService *asdbcev1alpha1.AerospikeServiceSpec) *asdbcev1alpha1.AerospikeCECluster {
		return &asdbcev1alpha1.AerospikeCECluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: asdbcev1alpha1.AerospikeCEClusterSpec{
				Size:       1,
				Image:      "aerospike:ce-8.1.1.1",
				PodService: podService,
			},
		}
	}

	createClusterCR := func(cluster *asdbcev1alpha1.AerospikeCECluster) {
		Expect(k8sClient.Create(ctx, cluster)).To(Succeed())
	}

	createPod := func(namespace, podName, clusterName string) *corev1.Pod {
		labels := utils.LabelsForCluster(clusterName)
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: namespace,
				Labels:    labels,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  podutil.AerospikeContainerName,
						Image: "aerospike:ce-8.1.1.1",
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, pod)).To(Succeed())
		return pod
	}

	Context("when podService is configured", func() {
		It("should create a per-pod service for each pod", func() {
			clusterName := "test-create"
			podSvc := &asdbcev1alpha1.AerospikeServiceSpec{
				Metadata: &asdbcev1alpha1.AerospikeObjectMeta{
					Annotations: map[string]string{"example.com/env": "test"},
				},
			}
			cluster := newCluster(ns.Name, clusterName, podSvc)
			createClusterCR(cluster)

			// Re-fetch to get UID (needed for owner reference).
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: clusterName, Namespace: ns.Name}, cluster)).To(Succeed())

			podName := fmt.Sprintf("%s-0", clusterName)
			createPod(ns.Name, podName, clusterName)

			// Reconcile pod services.
			err := reconciler.reconcilePodServices(ctx, cluster)
			Expect(err).NotTo(HaveOccurred())

			// Verify the service was created.
			svcName := fmt.Sprintf("%s-pod", podName)
			svc := &corev1.Service{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: svcName, Namespace: ns.Name}, svc)).To(Succeed())

			Expect(svc.Spec.Type).To(Equal(corev1.ServiceTypeClusterIP))
			Expect(svc.Spec.Selector).To(HaveKeyWithValue("statefulset.kubernetes.io/pod-name", podName))
			Expect(svc.Spec.Ports).To(HaveLen(1))
			Expect(svc.Spec.Ports[0].Port).To(Equal(podutil.ServicePort))
			Expect(svc.Annotations).To(HaveKeyWithValue("example.com/env", "test"))
			Expect(svc.Labels).To(HaveKeyWithValue("acko.io/pod-service", podName))
		})
	})

	Context("when annotations or labels change", func() {
		It("should update the existing per-pod service", func() {
			clusterName := "test-update"

			// Start with initial annotations.
			podSvc := &asdbcev1alpha1.AerospikeServiceSpec{
				Metadata: &asdbcev1alpha1.AerospikeObjectMeta{
					Annotations: map[string]string{"example.com/env": "staging"},
				},
			}
			cluster := newCluster(ns.Name, clusterName, podSvc)
			createClusterCR(cluster)
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: clusterName, Namespace: ns.Name}, cluster)).To(Succeed())

			podName := fmt.Sprintf("%s-0", clusterName)
			createPod(ns.Name, podName, clusterName)

			// First reconcile: create.
			Expect(reconciler.reconcilePodServices(ctx, cluster)).To(Succeed())

			svcName := fmt.Sprintf("%s-pod", podName)
			svc := &corev1.Service{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: svcName, Namespace: ns.Name}, svc)).To(Succeed())
			Expect(svc.Annotations).To(HaveKeyWithValue("example.com/env", "staging"))

			// Update cluster spec with new annotations.
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: clusterName, Namespace: ns.Name}, cluster)).To(Succeed())
			cluster.Spec.PodService = &asdbcev1alpha1.AerospikeServiceSpec{
				Metadata: &asdbcev1alpha1.AerospikeObjectMeta{
					Annotations: map[string]string{
						"example.com/env":    "production",
						"example.com/region": "us-west-2",
					},
					Labels: map[string]string{
						"custom-label": "custom-value",
					},
				},
			}
			Expect(k8sClient.Update(ctx, cluster)).To(Succeed())

			// Re-fetch to get latest version.
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: clusterName, Namespace: ns.Name}, cluster)).To(Succeed())

			// Second reconcile: update.
			Expect(reconciler.reconcilePodServices(ctx, cluster)).To(Succeed())

			// Verify updated service.
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: svcName, Namespace: ns.Name}, svc)).To(Succeed())
			Expect(svc.Annotations).To(HaveKeyWithValue("example.com/env", "production"))
			Expect(svc.Annotations).To(HaveKeyWithValue("example.com/region", "us-west-2"))
			Expect(svc.Labels).To(HaveKeyWithValue("custom-label", "custom-value"))
		})
	})

	Context("when annotations are removed from the CR", func() {
		It("should remove stale operator annotations from the service", func() {
			clusterName := "test-remove-ann"

			// Start with annotations.
			podSvc := &asdbcev1alpha1.AerospikeServiceSpec{
				Metadata: &asdbcev1alpha1.AerospikeObjectMeta{
					Annotations: map[string]string{
						"example.com/env":    "staging",
						"example.com/region": "us-east-1",
					},
				},
			}
			cluster := newCluster(ns.Name, clusterName, podSvc)
			createClusterCR(cluster)
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: clusterName, Namespace: ns.Name}, cluster)).To(Succeed())

			podName := fmt.Sprintf("%s-0", clusterName)
			createPod(ns.Name, podName, clusterName)

			// First reconcile: create with both annotations.
			Expect(reconciler.reconcilePodServices(ctx, cluster)).To(Succeed())

			svcName := fmt.Sprintf("%s-pod", podName)
			svc := &corev1.Service{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: svcName, Namespace: ns.Name}, svc)).To(Succeed())
			Expect(svc.Annotations).To(HaveKeyWithValue("example.com/env", "staging"))
			Expect(svc.Annotations).To(HaveKeyWithValue("example.com/region", "us-east-1"))

			// Update CR: remove all annotations.
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: clusterName, Namespace: ns.Name}, cluster)).To(Succeed())
			cluster.Spec.PodService = &asdbcev1alpha1.AerospikeServiceSpec{}
			Expect(k8sClient.Update(ctx, cluster)).To(Succeed())
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: clusterName, Namespace: ns.Name}, cluster)).To(Succeed())

			// Second reconcile: should remove stale annotations.
			Expect(reconciler.reconcilePodServices(ctx, cluster)).To(Succeed())

			// Verify annotations are removed.
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: svcName, Namespace: ns.Name}, svc)).To(Succeed())
			Expect(svc.Annotations).ToNot(HaveKey("example.com/env"))
			Expect(svc.Annotations).ToNot(HaveKey("example.com/region"))
		})
	})

	Context("when podService is nil", func() {
		It("should skip reconciliation and return nil", func() {
			clusterName := "test-skip"
			cluster := newCluster(ns.Name, clusterName, nil)
			createClusterCR(cluster)
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: clusterName, Namespace: ns.Name}, cluster)).To(Succeed())

			podName := fmt.Sprintf("%s-0", clusterName)
			createPod(ns.Name, podName, clusterName)

			// Reconcile pod services with nil PodService.
			err := reconciler.reconcilePodServices(ctx, cluster)
			Expect(err).NotTo(HaveOccurred())

			// Verify no service was created.
			svcName := fmt.Sprintf("%s-pod", podName)
			svc := &corev1.Service{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: svcName, Namespace: ns.Name}, svc)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})
	})
})
