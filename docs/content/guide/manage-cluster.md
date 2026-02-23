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
