---
sidebar_position: 1
title: AerospikeCECluster API Reference
---

# AerospikeCECluster API Reference

This page documents the `AerospikeCECluster` Custom Resource Definition (CRD) types.

**API Group:** `acko.io`
**API Version:** `v1alpha1`
**Kind:** `AerospikeCECluster`
**Short Names:** `asce`, `ascecluster`

---

## AerospikeCECluster

AerospikeCECluster is the Schema for the `aerospikececlusters` API. It manages the lifecycle of an Aerospike Community Edition cluster.

| Field | Type | Description |
|---|---|---|
| `apiVersion` | string | `acko.io/v1alpha1` |
| `kind` | string | `AerospikeCECluster` |
| `metadata` | [ObjectMeta](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/object-meta/) | Standard object metadata |
| `spec` | [AerospikeCEClusterSpec](#aerospikececlusterspec) | Desired state of the cluster |
| `status` | [AerospikeCEClusterStatus](#aerospikececlusterstatus) | Observed state of the cluster |

---

## AerospikeCEClusterSpec

Defines the desired state of an Aerospike CE cluster.

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `size` | int32 | Yes | — | Number of Aerospike pods. CE max: 8. |
| `image` | string | Yes | — | Aerospike CE container image (e.g., `aerospike:ce-8.1.1.1`). |
| `aerospikeConfig` | [AerospikeConfigSpec](#aerospikeconfigspec) | No | — | Raw Aerospike configuration map, converted to `aerospike.conf`. |
| `storage` | [AerospikeStorageSpec](#aerospikestoragespec) | No | — | Volume definitions for Aerospike pods. |
| `rackConfig` | [RackConfig](#rackconfig) | No | — | Rack-aware deployment topology. |
| `aerospikeNetworkPolicy` | [AerospikeNetworkPolicy](#aerospikenetworkpolicy) | No | — | Client access network configuration. |
| `podSpec` | [AerospikeCEPodSpec](#aerospikecepodspec) | No | — | Pod-level configuration. |
| `aerospikeAccessControl` | [AerospikeAccessControlSpec](#aerospikeaccesscontrolspec) | No | — | ACL roles and users. |
| `monitoring` | [AerospikeMonitoringSpec](#aerospikemonitoringspec) | No | — | Prometheus monitoring configuration. |
| `networkPolicyConfig` | [NetworkPolicyConfig](#networkpolicyconfig) | No | — | Automatic NetworkPolicy creation. |
| `bandwidthConfig` | [BandwidthConfig](#bandwidthconfig) | No | — | CNI bandwidth annotations. |
| `enableDynamicConfigUpdate` | *bool | No | — | Enable runtime config changes via `set-config`. |
| `rollingUpdateBatchSize` | *int32 | No | `1` | Number of pods to restart in parallel during rolling update. |
| `disablePDB` | *bool | No | `false` | Disable PodDisruptionBudget creation. |
| `maxUnavailable` | [IntOrString](https://pkg.go.dev/k8s.io/apimachinery/pkg/util/intstr#IntOrString) | No | `1` | Max pods unavailable during disruption. |
| `paused` | *bool | No | `false` | Stop reconciliation when true. |
| `seedsFinderServices` | [SeedsFinderServices](#seedsfinderservices) | No | — | LoadBalancer service for seed discovery. |
| `k8sNodeBlockList` | []string | No | — | Node names to exclude from scheduling. |
| `operations` | [][OperationSpec](#operationspec) | No | — | On-demand operations (WarmRestart, PodRestart). Max 1 at a time. |
| `validationPolicy` | [ValidationPolicySpec](#validationpolicyspec) | No | — | Controls webhook validation behavior. |
| `headlessService` | [AerospikeServiceSpec](#aerospikeservicespec) | No | — | Custom metadata for the headless service. |
| `podService` | [AerospikeServiceSpec](#aerospikeservicespec) | No | — | Custom metadata for per-pod services. Creates individual Service per pod when set. |
| `enableRackIDOverride` | *bool | No | `false` | Enable dynamic rack ID assignment via pod annotations. |
| `templateRef` | [TemplateRef](#templateref) | No | — | Reference to an `AerospikeCEClusterTemplate`. When set, the template spec is resolved and stored as a snapshot at creation time. |
| `overrides` | [AerospikeCEClusterTemplateSpec](./aerospikececlustertemplate#aerospikececlustertemplatetspec) | No | — | Fields that override the referenced template. Merge priority: overrides > template > operator defaults. |

---

## TemplateRef

Reference to an `AerospikeCEClusterTemplate` in the same namespace.

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name of the `AerospikeCEClusterTemplate` resource |

---

## TemplateSnapshotStatus

Recorded in `status.templateSnapshot` after a template is resolved.

| Field | Type | Description |
|---|---|---|
| `name` | string | Name of the referenced template |
| `resourceVersion` | string | ResourceVersion of the template at snapshot time |
| `snapshotTimestamp` | Time | When the snapshot was taken |
| `synced` | boolean | Whether the cluster uses the latest template version. Set to `false` when the template changes after the snapshot. |
| `spec` | object | Resolved template spec at snapshot time |

---

## AerospikeConfigSpec

Holds the raw Aerospike configuration as an unstructured JSON/YAML object. The operator converts this to `aerospike.conf` format.

This is a `map[string]interface{}` wrapper. Access via `.Value` in Go code. In YAML, write the Aerospike configuration directly:

```yaml
aerospikeConfig:
  service:
    cluster-name: my-cluster
    proto-fd-max: 15000
  network:
    service:
      port: 3000
    heartbeat:
      mode: mesh
      port: 3002
    fabric:
      port: 3001
  namespaces:
    - name: testns
      replication-factor: 2
      storage-engine:
        type: device
        file: /opt/aerospike/data/testns.dat
        filesize: 4294967296
  logging:
    - name: /var/log/aerospike/aerospike.log
      context: any info
```

---

## AerospikeCEClusterStatus

Observed state of the Aerospike CE cluster.

| Field | Type | Description |
|---|---|---|
| `phase` | string | Cluster phase: `InProgress`, `Completed`, or `Error`. |
| `size` | int32 | Current cluster size. |
| `conditions` | [][Condition](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/conditions/) | Latest observations of cluster state. |
| `pods` | map[string][AerospikePodStatus](#aerospikepodstatus) | Per-pod status information, keyed by pod name. |
| `observedGeneration` | int64 | Most recent generation observed by the controller. |
| `selector` | string | Label selector string for HPA compatibility. |
| `aerospikeConfig` | [AerospikeConfigSpec](#aerospikeconfigspec) | Last applied Aerospike configuration. |
| `operationStatus` | [OperationStatus](#operationstatus) | Current on-demand operation status. |

---

## AerospikePodStatus

Per-pod status information.

| Field | Type | Description |
|---|---|---|
| `podIP` | string | Pod IP address. |
| `hostIP` | string | Host node IP address. |
| `image` | string | Container image running on the pod. |
| `podPort` | int32 | Aerospike service port on the pod. |
| `servicePort` | int32 | Aerospike service port exposed via node/LB. |
| `rack` | int | Rack ID assigned to this pod. |
| `initializedVolumes` | []string | Volumes that have been initialized. |
| `isRunningAndReady` | bool | Whether the pod is running and ready. |
| `configHash` | string | SHA256 hash of the applied config. |
| `podSpecHash` | string | Hash of the pod template spec. |
| `dynamicConfigStatus` | string | Dynamic config update result: `Applied`, `Failed`, `Pending`, or empty. |
| `dirtyVolumes` | []string | Volumes needing initialization or cleanup. |

---

## AerospikeStorageSpec

Defines storage volumes for Aerospike pods.

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `volumes` | [][VolumeSpec](#volumespec) | No | — | List of volumes to attach. |
| `cleanupThreads` | int32 | No | `1` | Max threads for volume cleanup/init. |

---

## VolumeSpec

Defines a single volume attachment.

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `name` | string | Yes | — | Volume name. |
| `source` | [VolumeSource](#volumesource) | Yes | — | Volume source (PVC, emptyDir, secret, configMap). |
| `aerospike` | [AerospikeVolumeAttachment](#aerospikevolumeattachment) | No | — | Mount path in Aerospike container. |
| `sidecars` | [][VolumeAttachment](#volumeattachment) | No | — | Volume mounts for sidecar containers. |
| `initContainers` | [][VolumeAttachment](#volumeattachment) | No | — | Volume mounts for init containers. |
| `initMethod` | string | No | `none` | Init method: `none`, `deleteFiles`, `dd`, `blkdiscard`, `headerCleanup`. |
| `cascadeDelete` | bool | No | `false` | Delete PVC when CR is deleted. |

---

## VolumeSource

Describes the volume data source. Exactly one field should be set.

| Field | Type | Description |
|---|---|---|
| `persistentVolume` | [PersistentVolumeSpec](#persistentvolumespec) | Create a PVC. |
| `emptyDir` | [EmptyDirVolumeSource](https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/volume/#emptyDir) | Use emptyDir. |
| `secret` | [SecretVolumeSource](https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/volume/#secret) | Use a Kubernetes Secret. |
| `configMap` | [ConfigMapVolumeSource](https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/volume/#configMap) | Use a Kubernetes ConfigMap. |

---

## PersistentVolumeSpec

Defines a persistent volume claim template.

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `storageClass` | string | No | — | StorageClass name. |
| `volumeMode` | string | No | `Filesystem` | `Filesystem` or `Block`. |
| `size` | string | Yes | — | Storage size (e.g., `10Gi`). |
| `accessModes` | []string | No | — | Access modes (e.g., `ReadWriteOnce`). |
| `selector` | [LabelSelector](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/label-selector/) | No | — | Label selector for PV binding. |

---

## AerospikeVolumeAttachment

Defines how a volume is mounted in the Aerospike container.

| Field | Type | Required | Description |
|---|---|---|---|
| `path` | string | Yes | Mount path in the container. |

---

## VolumeAttachment

Defines a volume mount for sidecar or init containers.

| Field | Type | Required | Description |
|---|---|---|---|
| `containerName` | string | Yes | Target container name. |
| `path` | string | Yes | Mount path in the container. |

---

## AerospikeNetworkPolicy

Defines network access configuration.

| Field | Type | Default | Description |
|---|---|---|---|
| `accessType` | string | `pod` | Client access type: `pod`, `hostInternal`, `hostExternal`, `configuredIP`. |
| `alternateAccessType` | string | `pod` | Alternate access type. |
| `fabricType` | string | `pod` | Fabric (inter-node) network type. |
| `customAccessNetworkNames` | []string | — | Network names for `configuredIP` access. |
| `customAlternateAccessNetworkNames` | []string | — | Network names for `configuredIP` alternate access. |
| `customFabricNetworkNames` | []string | — | Network names for `configuredIP` fabric. |

---

## SeedsFinderServices

Configures external seed discovery via LoadBalancer.

| Field | Type | Description |
|---|---|---|
| `loadBalancer` | [LoadBalancerSpec](#loadbalancerspec) | LoadBalancer service configuration. |

---

## LoadBalancerSpec

Defines a LoadBalancer service.

| Field | Type | Default | Description |
|---|---|---|---|
| `annotations` | map[string]string | — | Service annotations. |
| `labels` | map[string]string | — | Service labels. |
| `externalTrafficPolicy` | string | — | `Cluster` or `Local`. |
| `port` | int32 | `3000` | External port. |
| `targetPort` | int32 | `3000` | Container target port. |
| `loadBalancerSourceRanges` | []string | — | Allowed source CIDRs. |

---

## AerospikeCEPodSpec

Pod-level customization for Aerospike pods.

| Field | Type | Description |
|---|---|---|
| `aerospikeContainer` | [AerospikeContainerSpec](#aerospikecontainerspec) | Aerospike container customization. |
| `sidecars` | [][Container](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#Container) | Sidecar containers. |
| `initContainers` | [][Container](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#Container) | Additional init containers. |
| `imagePullSecrets` | [][LocalObjectReference](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/local-object-reference/) | Image pull secrets. |
| `nodeSelector` | map[string]string | Node labels for scheduling. |
| `tolerations` | [][Toleration](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#scheduling) | Pod tolerations. |
| `affinity` | [Affinity](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#scheduling) | Affinity/anti-affinity rules. |
| `securityContext` | [PodSecurityContext](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#security-context) | Pod-level security attributes. |
| `serviceAccountName` | string | ServiceAccount name. |
| `dnsPolicy` | string | DNS policy for the pod. |
| `hostNetwork` | bool | Enable host networking. |
| `multiPodPerHost` | *bool | Allow multiple pods on the same node. |
| `terminationGracePeriodSeconds` | *int64 | Pod termination grace period. |
| `metadata` | [AerospikePodMetadata](#aerospikepodmetadata) | Additional pod labels/annotations. |

---

## AerospikeContainerSpec

Customizes the Aerospike server container.

| Field | Type | Description |
|---|---|---|
| `resources` | [ResourceRequirements](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#resources) | CPU/memory requests and limits. |
| `securityContext` | [SecurityContext](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#security-context-1) | Container-level security attributes. |

---

## AerospikePodMetadata

Extra labels and annotations for pods.

| Field | Type | Description |
|---|---|---|
| `labels` | map[string]string | Additional pod labels. |
| `annotations` | map[string]string | Additional pod annotations. |

---

## RackConfig

Defines rack-aware deployment configuration.

| Field | Type | Required | Description |
|---|---|---|---|
| `racks` | [][Rack](#rack) | Yes | List of rack definitions (min 1). |
| `namespaces` | []string | No | Aerospike namespace names that are rack-aware. |
| `scaleDownBatchSize` | [IntOrString](https://pkg.go.dev/k8s.io/apimachinery/pkg/util/intstr#IntOrString) | No | Pods to scale down simultaneously per rack. Int or percent string (e.g., `"25%"`). Default: 1. |
| `maxIgnorablePods` | [IntOrString](https://pkg.go.dev/k8s.io/apimachinery/pkg/util/intstr#IntOrString) | No | Max pending/failed pods to ignore during reconciliation. |
| `rollingUpdateBatchSize` | [IntOrString](https://pkg.go.dev/k8s.io/apimachinery/pkg/util/intstr#IntOrString) | No | Pods to restart simultaneously per rack. Int or percent string. Takes precedence over `spec.rollingUpdateBatchSize`. |

---

## Rack

Defines a single rack in the cluster topology.

| Field | Type | Required | Description |
|---|---|---|---|
| `id` | int | Yes | Unique rack identifier (>= 0). |
| `zone` | string | No | Zone label value (`topology.kubernetes.io/zone`). |
| `region` | string | No | Region label value (`topology.kubernetes.io/region`). |
| `nodeName` | string | No | Constrain to a specific node. |
| `rackLabel` | string | No | Custom label for rack affinity. Schedules to nodes with `acko.io/rack=<rackLabel>`. Must be unique across racks. |
| `revision` | string | No | Version identifier for controlled rack migrations. |
| `aerospikeConfig` | [AerospikeConfigSpec](#aerospikeconfigspec) | No | Per-rack Aerospike config override. |
| `storage` | [AerospikeStorageSpec](#aerospikestoragespec) | No | Per-rack storage override. |
| `podSpec` | [RackPodSpec](#rackpodspec) | No | Per-rack pod scheduling override. |

---

## RackPodSpec

Rack-level pod scheduling overrides.

| Field | Type | Description |
|---|---|---|
| `affinity` | [Affinity](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#scheduling) | Rack-level affinity override. |
| `tolerations` | [][Toleration](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#scheduling) | Rack-level tolerations override. |
| `nodeSelector` | map[string]string | Rack-level node selector override. |

---

## AerospikeAccessControlSpec

Defines ACL configuration.

| Field | Type | Description |
|---|---|---|
| `roles` | [][AerospikeRoleSpec](#aerospikerolespec) | Aerospike role definitions. |
| `users` | [][AerospikeUserSpec](#aerospikeuserspec) | Aerospike user definitions. |
| `adminPolicy` | [AerospikeClientAdminPolicy](#aerospikeclientadminpolicy) | Admin client timeout policy. |

---

## AerospikeRoleSpec

Defines an Aerospike role.

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Role name. |
| `privileges` | []string | Yes | Privilege strings: `read`, `write`, `read-write`, `read-write-udf`, `sys-admin`, `user-admin`, `data-admin`, `truncate`. Supports namespace scoping (e.g., `read-write.testns`). |
| `whitelist` | []string | No | Allowed CIDR ranges. |

---

## AerospikeUserSpec

Defines an Aerospike user.

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Username. |
| `secretName` | string | Yes | Kubernetes Secret name containing the password (key: `password`). |
| `roles` | []string | Yes | Assigned role names (min 1). |

---

## AerospikeClientAdminPolicy

Admin client timeout settings.

| Field | Type | Default | Description |
|---|---|---|---|
| `timeout` | int | `2000` | Admin operation timeout in milliseconds. |

---

## AerospikeMonitoringSpec

Prometheus monitoring configuration.

| Field | Type | Default | Description |
|---|---|---|---|
| `enabled` | bool | `false` | Enable Prometheus exporter sidecar. |
| `exporterImage` | string | `aerospike/aerospike-prometheus-exporter:latest` | Exporter container image. |
| `port` | int32 | `9145` | Metrics port. |
| `resources` | [ResourceRequirements](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#resources) | — | Exporter resource limits. |
| `serviceMonitor` | [ServiceMonitorSpec](#servicemonitorspec) | — | ServiceMonitor configuration. |

---

## ServiceMonitorSpec

ServiceMonitor configuration for Prometheus Operator.

| Field | Type | Default | Description |
|---|---|---|---|
| `enabled` | bool | `false` | Create ServiceMonitor resource. |
| `interval` | string | `30s` | Scrape interval. |
| `labels` | map[string]string | — | Additional labels for ServiceMonitor discovery. |

---

## NetworkPolicyConfig

Automatic NetworkPolicy creation.

| Field | Type | Default | Description |
|---|---|---|---|
| `enabled` | bool | `false` | Enable NetworkPolicy creation. |
| `type` | string | `kubernetes` | Policy type: `kubernetes` or `cilium`. |

---

## BandwidthConfig

Bandwidth annotations for CNI traffic shaping.

| Field | Type | Description |
|---|---|---|
| `ingress` | string | Max ingress bandwidth (e.g., `1Gbps`, `500Mbps`). |
| `egress` | string | Max egress bandwidth (e.g., `1Gbps`, `500Mbps`). |

---

## OperationSpec

Defines an on-demand operation to trigger on cluster pods.

| Field | Type | Required | Description |
|---|---|---|---|
| `kind` | string | Yes | Operation type: `WarmRestart` (SIGUSR1) or `PodRestart` (delete/recreate). |
| `id` | string | Yes | Unique operation identifier (1-20 characters). |
| `podList` | []string | No | Specific pod names to target. Empty means all pods. |

---

## OperationStatus

Tracks the status of an on-demand operation.

| Field | Type | Description |
|---|---|---|
| `id` | string | Operation identifier. |
| `kind` | string | Operation type: `WarmRestart` or `PodRestart`. |
| `phase` | string | Operation phase: `InProgress`, `Completed`, or `Error`. |
| `completedPods` | []string | Pods that have completed the operation. |
| `failedPods` | []string | Pods where the operation failed. |

---

## ValidationPolicySpec

Controls webhook validation behavior.

| Field | Type | Default | Description |
|---|---|---|---|
| `skipWorkDirValidate` | bool | `false` | Skip validation that the Aerospike work directory is on persistent storage. |

---

## AerospikeServiceSpec

Defines custom metadata for a Kubernetes Service.

| Field | Type | Required | Description |
|---|---|---|---|
| `metadata` | [AerospikeObjectMeta](#aerospikeobjectmeta) | No | Custom annotations and labels for the service. |

---

## AerospikeObjectMeta

Custom metadata for Kubernetes objects.

| Field | Type | Description |
|---|---|---|
| `annotations` | map[string]string | Custom annotations. |
| `labels` | map[string]string | Custom labels. |
