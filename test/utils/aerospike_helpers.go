/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CreateNamespaceIfNotExists creates a Kubernetes namespace if it does not already exist.
func CreateNamespaceIfNotExists(ns string) error {
	cmd := exec.Command("kubectl", "get", "ns", ns)
	if _, err := Run(cmd); err == nil {
		return nil // already exists
	}
	cmd = exec.Command("kubectl", "create", "ns", ns)
	_, err := Run(cmd)
	return err
}

// DeleteNamespaceIfExists deletes a Kubernetes namespace if it exists.
func DeleteNamespaceIfExists(ns string) error {
	cmd := exec.Command("kubectl", "delete", "ns", ns, "--ignore-not-found", "--timeout=120s")
	_, err := Run(cmd)
	return err
}

// ApplyFromFile applies a YAML file via kubectl.
func ApplyFromFile(yamlPath string) error {
	cmd := exec.Command("kubectl", "apply", "-f", yamlPath)
	_, err := Run(cmd)
	return err
}

// ApplyFromStdin applies YAML content from stdin via kubectl.
func ApplyFromStdin(yamlContent string) error {
	cmd := exec.Command("kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(yamlContent)
	dir, _ := GetProjectDir()
	cmd.Dir = dir
	_, err := cmd.CombinedOutput()
	return err
}

// DeleteAerospikeCluster deletes an AerospikeCECluster resource.
func DeleteAerospikeCluster(name, ns string) error {
	cmd := exec.Command("kubectl", "delete", "aerospikececluster", name,
		"-n", ns, "--ignore-not-found", "--timeout=120s")
	_, err := Run(cmd)
	return err
}

// WaitForClusterPhase waits until the AerospikeCECluster reaches the specified phase.
func WaitForClusterPhase(name, ns, phase string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		cmd := exec.Command("kubectl", "get", "aerospikececluster", name,
			"-n", ns, "-o", "jsonpath={.status.phase}")
		output, err := Run(cmd)
		if err == nil && strings.TrimSpace(output) == phase {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timeout waiting for cluster %s/%s to reach phase %s", ns, name, phase)
}

// WaitForPodCount waits until the expected number of pods are Running and Ready.
func WaitForPodCount(clusterName, ns string, expected int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	selector := fmt.Sprintf("app.kubernetes.io/instance=%s", clusterName)
	for time.Now().Before(deadline) {
		cmd := exec.Command("kubectl", "get", "pods", "-l", selector,
			"-n", ns, "--field-selector=status.phase=Running",
			"-o", "jsonpath={range .items[*]}{.status.conditions[?(@.type=='Ready')].status}{' '}{end}")
		output, err := Run(cmd)
		if err == nil {
			readyCount := 0
			for status := range strings.FieldsSeq(strings.TrimSpace(output)) {
				if status == "True" {
					readyCount++
				}
			}
			if readyCount >= expected {
				return nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timeout waiting for %d ready pods in cluster %s/%s", expected, ns, clusterName)
}

// GetClusterJSON returns the full AerospikeCECluster resource as JSON.
func GetClusterJSON(name, ns string) (string, error) {
	cmd := exec.Command("kubectl", "get", "aerospikececluster", name, "-n", ns, "-o", "json")
	return Run(cmd)
}

// GetClusterStatusField reads a specific jsonpath field from the cluster status.
func GetClusterStatusField(name, ns, jsonpath string) (string, error) {
	cmd := exec.Command("kubectl", "get", "aerospikececluster", name, "-n", ns,
		"-o", fmt.Sprintf("jsonpath=%s", jsonpath))
	return Run(cmd)
}

// GetPodNames returns pod names matching the cluster label selector.
func GetPodNames(clusterName, ns string) ([]string, error) {
	selector := fmt.Sprintf("app.kubernetes.io/instance=%s", clusterName)
	cmd := exec.Command("kubectl", "get", "pods", "-l", selector, "-n", ns,
		"-o", "jsonpath={.items[*].metadata.name}")
	output, err := Run(cmd)
	if err != nil {
		return nil, err
	}
	return strings.Fields(strings.TrimSpace(output)), nil
}

// GetPodAnnotation returns a specific annotation value from a pod.
func GetPodAnnotation(podName, ns, annotation string) (string, error) {
	cmd := exec.Command("kubectl", "get", "pod", podName, "-n", ns,
		"-o", fmt.Sprintf("jsonpath={.metadata.annotations['%s']}", annotation))
	return Run(cmd)
}

// GetPodLabel returns a specific label value from a pod.
func GetPodLabel(podName, ns, label string) (string, error) {
	cmd := exec.Command("kubectl", "get", "pod", podName, "-n", ns,
		"-o", fmt.Sprintf("jsonpath={.metadata.labels['%s']}", label))
	return Run(cmd)
}

// ResourceExists checks if a Kubernetes resource exists.
func ResourceExists(kind, name, ns string) bool {
	cmd := exec.Command("kubectl", "get", kind, name, "-n", ns)
	_, err := Run(cmd)
	return err == nil
}

// GetStatefulSetNames returns StatefulSet names matching the cluster label selector.
func GetStatefulSetNames(clusterName, ns string) ([]string, error) {
	selector := fmt.Sprintf("app.kubernetes.io/instance=%s", clusterName)
	cmd := exec.Command("kubectl", "get", "statefulset", "-l", selector, "-n", ns,
		"-o", "jsonpath={.items[*].metadata.name}")
	output, err := Run(cmd)
	if err != nil {
		return nil, err
	}
	return strings.Fields(strings.TrimSpace(output)), nil
}

// GetPVCNames returns PersistentVolumeClaim names matching the cluster label selector.
func GetPVCNames(clusterName, ns string) ([]string, error) {
	selector := fmt.Sprintf("app.kubernetes.io/instance=%s", clusterName)
	cmd := exec.Command("kubectl", "get", "pvc", "-l", selector, "-n", ns,
		"-o", "jsonpath={.items[*].metadata.name}")
	output, err := Run(cmd)
	if err != nil {
		return nil, err
	}
	fields := strings.Fields(strings.TrimSpace(output))
	if len(fields) == 1 && fields[0] == "" {
		return nil, nil
	}
	return fields, nil
}

// PatchClusterSpec patches the AerospikeCECluster spec with a JSON merge patch.
func PatchClusterSpec(name, ns, patch string) error {
	cmd := exec.Command("kubectl", "patch", "aerospikececluster", name,
		"-n", ns, "--type=merge", "-p", patch)
	_, err := Run(cmd)
	return err
}

// PodStatusMap represents the per-pod status from the cluster status.
type PodStatusMap map[string]struct {
	PodIP             string `json:"podIP"`
	HostIP            string `json:"hostIP"`
	Image             string `json:"image"`
	PodPort           int32  `json:"podPort"`
	Rack              int    `json:"rack"`
	IsRunningAndReady bool   `json:"isRunningAndReady"`
	ConfigHash        string `json:"configHash"`
	PodSpecHash       string `json:"podSpecHash"`
}

// GetPodStatusMap parses the status.pods field from the cluster.
func GetPodStatusMap(name, ns string) (PodStatusMap, error) {
	cmd := exec.Command("kubectl", "get", "aerospikececluster", name, "-n", ns,
		"-o", "jsonpath={.status.pods}")
	output, err := Run(cmd)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(output) == "" {
		return nil, fmt.Errorf("empty pod status for cluster %s/%s", ns, name)
	}

	var result PodStatusMap
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return nil, fmt.Errorf("parsing pod status JSON: %w", err)
	}
	return result, nil
}
