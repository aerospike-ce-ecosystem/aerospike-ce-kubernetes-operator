/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var aerospikececlusterlog = logf.Log.WithName("aerospikececluster-resource")

const (
	maxCEClusterSize     = 8
	maxCENamespaces      = 2
	defaultProtoFdMax    = 15000
	defaultHeartbeatMode = "mesh"

	defaultExporterImage  = "aerospike/aerospike-prometheus-exporter:v1.16.1"
	defaultExporterPort   = int32(9145)
	defaultScrapeInterval = "30s"
)

// SetupWebhookWithManager registers the webhooks for AerospikeCECluster.
func (r *AerospikeCECluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithDefaulter(&AerospikeCEClusterDefaulter{}).
		WithValidator(&AerospikeCEClusterValidator{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-acko-io-v1alpha1-aerospikececluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=acko.io,resources=aerospikececlusters,verbs=create;update,versions=v1alpha1,name=maerospikececluster.kb.io,admissionReviewVersions=v1

// AerospikeCEClusterDefaulter implements admission.Defaulter for AerospikeCECluster.
type AerospikeCEClusterDefaulter struct{}

var _ admission.Defaulter[*AerospikeCECluster] = &AerospikeCEClusterDefaulter{}

// Default implements admission.Defaulter[*AerospikeCECluster].
func (d *AerospikeCEClusterDefaulter) Default(ctx context.Context, cluster *AerospikeCECluster) error {
	aerospikececlusterlog.Info("Defaulting", "name", cluster.Name, "namespace", cluster.Namespace)

	if cluster.Spec.AerospikeConfig == nil {
		cluster.Spec.AerospikeConfig = &AerospikeConfigSpec{
			Value: make(map[string]any),
		}
	}
	if cluster.Spec.AerospikeConfig.Value == nil {
		cluster.Spec.AerospikeConfig.Value = make(map[string]any)
	}

	d.defaultServiceConfig(cluster)
	d.defaultNetworkConfig(cluster)
	d.defaultMonitoring(cluster)
	d.defaultHostNetwork(cluster)

	return nil
}

// defaultServiceConfig sets defaults in aerospikeConfig.service.
func (d *AerospikeCEClusterDefaulter) defaultServiceConfig(cluster *AerospikeCECluster) {
	config := cluster.Spec.AerospikeConfig.Value

	serviceSection := getOrCreateMapSection(config, "service")

	if _, exists := serviceSection["cluster-name"]; !exists {
		serviceSection["cluster-name"] = cluster.Name
	}

	if _, exists := serviceSection["proto-fd-max"]; !exists {
		serviceSection["proto-fd-max"] = defaultProtoFdMax
	}

	config["service"] = serviceSection
}

// defaultNetworkConfig sets defaults in aerospikeConfig.network.
func (d *AerospikeCEClusterDefaulter) defaultNetworkConfig(cluster *AerospikeCECluster) {
	config := cluster.Spec.AerospikeConfig.Value

	networkSection := getOrCreateMapSection(config, "network")

	// Default values for each network sub-section.
	networkDefaults := map[string]map[string]any{
		"service":   {"port": int(DefaultServicePort)},
		"heartbeat": {"port": int(DefaultHeartbeatPort), "mode": defaultHeartbeatMode},
		"fabric":    {"port": int(DefaultFabricPort)},
	}

	for name, defs := range networkDefaults {
		section := getOrCreateMapSection(networkSection, name)
		for k, v := range defs {
			if _, exists := section[k]; !exists {
				section[k] = v
			}
		}
		networkSection[name] = section
	}

	config["network"] = networkSection
}

// defaultMonitoring sets default values for the monitoring spec when enabled.
func (d *AerospikeCEClusterDefaulter) defaultMonitoring(cluster *AerospikeCECluster) {
	if cluster.Spec.Monitoring == nil || !cluster.Spec.Monitoring.Enabled {
		return
	}

	m := cluster.Spec.Monitoring
	if m.ExporterImage == "" {
		m.ExporterImage = defaultExporterImage
	}
	if m.Port == 0 {
		m.Port = defaultExporterPort
	}
	if m.ServiceMonitor != nil && m.ServiceMonitor.Enabled && m.ServiceMonitor.Interval == "" {
		m.ServiceMonitor.Interval = defaultScrapeInterval
	}
}

// defaultHostNetwork sets defaults for hostNetwork mode.
func (d *AerospikeCEClusterDefaulter) defaultHostNetwork(cluster *AerospikeCECluster) {
	if cluster.Spec.PodSpec == nil || !cluster.Spec.PodSpec.HostNetwork {
		return
	}

	// Default multiPodPerHost to false when hostNetwork is enabled
	if cluster.Spec.PodSpec.MultiPodPerHost == nil {
		f := false
		cluster.Spec.PodSpec.MultiPodPerHost = &f
	}

	// Default dnsPolicy to ClusterFirstWithHostNet
	if cluster.Spec.PodSpec.DNSPolicy == "" {
		cluster.Spec.PodSpec.DNSPolicy = corev1.DNSClusterFirstWithHostNet
	}
}

// getOrCreateMapSection returns the sub-map at key or creates a new one.
func getOrCreateMapSection(m map[string]any, key string) map[string]any {
	if existing, ok := m[key]; ok {
		if existingMap, ok := existing.(map[string]any); ok {
			return existingMap
		}
	}
	newMap := make(map[string]any)
	m[key] = newMap
	return newMap
}

// +kubebuilder:webhook:path=/validate-acko-io-v1alpha1-aerospikececluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=acko.io,resources=aerospikececlusters,verbs=create;update,versions=v1alpha1,name=vaerospikececluster.kb.io,admissionReviewVersions=v1

// AerospikeCEClusterValidator implements admission.Validator for AerospikeCECluster.
type AerospikeCEClusterValidator struct{}

var _ admission.Validator[*AerospikeCECluster] = &AerospikeCEClusterValidator{}

// ValidateCreate implements admission.Validator[*AerospikeCECluster].
func (v *AerospikeCEClusterValidator) ValidateCreate(ctx context.Context, cluster *AerospikeCECluster) (admission.Warnings, error) {
	aerospikececlusterlog.Info("Validating create", "name", cluster.Name)
	return v.validate(cluster)
}

// ValidateUpdate implements admission.Validator[*AerospikeCECluster].
func (v *AerospikeCEClusterValidator) ValidateUpdate(ctx context.Context, oldCluster, cluster *AerospikeCECluster) (admission.Warnings, error) {
	aerospikececlusterlog.Info("Validating update", "name", cluster.Name)

	// Don't allow changing operations while one is InProgress
	if oldCluster.Status.OperationStatus != nil &&
		oldCluster.Status.OperationStatus.Phase == AerospikePhaseInProgress {
		oldOps := oldCluster.Spec.Operations
		newOps := cluster.Spec.Operations
		// Block if operation list changed (added, removed, or replaced)
		if len(oldOps) != len(newOps) {
			return nil, fmt.Errorf("cannot change operations while operation %q is InProgress", oldCluster.Status.OperationStatus.ID)
		}
		for i := range oldOps {
			if oldOps[i].ID != newOps[i].ID || oldOps[i].Kind != newOps[i].Kind {
				return nil, fmt.Errorf("cannot change operations while operation %q is InProgress", oldCluster.Status.OperationStatus.ID)
			}
		}
	}

	// Prevent renaming existing rack IDs (which would cause data loss).
	// Adding or removing racks is fine, but changing an existing rack's ID is not.
	if oldCluster.Spec.RackConfig != nil && cluster.Spec.RackConfig != nil {
		oldIDs := make(map[int]bool, len(oldCluster.Spec.RackConfig.Racks))
		for _, rack := range oldCluster.Spec.RackConfig.Racks {
			oldIDs[rack.ID] = true
		}
		newIDs := make(map[int]bool, len(cluster.Spec.RackConfig.Racks))
		for _, rack := range cluster.Spec.RackConfig.Racks {
			newIDs[rack.ID] = true
		}
		// If the total rack count is unchanged but IDs differ, it's likely a rename
		if len(oldIDs) == len(newIDs) {
			for id := range oldIDs {
				if !newIDs[id] {
					return nil, fmt.Errorf("rackConfig rack IDs cannot be changed (rack ID %d was removed and replaced); remove old racks and add new ones in separate updates to avoid data loss", id)
				}
			}
		}
	}

	return v.validate(cluster)
}

// ValidateDelete implements admission.Validator[*AerospikeCECluster].
func (v *AerospikeCEClusterValidator) ValidateDelete(ctx context.Context, cluster *AerospikeCECluster) (admission.Warnings, error) {
	return nil, nil
}

// validate performs all CE-specific validations.
func (v *AerospikeCEClusterValidator) validate(cluster *AerospikeCECluster) (admission.Warnings, error) {
	var allErrors []string
	var warnings admission.Warnings

	// Validate cluster size
	if cluster.Spec.Size > maxCEClusterSize {
		allErrors = append(allErrors, fmt.Sprintf("spec.size %d exceeds CE maximum of %d", cluster.Spec.Size, maxCEClusterSize))
	}

	// Validate image is not empty
	if cluster.Spec.Image == "" {
		allErrors = append(allErrors, "spec.image must not be empty")
	}

	// Validate image is not enterprise (legacy "enterprise" in name or new ":ee-" tag prefix)
	imageLower := strings.ToLower(cluster.Spec.Image)
	if strings.Contains(imageLower, "enterprise") || isEnterpriseTag(cluster.Spec.Image) {
		allErrors = append(allErrors, fmt.Sprintf("spec.image %q is an Enterprise Edition image; only Community Edition images are allowed", cluster.Spec.Image))
	}

	// Warn about untagged or "latest" images
	if !strings.Contains(cluster.Spec.Image, ":") {
		warnings = append(warnings, fmt.Sprintf("spec.image %q has no tag; use an explicit version tag (e.g., aerospike:ce-8.1.1.1) for reproducible deployments", cluster.Spec.Image))
	} else {
		parts := strings.SplitN(cluster.Spec.Image, ":", 2)
		if parts[1] == "latest" {
			warnings = append(warnings, "spec.image uses 'latest' tag; use an explicit version tag for reproducible deployments")
		}
	}

	// Validate aerospikeConfig
	if cluster.Spec.AerospikeConfig != nil {
		configErrors, configWarnings := v.validateAerospikeConfig(cluster.Spec.AerospikeConfig.Value)
		allErrors = append(allErrors, configErrors...)
		warnings = append(warnings, configWarnings...)
	}

	// Validate access control
	if cluster.Spec.AerospikeAccessControl != nil {
		acErrors := v.validateAccessControl(cluster.Spec.AerospikeAccessControl)
		allErrors = append(allErrors, acErrors...)
	}

	// Validate hostNetwork + multiPodPerHost
	if cluster.Spec.PodSpec != nil && cluster.Spec.PodSpec.HostNetwork {
		if cluster.Spec.PodSpec.MultiPodPerHost != nil && *cluster.Spec.PodSpec.MultiPodPerHost {
			warnings = append(warnings, "hostNetwork=true with multiPodPerHost=true may cause port conflicts")
		}
		if cluster.Spec.PodSpec.DNSPolicy != "" && cluster.Spec.PodSpec.DNSPolicy != corev1.DNSClusterFirstWithHostNet {
			warnings = append(warnings, "hostNetwork=true with dnsPolicy other than ClusterFirstWithHostNet may cause DNS resolution issues")
		}
	}

	// Validate rack config
	if cluster.Spec.RackConfig != nil {
		rackErrors := v.validateRackConfig(cluster.Spec.RackConfig)
		allErrors = append(allErrors, rackErrors...)
	}

	// Validate monitoring
	if cluster.Spec.Monitoring != nil {
		if cluster.Spec.Monitoring.Enabled {
			mErrors, mWarnings := v.validateMonitoring(cluster.Spec.Monitoring)
			allErrors = append(allErrors, mErrors...)
			warnings = append(warnings, mWarnings...)
		} else {
			// Warn if sub-features are enabled but monitoring itself is disabled
			if cluster.Spec.Monitoring.ServiceMonitor != nil && cluster.Spec.Monitoring.ServiceMonitor.Enabled {
				warnings = append(warnings, "monitoring.serviceMonitor.enabled is true but monitoring.enabled is false; ServiceMonitor will not be created")
			}
			if cluster.Spec.Monitoring.PrometheusRule != nil && cluster.Spec.Monitoring.PrometheusRule.Enabled {
				warnings = append(warnings, "monitoring.prometheusRule.enabled is true but monitoring.enabled is false; PrometheusRule will not be created")
			}
		}
	}

	// Validate storage
	if cluster.Spec.Storage != nil {
		storageErrors, storageWarnings := v.validateStorage(cluster.Spec.Storage)
		allErrors = append(allErrors, storageErrors...)
		warnings = append(warnings, storageWarnings...)
	}

	// Validate replication-factor, work directory, batch size, max unavailable, and operations
	rfErrors := v.validateReplicationFactor(cluster)
	allErrors = append(allErrors, rfErrors...)
	warnings = append(warnings, v.validateWorkDirectory(cluster)...)
	warnings = append(warnings, v.validateBatchSize(cluster)...)
	warnings = append(warnings, v.validateMaxUnavailable(cluster)...)
	if len(cluster.Spec.Operations) > 0 {
		allErrors = append(allErrors, v.validateOperations(cluster.Spec.Operations)...)
	}

	// Validate overrides requires templateRef
	if cluster.Spec.Overrides != nil && cluster.Spec.TemplateRef == nil {
		allErrors = append(allErrors, "spec.overrides can only be set when spec.templateRef is specified")
	}

	if len(allErrors) > 0 {
		return warnings, fmt.Errorf("validation failed: %s", strings.Join(allErrors, "; "))
	}

	return warnings, nil
}

// validateAerospikeConfig checks the Aerospike configuration map.
func (v *AerospikeCEClusterValidator) validateAerospikeConfig(config map[string]any) ([]string, admission.Warnings) {
	var errors []string
	var warnings admission.Warnings

	// CE does not support XDR
	if _, exists := config["xdr"]; exists {
		errors = append(errors, "aerospikeConfig must not contain 'xdr' section (XDR is Enterprise-only)")
	}

	// CE does not support TLS
	if _, exists := config["tls"]; exists {
		errors = append(errors, "aerospikeConfig must not contain 'tls' section (TLS is Enterprise-only)")
	}

	// Count namespaces (CE limit: 2)
	if nsSection, exists := config["namespaces"]; exists {
		switch ns := nsSection.(type) {
		case []any:
			if len(ns) > maxCENamespaces {
				errors = append(errors, fmt.Sprintf("aerospikeConfig.namespaces count %d exceeds CE maximum of %d", len(ns), maxCENamespaces))
			}
			// Validate each namespace's config
			for i, nsEntry := range ns {
				if nsMap, ok := nsEntry.(map[string]any); ok {
					nsErrors, nsWarnings := v.validateNamespaceConfig(nsMap, i)
					errors = append(errors, nsErrors...)
					warnings = append(warnings, nsWarnings...)
				}
			}
		case map[string]any:
			if len(ns) > maxCENamespaces {
				errors = append(errors, fmt.Sprintf("aerospikeConfig.namespaces count %d exceeds CE maximum of %d", len(ns), maxCENamespaces))
			}
		}
	}

	// CE does not support security stanza (Enterprise-only in Aerospike 8.x)
	if _, exists := config["security"]; exists {
		errors = append(errors, "aerospikeConfig must not contain 'security' section (security/ACL is Enterprise-only in Aerospike CE 8.x)")
	}

	// Validate heartbeat mode is mesh (CE only supports mesh)
	if netCfg, ok := config["network"].(map[string]any); ok {
		if hbCfg, ok := netCfg["heartbeat"].(map[string]any); ok {
			if mode, ok := hbCfg["mode"].(string); ok && mode != "mesh" {
				errors = append(errors, fmt.Sprintf("aerospikeConfig.network.heartbeat.mode must be 'mesh' for CE (got %q); multicast is Enterprise-only", mode))
			}
		}
	}

	return errors, warnings
}

// enterpriseOnlyNamespaceKeys lists namespace-level config keys that are Enterprise-only.
var enterpriseOnlyNamespaceKeys = map[string]string{
	"compression":              "data compression is Enterprise-only",
	"compression-level":        "data compression is Enterprise-only",
	"durable-delete":           "durable deletes is Enterprise-only",
	"fast-restart":             "fast restart is Enterprise-only",
	"index-type":               "index-type flash/pmem is Enterprise-only",
	"sindex-type":              "sindex-type flash/pmem is Enterprise-only",
	"rack-id":                  "rack-id in namespace is Enterprise-only; use operator rackConfig instead",
	"strong-consistency":       "strong consistency is Enterprise-only",
	"tomb-raider-eligible-age": "tomb-raider is Enterprise-only",
	"tomb-raider-period":       "tomb-raider is Enterprise-only",
}

// validateNamespaceConfig checks individual namespace config for CE-incompatible options.
func (v *AerospikeCEClusterValidator) validateNamespaceConfig(nsMap map[string]any, index int) ([]string, admission.Warnings) {
	var errors []string
	var warnings admission.Warnings

	nsName := "<unknown>"
	if name, ok := nsMap["name"].(string); ok {
		nsName = name
	}

	// Check for enterprise-only keys
	for key, reason := range enterpriseOnlyNamespaceKeys {
		if _, exists := nsMap[key]; exists {
			errors = append(errors, fmt.Sprintf("namespace[%d] %q: '%s' is not allowed (%s)", index, nsName, key, reason))
		}
	}

	// Warn about data-in-memory usage in storage-engine device
	if se, ok := nsMap["storage-engine"].(map[string]any); ok {
		if dim, ok := se["data-in-memory"]; ok {
			if dimBool, ok := dim.(bool); ok && dimBool {
				warnings = append(warnings, fmt.Sprintf(
					"namespace %q: data-in-memory=true doubles memory usage (data stored in both memory and disk); ensure sufficient memory-size",
					nsName,
				))
			}
		}
	}

	// Validate replication-factor: single-node clusters should use 1
	if rf, ok := nsMap["replication-factor"]; ok {
		switch v := rf.(type) {
		case int:
			if v < 1 || v > 4 {
				errors = append(errors, fmt.Sprintf("namespace[%d] %q: replication-factor must be between 1 and 4 (got %d)", index, nsName, v))
			}
		case float64:
			if v < 1 || v > 4 {
				errors = append(errors, fmt.Sprintf("namespace[%d] %q: replication-factor must be between 1 and 4 (got %v)", index, nsName, v))
			}
		}
	}

	return errors, warnings
}

// aerospikeCEBuiltinRoles lists the predefined role/privilege names in Aerospike CE.
// In CE, every builtin role name is also a valid privilege code, so a single
// set serves both purposes. Enterprise adds "superuser" which is excluded here.
// Reference: https://aerospike.com/docs/server/operations/configure/security/access-control/index.html
var aerospikeCEBuiltinRoles = map[string]bool{
	"user-admin":     true,
	"sys-admin":      true,
	"data-admin":     true,
	"read":           true,
	"write":          true,
	"read-write":     true,
	"read-write-udf": true,
	"truncate":       true,
}

// validateAccessControl validates the ACL configuration.
func (v *AerospikeCEClusterValidator) validateAccessControl(acl *AerospikeAccessControlSpec) []string {
	var errors []string

	// Check for duplicate user names
	seenUsers := make(map[string]bool)
	for _, user := range acl.Users {
		if seenUsers[user.Name] {
			errors = append(errors, fmt.Sprintf("accessControl.users: duplicate user name %q", user.Name))
		}
		seenUsers[user.Name] = true
	}

	hasAdmin := false
	for _, user := range acl.Users {
		if user.SecretName == "" {
			errors = append(errors, fmt.Sprintf("user %q must have a secretName for password", user.Name))
		}
		hasSysAdmin := false
		hasUserAdmin := false
		for _, role := range user.Roles {
			if role == "sys-admin" {
				hasSysAdmin = true
			}
			if role == "user-admin" {
				hasUserAdmin = true
			}
		}
		if hasSysAdmin && hasUserAdmin {
			hasAdmin = true
		}
	}

	if !hasAdmin {
		errors = append(errors, "aerospikeAccessControl must have at least one user with both 'sys-admin' and 'user-admin' roles (required for operator-managed cluster administration)")
	}

	// Check for duplicate role names
	seenRoles := make(map[string]bool)
	for _, role := range acl.Roles {
		if seenRoles[role.Name] {
			errors = append(errors, fmt.Sprintf("accessControl.roles: duplicate role name %q", role.Name))
		}
		seenRoles[role.Name] = true
	}

	// Validate that users reference only built-in or explicitly defined roles
	definedRoles := make(map[string]bool)
	for _, role := range acl.Roles {
		definedRoles[role.Name] = true
	}
	for _, user := range acl.Users {
		for _, roleName := range user.Roles {
			if !aerospikeCEBuiltinRoles[roleName] && !definedRoles[roleName] {
				errors = append(errors, fmt.Sprintf("user %q references undefined role %q", user.Name, roleName))
			}
		}
	}

	// Validate privilege codes in role definitions
	for _, role := range acl.Roles {
		for i, privStr := range role.Privileges {
			trimmed := strings.TrimSpace(privStr)
			if trimmed == "" {
				errors = append(errors, fmt.Sprintf("role %q privileges[%d]: privilege string must not be empty or whitespace-only", role.Name, i))
				continue
			}
			// Reject strings with leading/trailing whitespace: the original value is
			// stored as-is, so " read-write" would be sent to Aerospike verbatim and
			// rejected at runtime even though it looks valid after trimming.
			if privStr != trimmed {
				errors = append(errors, fmt.Sprintf("role %q privileges[%d]: privilege string %q must not have leading or trailing whitespace", role.Name, i, privStr))
				continue
			}
			// Format: "<code>" or "<code>.<namespace>" or "<code>.<namespace>.<set>"
			code := strings.SplitN(privStr, ".", 2)[0]
			if !aerospikeCEBuiltinRoles[code] {
				errors = append(errors, fmt.Sprintf("role %q has invalid privilege code %q; valid codes: read, write, read-write, read-write-udf, sys-admin, user-admin, data-admin, truncate", role.Name, code))
			}
		}
	}

	return errors
}

// isEnterpriseTag returns true if the image tag starts with "ee-" (e.g., "aerospike:ee-8.0.0.1_1").
func isEnterpriseTag(image string) bool {
	parts := strings.SplitN(image, ":", 2)
	if len(parts) != 2 {
		return false
	}

	return strings.HasPrefix(strings.ToLower(parts[1]), "ee-")
}

// hasVolumeForPath checks if any volume mounts to the given path.
func hasVolumeForPath(storage *AerospikeStorageSpec, path string) bool {
	if storage == nil {
		return false
	}
	for _, vol := range storage.Volumes {
		if vol.Aerospike != nil && vol.Aerospike.Path == path {
			return true
		}
	}
	return false
}

// validateReplicationFactor validates that replication-factor does not exceed cluster size.
func (v *AerospikeCEClusterValidator) validateReplicationFactor(cluster *AerospikeCECluster) []string {
	if cluster.Spec.AerospikeConfig == nil {
		return nil
	}
	nsList, ok := cluster.Spec.AerospikeConfig.Value["namespaces"].([]any)
	if !ok {
		return nil
	}
	var errors []string
	for _, ns := range nsList {
		nsMap, ok := ns.(map[string]any)
		if !ok {
			continue
		}
		nsName, _ := nsMap["name"].(string)
		rf, ok := nsMap["replication-factor"]
		if !ok {
			continue
		}
		rfInt := 0
		switch val := rf.(type) {
		case int:
			rfInt = val
		case int64:
			rfInt = int(val)
		case float64:
			if val != float64(int(val)) || val < 0 {
				errors = append(errors, fmt.Sprintf(
					"namespace %q: replication-factor must be a positive integer, got %v", nsName, val))
				continue
			}
			rfInt = int(val)
		}
		if rfInt > int(cluster.Spec.Size) {
			errors = append(errors, fmt.Sprintf(
				"namespace %q: replication-factor %d exceeds cluster size %d",
				nsName, rfInt, cluster.Spec.Size))
		}
	}
	return errors
}

// validateWorkDirectory checks that the work directory has persistent storage.
func (v *AerospikeCEClusterValidator) validateWorkDirectory(cluster *AerospikeCECluster) admission.Warnings {
	if cluster.Spec.ValidationPolicy != nil && cluster.Spec.ValidationPolicy.SkipWorkDirValidate {
		return nil
	}
	if cluster.Spec.AerospikeConfig == nil {
		return nil
	}
	svcCfg, ok := cluster.Spec.AerospikeConfig.Value["service"].(map[string]any)
	if !ok {
		return nil
	}
	workDir, ok := svcCfg["work-directory"].(string)
	if !ok || workDir == "" {
		return nil
	}
	if !hasVolumeForPath(cluster.Spec.Storage, workDir) {
		return admission.Warnings{fmt.Sprintf(
			"work-directory %q has no persistent volume; data may be lost on pod restart (set validationPolicy.skipWorkDirValidate to suppress)", workDir)}
	}
	return nil
}

// validateBatchSize checks the rolling update batch size against cluster size.
func (v *AerospikeCEClusterValidator) validateBatchSize(cluster *AerospikeCECluster) admission.Warnings {
	if cluster.Spec.RollingUpdateBatchSize == nil {
		return nil
	}
	bs := *cluster.Spec.RollingUpdateBatchSize
	if bs > cluster.Spec.Size {
		return admission.Warnings{fmt.Sprintf("rollingUpdateBatchSize (%d) is greater than cluster size (%d); all pods may restart simultaneously", bs, cluster.Spec.Size)}
	}
	return nil
}

// validateMaxUnavailable warns if maxUnavailable is >= cluster size.
func (v *AerospikeCEClusterValidator) validateMaxUnavailable(cluster *AerospikeCECluster) admission.Warnings {
	if cluster.Spec.MaxUnavailable == nil {
		return nil
	}
	mu := *cluster.Spec.MaxUnavailable
	if mu.Type == intstr.Int {
		if mu.IntVal >= cluster.Spec.Size {
			return admission.Warnings{fmt.Sprintf(
				"maxUnavailable (%d) is >= cluster size (%d); PodDisruptionBudget will not prevent full disruption",
				mu.IntVal, cluster.Spec.Size)}
		}
	} else {
		s := mu.StrVal
		if numStr, ok := strings.CutSuffix(s, "%"); ok {
			num, err := strconv.Atoi(numStr)
			if err == nil && num >= 100 {
				return admission.Warnings{fmt.Sprintf(
					"maxUnavailable (%s) allows 100%% disruption; PodDisruptionBudget will not protect availability", s)}
			}
		}
	}
	return nil
}

// validateRackConfig validates the rack configuration.
func (v *AerospikeCEClusterValidator) validateRackConfig(rackConfig *RackConfig) []string {
	var errors []string

	rackIDs := make(map[int]bool)
	rackLabels := make(map[string]bool)
	nodeNames := make(map[string]int)
	for i, rack := range rackConfig.Racks {
		if rack.ID <= 0 {
			errors = append(errors, fmt.Sprintf("rack ID must be > 0, got %d (rack ID 0 is reserved for the default rack)", rack.ID))
		}
		if rackIDs[rack.ID] {
			errors = append(errors, fmt.Sprintf("duplicate rack ID %d in rackConfig", rack.ID))
		}
		rackIDs[rack.ID] = true

		// Validate RackLabel uniqueness across racks
		if rack.RackLabel != "" {
			if rackLabels[rack.RackLabel] {
				errors = append(errors, fmt.Sprintf("duplicate rackLabel %q in rackConfig; each rack must have a unique rackLabel", rack.RackLabel))
			}
			rackLabels[rack.RackLabel] = true
		}

		// Validate NodeName uniqueness across racks
		if rack.NodeName != "" {
			if prevIdx, exists := nodeNames[rack.NodeName]; exists {
				errors = append(errors, fmt.Sprintf(
					"rackConfig.racks[%d] and racks[%d] both constrained to node %q; each rack must use a different nodeName",
					prevIdx, i, rack.NodeName))
			}
			nodeNames[rack.NodeName] = i
		}
	}

	// Validate ScaleDownBatchSize is positive if set
	if rackConfig.ScaleDownBatchSize != nil {
		if err := validateIntOrString(rackConfig.ScaleDownBatchSize, "rackConfig.scaleDownBatchSize", 1); err != "" {
			errors = append(errors, err)
		}
	}

	// Validate MaxIgnorablePods is non-negative if set
	if rackConfig.MaxIgnorablePods != nil {
		if err := validateIntOrString(rackConfig.MaxIgnorablePods, "rackConfig.maxIgnorablePods", 0); err != "" {
			errors = append(errors, err)
		}
	}

	// Validate RollingUpdateBatchSize is positive if set
	if rackConfig.RollingUpdateBatchSize != nil {
		if err := validateIntOrString(rackConfig.RollingUpdateBatchSize, "rackConfig.rollingUpdateBatchSize", 1); err != "" {
			errors = append(errors, err)
		}
	}

	return errors
}

// validateOperations validates the on-demand operations spec.
func (v *AerospikeCEClusterValidator) validateOperations(ops []OperationSpec) []string {
	var errors []string

	if len(ops) > 1 {
		errors = append(errors, "only one operation can be specified at a time")
	}

	seenIDs := make(map[string]bool)
	for _, op := range ops {
		if len(op.ID) < 1 || len(op.ID) > 20 {
			errors = append(errors, fmt.Sprintf("operation id %q must be 1-20 characters", op.ID))
		}
		if seenIDs[op.ID] {
			errors = append(errors, fmt.Sprintf("duplicate operation id %q", op.ID))
		}
		seenIDs[op.ID] = true
	}

	return errors
}

// validateIntOrString validates that an IntOrString value meets a minimum bound.
// Use minValue=1 for positive validation, minValue=0 for non-negative validation.
func validateIntOrString(val *intstr.IntOrString, fieldName string, minValue int) string {
	if val == nil {
		return ""
	}
	label := "positive"
	if minValue == 0 {
		label = "non-negative"
	}
	if val.Type == intstr.Int {
		if int(val.IntVal) < minValue {
			return fmt.Sprintf("%s must be a %s integer (got %d)", fieldName, label, val.IntVal)
		}
	} else {
		s := val.StrVal
		if !strings.HasSuffix(s, "%") {
			return fmt.Sprintf("%s must be a %s integer or a percentage string (e.g., \"25%%\"); got %q", fieldName, label, s)
		}
		numStr := strings.TrimSuffix(s, "%")
		num, err := strconv.Atoi(numStr)
		if err != nil {
			return fmt.Sprintf("%s percentage %q is not a valid integer", fieldName, s)
		}
		if num < minValue || num > 100 {
			return fmt.Sprintf("%s percentage must be between %d and 100 (got %d)", fieldName, minValue, num)
		}
	}
	return ""
}

// validateStorage validates the storage configuration.
func (v *AerospikeCEClusterValidator) validateStorage(storage *AerospikeStorageSpec) ([]string, admission.Warnings) {
	var errors []string
	var warnings admission.Warnings

	// Check for duplicate volume names
	namesSeen := make(map[string]bool, len(storage.Volumes))
	for _, vol := range storage.Volumes {
		if namesSeen[vol.Name] {
			errors = append(errors, fmt.Sprintf("storage.volumes: duplicate volume name %q", vol.Name))
		}
		namesSeen[vol.Name] = true
	}

	for i, vol := range storage.Volumes {
		volErrors, volWarnings := v.validateVolume(vol, i)
		errors = append(errors, volErrors...)
		warnings = append(warnings, volWarnings...)
	}

	// Validate deleteLocalStorageOnRestart requires localStorageClasses
	if storage.DeleteLocalStorageOnRestart != nil && *storage.DeleteLocalStorageOnRestart {
		if len(storage.LocalStorageClasses) == 0 {
			errors = append(errors, "storage.deleteLocalStorageOnRestart is true but storage.localStorageClasses is empty; specify which storage classes are local")
		}
	}

	// Warn if local storage class is used but deleteLocalStorageOnRestart is not set
	if len(storage.LocalStorageClasses) > 0 && storage.DeleteLocalStorageOnRestart == nil {
		warnings = append(warnings, "storage.localStorageClasses is set but storage.deleteLocalStorageOnRestart is not configured; local PVCs will not be deleted on pod restart")
	}

	return errors, warnings
}

// validateVolume validates a single volume spec.
func (v *AerospikeCEClusterValidator) validateVolume(vol VolumeSpec, index int) ([]string, admission.Warnings) {
	var errors []string
	var warnings admission.Warnings

	// Validate exactly one volume source is specified
	sourceCount := 0
	if vol.Source.PersistentVolume != nil {
		sourceCount++
	}
	if vol.Source.EmptyDir != nil {
		sourceCount++
	}
	if vol.Source.Secret != nil {
		sourceCount++
	}
	if vol.Source.ConfigMap != nil {
		sourceCount++
	}
	if vol.Source.HostPath != nil {
		sourceCount++
	}
	if sourceCount == 0 {
		errors = append(errors, fmt.Sprintf("storage.volumes[%d] %q: exactly one volume source must be specified", index, vol.Name))
	} else if sourceCount > 1 {
		errors = append(errors, fmt.Sprintf("storage.volumes[%d] %q: only one volume source can be specified (found %d)", index, vol.Name, sourceCount))
	}

	// Warn about HostPath usage
	if vol.Source.HostPath != nil {
		warnings = append(warnings, fmt.Sprintf(
			"storage.volumes[%d] %q: hostPath volumes are not recommended for production; data is tied to a specific node and not portable",
			index, vol.Name))
	}

	// Warn about cascadeDelete on non-persistent volumes (has no effect)
	if vol.CascadeDelete != nil && *vol.CascadeDelete && vol.Source.PersistentVolume == nil {
		warnings = append(warnings, fmt.Sprintf(
			"storage.volumes[%d] %q: cascadeDelete has no effect on non-persistent volumes",
			index, vol.Name))
	}

	// Validate PV size is a valid Kubernetes quantity
	if vol.Source.PersistentVolume != nil {
		if vol.Source.PersistentVolume.Size == "" {
			errors = append(errors, fmt.Sprintf(
				"storage.volumes[%d] %q: persistentVolume.size must not be empty", index, vol.Name))
		} else if qty, err := resource.ParseQuantity(vol.Source.PersistentVolume.Size); err != nil {
			errors = append(errors, fmt.Sprintf(
				"storage.volumes[%d] %q: persistentVolume.size %q is not a valid Kubernetes quantity: %v",
				index, vol.Name, vol.Source.PersistentVolume.Size, err))
		} else if qty.Sign() <= 0 {
			errors = append(errors, fmt.Sprintf(
				"storage.volumes[%d] %q: persistentVolume.size must be a positive quantity (got %q)",
				index, vol.Name, vol.Source.PersistentVolume.Size))
		}
	}

	// Validate Aerospike mount path is absolute
	if vol.Aerospike != nil && vol.Aerospike.Path != "" && !strings.HasPrefix(vol.Aerospike.Path, "/") {
		errors = append(errors, fmt.Sprintf(
			"storage.volumes[%d] %q: aerospike.path must be an absolute path (got %q)",
			index, vol.Name, vol.Aerospike.Path))
	}

	// Validate SubPath and SubPathExpr are mutually exclusive (Aerospike attachment)
	if vol.Aerospike != nil && vol.Aerospike.SubPath != "" && vol.Aerospike.SubPathExpr != "" {
		errors = append(errors, fmt.Sprintf(
			"storage.volumes[%d] %q: aerospike.subPath and aerospike.subPathExpr are mutually exclusive",
			index, vol.Name))
	}

	// Validate SubPath and SubPathExpr in sidecar attachments
	for j, sc := range vol.Sidecars {
		if sc.SubPath != "" && sc.SubPathExpr != "" {
			errors = append(errors, fmt.Sprintf(
				"storage.volumes[%d] %q: sidecars[%d] %q subPath and subPathExpr are mutually exclusive",
				index, vol.Name, j, sc.ContainerName))
		}
	}

	// Validate SubPath and SubPathExpr in init container attachments
	for j, ic := range vol.InitContainers {
		if ic.SubPath != "" && ic.SubPathExpr != "" {
			errors = append(errors, fmt.Sprintf(
				"storage.volumes[%d] %q: initContainers[%d] %q subPath and subPathExpr are mutually exclusive",
				index, vol.Name, j, ic.ContainerName))
		}
	}

	return errors, warnings
}

// aerospikeReservedPorts lists ports used by Aerospike server that must not conflict.
var aerospikeReservedPorts = map[int32]string{
	3000: "service",
	3001: "fabric",
	3002: "heartbeat",
	3003: "info",
}

// validateMonitoring validates the monitoring configuration.
func (v *AerospikeCEClusterValidator) validateMonitoring(m *AerospikeMonitoringSpec) ([]string, admission.Warnings) {
	var errors []string
	var warnings admission.Warnings

	// Validate port is in valid TCP range.
	if m.Port < 1 || m.Port > 65535 {
		errors = append(errors, fmt.Sprintf("monitoring.port must be in range 1-65535 (got %d)", m.Port))
	}

	// Port conflict check with Aerospike reserved ports.
	if portName, ok := aerospikeReservedPorts[m.Port]; ok {
		errors = append(errors, fmt.Sprintf("monitoring.port %d conflicts with Aerospike %s port", m.Port, portName))
	}

	// Empty image check.
	if m.ExporterImage == "" {
		errors = append(errors, "monitoring.exporterImage must not be empty when monitoring is enabled")
	}

	// Warn about 'latest' tag on exporter image.
	if strings.Contains(m.ExporterImage, ":") {
		parts := strings.SplitN(m.ExporterImage, ":", 2)
		if parts[1] == "latest" {
			warnings = append(warnings, "monitoring.exporterImage uses 'latest' tag; use an explicit version tag for reproducible deployments")
		}
	} else if m.ExporterImage != "" {
		warnings = append(warnings, fmt.Sprintf("monitoring.exporterImage %q has no tag; use an explicit version tag for reproducible deployments", m.ExporterImage))
	}

	// Validate MetricLabels keys and values do not contain reserved characters.
	for k, val := range m.MetricLabels {
		if strings.ContainsAny(k, "=,") {
			errors = append(errors, fmt.Sprintf("monitoring.metricLabels key %q must not contain '=' or ','", k))
		}
		if strings.ContainsAny(val, "=,") {
			errors = append(errors, fmt.Sprintf("monitoring.metricLabels[%q] value %q must not contain '=' or ','", k, val))
		}
	}

	// Validate CustomRules structure.
	if m.PrometheusRule != nil {
		for i, raw := range m.PrometheusRule.CustomRules {
			var ruleGroup map[string]any
			if err := json.Unmarshal(raw.Raw, &ruleGroup); err != nil {
				errors = append(errors, fmt.Sprintf("monitoring.prometheusRule.customRules[%d]: invalid JSON: %v", i, err))
				continue
			}
			if _, ok := ruleGroup["name"]; !ok {
				errors = append(errors, fmt.Sprintf("monitoring.prometheusRule.customRules[%d]: missing required field 'name'", i))
			}
			if _, ok := ruleGroup["rules"]; !ok {
				errors = append(errors, fmt.Sprintf("monitoring.prometheusRule.customRules[%d]: missing required field 'rules'", i))
			}
		}
	}

	return errors, warnings
}
