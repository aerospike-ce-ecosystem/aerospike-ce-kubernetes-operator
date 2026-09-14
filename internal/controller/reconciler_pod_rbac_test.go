package controller

import (
	"context"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	ackov1alpha1 "github.com/aerospike-ce-ecosystem/aerospike-ce-kubernetes-operator/api/v1alpha1"
	"github.com/aerospike-ce-ecosystem/aerospike-ce-kubernetes-operator/internal/podutil"
)

func podRBACTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme(client-go) error = %v", err)
	}
	if err := ackov1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme(acko) error = %v", err)
	}
	return scheme
}

func podRBACTestCluster(serviceAccountName string) *ackov1alpha1.AerospikeCluster {
	cluster := &ackov1alpha1.AerospikeCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "aerospike",
		},
		Spec: ackov1alpha1.AerospikeClusterSpec{
			PodService: &ackov1alpha1.AerospikeServiceSpec{
				ServiceType: string(corev1.ServiceTypeLoadBalancer),
			},
		},
	}
	if serviceAccountName != "" {
		cluster.Spec.PodSpec = &ackov1alpha1.AerospikePodSpec{
			ServiceAccountName: serviceAccountName,
		}
	}
	return cluster
}

// TestReconcilePodServiceRBACBindsConfiguredServiceAccount asserts that the
// reader RoleBinding follows spec.podSpec.serviceAccountName, so the init
// container can actually GET its per-pod Service.
func TestReconcilePodServiceRBACBindsConfiguredServiceAccount(t *testing.T) {
	tests := []struct {
		name               string
		serviceAccountName string
		wantSubjectName    string
	}{
		{
			name:               "custom service account",
			serviceAccountName: "aerospike-sa",
			wantSubjectName:    "aerospike-sa",
		},
		{
			name:               "unset service account falls back to default",
			serviceAccountName: "",
			wantSubjectName:    "default",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			scheme := podRBACTestScheme(t)
			cluster := podRBACTestCluster(tc.serviceAccountName)

			reconciler := &AerospikeClusterReconciler{
				Client: fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(cluster).
					Build(),
				Scheme:   scheme,
				Recorder: record.NewFakeRecorder(8),
			}

			if err := reconciler.reconcilePodServiceRBAC(context.Background(), cluster); err != nil {
				t.Fatalf("reconcilePodServiceRBAC() error = %v", err)
			}

			bindingName := fmt.Sprintf("%s-%s", cluster.Name, podServiceReaderRole)
			binding := &rbacv1.RoleBinding{}
			if err := reconciler.Get(context.Background(),
				types.NamespacedName{Name: bindingName, Namespace: cluster.Namespace}, binding); err != nil {
				t.Fatalf("Get(RoleBinding) error = %v", err)
			}

			if len(binding.Subjects) != 1 {
				t.Fatalf("Subjects = %v, want exactly one", binding.Subjects)
			}
			subject := binding.Subjects[0]
			if subject.Kind != "ServiceAccount" {
				t.Fatalf("Subject.Kind = %q, want %q", subject.Kind, "ServiceAccount")
			}
			if subject.Name != tc.wantSubjectName {
				t.Fatalf("Subject.Name = %q, want %q", subject.Name, tc.wantSubjectName)
			}
			if subject.Namespace != cluster.Namespace {
				t.Fatalf("Subject.Namespace = %q, want %q", subject.Namespace, cluster.Namespace)
			}
		})
	}
}

// TestReconcilePodServiceRBACRebindsOnServiceAccountChange asserts that an
// existing binding pointing at the old service account is updated in place.
func TestReconcilePodServiceRBACRebindsOnServiceAccountChange(t *testing.T) {
	scheme := podRBACTestScheme(t)
	cluster := podRBACTestCluster("aerospike-sa")
	bindingName := fmt.Sprintf("%s-%s", cluster.Name, podServiceReaderRole)

	stale := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bindingName,
			Namespace: cluster.Namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     bindingName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "default",
				Namespace: cluster.Namespace,
			},
		},
	}

	reconciler := &AerospikeClusterReconciler{
		Client: fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(cluster, stale).
			Build(),
		Scheme:   scheme,
		Recorder: record.NewFakeRecorder(8),
	}

	if err := reconciler.reconcilePodServiceRBAC(context.Background(), cluster); err != nil {
		t.Fatalf("reconcilePodServiceRBAC() error = %v", err)
	}

	updated := &rbacv1.RoleBinding{}
	if err := reconciler.Get(context.Background(),
		types.NamespacedName{Name: bindingName, Namespace: cluster.Namespace}, updated); err != nil {
		t.Fatalf("Get(RoleBinding) error = %v", err)
	}

	if len(updated.Subjects) != 1 || updated.Subjects[0].Name != "aerospike-sa" {
		t.Fatalf("Subjects = %v, want a single subject named %q", updated.Subjects, "aerospike-sa")
	}
	if updated.UID != stale.UID {
		t.Fatalf("RoleBinding was recreated (UID %q -> %q), want an in-place Subjects update", stale.UID, updated.UID)
	}
}

// TestReconcilePodServiceRBACSubjectMatchesPodSpec guards against the binding
// subject and the pod's actual service account drifting apart.
func TestReconcilePodServiceRBACSubjectMatchesPodSpec(t *testing.T) {
	scheme := podRBACTestScheme(t)
	cluster := podRBACTestCluster("aerospike-sa")

	reconciler := &AerospikeClusterReconciler{
		Client: fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(cluster).
			Build(),
		Scheme:   scheme,
		Recorder: record.NewFakeRecorder(8),
	}

	if err := reconciler.reconcilePodServiceRBAC(context.Background(), cluster); err != nil {
		t.Fatalf("reconcilePodServiceRBAC() error = %v", err)
	}

	bindingName := fmt.Sprintf("%s-%s", cluster.Name, podServiceReaderRole)
	binding := &rbacv1.RoleBinding{}
	if err := reconciler.Get(context.Background(),
		types.NamespacedName{Name: bindingName, Namespace: cluster.Namespace}, binding); err != nil {
		t.Fatalf("Get(RoleBinding) error = %v", err)
	}

	template := podutil.BuildPodTemplateSpec(cluster, nil, 0, "demo-0-config", "")
	podSA := template.Spec.ServiceAccountName
	if podSA == "" {
		podSA = "default"
	}

	if len(binding.Subjects) != 1 || binding.Subjects[0].Name != podSA {
		t.Fatalf("RoleBinding subjects = %v, but the pod runs as service account %q",
			binding.Subjects, podSA)
	}
}
