package utils

// DeepMerge merges override into base and returns the result.
// Override values take precedence. Nested maps are merged recursively.
// Neither base nor override is modified; the result is a new map.
func DeepMerge(base, override map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(base))

	for k, v := range base {
		result[k] = v
	}

	for k, overrideVal := range override {
		baseVal, exists := result[k]
		if !exists {
			result[k] = overrideVal
			continue
		}

		baseMap, baseIsMap := baseVal.(map[string]interface{})
		overrideMap, overrideIsMap := overrideVal.(map[string]interface{})

		if baseIsMap && overrideIsMap {
			result[k] = DeepMerge(baseMap, overrideMap)
		} else {
			result[k] = overrideVal
		}
	}

	return result
}
