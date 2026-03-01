package controller

// Readiness gate event reason constants.
const (
	// EventReadinessGateSatisfied is emitted when a pod's readiness gate
	// transitions from False to True (Aerospike joined mesh, migrations done).
	EventReadinessGateSatisfied = "ReadinessGateSatisfied"

	// EventReadinessGateBlocking is emitted (Warning) when the rolling restart
	// is paused because a pod's readiness gate is not yet satisfied.
	EventReadinessGateBlocking = "ReadinessGateBlocking"
)
