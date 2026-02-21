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
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var aerospikececlusterlog = logf.Log.WithName("aerospikececluster-resource")

const (
	maxCEClusterSize     = 8
	maxCENamespaces      = 2
	defaultServicePort   = 3000
	defaultFabricPort    = 3001
	defaultHeartbeatPort = 3002
	defaultProtoFdMax    = 15000
	defaultHeartbeatMode = "mesh"

	defaultExporterImage  = "aerospike/aerospike-prometheus-exporter:latest"
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
		"service":   {"port": defaultServicePort},
		"heartbeat": {"port": defaultHeartbeatPort, "mode": defaultHeartbeatMode},
		"fabric":    {"port": defaultFabricPort},
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

	// Validate image is not enterprise (legacy "enterprise" in name or new ":ee-" tag prefix)
	imageLower := strings.ToLower(cluster.Spec.Image)
	if strings.Contains(imageLower, "enterprise") || isEnterpriseTag(cluster.Spec.Image) {
		allErrors = append(allErrors, fmt.Sprintf("spec.image %q is an Enterprise Edition image; only Community Edition images are allowed", cluster.Spec.Image))
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

	// Validate rolling update batch size
	if cluster.Spec.RollingUpdateBatchSize != nil {
		bs := *cluster.Spec.RollingUpdateBatchSize
		if bs > cluster.Spec.Size {
			warnings = append(warnings, fmt.Sprintf("rollingUpdateBatchSize (%d) is greater than cluster size (%d); all pods may restart simultaneously", bs, cluster.Spec.Size))
		}
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

// validateAccessControl validates the ACL configuration.
func (v *AerospikeCEClusterValidator) validateAccessControl(acl *AerospikeAccessControlSpec) []string {
	var errors []string

	hasAdmin := false
	for _, user := range acl.Users {
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
			break
		}
	}

	if !hasAdmin {
		errors = append(errors, "aerospikeAccessControl must have at least one user with both 'sys-admin' and 'user-admin' roles")
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

// validateRackConfig validates the rack configuration.
func (v *AerospikeCEClusterValidator) validateRackConfig(rackConfig *RackConfig) []string {
	var errors []string

	rackIDs := make(map[int]bool)
	for _, rack := range rackConfig.Racks {
		if rackIDs[rack.ID] {
			errors = append(errors, fmt.Sprintf("duplicate rack ID %d in rackConfig", rack.ID))
		}
		rackIDs[rack.ID] = true
	}

	return errors
}
