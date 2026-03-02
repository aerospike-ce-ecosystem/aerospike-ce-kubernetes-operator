---
sidebar_position: 1
title: Installation
---

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

# Installation

This guide covers two methods to install the ACKO operator.

## Prerequisites

- Kubernetes cluster v1.26+
- kubectl configured to access the cluster
- [cert-manager](https://cert-manager.io/) installed (required for webhook TLS)

### cert-manager

cert-manager is required for webhook TLS. Install it before the operator:

```bash
helm repo add jetstack https://charts.jetstack.io
helm repo update
helm install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --set crds.enabled=true
```

Wait for cert-manager to be ready:

```bash
kubectl -n cert-manager wait --for=condition=Available deployment/cert-manager --timeout=60s
kubectl -n cert-manager wait --for=condition=Available deployment/cert-manager-webhook --timeout=60s
```

## Install the Operator

<Tabs groupId="install-method">
<TabItem value="helm-oci" label="Helm OCI (Recommended)" default>

The simplest installation method using the published OCI Helm chart.

```bash
helm install aerospike-ce-operator oci://ghcr.io/kimsoungryoul/charts/aerospike-ce-operator \
  -n aerospike-operator --create-namespace
```

### Customizing Helm Values

You can override default values:

```bash
helm install aerospike-ce-operator oci://ghcr.io/kimsoungryoul/charts/aerospike-ce-operator \
  -n aerospike-operator --create-namespace \
  --set replicaCount=2 \
  --set resources.limits.memory=256Mi
```

To see all available values:

```bash
helm show values oci://ghcr.io/kimsoungryoul/charts/aerospike-ce-operator
```

</TabItem>
<TabItem value="helm-gitops" label="Helm + GitOps (ArgoCD / Flux)">

For GitOps environments, install CRDs separately so they can be managed independently
of the operator lifecycle.

**Step 1: Install CRDs once per cluster**

```bash
helm install aerospike-ce-operator-crds oci://ghcr.io/kimsoungryoul/charts/aerospike-ce-operator-crds \
  --version 0.1.0
```

CRDs carry the `helm.sh/resource-policy: keep` annotation — they are **not** deleted
on `helm uninstall`, protecting your cluster data.

**Step 2: Install the operator (skip CRD installation)**

```bash
helm install aerospike-ce-operator oci://ghcr.io/kimsoungryoul/charts/aerospike-ce-operator \
  --version 0.1.0 \
  --set crds.install=false \
  -n aerospike-operator --create-namespace
```

**ArgoCD example (sync-wave)**

```yaml
# Application 1: CRDs — never auto-prune
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: aerospike-ce-operator-crds
  annotations:
    argocd.argoproj.io/sync-options: Replace=true
spec:
  source:
    repoURL: ghcr.io/kimsoungryoul/charts
    chart: aerospike-ce-operator-crds
    targetRevision: "0.1.0"
  syncPolicy:
    automated:
      prune: false
      selfHeal: true
---
# Application 2: Operator
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: aerospike-ce-operator
spec:
  source:
    repoURL: ghcr.io/kimsoungryoul/charts
    chart: aerospike-ce-operator
    targetRevision: "0.1.0"
    helm:
      values: |
        crds:
          install: false
  destination:
    namespace: aerospike-operator
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

**Flux example**

```yaml
# HelmRepository (OCI) — shared by both HelmReleases
apiVersion: source.toolkit.fluxcd.io/v1
kind: HelmRepository
metadata:
  name: aerospike-ce-operator
  namespace: flux-system
spec:
  type: oci
  url: oci://ghcr.io/kimsoungryoul/charts
---
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: aerospike-ce-operator-crds
  namespace: flux-system
spec:
  chart:
    spec:
      chart: aerospike-ce-operator-crds
      version: "0.1.0"
      sourceRef:
        kind: HelmRepository
        name: aerospike-ce-operator
  install:
    crds: CreateReplace
  upgrade:
    crds: CreateReplace
---
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: aerospike-ce-operator
  namespace: flux-system
spec:
  dependsOn:
    - name: aerospike-ce-operator-crds
  targetNamespace: aerospike-operator
  chart:
    spec:
      chart: aerospike-ce-operator
      version: "0.1.0"
      sourceRef:
        kind: HelmRepository
        name: aerospike-ce-operator
  values:
    crds:
      install: false
```

</TabItem>
<TabItem value="local-build" label="Local Build">

For developers and contributors building from source.

```bash
git clone https://github.com/KimSoungRyoul/aerospike-ce-kubernetes-operator.git
cd aerospike-ce-kubernetes-operator

# Build and push the operator image to your registry
make docker-build docker-push IMG=<your-registry>/aerospike-ce-operator:latest

# Install CRDs
make install

# Deploy the operator
make deploy IMG=<your-registry>/aerospike-ce-operator:latest
```

</TabItem>
</Tabs>

## Monitoring (Optional)

The Helm chart includes built-in support for Prometheus Operator monitoring resources.
All monitoring features are **disabled by default** and require [Prometheus Operator](https://github.com/prometheus-operator/prometheus-operator) (commonly installed via [kube-prometheus-stack](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack)).

### ServiceMonitor

Creates a `ServiceMonitor` resource so Prometheus automatically scrapes operator metrics.

```bash
helm install aerospike-ce-operator oci://ghcr.io/kimsoungryoul/charts/aerospike-ce-operator \
  -n aerospike-operator --create-namespace \
  --set serviceMonitor.enabled=true \
  --set serviceMonitor.additionalLabels.release=prometheus
```

:::tip
The `additionalLabels.release=prometheus` label must match your Prometheus Operator's `serviceMonitorSelector`. Check with:
```bash
kubectl get prometheus -A -o jsonpath='{.items[*].spec.serviceMonitorSelector}'
```
:::

| Parameter | Default | Description |
|-----------|---------|-------------|
| `serviceMonitor.enabled` | `false` | Create ServiceMonitor resource |
| `serviceMonitor.interval` | — | Scrape interval (e.g., `"30s"`) |
| `serviceMonitor.scrapeTimeout` | — | Scrape timeout (e.g., `"10s"`) |
| `serviceMonitor.additionalLabels` | `{}` | Extra labels for Prometheus selector matching |

### PrometheusRule

Creates a `PrometheusRule` resource with built-in alerting rules for the operator.

```bash
helm install aerospike-ce-operator oci://ghcr.io/kimsoungryoul/charts/aerospike-ce-operator \
  -n aerospike-operator --create-namespace \
  --set serviceMonitor.enabled=true \
  --set prometheusRule.enabled=true
```

**Built-in alerts** (used when `prometheusRule.rules` is empty):

| Alert | Condition | Severity |
|-------|-----------|----------|
| `AerospikeOperatorDown` | Operator unreachable for 5m | critical |
| `AerospikeOperatorReconcileErrors` | Reconcile error rate > 0 for 15m | warning |
| `AerospikeOperatorSlowReconcile` | p99 reconcile time > 60s for 10m | warning |
| `AerospikeOperatorWorkQueueDepth` | Queue depth > 10 for 10m | warning |
| `AerospikeOperatorHighMemory` | Memory > 256Mi for 10m | warning |
| `AerospikeOperatorPodRestarts` | > 3 restarts in 1h | warning |

To provide **custom rules** instead of the built-in defaults, use a `values.yaml` file:

```yaml
prometheusRule:
  enabled: true
  rules:
    - alert: CustomAerospikeAlert
      expr: up{job="aerospike"} == 0
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "Custom Aerospike alert"
```

| Parameter | Default | Description |
|-----------|---------|-------------|
| `prometheusRule.enabled` | `false` | Create PrometheusRule resource |
| `prometheusRule.additionalLabels` | `{}` | Extra labels for Prometheus selector matching |
| `prometheusRule.rules` | `[]` | Custom rules; when empty, built-in defaults are used |

### Grafana Dashboard

Creates a `ConfigMap` with a pre-built Grafana dashboard. Requires the [Grafana sidecar](https://github.com/grafana/helm-charts/tree/main/charts/grafana#sidecar-for-dashboards) to be configured for auto-discovery.

```bash
helm install aerospike-ce-operator oci://ghcr.io/kimsoungryoul/charts/aerospike-ce-operator \
  -n aerospike-operator --create-namespace \
  --set grafanaDashboard.enabled=true
```

The dashboard includes panels for: **Reconcile Rate**, **Reconcile Errors**, **Reconcile Duration (p99/p50)**, **Work Queue Depth**, **Operator Memory Usage**, and **Operator CPU Usage**.

| Parameter | Default | Description |
|-----------|---------|-------------|
| `grafanaDashboard.enabled` | `false` | Create dashboard ConfigMap |
| `grafanaDashboard.sidecarLabel` | `grafana_dashboard` | Label key for Grafana sidecar discovery |
| `grafanaDashboard.sidecarLabelValue` | `"1"` | Label value for Grafana sidecar discovery |
| `grafanaDashboard.folder` | `""` | Grafana folder name for organizing dashboards |

### Setting Up Grafana with Dashboard Auto-Discovery

A step-by-step guide to install Grafana with sidecar enabled and view the operator dashboard via port-forward.

**1. Add Grafana Helm repository:**

```bash
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update
```

**2. Install Grafana with sidecar enabled:**

```bash
helm install grafana grafana/grafana \
  -n monitoring --create-namespace \
  --set sidecar.dashboards.enabled=true \
  --set sidecar.dashboards.label=grafana_dashboard \
  --set sidecar.dashboards.labelValue="1" \
  --set sidecar.dashboards.searchNamespace=ALL \
  --set sidecar.datasources.enabled=true
```

:::info
`sidecar.dashboards.searchNamespace=ALL` enables the sidecar to discover dashboard ConfigMaps across **all namespaces**, including `aerospike-operator` where the operator chart deploys the dashboard ConfigMap. Without this, the sidecar only watches its own namespace.
:::

**3. Install (or upgrade) the operator with the dashboard enabled:**

```bash
helm install aerospike-ce-operator oci://ghcr.io/kimsoungryoul/charts/aerospike-ce-operator \
  -n aerospike-operator --create-namespace \
  --set grafanaDashboard.enabled=true
```

**4. Get the Grafana admin password:**

```bash
kubectl -n monitoring get secret grafana -o jsonpath="{.data.admin-password}" | base64 -d; echo
```

**5. Port-forward to access Grafana:**

```bash
kubectl -n monitoring port-forward svc/grafana 3000:80
```

**6.** Open [http://localhost:3000](http://localhost:3000) in your browser. Log in with:
- **Username:** `admin`
- **Password:** the value from step 4

The **"Aerospike CE Operator"** dashboard will appear automatically under **Dashboards**. If you set `grafanaDashboard.folder`, it will be organized in the specified folder.

:::tip
If the dashboard does not appear, verify the ConfigMap was created and has the correct label:
```bash
kubectl -n aerospike-operator get configmap -l grafana_dashboard=1
```
:::

### Full Monitoring Stack Example

Enable all monitoring features at once:

```bash
helm install aerospike-ce-operator oci://ghcr.io/kimsoungryoul/charts/aerospike-ce-operator \
  -n aerospike-operator --create-namespace \
  --set serviceMonitor.enabled=true \
  --set serviceMonitor.additionalLabels.release=prometheus \
  --set prometheusRule.enabled=true \
  --set grafanaDashboard.enabled=true \
  --set grafanaDashboard.folder=Aerospike
```

## Cluster Manager UI (Optional)

[Aerospike Cluster Manager](https://github.com/KimSoungRyoul/aerospike-cluster-manager) is a web-based GUI tool for browsing records, managing indexes, executing AQL queries, and monitoring your Aerospike CE cluster.

**Features:** Cluster dashboard, namespace/set browsing, record CRUD, query builder, secondary index management, user/role admin, UDF management, AQL terminal.

The UI is bundled with the Helm chart and deployed alongside the operator. It includes an embedded PostgreSQL sidecar (with PVC) for storing cluster connection profiles.

### Enable the UI

Add `--set ui.enabled=true` to your Helm install command:

```bash
helm install aerospike-ce-operator oci://ghcr.io/kimsoungryoul/charts/aerospike-ce-operator \
  -n aerospike-operator --create-namespace \
  --set ui.enabled=true
```

### Access via Port-Forward

```bash
kubectl -n aerospike-operator port-forward svc/aerospike-ce-operator-ui 3000:3000
```

Open [http://localhost:3000](http://localhost:3000) in your browser.

:::tip
The service name is `<release>-aerospike-ce-operator-ui`. If you used a custom release name, adjust accordingly:
```bash
kubectl -n aerospike-operator port-forward svc/<release>-aerospike-ce-operator-ui 3000:3000
```
:::

### Access via Ingress

For persistent external access, enable Ingress:

```bash
helm install aerospike-ce-operator oci://ghcr.io/kimsoungryoul/charts/aerospike-ce-operator \
  -n aerospike-operator --create-namespace \
  --set ui.enabled=true \
  --set ui.ingress.enabled=true \
  --set ui.ingress.className=nginx \
  --set "ui.ingress.hosts[0].host=aerospike-admin.example.com" \
  --set "ui.ingress.hosts[0].paths[0].path=/" \
  --set "ui.ingress.hosts[0].paths[0].pathType=Prefix"
```

### Key Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `ui.enabled` | `false` | Enable the Cluster Manager UI |
| `ui.replicaCount` | `1` | Number of UI replicas |
| `ui.service.type` | `ClusterIP` | Service type (`ClusterIP`, `NodePort`, `LoadBalancer`) |
| `ui.service.frontendPort` | `3000` | Frontend (Next.js) port |
| `ui.service.backendPort` | `8000` | Backend (FastAPI) port |
| `ui.ingress.enabled` | `false` | Create an Ingress for external access |
| `ui.postgresql.enabled` | `true` | Deploy embedded PostgreSQL sidecar |
| `ui.persistence.enabled` | `true` | Enable PVC for PostgreSQL data |
| `ui.persistence.size` | `1Gi` | PVC storage size |
| `ui.k8s.enabled` | `true` | Enable K8s cluster management (Create Cluster from UI) |
| `ui.env.databaseUrl` | `""` | External PostgreSQL URL (when `postgresql.enabled=false`) |

### Using an External PostgreSQL

To use an existing PostgreSQL instance instead of the embedded sidecar:

```bash
helm install aerospike-ce-operator oci://ghcr.io/kimsoungryoul/charts/aerospike-ce-operator \
  -n aerospike-operator --create-namespace \
  --set ui.enabled=true \
  --set ui.postgresql.enabled=false \
  --set ui.env.databaseUrl="postgresql://user:pass@db-host:5432/aerospike_manager"
```

## Verify Installation

Check the operator pod is running:

```bash
kubectl -n aerospike-operator get pods
```

Expected output:

```
NAME                                    READY   STATUS    RESTARTS   AGE
aerospike-ce-operator-controller-manager-xxxxx-yyyyy     1/1     Running   0          30s
```

Check the CRD is registered:

```bash
kubectl get crd aerospikeclusters.acko.io
```

## Quick Start: Full Installation Script

A single copy-paste script that sets up everything from scratch on a Kind cluster — cert-manager, Prometheus, the operator (with all monitoring enabled), Grafana, a sample Aerospike cluster, and verification.

:::info Prerequisites
- [Kind](https://kind.sigs.k8s.io/) installed
- [Helm](https://helm.sh/) v3 installed
- [kubectl](https://kubernetes.io/docs/tasks/tools/) installed

Create a Kind cluster first:
```bash
kind create cluster --config kind-config.yaml --name kind
```
:::

```bash
# =============================================================================
# 1. Install cert-manager
# =============================================================================
helm repo add jetstack https://charts.jetstack.io
helm repo update jetstack
helm install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --set crds.enabled=true \
  --wait

# =============================================================================
# 2. Install Prometheus Operator (kube-prometheus-stack)
# =============================================================================
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update prometheus-community
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --set grafana.enabled=false \
  --wait

# =============================================================================
# 3. Install Aerospike Operator (all monitoring enabled)
# =============================================================================
helm install aerospike-ce-operator oci://ghcr.io/kimsoungryoul/charts/aerospike-ce-operator \
  -n aerospike-operator --create-namespace \
  --set serviceMonitor.enabled=true \
  --set serviceMonitor.additionalLabels.release=prometheus \
  --set prometheusRule.enabled=true \
  --set grafanaDashboard.enabled=true \
  --set grafanaDashboard.folder=Aerospike \
  --wait

# =============================================================================
# 4. Install Grafana with sidecar dashboard auto-discovery
# =============================================================================
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update grafana
helm install grafana grafana/grafana \
  --namespace monitoring \
  --set sidecar.dashboards.enabled=true \
  --set sidecar.dashboards.label=grafana_dashboard \
  --set sidecar.dashboards.labelValue="1" \
  --set sidecar.dashboards.searchNamespace=ALL \
  --set sidecar.datasources.enabled=true \
  --wait

# =============================================================================
# 5. Deploy an Aerospike CE cluster
# =============================================================================
kubectl create namespace aerospike --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f config/samples/acko_v1alpha1_aerospikecluster.yaml

echo "Waiting for Aerospike pod to be ready..."
kubectl -n aerospike wait --for=condition=Ready pod/aerospike-ce-basic-0-0 --timeout=120s

# =============================================================================
# 6. Verify: run asinfo inside the Aerospike pod
# =============================================================================
echo "=== Aerospike cluster info ==="
kubectl -n aerospike exec -it aerospike-ce-basic-0-0 -- asinfo -v status
kubectl -n aerospike exec -it aerospike-ce-basic-0-0 -- asinfo -v build

# =============================================================================
# 7. Port-forward Grafana (access at http://localhost:3000)
# =============================================================================
GRAFANA_PASSWORD=$(kubectl -n monitoring get secret grafana \
  -o jsonpath="{.data.admin-password}" | base64 -d)
echo ""
echo "=== Grafana ==="
echo "URL:      http://localhost:3000"
echo "Username: admin"
echo "Password: ${GRAFANA_PASSWORD}"
echo ""
kubectl -n monitoring port-forward svc/grafana 3000:80
```

:::tip
The script uses `--wait` on each Helm install so subsequent steps don't start until the previous component is fully ready. If you want to run this in CI without the final `port-forward`, remove the last command.
:::

## Uninstall

:::warning
Always delete AerospikeCluster resources before uninstalling the operator. Removing the operator first will leave orphaned StatefulSets and PVCs.
:::

<Tabs groupId="install-method">
<TabItem value="helm-oci" label="Helm" default>

```bash
# Delete all Aerospike clusters first
kubectl delete asc --all --all-namespaces

# Uninstall the operator
helm uninstall aerospike-ce-operator -n aerospike-operator

# (Optional) Uninstall CRDs — WARNING: this deletes all AerospikeCluster data
# helm uninstall aerospike-ce-operator-crds

# (Optional) Delete the namespace
kubectl delete namespace aerospike-operator
```

</TabItem>
<TabItem value="local-build" label="Local Build">

```bash
# Delete all Aerospike clusters first
kubectl delete asc --all --all-namespaces

# Remove the operator
make undeploy

# Remove CRDs
make uninstall
```

</TabItem>
</Tabs>
