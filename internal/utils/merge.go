package utils

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"maps"
)

// IntFromAny extracts an int from a value that may be int, int64, or float64.
// Returns the fallback value if conversion is not possible.
func IntFromAny(v any, fallback int) int {
	switch p := v.(type) {
	case int:
		return p
	case int64:
		return int(p)
	case float64:
		return int(p)
	default:
		return fallback
	}
}

// ShortSHA256 computes a deterministic short SHA256 hash (first 8 bytes, hex-encoded)
// of the given value by JSON-marshaling it. Returns an empty string if marshaling fails.
func ShortSHA256(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:8])
}

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
