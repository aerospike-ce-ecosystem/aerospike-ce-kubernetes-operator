package controller

// Event reason constants for Kubernetes Events recorded by the AerospikeCECluster controller.
// Use these constants instead of hardcoded strings to avoid typos and enable consistent monitoring.
const (
	// Rolling restart lifecycle
	EventRollingRestartStarted   = "RollingRestartStarted"
	EventRollingRestartCompleted = "RollingRestartCompleted"
	EventRestartFailed           = "RestartFailed"
	EventPodWarmRestarted        = "PodWarmRestarted"
	EventPodColdRestarted        = "PodColdRestarted"
	EventLocalPVCDeleteFailed    = "LocalPVCDeleteFailed"

	// Config management
	EventConfigMapCreated     = "ConfigMapCreated"
	EventConfigMapUpdated     = "ConfigMapUpdated"
	EventDynamicConfigApplied = "DynamicConfigApplied"
	EventDynamicConfigFailed  = "DynamicConfigFailed"

	// StatefulSet / Rack management
	EventStatefulSetCreated = "StatefulSetCreated"
	EventStatefulSetUpdated = "StatefulSetUpdated"
	EventRackScaled         = "RackScaled"

	// ACL synchronization
	EventACLSyncStarted   = "ACLSyncStarted"
	EventACLSyncCompleted = "ACLSyncCompleted"
	EventACLSyncError     = "ACLSyncError"

	// PodDisruptionBudget
	EventPDBCreated = "PDBCreated"
	EventPDBUpdated = "PDBUpdated"

	// Service management
	EventServiceCreated = "ServiceCreated"
	EventServiceUpdated = "ServiceUpdated"

	// Cluster lifecycle
	EventClusterDeletionStarted = "ClusterDeletionStarted"
	EventFinalizerRemoved       = "FinalizerRemoved"

	// Template
	EventTemplateApplied         = "TemplateApplied"
	EventTemplateResolutionError = "TemplateResolutionError"
	EventTemplateDrifted         = "TemplateDrifted"

	// Miscellaneous
	EventValidationWarning = "ValidationWarning"
	EventReconcileError    = "ReconcileError"
	EventOperation         = "Operation"
)
