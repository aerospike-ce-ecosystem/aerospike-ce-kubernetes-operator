package v1alpha1

// Default Aerospike network ports and container constants.
// These are used by both the webhook defaulter and podutil package.
const (
	DefaultServicePort   int32 = 3000
	DefaultFabricPort    int32 = 3001
	DefaultHeartbeatPort int32 = 3002

	AerospikeContainerName = "aerospike-server"
)
