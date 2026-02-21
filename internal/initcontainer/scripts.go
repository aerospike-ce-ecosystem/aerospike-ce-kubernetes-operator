package initcontainer

import (
	_ "embed"
)

//go:embed scripts/aerospike-init.sh
var initScript string

// GetInitScript returns the embedded init container shell script.
func GetInitScript() string {
	return initScript
}

// GetConfigMapData returns the data map for the aerospike config ConfigMap.
// It includes the aerospike.conf content and the init script.
func GetConfigMapData(aerospikeConf string) map[string]string {
	return map[string]string{
		"aerospike.conf":    aerospikeConf,
		"aerospike-init.sh": initScript,
	}
}
