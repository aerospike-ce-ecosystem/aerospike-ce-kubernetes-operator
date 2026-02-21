package configgen

import (
	"fmt"
	"strings"
)

// generateNetworkSection generates the network stanza with mesh seeds injected
// for all pods in the StatefulSet.
func generateNetworkSection(
	networkConfig map[string]any,
	_, serviceName, namespace string,
	podNames []string,
	heartbeatPort int,
) string {
	var b strings.Builder
	b.WriteString("network {\n")

	keys := sortedKeys(networkConfig)
	for _, key := range keys {
		val := networkConfig[key]

		if key == "heartbeat" {
			hbMap, ok := val.(map[string]any)
			if !ok {
				hbMap = make(map[string]any)
			}
			b.WriteString(generateHeartbeatSubsection(hbMap, serviceName, namespace, podNames, heartbeatPort))
		} else if subMap, ok := val.(map[string]any); ok {
			b.WriteString("\t")
			b.WriteString(key)
			b.WriteString(" {\n")
			writeMapEntries(&b, subMap, 2)
			b.WriteString("\t}\n")
		} else {
			b.WriteString("\t")
			b.WriteString(key)
			b.WriteString(" ")
			b.WriteString(formatValue(val))
			b.WriteString("\n")
		}
	}

	b.WriteString("}\n")
	return b.String()
}

// generateHeartbeatSubsection generates the heartbeat sub-stanza with mesh seed entries.
func generateHeartbeatSubsection(
	hbConfig map[string]any,
	serviceName, namespace string,
	podNames []string,
	heartbeatPort int,
) string {
	var b strings.Builder
	b.WriteString("\theartbeat {\n")

	// Write existing heartbeat config entries (excluding mesh-seed-address-port).
	keys := sortedKeys(hbConfig)
	for _, key := range keys {
		if key == "mesh-seed-address-port" {
			continue
		}
		val := hbConfig[key]
		if subMap, ok := val.(map[string]any); ok {
			b.WriteString("\t\t")
			b.WriteString(key)
			b.WriteString(" {\n")
			writeMapEntries(&b, subMap, 3)
			b.WriteString("\t\t}\n")
		} else {
			b.WriteString("\t\t")
			b.WriteString(key)
			b.WriteString(" ")
			b.WriteString(formatValue(val))
			b.WriteString("\n")
		}
	}

	// Inject mesh-seed-address-port for all pods.
	for _, pName := range podNames {
		fqdn := fmt.Sprintf("%s.%s.%s.svc.cluster.local", pName, serviceName, namespace)
		b.WriteString(fmt.Sprintf("\t\tmesh-seed-address-port %s %d\n", fqdn, heartbeatPort))
	}

	b.WriteString("\t}\n")
	return b.String()
}
