---
sidebar_position: 3
title: Manage Cluster
---

# Manage an Aerospike Cluster

This guide covers day-2 operations: scaling, updates, configuration changes, and troubleshooting.

## Scaling

Change `spec.size` to scale the cluster up or down.

```bash
kubectl -n aerospike patch asce aerospike-ce-3node --type merge -p '{"spec":{"size":5}}'
```

The operator creates or removes pods to match the desired size. For multi-rack deployments, pods are distributed evenly across racks.

:::warning
`spec.size` must not exceed 8 (CE limit). `replication-factor` must not exceed the new cluster size.
:::

## Rolling Updates

### Image Update

Change `spec.image` to trigger a rolling restart with the new image:

```yaml
spec:
  image: aerospike:ce-8.1.1.1   # Change to new version
```

The operator uses `OnDelete` update strategy — it deletes pods one at a time (or in batches) and waits for the new pod to become ready before proceeding.

### Batch Size

Control how many pods restart simultaneously:

```yaml
spec:
  rollingUpdateBatchSize: 2   # Restart 2 pods at a time (default: 1)
```

For rack-aware deployments, you can set batch size per rack in `rackConfig`. This takes precedence over `spec.rollingUpdateBatchSize`:

```yaml
spec:
  rackConfig:
    rollingUpdateBatchSize: "50%"   # Restart 50% of pods per rack simultaneously
```

#### Batch Size: Integer vs Percentage

Batch size accepts either an integer or a percentage string:

| Format | Example | Behavior (size=6) |
|--------|---------|-------------------|
| Integer | `2` | Exactly 2 pods at a time |
| Percentage | `"50%"` | 50% of 6 = 3 pods at a time |
| Percentage | `"25%"` | 25% of 6 = 2 pods at a time (rounded up) |

:::tip
A percentage string must include the `%` suffix (e.g., `"50%"`). The percentage is calculated against the total pod count per rack, rounded up to at least 1.
:::

### Scale Down Batch Size

Control how many pods are removed simultaneously per rack during scale-down:

```yaml
spec:
  rackConfig:
    scaleDownBatchSize: 2            # Remove 2 pods per rack at a time
    # scaleDownBatchSize: "25%"      # Or use percentage
```

`scaleDownBatchSize` applies **per rack** during scale-down operations. This prevents removing too many pods at once, which could cause data unavailability.

### Max Ignorable Pods

Allow reconciliation to continue even when some pods are in Pending/Failed state:

```yaml
spec:
  rackConfig:
    maxIgnorablePods: 1   # Ignore up to 1 stuck pod and continue reconciling
```

This is useful when pods are stuck due to scheduling issues and you don't want to block the entire reconciliation.

#### Batch Size Summary

| Field | Scope | Default | Description |
|-------|-------|---------|-------------|
| `spec.rollingUpdateBatchSize` | Cluster-wide | 1 | Pods restarted simultaneously during rolling update |
| `rackConfig.rollingUpdateBatchSize` | Per rack | inherits from spec | Overrides cluster-level batch size per rack |
| `rackConfig.scaleDownBatchSize` | Per rack | all at once | Pods removed simultaneously during scale-down |
| `rackConfig.maxIgnorablePods` | Per rack | 0 | Stuck pods to ignore during reconciliation |

## Configuration Updates

### Static Configuration Changes

Any change to `spec.aerospikeConfig` triggers a rolling restart to apply the new configuration. The operator regenerates `aerospike.conf` in each pod's ConfigMap.

### Dynamic Configuration Updates

Enable runtime configuration changes without pod restarts:

```yaml
spec:
  enableDynamicConfigUpdate: true
```

When enabled, the operator uses Aerospike's `set-config` command to apply configuration changes at runtime where possible. Only changes that cannot be applied dynamically trigger a rolling restart.

#### Which settings are dynamic?

Most Aerospike service and namespace parameters are dynamically configurable. Examples include:

| Category | Dynamic Parameters |
|----------|-------------------|
| Service | `proto-fd-max`, `transaction-pending-limit`, `batch-max-buffers-per-queue` |
| Namespace | `high-water-memory-pct`, `high-water-disk-pct`, `stop-writes-pct`, `nsup-period`, `default-ttl` |
| Not Dynamic | `replication-factor`, `storage-engine type`, `name` (requires restart) |

#### Checking dynamicConfigStatus

After a config change with `enableDynamicConfigUpdate: true`, check per-pod status:

```bash
kubectl -n aerospike get asce aerospike-ce-3node -o jsonpath='{.status.pods}' | jq '.[] | {name: .podName, dynamicConfig: .dynamicConfigStatus}'
```

| Status | Meaning |
|--------|---------|
| `Applied` | Dynamic config was applied successfully at runtime |
| `Failed` | Dynamic update failed — a rolling restart will be triggered |
| `Pending` | Waiting for the operator to apply the change |
| (empty) | No dynamic config change was attempted |

## Pausing Reconciliation

Temporarily stop the operator from reconciling:

```yaml
spec:
  paused: true
```

While paused, the operator skips all reconciliation. Set back to `false` (or remove the field) to resume.

```bash
# Pause
kubectl -n aerospike patch asce aerospike-ce-3node --type merge -p '{"spec":{"paused":true}}'

# Resume
kubectl -n aerospike patch asce aerospike-ce-3node --type merge -p '{"spec":{"paused":null}}'
```

## On-Demand Operations

Trigger pod restarts declaratively via `spec.operations`. Only one operation can be active at a time.

### WarmRestart

Sends SIGUSR1 to the Aerospike process for a graceful restart without pod deletion:

```yaml
spec:
  operations:
    - kind: WarmRestart
      id: "config-reload-v2"       # Unique ID (1-20 chars)
      podList:                      # Optional: empty = all pods
        - aerospike-ce-3node-0
        - aerospike-ce-3node-1
```

### PodRestart

Deletes and recreates pods (cold restart):

```yaml
spec:
  operations:
    - kind: PodRestart
      id: "cold-restart-01"
      podList:
        - aerospike-ce-3node-2
```

### Checking Operation Status

```bash
kubectl -n aerospike get asce aerospike-ce-3node -o jsonpath='{.status.operationStatus}' | jq .
```

The status includes `phase` (`InProgress`, `Completed`, `Error`), `completedPods`, and `failedPods`.

:::warning
- Operations cannot be modified while InProgress
- The operation `id` must be unique (1-20 characters)
- Remove the operation from the spec after it completes
:::

## Service Customization

### Headless Service Metadata

Add custom annotations and labels to the headless service (used for pod discovery):

```yaml
spec:
  headlessService:
    metadata:
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8687"
      labels:
        monitoring: enabled
```

### Per-Pod Services

When `podService` is set, the operator creates an individual ClusterIP Service for each pod, enabling direct pod-level access:

```yaml
spec:
  podService:
    metadata:
      annotations:
        external-dns.alpha.kubernetes.io/hostname: "aero.example.com"
      labels:
        service-type: pod-local
```

**Use case:** External DNS integration, pod-level load balancing, or direct client access to specific pods.

## Validation Policy

Control webhook validation behavior:

```yaml
spec:
  validationPolicy:
    skipWorkDirValidate: true   # Skip work directory PV validation
```

| Field | Default | Description |
|---|---|---|
| `skipWorkDirValidate` | `false` | Skip validation that work directory is on persistent storage |

This is useful for development environments or in-memory deployments that don't require persistent work directories.

## Rack ID Override

Enable dynamic rack ID assignment via pod annotations:

```yaml
spec:
  enableRackIDOverride: true
```

When enabled, the operator allows rack IDs to be overridden by pod annotations instead of being strictly managed by the operator. This is useful for manual rack management scenarios.

## Storage

### Volume Types

| Source | Use Case |
|---|---|
| `persistentVolume` | Durable data (survives pod restarts) |
| `emptyDir` | Temporary scratch space |
| `secret` | Credentials, TLS certs |
| `configMap` | Custom config files |
| `hostPath` | Node-local path (dev/test only) |

### Global Volume Policy

Set default policies for all persistent volumes by category (filesystem or block). Per-volume settings always override the global policy.

```yaml
storage:
  filesystemVolumePolicy:
    initMethod: deleteFiles
    wipeMethod: deleteFiles
    cascadeDelete: true
  blockVolumePolicy:
    initMethod: blkdiscard
    wipeMethod: blkdiscardWithHeaderCleanup
```

The operator resolves settings with this precedence:
1. **Per-volume** `initMethod` / `wipeMethod` / `cascadeDelete`
2. **Global policy** (based on `volumeMode`: `filesystemVolumePolicy` or `blockVolumePolicy`)
3. **Default**: `none` / `none` / `false`

### Cascade Delete

When `cascadeDelete: true`, PVCs are automatically deleted when the AerospikeCECluster CR is deleted. This can be set per-volume or via global policy.

```yaml
storage:
  # Global: delete all filesystem PVCs on CR deletion
  filesystemVolumePolicy:
    cascadeDelete: true
  volumes:
    - name: data-vol
      source:
        persistentVolume:
          storageClass: standard
          size: 10Gi
      # Inherits cascadeDelete: true from filesystemVolumePolicy
```

### Volume Init Methods

| Method | Description |
|---|---|
| `none` | No initialization (default) |
| `deleteFiles` | Delete all files in the volume |
| `dd` | Zero-fill the volume with `dd` |
| `blkdiscard` | Discard blocks (block devices only) |
| `headerCleanup` | Clear Aerospike file headers only |

```yaml
storage:
  volumes:
    - name: data-vol
      initMethod: deleteFiles
```

### Wipe Methods

Wipe methods are similar to init methods but apply to **dirty volumes** (volumes that need cleanup after unclean shutdown). The `wipeMethod` field supports the following values:

| Method | Description |
|---|---|
| `none` | No wiping (default) |
| `deleteFiles` | Delete all files in the volume |
| `dd` | Zero-fill the device using `dd` |
| `blkdiscard` | Discard all blocks on the device |
| `headerCleanup` | Clear Aerospike file headers only |
| `blkdiscardWithHeaderCleanup` | Discard blocks and then clear Aerospike headers |

```yaml
storage:
  volumes:
    - name: data-vol
      wipeMethod: headerCleanup
```

### HostPath Volumes

:::warning
HostPath volumes are **not recommended for production**. Data is tied to a specific node and is not portable across pod rescheduling.
:::

```yaml
storage:
  volumes:
    - name: host-logs
      source:
        hostPath:
          path: /var/log/aerospike
          type: DirectoryOrCreate
      aerospike:
        path: /opt/aerospike/logs
```

### PVC Custom Metadata

Add custom labels and annotations to PersistentVolumeClaims:

```yaml
storage:
  volumes:
    - name: data-vol
      source:
        persistentVolume:
          storageClass: standard
          size: 50Gi
          metadata:
            labels:
              backup-policy: "daily"
            annotations:
              volume.kubernetes.io/storage-provisioner: "ebs.csi.aws.com"
```

### Volume Mount Options

Advanced mount options for Aerospike and sidecar containers:

```yaml
storage:
  volumes:
    - name: shared-data
      source:
        emptyDir: {}
      aerospike:
        path: /opt/aerospike/shared
        readOnly: false
        subPath: "aerospike-data"       # Mount a sub-directory
        mountPropagation: HostToContainer
      sidecars:
        - containerName: exporter
          path: /shared
          readOnly: true
```

| Option | Description |
|---|---|
| `readOnly` | Mount the volume as read-only |
| `subPath` | Mount a specific sub-directory of the volume |
| `subPathExpr` | Like subPath but supports environment variable expansion |
| `mountPropagation` | Control mount propagation (`None`, `HostToContainer`, `Bidirectional`) |

:::note
`subPath` and `subPathExpr` are mutually exclusive.
:::

### Local Storage

Mark storage classes as local to enable special handling during pod restarts:

```yaml
storage:
  localStorageClasses:
    - local-path
    - openebs-hostpath
  deleteLocalStorageOnRestart: true
```

When `deleteLocalStorageOnRestart: true`, the operator deletes PVCs backed by local storage classes **before** pod deletion during a cold restart. This forces re-provisioning on the new node, which is necessary because local storage is node-bound.

## Network Configuration

### Access Type

Control how clients discover and connect to Aerospike nodes:

| Type | Description |
|---|---|
| `pod` | Pod IP (default, in-cluster clients) |
| `hostInternal` | Node internal IP |
| `hostExternal` | Node external IP |
| `configuredIP` | Custom IP from pod annotations |

```yaml
spec:
  aerospikeNetworkPolicy:
    accessType: pod
    alternateAccessType: hostExternal
    fabricType: pod
```

### LoadBalancer

Expose the cluster via a LoadBalancer service:

```yaml
spec:
  seedsFinderServices:
    loadBalancer:
      port: 3000
      targetPort: 3000
      annotations:
        service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
      loadBalancerSourceRanges:
        - "10.0.0.0/8"
```

### NetworkPolicy

Enable automatic NetworkPolicy creation:

```yaml
spec:
  networkPolicyConfig:
    enabled: true
    type: kubernetes    # or "cilium" for CiliumNetworkPolicy
```

### Bandwidth Shaping

Set bandwidth limits via CNI annotations (e.g., Cilium):

```yaml
spec:
  bandwidthConfig:
    ingress: "1Gbps"
    egress: "1Gbps"
```

## Pod Disruption Budget

By default, the operator creates a PodDisruptionBudget to protect the cluster during maintenance.

### Disable PDB

```yaml
spec:
  disablePDB: true
```

### Custom MaxUnavailable

```yaml
spec:
  maxUnavailable: 1         # Can be integer or percentage string like "25%"
```

## Host Network

Enable host networking for direct node port access:

```yaml
spec:
  podSpec:
    hostNetwork: true
    # Defaults applied automatically:
    #   multiPodPerHost: false
    #   dnsPolicy: ClusterFirstWithHostNet
```

## Rack Label Scheduling

Use `rackLabel` to schedule pods to nodes with a specific label. The operator sets a node affinity for `acko.io/rack=<rackLabel>`:

```yaml
spec:
  rackConfig:
    racks:
      - id: 1
        rackLabel: zone-a    # Pods scheduled to nodes with acko.io/rack=zone-a
      - id: 2
        rackLabel: zone-b
      - id: 3
        rackLabel: zone-c
```

:::warning
`rackLabel` values must be unique across racks.
:::

You can also attach a `revision` to each rack for controlled migrations:

```yaml
spec:
  rackConfig:
    racks:
      - id: 1
        rackLabel: zone-a
        revision: "v1.0"     # Version identifier for migration tracking
```

## Node Scheduling

### Node Selector

```yaml
spec:
  podSpec:
    nodeSelector:
      node-type: aerospike
```

### Tolerations

```yaml
spec:
  podSpec:
    tolerations:
      - key: "dedicated"
        operator: "Equal"
        value: "aerospike"
        effect: "NoSchedule"
```

### Node Block List

Prevent scheduling on specific nodes:

```yaml
spec:
  k8sNodeBlockList:
    - node-maintenance-01
    - node-maintenance-02
```

## Troubleshooting

### Check Cluster Phase

```bash
kubectl -n aerospike get asce
```

| Phase | Meaning |
|---|---|
| `InProgress` | Reconciliation is in progress |
| `Completed` | Cluster is healthy and up-to-date |
| `Error` | Reconciliation encountered an error |

### Check Conditions

```bash
kubectl -n aerospike get asce aerospike-ce-3node -o jsonpath='{.status.conditions}' | jq .
```

### Check Pod Status

```bash
kubectl -n aerospike get asce aerospike-ce-3node -o jsonpath='{.status.pods}' | jq .
```

Each pod status includes:

| Field | Description |
|---|---|
| `podIP` | Pod IP address |
| `hostIP` | Node IP address |
| `image` | Running container image |
| `rack` | Rack ID |
| `isRunningAndReady` | Whether the pod is healthy |
| `configHash` | SHA256 of applied config |
| `dynamicConfigStatus` | `Applied`, `Failed`, `Pending`, or empty |

### Check Operator Logs

```bash
kubectl -n aerospike-operator logs -l control-plane=controller-manager -f
```

### Common Issues

**Phase stuck at InProgress:**
- Check operator logs for error details
- Verify storage class exists: `kubectl get sc`
- Verify image is pullable: `kubectl -n aerospike describe pod <pod-name>`

**Pod CrashLoopBackOff:**
- Check Aerospike logs: `kubectl -n aerospike logs <pod-name> -c aerospike`
- Verify `aerospikeConfig` is valid (namespace names, storage paths)

**Webhook rejection:**
- Read the error message — the webhook validates CE constraints
- Check [CE Validation Rules](./create-cluster#ce-validation-rules)
