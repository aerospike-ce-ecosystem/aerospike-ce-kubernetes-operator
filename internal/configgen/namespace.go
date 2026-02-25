package configgen

import (
	"fmt"
	"slices"
	"strings"
)

// generateNamespaceSections generates one `namespace <name> { ... }` block per namespace entry.
func generateNamespaceSections(namespaces []any) (string, error) {
	var b strings.Builder

	for i, ns := range namespaces {
		nsMap, ok := ns.(map[string]any)
		if !ok {
			return "", fmt.Errorf("namespace entry %d is not a map", i)
		}

		name, ok := nsMap["name"].(string)
		if !ok || name == "" {
			return "", fmt.Errorf("namespace entry %d missing 'name' key", i)
		}

		b.WriteString("namespace ")
		b.WriteString(name)
		b.WriteString(" {\n")

		// Write all keys except "name".
		keys := sortedKeys(nsMap)
		for _, key := range keys {
			if key == "name" {
				continue
			}
			val := nsMap[key]
			writeNamespaceEntry(&b, key, val, 1)
		}

		b.WriteString("}\n")
	}

	return b.String(), nil
}

// writeNamespaceEntry writes a single namespace config entry at the given indent level.
func writeNamespaceEntry(b *strings.Builder, key string, val any, indent int) {
	// storage-engine uses special syntax: "storage-engine <type> { ... }"
	if key == "storage-engine" {
		writeStorageEngine(b, val, indent)
		return
	}

	prefix := strings.Repeat("\t", indent)

	switch v := val.(type) {
	case map[string]any:
		b.WriteString(prefix)
		b.WriteString(key)
		b.WriteString(" {\n")
		keys := sortedKeys(v)
		for _, k := range keys {
			writeNamespaceEntry(b, k, v[k], indent+1)
		}
		b.WriteString(prefix)
		b.WriteString("}\n")
	case []any:
		for _, item := range v {
			if subMap, ok := item.(map[string]any); ok {
				b.WriteString(prefix)
				b.WriteString(key)
				b.WriteString(" {\n")
				keys := sortedKeys(subMap)
				for _, k := range keys {
					writeNamespaceEntry(b, k, subMap[k], indent+1)
				}
				b.WriteString(prefix)
				b.WriteString("}\n")
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

// writeStorageEngine handles the special aerospike.conf syntax for storage-engine.
// Aerospike expects: "storage-engine memory" or "storage-engine device { file ... }"
// NOT: "storage-engine { type memory ... }"
func writeStorageEngine(b *strings.Builder, val any, indent int) {
	prefix := strings.Repeat("\t", indent)

	m, ok := val.(map[string]any)
	if !ok {
		// Simple value like "storage-engine memory"
		b.WriteString(prefix)
		b.WriteString("storage-engine ")
		b.WriteString(formatValue(val))
		b.WriteString("\n")
		return
	}

	seType := inferStorageEngineType(m)

	// Collect remaining keys (excluding "type").
	var remainingKeys []string
	for k := range m {
		if k != "type" {
			remainingKeys = append(remainingKeys, k)
		}
	}
	slices.Sort(remainingKeys)

	if len(remainingKeys) == 0 {
		// No additional settings: "storage-engine memory"
		b.WriteString(prefix)
		b.WriteString("storage-engine ")
		b.WriteString(seType)
		b.WriteString("\n")
	} else {
		// With settings: "storage-engine device { ... }"
		b.WriteString(prefix)
		b.WriteString("storage-engine ")
		b.WriteString(seType)
		b.WriteString(" {\n")
		for _, k := range remainingKeys {
			writeNamespaceEntry(b, k, m[k], indent+1)
		}
		b.WriteString(prefix)
		b.WriteString("}\n")
	}
}

// inferStorageEngineType determines the storage-engine type from the config map.
func inferStorageEngineType(m map[string]any) string {
	if t, ok := m["type"].(string); ok {
		return t
	}
	// Infer from context: presence of "file" or "device" keys implies device storage.
	for k := range m {
		if k == "file" || k == "device" {
			return "device"
		}
	}
	return "memory"
}
