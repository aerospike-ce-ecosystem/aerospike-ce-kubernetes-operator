---
sidebar_position: 5
title: Advanced Configuration
---

# Advanced Configuration

This guide covers advanced operator features for fine-tuning Aerospike cluster behavior, pod scheduling, rack-level overrides, and validation policies.

---

## EnableRackIDOverride

By default, the operator assigns rack IDs to pods based on the StatefulSet ordinal and the rack definition. When `enableRackIDOverride` is enabled, the operator allows dynamic rack ID assignment via pod annotations, giving you manual control over which rack a pod belongs to.

This is useful when you need to migrate pods between racks without scaling down and back up, or when integrating with external orchestration tools that manage rack placement.

```yaml
apiVersion: acko.io/v1alpha1
kind: AerospikeCluster
metadata:
  name: aerospike-cluster
spec:
  size: 4
  image: aerospike:ce-8.1.1.1
  enableRackIDOverride: true
  rackConfig:
    racks:
      - id: 1
        zone: us-east-1a
      - id: 2
        zone: us-east-1b
  aerospikeConfig:
    service:
      cluster-name: my-cluster
    namespaces:
      - name: test
        memory-size: 1073741824
        replication-factor: 2
        storage-engine:
          type: memory
    network:
      service:
        port: 3000
      heartbeat:
        port: 3002
      fabric:
        port: 3001
```

:::caution
Use `enableRackIDOverride` only when you have a specific need for manual rack assignment. In most cases, the operator's automatic rack distribution is sufficient and safer.
:::

---

## ValidationPolicy

The `validationPolicy` field controls which validation checks the operator performs. Currently it supports one option: `skipWorkDirValidate`.

### skipWorkDirValidate

By default, the operator validates that the Aerospike work directory (`/opt/aerospike`) is backed by persistent storage. This prevents data loss from ephemeral volumes. Setting `skipWorkDirValidate: true` disables this check.

**When to use:**

- Development or test clusters where persistence is not required
- Clusters using `emptyDir` or host-path volumes for the work directory
- Ephemeral benchmarking environments

```yaml
apiVersion: acko.io/v1alpha1
kind: AerospikeCluster
metadata:
  name: aerospike-dev
spec:
  size: 1
  image: aerospike:ce-8.1.1.1
  validationPolicy:
    skipWorkDirValidate: true
  aerospikeConfig:
    service:
      cluster-name: dev-cluster
    namespaces:
      - name: test
        memory-size: 1073741824
        replication-factor: 1
        storage-engine:
          type: memory
    network:
      service:
        port: 3000
      heartbeat:
        port: 3002
      fabric:
        port: 3001
```

---

## Pod Scheduling

The `spec.podSpec` field provides full control over Kubernetes scheduling primitives. All standard Kubernetes scheduling fields are supported.

### nodeSelector

Constrain Aerospike pods to nodes with specific labels.

```yaml
spec:
  podSpec:
    nodeSelector:
      disktype: ssd
      kubernetes.io/arch: amd64
```

### Tolerations

Allow Aerospike pods to be scheduled on tainted nodes.

```yaml
spec:
  podSpec:
    tolerations:
      - key: "dedicated"
        operator: "Equal"
        value: "aerospike"
        effect: "NoSchedule"
      - key: "node.kubernetes.io/disk-pressure"
        operator: "Exists"
        effect: "NoSchedule"
```

### Affinity

Define pod affinity and anti-affinity rules for precise scheduling control.

```yaml
spec:
  podSpec:
    affinity:
      podAntiAffinity:
        preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchExpressions:
                  - key: app
                    operator: In
                    values:
                      - aerospike
              topologyKey: kubernetes.io/hostname
      nodeAffinity:
        requiredDuringSchedulingIgnoredDuringExecution:
          nodeSelectorTerms:
            - matchExpressions:
                - key: topology.kubernetes.io/zone
                  operator: In
                  values:
                    - us-east-1a
                    - us-east-1b
```

:::tip
When `podSpec.multiPodPerHost` is `false` (or `nil` with `hostNetwork: true`), the operator automatically injects a `RequiredDuringSchedulingIgnoredDuringExecution` pod anti-affinity rule to ensure one Aerospike pod per node. You do not need to manually configure this.
:::

### topologySpreadConstraints

Distribute pods evenly across failure domains (zones, nodes, etc.).

```yaml
spec:
  podSpec:
    topologySpreadConstraints:
      - maxSkew: 1
        topologyKey: topology.kubernetes.io/zone
        whenUnsatisfiable: DoNotSchedule
        labelSelector:
          matchLabels:
            app: aerospike
      - maxSkew: 1
        topologyKey: kubernetes.io/hostname
        whenUnsatisfiable: ScheduleAnyway
        labelSelector:
          matchLabels:
            app: aerospike
```

### podManagementPolicy

Controls how pods are created during scaling. Valid values:

| Value | Behavior |
|-------|----------|
| `OrderedReady` | Pods are created sequentially, each must be ready before the next is started (default) |
| `Parallel` | All pods are created simultaneously, useful for faster initial deployment |

```yaml
spec:
  podSpec:
    podManagementPolicy: Parallel
```

### Full Pod Scheduling Example

```yaml
apiVersion: acko.io/v1alpha1
kind: AerospikeCluster
metadata:
  name: aerospike-production
spec:
  size: 6
  image: aerospike:ce-8.1.1.1
  podSpec:
    podManagementPolicy: OrderedReady
    nodeSelector:
      disktype: ssd
    tolerations:
      - key: "dedicated"
        operator: "Equal"
        value: "aerospike"
        effect: "NoSchedule"
    affinity:
      podAntiAffinity:
        requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchExpressions:
                - key: app
                  operator: In
                  values:
                    - aerospike
            topologyKey: kubernetes.io/hostname
    topologySpreadConstraints:
      - maxSkew: 1
        topologyKey: topology.kubernetes.io/zone
        whenUnsatisfiable: DoNotSchedule
        labelSelector:
          matchLabels:
            app: aerospike
    terminationGracePeriodSeconds: 600
  aerospikeConfig:
    service:
      cluster-name: production
    namespaces:
      - name: data
        memory-size: 4294967296
        replication-factor: 2
        storage-engine:
          type: memory
    network:
      service:
        port: 3000
      heartbeat:
        port: 3002
      fabric:
        port: 3001
```

---

## Rack-level Overrides

Each rack in `spec.rackConfig.racks` can override cluster-level settings for `aerospikeConfig`, `storage`, and `podSpec`. This enables heterogeneous configurations across failure domains.

### Per-rack aerospikeConfig

Override Aerospike configuration at the rack level. The rack-level config is merged with the cluster-level config, with rack values taking precedence.

```yaml
spec:
  aerospikeConfig:
    service:
      cluster-name: my-cluster
    namespaces:
      - name: data
        memory-size: 4294967296
        replication-factor: 2
        storage-engine:
          type: memory
    network:
      service:
        port: 3000
      heartbeat:
        port: 3002
      fabric:
        port: 3001
  rackConfig:
    racks:
      - id: 1
        aerospikeConfig:
          namespaces:
            - name: data
              memory-size: 8589934592
              replication-factor: 2
              storage-engine:
                type: memory
      - id: 2
```

In this example, rack 1 uses 8 GB of memory for the `data` namespace while rack 2 inherits the cluster-level 4 GB setting.

### Per-rack storage

Override the storage configuration for a specific rack, for example to use a different StorageClass in different availability zones.

```yaml
spec:
  storage:
    volumes:
      - name: data-vol
        storageClass: gp3
        size: 50Gi
        path: /opt/aerospike/data
        volumeMode: Filesystem
  rackConfig:
    racks:
      - id: 1
        storage:
          volumes:
            - name: data-vol
              storageClass: io2
              size: 100Gi
              path: /opt/aerospike/data
              volumeMode: Filesystem
      - id: 2
```

### Per-rack podSpec (affinity, tolerations, nodeSelector)

The `Rack.podSpec` field (`RackPodSpec`) supports three scheduling overrides:

| Field | Type | Description |
|-------|------|-------------|
| `affinity` | `corev1.Affinity` | Override cluster-level affinity rules |
| `tolerations` | `[]corev1.Toleration` | Override cluster-level tolerations |
| `nodeSelector` | `map[string]string` | Override cluster-level node selector |

```yaml
spec:
  rackConfig:
    racks:
      - id: 1
        zone: us-east-1a
        podSpec:
          nodeSelector:
            topology.kubernetes.io/zone: us-east-1a
          tolerations:
            - key: "zone-a-dedicated"
              operator: "Exists"
              effect: "NoSchedule"
          affinity:
            nodeAffinity:
              requiredDuringSchedulingIgnoredDuringExecution:
                nodeSelectorTerms:
                  - matchExpressions:
                      - key: node.kubernetes.io/instance-type
                        operator: In
                        values:
                          - m5.2xlarge
                          - m5.4xlarge
      - id: 2
        zone: us-east-1b
        podSpec:
          nodeSelector:
            topology.kubernetes.io/zone: us-east-1b
```

### Rack topology shortcuts

Each rack also provides shortcut fields for common scheduling scenarios:

| Field | Description |
|-------|-------------|
| `zone` | Equivalent to a node affinity on `topology.kubernetes.io/zone` |
| `region` | Equivalent to a node affinity on `topology.kubernetes.io/region` |
| `nodeName` | Constrains the rack to a specific Kubernetes node |
| `rackLabel` | Schedules pods to nodes with `acko.io/rack=<rackLabel>` |

```yaml
spec:
  rackConfig:
    racks:
      - id: 1
        zone: us-east-1a
      - id: 2
        zone: us-east-1b
      - id: 3
        region: us-west-2
        rackLabel: high-perf
```

---

## Rack Configuration

The `spec.rackConfig` top-level fields control rack-wide behavior for namespace awareness, scaling, and rolling updates.

### namespaces

Specifies which Aerospike namespaces are rack-aware. If empty, all namespaces use the default replication factor. CE supports a maximum of 2 namespaces.

```yaml
spec:
  rackConfig:
    namespaces:
      - data
      - cache
    racks:
      - id: 1
      - id: 2
```

### scaleDownBatchSize

Controls how many pods are removed simultaneously per rack during scale-down operations. Accepts an absolute integer or a percentage string. Defaults to 1.

```yaml
spec:
  rackConfig:
    scaleDownBatchSize: 2
    racks:
      - id: 1
      - id: 2
```

With a percentage:

```yaml
spec:
  rackConfig:
    scaleDownBatchSize: "25%"
    racks:
      - id: 1
      - id: 2
```

### maxIgnorablePods

The maximum number of pending or failed pods to ignore during reconciliation. This is useful when pods are stuck due to scheduling issues (e.g., insufficient resources, node affinity mismatch) and you want the operator to continue reconciling healthy pods. Accepts an absolute integer or a percentage string.

```yaml
spec:
  rackConfig:
    maxIgnorablePods: 1
    racks:
      - id: 1
      - id: 2
```

### rollingUpdateBatchSize

Controls how many pods are restarted simultaneously per rack during a rolling restart. Accepts an absolute integer or a percentage string. Defaults to 1. This field takes precedence over `spec.rollingUpdateBatchSize` when set.

```yaml
spec:
  rackConfig:
    rollingUpdateBatchSize: 2
    racks:
      - id: 1
      - id: 2
```

:::note
There are two `rollingUpdateBatchSize` fields:
- `spec.rollingUpdateBatchSize` (integer only, cluster-wide default)
- `spec.rackConfig.rollingUpdateBatchSize` (integer or percentage, per-rack override, takes precedence)
:::

---

## Pod Metadata

Add custom labels and annotations to Aerospike pods via `spec.podSpec.metadata`. These are applied to every pod managed by the operator, in addition to the operator's own labels.

This is useful for:

- Service mesh injection (e.g., Istio sidecar annotations)
- Monitoring label selectors
- Cost allocation tags
- External tool integration

```yaml
spec:
  podSpec:
    metadata:
      labels:
        team: platform
        cost-center: "12345"
        environment: production
      annotations:
        sidecar.istio.io/inject: "true"
        prometheus.io/scrape: "true"
        prometheus.io/port: "9145"
```

---

## ReadinessGateEnabled

When `readinessGateEnabled` is set to `true`, the operator injects a custom Pod Readiness Gate with the condition type `acko.io/aerospike-ready` into each pod's spec. The operator patches the pod's `status.conditions` to set this gate to `True` only after the Aerospike node has:

1. Joined the cluster mesh
2. Finished all pending data migrations

Until the readiness gate is satisfied, the pod is excluded from Service endpoints, preventing traffic from being routed to nodes that are not fully ready to serve requests.

Defaults to `false` for backward compatibility.

```yaml
apiVersion: acko.io/v1alpha1
kind: AerospikeCluster
metadata:
  name: aerospike-with-readiness-gate
spec:
  size: 3
  image: aerospike:ce-8.1.1.1
  podSpec:
    readinessGateEnabled: true
  aerospikeConfig:
    service:
      cluster-name: my-cluster
    namespaces:
      - name: data
        memory-size: 2147483648
        replication-factor: 2
        storage-engine:
          type: memory
    network:
      service:
        port: 3000
      heartbeat:
        port: 3002
      fabric:
        port: 3001
```

You can verify the readiness gate status on individual pods:

```bash
kubectl get pod aerospike-with-readiness-gate-1-0 -o jsonpath='{.status.conditions[?(@.type=="acko.io/aerospike-ready")]}'
```

The per-pod status field `readinessGateSatisfied` in `status.pods` also reflects whether the gate is currently `True`.

:::tip
Enable readiness gates in production clusters to ensure zero-downtime rolling updates. During a rolling restart, the new pod will not receive client traffic until it has fully joined the cluster and completed data migrations.
:::
