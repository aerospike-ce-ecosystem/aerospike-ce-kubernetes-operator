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

// RackConfig defines rack-aware deployment configuration.
type RackConfig struct {
	// Racks is the list of rack definitions.
	// +kubebuilder:validation:MinItems=1
	Racks []Rack `json:"racks"`

	// Namespaces lists Aerospike namespace names that are rack-aware.
	// If empty, all namespaces use the default replication factor.
	// +optional
	Namespaces []string `json:"namespaces,omitempty"`
}

// Rack defines a single rack in the cluster topology.
type Rack struct {
	// ID is a unique rack identifier (integer).
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Required
	ID int `json:"id"`

	// Zone is the cloud provider zone label value for scheduling.
	// Maps to topology.kubernetes.io/zone.
	// +optional
	Zone string `json:"zone,omitempty"`

	// Region is the cloud provider region label value for scheduling.
	// Maps to topology.kubernetes.io/region.
	// +optional
	Region string `json:"region,omitempty"`

	// NodeName constrains this rack to a specific Kubernetes node.
	// +optional
	NodeName string `json:"nodeName,omitempty"`

	// AerospikeConfig overrides the cluster-level Aerospike config for this rack.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	AerospikeConfig *AerospikeConfigSpec `json:"aerospikeConfig,omitempty"`

	// Storage overrides the cluster-level storage config for this rack.
	// +optional
	Storage *AerospikeStorageSpec `json:"storage,omitempty"`

	// PodSpec overrides the cluster-level pod spec for this rack.
	// +optional
	PodSpec *RackPodSpec `json:"podSpec,omitempty"`
}

// RackPodSpec defines rack-level pod scheduling overrides.
type RackPodSpec struct {
	// Affinity overrides the cluster-level affinity for this rack.
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// Tolerations overrides the cluster-level tolerations for this rack.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// NodeSelector overrides the cluster-level node selector for this rack.
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
}

// AerospikeAccessControlSpec defines ACL configuration for Aerospike CE 7.x+.
type AerospikeAccessControlSpec struct {
	// Roles defines Aerospike roles with privileges.
	// +optional
	Roles []AerospikeRoleSpec `json:"roles,omitempty"`

	// Users defines Aerospike users with role bindings.
	// +optional
	Users []AerospikeUserSpec `json:"users,omitempty"`

	// AdminPolicy defines the client policy for admin operations.
	// +optional
	AdminPolicy *AerospikeClientAdminPolicy `json:"adminPolicy,omitempty"`
}

// AerospikeRoleSpec defines an Aerospike role.
type AerospikeRoleSpec struct {
	// Name is the role name.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Privileges is a list of privilege strings (e.g., "read-write", "sys-admin").
	// +kubebuilder:validation:MinItems=1
	Privileges []string `json:"privileges"`

	// Whitelist is a list of allowed CIDR ranges for this role.
	// +optional
	Whitelist []string `json:"whitelist,omitempty"`
}

// AerospikeUserSpec defines an Aerospike user.
type AerospikeUserSpec struct {
	// Name is the username.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// SecretName is the Kubernetes Secret containing the user's password.
	// The secret must have a "password" key.
	// +kubebuilder:validation:Required
	SecretName string `json:"secretName"`

	// Roles is a list of role names assigned to this user.
	// +kubebuilder:validation:MinItems=1
	Roles []string `json:"roles"`
}

// AerospikeClientAdminPolicy defines timeout settings for admin client operations.
type AerospikeClientAdminPolicy struct {
	// Timeout is the admin operation timeout in milliseconds.
	// +kubebuilder:default=2000
	// +optional
	Timeout int `json:"timeout,omitempty"`
}
