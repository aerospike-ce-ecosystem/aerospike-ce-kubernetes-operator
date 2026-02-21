package controller

import (
	"bytes"
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	asdbcev1alpha1 "github.com/ksr/aerospike-ce-kubernetes-operator/api/v1alpha1"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/podutil"
	"github.com/ksr/aerospike-ce-kubernetes-operator/internal/utils"
)

const (
	warmRestartReadyTimeout = 120 * time.Second
	warmRestartPollInterval = 5 * time.Second
)

// shouldWarmRestart decides whether a pod can be warm-restarted.
// Warm restart is possible when:
// - Only config changed (same image, same pod template spec)
// - The pod is currently running and ready
// - RestConfig is available (for exec API)
func (r *AerospikeCEClusterReconciler) shouldWarmRestart(
	cluster *asdbcev1alpha1.AerospikeCECluster,
	pod *corev1.Pod,
	sts *appsv1.StatefulSet,
) bool {
	if r.RestConfig == nil {
		return false
	}

	if !isPodReady(pod) {
		return false
	}

	// Check if the image changed — if so, cold restart is required
	desiredImage := cluster.Spec.Image
	for _, c := range pod.Spec.Containers {
		if c.Name == podutil.AerospikeContainerName {
			if c.Image != desiredImage {
				return false
			}
			break
		}
	}

	// Check if pod spec hash changed (non-config change) — if so, cold restart needed
	currentPodSpecHash := ""
	if pod.Annotations != nil {
		currentPodSpecHash = pod.Annotations[utils.PodSpecHashAnnotation]
	}
	desiredPodSpecHash := ""
	if sts.Spec.Template.Annotations != nil {
		desiredPodSpecHash = sts.Spec.Template.Annotations[utils.PodSpecHashAnnotation]
	}
	if currentPodSpecHash != desiredPodSpecHash && desiredPodSpecHash != "" {
		return false
	}

	return true
}

// warmRestartPod sends SIGUSR1 to PID 1 in the Aerospike container to trigger
// a warm restart, then waits for the pod to become ready again.
func (r *AerospikeCEClusterReconciler) warmRestartPod(ctx context.Context, pod *corev1.Pod) error {
	log := logf.FromContext(ctx)

	if r.RestConfig == nil {
		return fmt.Errorf("RestConfig not available for exec")
	}

	clientset, err := kubernetes.NewForConfig(r.RestConfig)
	if err != nil {
		return fmt.Errorf("creating kubernetes clientset: %w", err)
	}

	// Execute "kill -USR1 1" in the aerospike container
	req := clientset.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Namespace(pod.Namespace).
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: podutil.AerospikeContainerName,
			Command:   []string{"kill", "-USR1", "1"},
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(r.RestConfig, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("creating SPDY executor: %w", err)
	}

	var stdout, stderr bytes.Buffer
	if err := exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	}); err != nil {
		return fmt.Errorf("executing SIGUSR1: %w (stderr: %s)", err, stderr.String())
	}

	log.Info("SIGUSR1 sent, waiting for pod readiness", "pod", pod.Name)

	// Wait for the pod to become ready again
	return r.waitForPodReady(ctx, pod.Namespace, pod.Name)
}

// waitForPodReady polls the pod until it becomes ready or times out.
func (r *AerospikeCEClusterReconciler) waitForPodReady(ctx context.Context, namespace, name string) error {
	deadline := time.Now().Add(warmRestartReadyTimeout)

	for time.Now().Before(deadline) {
		pod := &corev1.Pod{}
		if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, pod); err != nil {
			return fmt.Errorf("getting pod %s: %w", name, err)
		}

		if isPodReady(pod) {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(warmRestartPollInterval):
		}
	}

	return fmt.Errorf("pod %s did not become ready within %v after warm restart", name, warmRestartReadyTimeout)
}
