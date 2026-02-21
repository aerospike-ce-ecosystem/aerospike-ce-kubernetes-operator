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
	corev1 "k8s.io/api/core/v1"
)

// AerospikeMonitoringSpec defines Prometheus monitoring configuration for the cluster.
type AerospikeMonitoringSpec struct {
	// Enabled enables the Prometheus exporter sidecar.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// ExporterImage is the Aerospike Prometheus exporter container image.
	// Defaults to "aerospike/aerospike-prometheus-exporter:latest".
	// +optional
	ExporterImage string `json:"exporterImage,omitempty"`

	// Port is the metrics port for the exporter. Defaults to 9145.
	// +optional
	Port int32 `json:"port,omitempty"`

	// Resources defines CPU and memory resource requests/limits for the exporter.
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// ServiceMonitor configures automatic ServiceMonitor creation for Prometheus Operator.
	// +optional
	ServiceMonitor *ServiceMonitorSpec `json:"serviceMonitor,omitempty"`
}

// ServiceMonitorSpec defines ServiceMonitor configuration.
type ServiceMonitorSpec struct {
	// Enabled enables ServiceMonitor creation.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Interval is the Prometheus scrape interval. Defaults to "30s".
	// +optional
	Interval string `json:"interval,omitempty"`

	// Labels are additional labels to add to the ServiceMonitor for discovery.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}
