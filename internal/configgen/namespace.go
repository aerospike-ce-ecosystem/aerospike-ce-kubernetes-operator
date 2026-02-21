package configgen

import (
	"fmt"
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
