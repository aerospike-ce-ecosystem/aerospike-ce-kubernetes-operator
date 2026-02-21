package utils

import "maps"

// DeepMerge merges override into base and returns the result.
// Override values take precedence. Nested maps are merged recursively.
// Neither base nor override is modified; the result is a new map.
func DeepMerge(base, override map[string]any) map[string]any {
	result := make(map[string]any, len(base))
	maps.Copy(result, base)

	for k, overrideVal := range override {
		baseVal, exists := result[k]
		if !exists {
			result[k] = overrideVal
			continue
		}

		baseMap, baseIsMap := baseVal.(map[string]any)
		overrideMap, overrideIsMap := overrideVal.(map[string]any)

		if baseIsMap && overrideIsMap {
			result[k] = DeepMerge(baseMap, overrideMap)
		} else {
			result[k] = overrideVal
		}
	}

	return result
}
