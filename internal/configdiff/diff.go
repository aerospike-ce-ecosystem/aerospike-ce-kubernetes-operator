package configdiff

import (
	"fmt"
)

// Change represents a single configuration parameter change.
type Change struct {
	// Path is the dot-separated config path (e.g., "service.proto-fd-max").
	Path string
	// Context is the Aerospike config context (e.g., "service", "network", "namespace").
	Context string
	// Key is the parameter name within the context.
	Key string
	// OldValue is the previous value.
	OldValue any
	// NewValue is the desired value.
	NewValue any
	// Namespace is the Aerospike namespace name (for namespace-level params).
	Namespace string
}

// DiffResult contains the categorized configuration changes.
type DiffResult struct {
	// Dynamic changes that can be applied via set-config without restart.
	Dynamic []Change
	// Static changes that require a pod restart.
	Static []Change
}

// HasChanges returns true if there are any changes.
func (d *DiffResult) HasChanges() bool {
	return len(d.Dynamic) > 0 || len(d.Static) > 0
}

// HasStaticChanges returns true if any changes require a restart.
func (d *DiffResult) HasStaticChanges() bool {
	return len(d.Static) > 0
}

// Diff compares old and new Aerospike config maps and categorizes changes
// as dynamic (runtime-changeable) or static (requires restart).
func Diff(oldConfig, newConfig map[string]any) *DiffResult {
	result := &DiffResult{}
	diffSection(result, "", oldConfig, newConfig)
	return result
}

// diffSection recursively compares two config sections.
func diffSection(result *DiffResult, prefix string, oldSection, newSection map[string]any) {
	if oldSection == nil {
		oldSection = make(map[string]any)
	}
	if newSection == nil {
		newSection = make(map[string]any)
	}

	// Check for changed or added keys
	for key, newVal := range newSection {
		path := joinPath(prefix, key)
		oldVal, exists := oldSection[key]

		// Handle namespace arrays specially
		if key == "namespaces" {
			diffNamespaces(result, asSlice(oldVal), asSlice(newVal))
			continue
		}

		if !exists {
			// New key added
			classifyChange(result, path, nil, newVal, "")
			continue
		}

		// Both exist: compare
		newMap, newIsMap := newVal.(map[string]any)
		oldMap, oldIsMap := oldVal.(map[string]any)

		if newIsMap && oldIsMap {
			diffSection(result, path, oldMap, newMap)
		} else if !valuesEqual(oldVal, newVal) {
			classifyChange(result, path, oldVal, newVal, "")
		}
	}

	// Check for removed keys
	for key, oldVal := range oldSection {
		if key == "namespaces" {
			continue
		}
		if _, exists := newSection[key]; !exists {
			path := joinPath(prefix, key)
			classifyChange(result, path, oldVal, nil, "")
		}
	}
}

// diffNamespaces handles namespace-level config diff.
func diffNamespaces(result *DiffResult, oldNS, newNS []any) {
	oldByName := namespacesByName(oldNS)
	newByName := namespacesByName(newNS)

	for name, newCfg := range newByName {
		oldCfg, exists := oldByName[name]
		if !exists {
			// New namespace added — this is static (requires restart)
			result.Static = append(result.Static, Change{
				Path:    fmt.Sprintf("namespaces.%s", name),
				Context: "namespace",
				Key:     name,
				NewValue: newCfg,
			})
			continue
		}

		// Compare namespace params
		for key, newVal := range newCfg {
			if key == "name" {
				continue
			}
			oldVal, exists := oldCfg[key]
			path := fmt.Sprintf("namespace.%s", key)

			if !exists || !valuesEqual(oldVal, newVal) {
				change := Change{
					Path:      path,
					Context:   "namespace",
					Key:       key,
					OldValue:  oldVal,
					NewValue:  newVal,
					Namespace: name,
				}
				if IsDynamic(path) {
					result.Dynamic = append(result.Dynamic, change)
				} else {
					result.Static = append(result.Static, change)
				}
			}
		}

		// Check removed keys
		for key, oldVal := range oldCfg {
			if key == "name" {
				continue
			}
			if _, exists := newCfg[key]; !exists {
				path := fmt.Sprintf("namespace.%s", key)
				result.Static = append(result.Static, Change{
					Path:      path,
					Context:   "namespace",
					Key:       key,
					OldValue:  oldVal,
					Namespace: name,
				})
			}
		}
	}

	// Check removed namespaces
	for name := range oldByName {
		if _, exists := newByName[name]; !exists {
			result.Static = append(result.Static, Change{
				Path:    fmt.Sprintf("namespaces.%s", name),
				Context: "namespace",
				Key:     name,
				OldValue: oldByName[name],
			})
		}
	}
}

// classifyChange categorizes a change as dynamic or static.
func classifyChange(result *DiffResult, path string, oldVal, newVal any, namespace string) {
	change := Change{
		Path:      path,
		Key:       lastSegment(path),
		Context:   firstSegment(path),
		OldValue:  oldVal,
		NewValue:  newVal,
		Namespace: namespace,
	}

	if IsDynamic(path) {
		result.Dynamic = append(result.Dynamic, change)
	} else {
		result.Static = append(result.Static, change)
	}
}

// Helper functions

func joinPath(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

func firstSegment(path string) string {
	for i, c := range path {
		if c == '.' {
			return path[:i]
		}
	}
	return path
}

func lastSegment(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			return path[i+1:]
		}
	}
	return path
}

func valuesEqual(a, b any) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func asSlice(v any) []any {
	if v == nil {
		return nil
	}
	if s, ok := v.([]any); ok {
		return s
	}
	return nil
}

func namespacesByName(namespaces []any) map[string]map[string]any {
	result := make(map[string]map[string]any)
	for _, ns := range namespaces {
		if nsMap, ok := ns.(map[string]any); ok {
			if name, ok := nsMap["name"].(string); ok {
				result[name] = nsMap
			}
		}
	}
	return result
}
