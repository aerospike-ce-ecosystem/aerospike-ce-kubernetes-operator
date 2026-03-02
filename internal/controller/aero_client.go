package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	aero "github.com/aerospike/aerospike-client-go/v8"

	ackov1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/utils"
)

const (
	aeroClientTimeout = 30 * time.Second
	aeroLoginTimeout  = 10 * time.Second
	aeroInfoTimeout   = 10 * time.Second
	defaultAeroPort   = 3000
)

// getServicePort returns the configured Aerospike service port from the cluster
// config, falling back to the default port.
func getServicePort(cluster *ackov1alpha1.AerospikeCluster) int {
	if cluster.Spec.AerospikeConfig != nil {
		if netCfg, ok := cluster.Spec.AerospikeConfig.Value["network"].(map[string]any); ok {
			if svcCfg, ok := netCfg["service"].(map[string]any); ok {
				if port, ok := svcCfg["port"]; ok {
					return utils.IntFromAny(port, defaultAeroPort)
				}
			}
		}
	}
	return defaultAeroPort
}

// getAerospikeClient creates an Aerospike client connected to the cluster.
func (r *AerospikeClusterReconciler) getAerospikeClient(
	ctx context.Context,
	cluster *ackov1alpha1.AerospikeCluster,
) (*aero.Client, error) {
	log := logf.FromContext(ctx)

	serviceName := utils.HeadlessServiceName(cluster.Name)
	host := fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, cluster.Namespace)
	port := getServicePort(cluster)

	policy := aero.NewClientPolicy()
	policy.Timeout = aeroClientTimeout
	policy.LoginTimeout = aeroLoginTimeout

	// If ACL is enabled, find the admin user dynamically by roles.
	if adminUser := utils.FindAdminUser(cluster.Spec.AerospikeAccessControl); adminUser != nil {
		password, err := r.getPasswordFromSecret(ctx, cluster.Namespace, adminUser.SecretName)
		if err != nil {
			return nil, fmt.Errorf("getting admin password: %w", err)
		}
		policy.User = adminUser.Name
		policy.Password = password
	}

	log.Info("Connecting to Aerospike cluster", "host", host, "port", port)
	client, err := aero.NewClientWithPolicy(policy, host, port)
	if err != nil {
		return nil, fmt.Errorf("connecting to Aerospike: %w", err)
	}

	return client, nil
}

// closeAerospikeClient safely closes an Aerospike client.
func closeAerospikeClient(client *aero.Client) {
	if client != nil {
		client.Close()
	}
}

// getPasswordFromSecret reads a password from a Kubernetes Secret.
func (r *AerospikeClusterReconciler) getPasswordFromSecret(
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
