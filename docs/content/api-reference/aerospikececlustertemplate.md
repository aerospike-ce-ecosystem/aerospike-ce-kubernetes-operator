---
sidebar_position: 2
title: AerospikeCEClusterTemplate API Reference
---

# AerospikeCEClusterTemplate API Reference

This page documents the `AerospikeCEClusterTemplate` Custom Resource Definition (CRD) types.

**API Group:** `acko.io`
**API Version:** `v1alpha1`
**Kind:** `AerospikeCEClusterTemplate`
**Short Names:** `ascet`, `ascetemplate`

---

## Overview

`AerospikeCEClusterTemplate` is a reusable configuration profile for `AerospikeCECluster`. It lets you define shared settings (scheduling, storage, resources, Aerospike config) once and reference them from multiple clusters via `spec.templateRef`.

**Snapshot strategy:** The template spec is copied into `status.templateSnapshot` at cluster creation time. Subsequent template changes are **not** automatically propagated. To resync, set the annotation `acko.io/resync-template: "true"` on the cluster object.

---

## AerospikeCEClusterTemplate

| Field | Type | Description |
|---|---|---|
| `apiVersion` | string | `acko.io/v1alpha1` |
| `kind` | string | `AerospikeCEClusterTemplate` |
| `metadata` | [ObjectMeta](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/object-meta/) | Standard object metadata |
| `spec` | [AerospikeCEClusterTemplateSpec](#aerospikececlustertemplatetspec) | Configuration profile |
| `status` | [AerospikeCEClusterTemplateStatus](#aerospikececlustertemplatestatus) | Observed state |

---

## AerospikeCEClusterTemplateSpec

| Field | Type | Description |
|---|---|---|
| `aerospikeConfig` | [TemplateAerospikeConfig](#templateaerospikeconfig) | Aerospike configuration defaults |
| `scheduling` | [TemplateScheduling](#templatescheduling) | Pod scheduling defaults |
| `storage` | [TemplateStorage](#templatestorage) | Data volume defaults |
| `resources` | [ResourceRequirements](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#resources) | Container CPU/memory defaults |
| `rackConfig` | [TemplateRackConfig](#templaterackconfig) | Rack configuration defaults |

---

## TemplateAerospikeConfig

| Field | Type | Description |
|---|---|---|
| `namespaceDefaults` | object | Base configuration merged into every namespace defined in the cluster's `aerospikeConfig.namespaces`. Cluster-level settings override these defaults. |
| `service` | object | Defaults for the `service` section of `aerospikeConfig`. Cluster-level settings override these defaults. |
| `network` | [TemplateNetworkConfig](#templatenetworkconfig) | Network configuration defaults |

---

## TemplateNetworkConfig

| Field | Type | Description |
|---|---|---|
| `heartbeat` | [TemplateHeartbeatConfig](#templateheartbeatconfig) | Heartbeat configuration defaults |

---

## TemplateHeartbeatConfig

| Field | Type | Description |
|---|---|---|
| `mode` | string | Heartbeat mode. Must be `mesh` for CE. |
| `interval` | integer | Heartbeat interval in milliseconds |
| `timeout` | integer | Heartbeat timeout in milliseconds |

---

## TemplateScheduling

| Field | Type | Description |
|---|---|---|
| `podAntiAffinityLevel` | string | Pod anti-affinity policy: `none`, `preferred`, or `required`. `required` enforces one Aerospike pod per node. |
| `nodeAffinity` | [NodeAffinity](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#NodeAffinity) | Node affinity rules for pod scheduling |
| `tolerations` | []Toleration | Pod scheduling tolerations |
| `topologySpreadConstraints` | []TopologySpreadConstraint | How pods are spread across topology domains |
| `podManagementPolicy` | string | StatefulSet pod management policy: `OrderedReady` or `Parallel` |

### PodAntiAffinityLevel values

| Value | Behavior |
|---|---|
| `none` | No anti-affinity rules are injected |
| `preferred` | Soft rule (weight=100): prefer spreading pods across nodes |
| `required` | Hard rule: exactly one Aerospike pod per node |

---

## TemplateStorage

| Field | Type | Description |
|---|---|---|
| `storageClassName` | string | Kubernetes StorageClass for the data PVC |
| `volumeMode` | string | `Filesystem` (default) or `Block` |
| `accessModes` | []string | PVC access modes (default: `ReadWriteOnce`) |
| `resources` | [VolumeResourceRequirements](https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/persistent-volume-claim-v1/#VolumeResourceRequirements) | Storage size request |
| `localPVRequired` | boolean | When `true`, asserts that the StorageClass uses `WaitForFirstConsumer` binding mode (local PV) |

---

## TemplateRackConfig

| Field | Type | Description |
|---|---|---|
| `maxRacksPerNode` | integer | Maximum racks per Kubernetes node. When set to `1`, a warning is raised if `podAntiAffinityLevel` is not `required`. |

---

## AerospikeCEClusterTemplateStatus

| Field | Type | Description |
|---|---|---|
| `usedBy` | []string | List of `AerospikeCECluster` names that reference this template |

---

## Using Templates in a Cluster

Reference a template via `spec.templateRef` in `AerospikeCECluster`:

```yaml
apiVersion: acko.io/v1alpha1
kind: AerospikeCECluster
metadata:
  name: my-cluster
spec:
  size: 3
  image: aerospike:ce-8.1.1.1
  templateRef:
    name: prod          # references AerospikeCEClusterTemplate named "prod"
  overrides:            # optional: override specific fields from the template
    resources:
      requests:
        cpu: "1"
        memory: "2Gi"
      limits:
        cpu: "2"
        memory: "2Gi"
```

### Resync after template update

Template changes are not automatically applied to clusters. To resync:

```bash
kubectl annotate aerospikececluster my-cluster acko.io/resync-template=true
```

The operator will re-fetch the template, update `status.templateSnapshot`, emit a `TemplateApplied` event, and remove the annotation.

---

## Validation Rules

| Rule | Description |
|---|---|
| V-T01 | `scheduling.podAntiAffinityLevel` must be `none`, `preferred`, or `required` |
| V-T02 | `rackConfig.maxRacksPerNode` must be >= 0 |
| V-T03 | `storage.localPVRequired=true` without `storageClassName` raises a warning |
| V-T04 | For Guaranteed QoS, resource requests should equal limits (warning) |
| V-T05 | `scheduling.podManagementPolicy` must be `OrderedReady` or `Parallel` |
