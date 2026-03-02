---
sidebar_position: 4
title: Storage Configuration
---

# Storage Configuration

Aerospike clusters use persistent storage for namespace data, secondary indexes, and logs. The `spec.storage` field controls how volumes are provisioned, initialized, and cleaned up.

---

## Global Volume Policies

Global policies apply to all volumes unless overridden at the individual volume level.

### filesystemVolumePolicy

Applied to PVC volumes with `volumeMode: Filesystem` (the default).

| Field | Options | Description |
|-------|---------|-------------|
| `initMethod` | `none`, `dd`, `deleteFiles`, `blkdiscard` | How to initialize the volume on first use |
| `wipeMethod` | `none`, `dd`, `deleteFiles`, `blkdiscard`, `blkdiscardWithHeaderCleanup`, `headerCleanup` | How to wipe the volume before pod deletion |
| `cascadeDelete` | `true` / `false` | Delete the PVC when the pod is deleted |

### blockVolumePolicy

Applied to PVC volumes with `volumeMode: Block`.

Same fields as `filesystemVolumePolicy`. The recommended `initMethod` for block devices is `blkdiscard` (fast, hardware-accelerated) and `blkdiscardWithHeaderCleanup` for `wipeMethod`.

### localStorageClasses

List of StorageClass names considered "local" storage (e.g., `local-path`, `openebs-hostpath`). Local PVCs are deleted before pod restart to ensure a clean state.

```yaml
storage:
  localStorageClasses:
    - local-path
    - openebs-hostpath
  deleteLocalStorageOnRestart: true
  cleanupThreads: 2
```

---

## Volume Types

### PVC — Filesystem (default)

Standard block-backed filesystem. Suitable for data, logs, and WAL files.

```yaml
volumes:
  - name: data-vol
    source:
      persistentVolume:
        storageClass: standard
        size: 50Gi
        volumeMode: Filesystem   # default
    aerospike:
      path: /opt/aerospike/data
```

### PVC — Block Device

Raw block volume (no filesystem). Use for Aerospike device-mode storage engines that access storage directly.

```yaml
volumes:
  - name: sindex-vol
    source:
      persistentVolume:
        storageClass: fast-ssd
        size: 20Gi
        volumeMode: Block
    aerospike:
      path: /opt/aerospike/sindex
```

### EmptyDir

Ephemeral in-pod storage. Useful for shared data between the Aerospike container and sidecars (e.g., exporters). Data is lost on pod restart.

```yaml
volumes:
  - name: shared-data
    source:
      emptyDir: {}
    aerospike:
      path: /opt/aerospike/shared
      subPath: "aerospike-data"
    sidecars:
      - containerName: aerospike-prometheus-exporter
        path: /shared
        readOnly: true
```

### HostPath

Mount a directory from the host node. For development/testing only — data is tied to a specific node.

```yaml
volumes:
  - name: host-logs
    source:
      hostPath:
        path: /var/log/aerospike
        type: DirectoryOrCreate
    aerospike:
      path: /opt/aerospike/logs
```

### CSI Volume (mountPropagation)

For CSI drivers that require `HostToContainer` propagation (e.g., FUSE-based storage).

```yaml
volumes:
  - name: csi-data
    source:
      persistentVolume:
        storageClass: csi-driver
        size: 100Gi
    aerospike:
      path: /opt/aerospike/csi
      mountPropagation: HostToContainer
```

---

## Per-Volume Override

Each volume can override the global policy with its own `initMethod`, `wipeMethod`, and `cascadeDelete`:

```yaml
storage:
  filesystemVolumePolicy:
    initMethod: deleteFiles   # global default
  volumes:
    - name: data-vol
      source:
        persistentVolume:
          storageClass: standard
          size: 50Gi
      aerospike:
        path: /opt/aerospike/data
      initMethod: dd          # override: use dd for this volume only
```

Priority: **per-volume** > **global policy** > **operator default**

---

## PVC Metadata

You can add custom labels and annotations to PVCs — useful for backup tools, storage policy, and cost attribution:

```yaml
volumes:
  - name: data-vol
    source:
      persistentVolume:
        storageClass: standard
        size: 50Gi
        metadata:
          labels:
            app.kubernetes.io/data-tier: "aerospike"
            backup-policy: "daily"
          annotations:
            volume.kubernetes.io/storage-provisioner: "ebs.csi.aws.com"
    aerospike:
      path: /opt/aerospike/data
```

---

## Complete Example

The following example (`config/samples/aerospike-cluster-storage-advanced.yaml`) demonstrates all volume types and policy options in a single cluster:

```yaml
apiVersion: acko.io/v1alpha1
kind: AerospikeCluster
metadata:
  name: aerospike-storage-advanced
  namespace: aerospike
spec:
  size: 3
  image: aerospike:ce-8.1.1.1

  storage:
    # Global volume policies: applied to all persistent volumes unless overridden per-volume
    filesystemVolumePolicy:
      initMethod: deleteFiles
      wipeMethod: deleteFiles
      cascadeDelete: true
    blockVolumePolicy:
      initMethod: blkdiscard
      wipeMethod: blkdiscardWithHeaderCleanup

    # Local storage recognition: delete local PVCs before pod restart
    localStorageClasses:
      - local-path
      - openebs-hostpath
    deleteLocalStorageOnRestart: true

    cleanupThreads: 2

    volumes:
      # PVC with custom metadata (labels and annotations on the PVC itself)
      - name: data-vol
        source:
          persistentVolume:
            storageClass: standard
            size: 50Gi
            volumeMode: Filesystem
            metadata:
              labels:
                app.kubernetes.io/data-tier: "aerospike"
                backup-policy: "daily"
              annotations:
                volume.kubernetes.io/storage-provisioner: "ebs.csi.aws.com"
        aerospike:
          path: /opt/aerospike/data
        # Per-volume override: uses dd instead of global policy's deleteFiles
        initMethod: dd

      # Block device volume (uses blockVolumePolicy defaults)
      - name: sindex-vol
        source:
          persistentVolume:
            storageClass: fast-ssd
            size: 20Gi
            volumeMode: Block
        aerospike:
          path: /opt/aerospike/sindex
        # initMethod not set → falls back to blockVolumePolicy.initMethod (blkdiscard)

      # Volume with advanced mount options
      - name: shared-data
        source:
          emptyDir: {}
        aerospike:
          path: /opt/aerospike/shared
          readOnly: false
          subPath: "aerospike-data"
        sidecars:
          - containerName: aerospike-prometheus-exporter
            path: /shared
            readOnly: true

      # HostPath volume (for development/testing only)
      - name: host-logs
        source:
          hostPath:
            path: /var/log/aerospike
            type: DirectoryOrCreate
        aerospike:
          path: /opt/aerospike/logs

      # Volume with mountPropagation
      - name: csi-data
        source:
          persistentVolume:
            storageClass: csi-driver
            size: 100Gi
        aerospike:
          path: /opt/aerospike/csi
          mountPropagation: HostToContainer

      # Local storage volume
      - name: local-data
        source:
          persistentVolume:
            storageClass: local-path
            size: 10Gi
        aerospike:
          path: /opt/aerospike/local
        initMethod: deleteFiles
        wipeMethod: deleteFiles

  aerospikeConfig:
    service:
      cluster-name: storage-advanced-demo
    namespaces:
      - name: test
        memory-size: 1073741824
        replication-factor: 2
        storage-engine:
          type: device
          devices:
            - /opt/aerospike/data/test.dat
          filesize: 4294967296
          data-in-memory: true
```

Apply it with:

```bash
kubectl apply -f config/samples/aerospike-cluster-storage-advanced.yaml
```
