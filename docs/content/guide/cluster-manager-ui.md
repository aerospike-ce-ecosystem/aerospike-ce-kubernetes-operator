---
sidebar_position: 6
title: Cluster Manager UI
---

# Aerospike Cluster Manager UI

The [Aerospike Cluster Manager](https://github.com/KimSoungRyoul/aerospike-cluster-manager) is a web-based GUI for managing Aerospike CE clusters. It is bundled with the operator Helm chart and can be deployed alongside the operator as an optional component.

The UI includes an embedded PostgreSQL sidecar (with PVC) for storing cluster connection profiles.

---

## Installation

Enable the UI when installing the operator:

```bash
helm install acko oci://ghcr.io/kimsoungryoul/charts/aerospike-ce-kubernetes-operator \
  --namespace aerospike-operator --create-namespace \
  --set ui.enabled=true
```

Verify the UI pod is running:

```bash
kubectl -n aerospike-operator get pods -l app.kubernetes.io/component=ui
```

---

## Access the UI

### Port-Forward (Development)

```bash
kubectl -n aerospike-operator port-forward svc/acko-aerospike-ce-kubernetes-operator-ui 3000:3000
```

Open [http://localhost:3000](http://localhost:3000) in your browser.

:::tip
The service name follows the pattern `<release>-aerospike-ce-kubernetes-operator-ui`. If you used a different release name, adjust accordingly:
```bash
kubectl -n aerospike-operator port-forward svc/<release>-aerospike-ce-kubernetes-operator-ui 3000:3000
```
:::

### Ingress (Production)

For persistent external access, enable Ingress:

```bash
helm install acko oci://ghcr.io/kimsoungryoul/charts/aerospike-ce-kubernetes-operator \
  --namespace aerospike-operator --create-namespace \
  --set ui.enabled=true \
  --set ui.ingress.enabled=true \
  --set ui.ingress.className=nginx \
  --set "ui.ingress.hosts[0].host=aerospike-admin.example.com" \
  --set "ui.ingress.hosts[0].paths[0].path=/" \
  --set "ui.ingress.hosts[0].paths[0].pathType=Prefix"
```

---

## Configuration Options

| Parameter | Description | Default |
|-----------|-------------|---------|
| `ui.enabled` | Enable the Cluster Manager UI | `false` |
| `ui.replicaCount` | Number of UI replicas | `1` |
| `ui.image.repository` | UI container image | `ghcr.io/kimsoungryoul/aerospike-cluster-manager` |
| `ui.image.tag` | Image tag (defaults to Chart appVersion when empty) | `""` |
| `ui.service.type` | Service type (`ClusterIP`, `NodePort`, `LoadBalancer`) | `ClusterIP` |
| `ui.service.frontendPort` | Frontend (Next.js) port | `3000` |
| `ui.service.backendPort` | Backend (FastAPI) port | `8000` |
| `ui.postgresql.enabled` | Deploy embedded PostgreSQL sidecar | `true` |
| `ui.k8s.enabled` | Enable K8s cluster management features | `true` |
| `ui.ingress.enabled` | Create an Ingress for external access | `false` |
| `ui.persistence.enabled` | Enable PVC for PostgreSQL data | `true` |
| `ui.persistence.size` | PVC storage size | `1Gi` |
| `ui.env.databaseUrl` | External PostgreSQL URL (when `postgresql.enabled=false`) | `""` |
| `ui.rbac.create` | Create ClusterRole and ClusterRoleBinding for K8s API access | `true` |
| `ui.serviceAccount.create` | Create a ServiceAccount for the UI pod | `true` |
| `ui.networkPolicy.enabled` | Restrict network traffic to the UI pod | `false` |
| `ui.image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `ui.persistence.storageClassName` | Storage class for PostgreSQL PVC | `""` (default) |
| `ui.postgresql.existingSecret` | Use an existing Secret for database credentials | `""` |

:::tip
For the full list of configuration options (probes, security contexts, tolerations, affinity, autoscaling, etc.), run:
```bash
helm show values oci://ghcr.io/kimsoungryoul/charts/aerospike-ce-kubernetes-operator | grep -A 500 "^ui:"
```
:::

---

## Features

### Connection Management

Manage multiple Aerospike cluster connections with color-coded profiles. Each connection stores host, port, and optional credentials. The profiles are persisted in the embedded PostgreSQL database.

### Cluster Monitoring

Real-time dashboard showing TPS, client connections, and success rates. Monitor namespace usage, storage utilization, and cluster health at a glance.

### Record Browser

Browse, create, update, and delete records with pagination support. Navigate through namespaces and sets, and inspect individual records with full metadata display.

### Query Builder

Build and execute scan/query operations with predicates. Construct queries visually without writing AQL manually.

### K8s Cluster Management

When `ui.k8s.enabled=true` (the default), the UI provides Kubernetes-native cluster management:

- **Create clusters** -- Deploy new AerospikeCluster CRs with a guided wizard
- **Edit clusters** -- Modify running cluster settings (image, size, dynamic config, aerospike config) via an edit dialog
- **Scale clusters** -- Adjust cluster size (1-8 nodes for CE)
- **Monitor status** -- View cluster phase, conditions, and pod details
- **Manage templates** -- Browse available AerospikeClusterTemplates with sync status
- **Template snapshot** -- View resolved template spec with sync status (Synced/Out of Sync)
- **Trigger operations** -- Initiate warm restarts and pod restarts with optional pod selection
- **Pod selection** -- Select specific pods via checkboxes for targeted restart operations
- **Pause/Resume** -- Pause and resume reconciliation
- **Configure ACL** -- Set up access control with roles, users, and K8s secret-based credentials (with secrets picker dropdown)
- **Rolling update strategy** -- Configure batch size, max unavailable, and PDB settings
- **Monitor operations** -- View active operation status, completed/failed pods in real-time
- **Dynamic config status** -- View per-pod dynamic config status (Applied/Failed/Pending) and last restart reason
- **Track reconciliation** -- See reconcile error counts and failure reasons
- **View events** -- Browse K8s events timeline for each cluster (auto-refreshes during transitional phases)
- **Pod logs** -- View container logs directly from the pod table with tail lines selection, copy, and download
- **Export CR YAML** -- Copy the cluster's AerospikeCluster CR as clean YAML for debugging or migration
- **Health dashboard** -- At-a-glance cluster health: pod readiness, migration status, config state, availability, and rack distribution
- **Storage policies** -- Configure volume init method (deleteFiles/dd/blkdiscard/headerCleanup), wipe method, and cascade delete behavior for PVCs
- **Network access type** -- Choose how clients access the cluster: Pod IP (default), Host Internal, Host External, or Configured IP; configure fabric type for inter-node communication
- **Node block list** -- Specify Kubernetes nodes where Aerospike pods should not be scheduled (via the edit dialog)

:::info
K8s cluster management requires the UI service account to have RBAC access to AerospikeCluster resources. This is configured automatically when `ui.rbac.create=true` (the default).
:::

### Rack Configuration

The wizard includes a **Rack Config** step for multi-rack, zone-aware deployments:

- **Add/Remove Racks**: Configure multiple racks with unique IDs
- **Zone Affinity**: Select K8s availability zones from live node data
- **Pod Distribution**: Set max pods per node for each rack
- **Distribution Preview**: See estimated pod distribution across racks

Each rack creates a separate StatefulSet, enabling zone-aware high availability.

### Storage Policies

When using persistent storage (device mode), the wizard lets you configure:

- **Init Method**: How volumes are prepared on first use (`none`, `deleteFiles`, `dd`, `blkdiscard`, `headerCleanup`)
- **Wipe Method**: How dirty volumes are cleaned on pod restart (`none`, `deleteFiles`, `dd`, `blkdiscard`, `headerCleanup`, `blkdiscardWithHeaderCleanup`)
- **Cascade Delete**: Whether PVCs are automatically deleted when the cluster CR is deleted (default: enabled)

### Network Access

Configure client-to-cluster and node-to-node communication:

- **Client Access Type**: `pod` (default — uses Pod IP), `hostInternal` (node internal IP), `hostExternal` (node external IP), or `configuredIP` (annotation-based)
- **Fabric Type**: Network type for inter-node fabric communication (defaults to `pod`)

### Index Management

Create, view, and delete secondary indexes. Monitor index build progress and view index statistics.

### User/Role Management (ACL)

Manage Aerospike users and roles through the UI. Create users, assign roles, and update passwords without using command-line tools.

### UDF Management

Upload, view, and delete User-Defined Functions (Lua modules) registered with the Aerospike cluster.

### AQL Terminal

Execute AQL commands directly from the browser with syntax highlighting and result formatting.

### Light/Dark Theme

Toggle between light and dark themes to suit your preference.

---

## Using an External PostgreSQL

To use an existing PostgreSQL instance instead of the embedded sidecar:

```bash
helm install acko oci://ghcr.io/kimsoungryoul/charts/aerospike-ce-kubernetes-operator \
  --namespace aerospike-operator --create-namespace \
  --set ui.enabled=true \
  --set ui.postgresql.enabled=false \
  --set ui.env.databaseUrl="postgresql://user:pass@db-host:5432/aerospike_manager"
```

:::tip
You can also use an existing Kubernetes Secret for the database credentials by setting `ui.postgresql.existingSecret` to the Secret name. The Secret must contain `POSTGRES_PASSWORD` and `DATABASE_URL` keys.
:::

---

## Security

### RBAC

When `ui.rbac.create=true` (the default), the Helm chart creates a ClusterRole and ClusterRoleBinding granting the UI service account:

- **Read/write** access to `AerospikeCluster` resources (create, scale, update, delete)
- **Read-only** access to `AerospikeClusterTemplate` resources (browse templates)
- **Read-only** access to Pods, Services, Events, and Namespaces (for cluster monitoring, events timeline, and wizard dropdowns)
- **Read-only** access to Pod logs (`pods/log`) for viewing container logs from the UI
- **List-only** access to Secrets (name enumeration for ACL credential selection — contents are never read)
- **List-only** access to StorageClasses (for storage wizard dropdowns)
- **Read-only** access to Nodes (`get`, `list`) for retrieving availability zone information used in rack configuration

### Pod Security

The UI runs as non-root by default (`runAsUser: 1001`) with a read-only root filesystem disabled to support Next.js runtime requirements. Privilege escalation is blocked and all Linux capabilities are dropped.

### Network Policy

Restrict traffic to the UI pod:

```bash
helm install acko oci://ghcr.io/kimsoungryoul/charts/aerospike-ce-kubernetes-operator \
  --namespace aerospike-operator --create-namespace \
  --set ui.enabled=true \
  --set ui.networkPolicy.enabled=true
```

---

## Full Example

Deploy the operator with the UI, monitoring, and Ingress all enabled:

```bash
helm install acko oci://ghcr.io/kimsoungryoul/charts/aerospike-ce-kubernetes-operator \
  --namespace aerospike-operator --create-namespace \
  --set ui.enabled=true \
  --set ui.ingress.enabled=true \
  --set ui.ingress.className=nginx \
  --set "ui.ingress.hosts[0].host=aerospike-admin.example.com" \
  --set "ui.ingress.hosts[0].paths[0].path=/" \
  --set "ui.ingress.hosts[0].paths[0].pathType=Prefix" \
  --set serviceMonitor.enabled=true \
  --set grafanaDashboard.enabled=true
```
