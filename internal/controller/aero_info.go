package controller

import (
	"fmt"
	"strconv"
	"strings"

	aero "github.com/aerospike/aerospike-client-go/v8"
)

// AsinfoCommand executes an asinfo command on a specific node.
func AsinfoCommand(client *aero.Client, cmd string) (string, error) {
	nodes := client.GetNodes()
	if len(nodes) == 0 {
		return "", fmt.Errorf("no nodes available")
	}

	policy := aero.NewInfoPolicy()
	policy.Timeout = aeroInfoTimeout

	result, err := nodes[0].RequestInfo(policy, cmd)
	if err != nil {
		return "", fmt.Errorf("asinfo command %q failed: %w", cmd, err)
	}

	if val, ok := result[cmd]; ok {
		return val, nil
	}

	return "", fmt.Errorf("no result for command %q", cmd)
}

// asinfoCommandOnNode executes an asinfo command on a specific node by address.
func asinfoCommandOnNode(node *aero.Node, cmd string) (string, error) {
	policy := aero.NewInfoPolicy()
	policy.Timeout = aeroInfoTimeout

	result, err := node.RequestInfo(policy, cmd)
	if err != nil {
		return "", fmt.Errorf("asinfo command %q on node %s failed: %w", cmd, node.GetName(), err)
	}

	if val, ok := result[cmd]; ok {
		return val, nil
	}

	return "", fmt.Errorf("no result for command %q on node %s", cmd, node.GetName())
}

// IsMigrating checks if the cluster has any pending migrations.
func IsMigrating(client *aero.Client) (bool, error) {
	result, err := AsinfoCommand(client, "cluster-stable:")
	if err != nil {
		return true, err
	}

	// If cluster-stable returns a cluster key, the cluster is stable (no migrations).
	// If it returns an error or empty, migrations may be in progress.
	return strings.TrimSpace(result) == "", nil
}

// Recluster sends the recluster command to the cluster.
func Recluster(client *aero.Client) error {
	_, err := AsinfoCommand(client, "recluster:")
	return err
}

// ClusterSize returns the number of nodes in the Aerospike cluster as reported by asinfo.
// Returns 0 and an error if the cluster is unreachable or the response cannot be parsed.
func ClusterSize(client *aero.Client) (int, error) {
	result, err := AsinfoCommand(client, "cluster-size")
	if err != nil {
		return 0, err
	}
	size, err := strconv.Atoi(strings.TrimSpace(result))
	if err != nil {
		return 0, fmt.Errorf("parsing cluster-size response %q: %w", result, err)
	}
	return size, nil
}
