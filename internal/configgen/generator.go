package configgen

import (
	"fmt"
	"slices"
	"strings"
)

// Aerospike configuration section keys used throughout config generation.
const (
	SectionNamespaces      = "namespaces"
	SectionLogging         = "logging"
	SectionSecurity        = "security"
	SectionService         = "service"
	SectionNetwork         = "network"
	SectionHeartbeat       = "heartbeat"
	KeyMeshSeedAddressPort = "mesh-seed-address-port"
)

type networkWriter func(netMap map[string]any) string

func generateConfigCore(config map[string]any, writeNetwork networkWriter) (string, error) {
	if config == nil {
		return "", fmt.Errorf("config map is nil")
	}

	var b strings.Builder
	keys := sortedKeys(config)

	for _, key := range keys {
		val := config[key]
		if val == nil {
			continue
		}

		switch key {
		case SectionNamespaces:
			namespaces, ok := val.([]any)
			if !ok {
				return "", fmt.Errorf("namespaces must be a list")
			}
			s, err := generateNamespaceSections(namespaces)
			if err != nil {
				return "", err
			}
			b.WriteString(s)

		case SectionLogging:
			logs, ok := val.([]any)
			if !ok {
				return "", fmt.Errorf("logging must be a list")
			}
			b.WriteString(generateLoggingSection(logs))

		case SectionSecurity:
			continue

		case SectionService:
			svcMap, ok := val.(map[string]any)
			if !ok {
				return "", fmt.Errorf("service must be a map")
			}
			b.WriteString(generateServiceSection(svcMap))

		case SectionNetwork:
			netMap, ok := val.(map[string]any)
			if !ok {
				return "", fmt.Errorf("network must be a map")
			}
			b.WriteString(writeNetwork(netMap))

		default:
			switch v := val.(type) {
			case map[string]any:
				b.WriteString(generateStanza(key, v))
			default:
				b.WriteString(fmt.Sprintf("%s %s\n", key, formatValue(val)))
			}
		}
	}

	return b.String(), nil
}

// GenerateConfig converts an unstructured config map into aerospike.conf text format.
func GenerateConfig(config map[string]any) (string, error) {
	return generateConfigCore(config, func(netMap map[string]any) string {
		return generateStanza(SectionNetwork, netMap)
	})
}

// GenerateConfForPod generates an aerospike.conf with mesh seeds injected for the given pod.
func GenerateConfForPod(
	config map[string]any,
	serviceName, namespace string,
	podNames []string,
	heartbeatPort int,
) (string, error) {
	return generateConfigCore(config, func(netMap map[string]any) string {
		return generateNetworkSection(netMap, serviceName, namespace, podNames, heartbeatPort)
	})
}

// generateStanza outputs a named stanza block with its key-value pairs.
func generateStanza(name string, m map[string]any) string {
	var b strings.Builder
	writeBlock(&b, "", name, m, 0)
	return b.String()
}

// writeBlock writes a named `key { ... }` block with nested map entries.
func writeBlock(b *strings.Builder, prefix, key string, inner map[string]any, indent int) {
	b.WriteString(prefix)
	b.WriteString(key)
	b.WriteString(" {\n")
	writeMapEntries(b, inner, indent+1)
	b.WriteString(prefix)
	b.WriteString("}\n")
}

// writeMapEntries writes sorted map entries with the given indentation level.
func writeMapEntries(b *strings.Builder, m map[string]any, indent int) {
	prefix := strings.Repeat("\t", indent)
	keys := sortedKeys(m)

	for _, key := range keys {
		val := m[key]
		if val == nil {
			continue
		}
		switch v := val.(type) {
		case map[string]any:
			writeBlock(b, prefix, key, v, indent)
		case []any:
			// Lists of maps become repeated sub-stanzas; lists of primitives become repeated key-value lines.
			for _, item := range v {
				if subMap, ok := item.(map[string]any); ok {
					writeBlock(b, prefix, key, subMap, indent)
				} else {
					b.WriteString(prefix)
					b.WriteString(key)
					b.WriteString(" ")
					b.WriteString(formatValue(item))
					b.WriteString("\n")
				}
			}
		default:
			b.WriteString(prefix)
			b.WriteString(key)
			b.WriteString(" ")
			b.WriteString(formatValue(val))
			b.WriteString("\n")
		}
	}
}

// generateLoggingSection produces the logging stanza from a list of log sink configs.
// Supports three sink types via the "name" key:
//   - "console" or "stderr" -- generates a `console { ... }` block
//   - "syslog" -- generates a `syslog { ... }` block
//   - anything else -- treated as a file path, generating `file <name> { ... }`
func generateLoggingSection(logs []any) string {
	var b strings.Builder
	b.WriteString("logging {\n")

	for _, entry := range logs {
		logMap, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		name, _ := logMap["name"].(string)
		if name == "" {
			continue
		}

		// Determine sink type from the name field.
		switch name {
		case "console", "stderr":
			b.WriteString("\tconsole {\n")
		case "syslog":
			b.WriteString("\tsyslog {\n")
		default:
			b.WriteString("\tfile ")
			b.WriteString(name)
			b.WriteString(" {\n")
		}

		// Write context entries (all keys except "name").
		keys := sortedKeys(logMap)
		for _, k := range keys {
			if k == "name" {
				continue
			}
			b.WriteString("\t\t")
			b.WriteString(k)
			b.WriteString(" ")
			b.WriteString(formatValue(logMap[k]))
			b.WriteString("\n")
		}
		b.WriteString("\t}\n")
	}

	b.WriteString("}\n")
	return b.String()
}

// formatValue formats a config value for aerospike.conf output.
func formatValue(val any) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int:
		return fmt.Sprintf("%d", v)
	case int32:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		// If the float is a whole number, output without decimal.
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)
	case float32:
		f64 := float64(v)
		if f64 == float64(int64(f64)) {
			return fmt.Sprintf("%d", int64(f64))
		}
		return fmt.Sprintf("%g", f64)
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

// sortedKeys returns the keys of a map in sorted order.
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
