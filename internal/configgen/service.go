package configgen

import "strings"

// generateServiceSection generates the service stanza from a config map.
func generateServiceSection(serviceConfig map[string]any) string {
	var b strings.Builder
	b.WriteString("service {\n")
	writeMapEntries(&b, serviceConfig, 1)
	b.WriteString("}\n")
	return b.String()
}
