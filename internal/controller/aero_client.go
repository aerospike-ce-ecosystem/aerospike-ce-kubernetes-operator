package controller

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	aero "github.com/aerospike/aerospike-client-go/v8"

	ackov1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/podutil"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/utils"
)

const (
	aeroClientTimeout = 30 * time.Second
	aeroLoginTimeout  = 10 * time.Second
	aeroInfoTimeout   = 10 * time.Second
	defaultAeroPort   = 3000

	// quiesceTimeout is the maximum time to wait for a quiesce command to complete.
	quiesceTimeout = 15 * time.Second
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

// quiesceNode sends the "quiesce:" command to an Aerospike node via pod exec,
// so the node gracefully stops accepting new transactions before being removed.
// The command is executed inside the Aerospike container using asinfo.
// This is a best-effort operation: the caller should proceed with pod deletion
// even if quiesce fails (e.g., node is already down, asinfo not available).
func (r *AerospikeClusterReconciler) quiesceNode(
	ctx context.Context,
	pod *corev1.Pod,
	cluster *ackov1alpha1.AerospikeCluster,
) error {
	log := logf.FromContext(ctx)

	if !isPodReady(pod) {
		log.V(1).Info("Pod not ready, skipping quiesce", "pod", pod.Name)
		return nil
	}

	if r.RestConfig == nil {
		return fmt.Errorf("RestConfig not available for exec")
	}

	clientset, err := r.getOrCreateKubeClientset()
	if err != nil {
		return fmt.Errorf("creating kubernetes clientset for quiesce: %w", err)
	}

	port := getServicePort(cluster)

	// Build the asinfo command. If ACL is enabled, include auth flags.
	cmd := buildQuiesceCommand(cluster, port)

	// Apply quiesce-specific timeout to prevent blocking pod deletion indefinitely.
	execCtx, cancel := context.WithTimeout(ctx, quiesceTimeout)
	defer cancel()

	req := clientset.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Namespace(pod.Namespace).
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: podutil.AerospikeContainerName,
			Command:   cmd,
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(r.RestConfig, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("creating SPDY executor for quiesce: %w", err)
	}

	var stdout, stderr bytes.Buffer
	if err := exec.StreamWithContext(execCtx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	}); err != nil {
		return fmt.Errorf("executing quiesce command: %w (stderr: %s)", err, stderr.String())
	}

	output := strings.TrimSpace(stdout.String())
	log.Info("Quiesce command completed", "pod", pod.Name, "output", output)

	// Aerospike returns "ok" on success. Any other response indicates a problem.
	if output != "ok" && output != "" {
		return fmt.Errorf("quiesce returned unexpected response: %s", output)
	}

	return nil
}

// buildQuiesceCommand constructs the asinfo command to quiesce a node.
// If ACL is enabled, authentication flags are included.
func buildQuiesceCommand(cluster *ackov1alpha1.AerospikeCluster, port int) []string {
	cmd := []string{
		"asinfo",
		"-v", "quiesce:",
		"-h", "localhost",
		"-p", fmt.Sprintf("%d", port),
	}

	// Add authentication flags if ACL is enabled.
	// We use the admin user name only; the password is read from the node's
	// local security context since asinfo running inside the pod can use
	// file-based auth or the default credentials.
	if adminUser := utils.FindAdminUser(cluster.Spec.AerospikeAccessControl); adminUser != nil {
		cmd = append(cmd, "-U", adminUser.Name)
	}

	return cmd
}
