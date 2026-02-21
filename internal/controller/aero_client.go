package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	aero "github.com/aerospike/aerospike-client-go/v8"

	asdbcev1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/utils"
)

// getAerospikeClient creates an Aerospike client connected to the cluster.
//
//nolint:unused // placeholder for future ACL integration
func (r *AerospikeCEClusterReconciler) getAerospikeClient(
	ctx context.Context,
	cluster *asdbcev1alpha1.AerospikeCECluster,
) (*aero.Client, error) {
	log := logf.FromContext(ctx)

	serviceName := utils.HeadlessServiceName(cluster.Name)
	host := fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, cluster.Namespace)

	policy := aero.NewClientPolicy()
	policy.Timeout = 0 // Use default timeout

	// If ACL is enabled, set admin credentials
	if cluster.Spec.AerospikeAccessControl != nil {
		for _, user := range cluster.Spec.AerospikeAccessControl.Users {
			if user.Name == "admin" {
				password, err := r.getPasswordFromSecret(ctx, cluster.Namespace, user.SecretName)
				if err != nil {
					return nil, fmt.Errorf("getting admin password: %w", err)
				}
				policy.User = "admin"
				policy.Password = password
				break
			}
		}
	}

	log.Info("Connecting to Aerospike cluster", "host", host)
	client, err := aero.NewClientWithPolicy(policy, host, 3000)
	if err != nil {
		return nil, fmt.Errorf("connecting to Aerospike: %w", err)
	}

	return client, nil
}

// getPasswordFromSecret reads a password from a Kubernetes Secret.
//
//nolint:unused // placeholder for future ACL integration
func (r *AerospikeCEClusterReconciler) getPasswordFromSecret(
	ctx context.Context,
	namespace, secretName string,
) (string, error) {
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, secret); err != nil {
		return "", fmt.Errorf("getting secret %s: %w", secretName, err)
	}

	password, ok := secret.Data["password"]
	if !ok {
		return "", fmt.Errorf("secret %s does not have 'password' key", secretName)
	}

	return string(password), nil
}
