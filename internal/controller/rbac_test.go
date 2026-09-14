//go:build integration

package controller

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/aerospike-ce-ecosystem/aerospike-ce-kubernetes-operator/internal/utils"
)

// --- the generated manager ClusterRole must admit every write the operator makes ---
//
// The reconciler stamps annotations onto Pods with r.Patch(...) on the main
// resource (updatePodConfigHash and adoptStorageHash in reconciler_restart.go),
// which the apiserver authorizes as verb=patch resource=pods — the pods/status
// grant used by the readiness gate does not cover it. The pods rule only listed
// get/list/watch/delete, so those two writes returned 403 in every real install
// (warm restart never stamped the new hash and looped; the storage-hash adoption
// for pre-upgrade pods never happened).
//
// envtest runs kube-apiserver with --authorization-mode=RBAC and the test client
// is a cluster admin, so unit tests could never see this. These specs load the
// real config/rbac/role.yaml, bind it to a throwaway user, and exercise the
// operator's pod writes through an impersonated client.

const rbacTestUser = "acko-rbac-test-manager"

// loadManagerClusterRole parses config/rbac/role.yaml (the file `make manifests`
// generates from the kubebuilder markers) into a ClusterRole.
func loadManagerClusterRole(name string) *rbacv1.ClusterRole {
	GinkgoHelper()

	raw, err := os.ReadFile(filepath.Join("..", "..", "config", "rbac", "role.yaml"))
	Expect(err).NotTo(HaveOccurred())

	role := &rbacv1.ClusterRole{}
	Expect(yaml.Unmarshal(raw, role)).To(Succeed())
	Expect(role.Rules).NotTo(BeEmpty())

	role.ObjectMeta = metav1.ObjectMeta{Name: name}
	return role
}

var _ = Describe("Manager ClusterRole", func() {
	var (
		managerClient client.Client
		pod           *corev1.Pod
	)

	BeforeEach(func() {
		role := loadManagerClusterRole(rbacTestUser + "-role")
		Expect(k8sClient.Create(ctx, role)).To(Succeed())
		DeferCleanup(func() {
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, role))).To(Succeed())
		})

		binding := &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: rbacTestUser + "-binding"},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     role.Name,
			},
			Subjects: []rbacv1.Subject{{
				APIGroup: rbacv1.GroupName,
				Kind:     rbacv1.UserKind,
				Name:     rbacTestUser,
			}},
		}
		Expect(k8sClient.Create(ctx, binding)).To(Succeed())
		DeferCleanup(func() {
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, binding))).To(Succeed())
		})

		impersonated := rest.CopyConfig(cfg)
		impersonated.Impersonate = rest.ImpersonationConfig{UserName: rbacTestUser}

		var err error
		managerClient, err = client.New(impersonated, client.Options{Scheme: scheme.Scheme})
		Expect(err).NotTo(HaveOccurred())

		pod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "acko-rbac-test-0",
				Namespace: "default",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:  "aerospike",
					Image: "aerospike:ce-8.1.1.1",
				}},
			},
		}
		Expect(k8sClient.Create(ctx, pod)).To(Succeed())
		DeferCleanup(func() {
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, pod))).To(Succeed())
		})

		// The RBAC authorizer is informer-backed, so the new binding takes a
		// moment to become visible to the apiserver.
		Eventually(func() error {
			return managerClient.Get(ctx, client.ObjectKeyFromObject(pod), &corev1.Pod{})
		}, 30*time.Second, 200*time.Millisecond).Should(Succeed())
	})

	It("admits the pod annotation patches the reconciler issues", func() {
		// updatePodConfigHash / adoptStorageHash: a merge patch on the pod
		// itself, authorized as verb=patch resource=pods.
		current := &corev1.Pod{}
		Expect(managerClient.Get(ctx, client.ObjectKeyFromObject(pod), current)).To(Succeed())

		podCopy := current.DeepCopy()
		if podCopy.Annotations == nil {
			podCopy.Annotations = map[string]string{}
		}
		podCopy.Annotations[utils.ConfigHashAnnotation] = "config-hash"
		podCopy.Annotations[utils.StorageHashAnnotation] = "storage-hash"
		Expect(managerClient.Patch(ctx, podCopy, client.MergeFrom(current))).To(Succeed())

		updated := &corev1.Pod{}
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(pod), updated)).To(Succeed())
		Expect(updated.Annotations).To(HaveKeyWithValue(utils.ConfigHashAnnotation, "config-hash"))
		Expect(updated.Annotations).To(HaveKeyWithValue(utils.StorageHashAnnotation, "storage-hash"))
	})

	It("admits the pod status patch the readiness gate issues", func() {
		current := &corev1.Pod{}
		Expect(managerClient.Get(ctx, client.ObjectKeyFromObject(pod), current)).To(Succeed())

		podCopy := current.DeepCopy()
		podCopy.Status.Conditions = append(podCopy.Status.Conditions, corev1.PodCondition{
			Type:   "acko.io/rbac-test",
			Status: corev1.ConditionTrue,
		})
		Expect(managerClient.Status().Patch(ctx, podCopy, client.MergeFrom(current))).To(Succeed())
	})

	It("admits the pod delete the cold restart path issues", func() {
		Expect(managerClient.Delete(ctx, pod)).To(Succeed())
	})
})
