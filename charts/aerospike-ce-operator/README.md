# Aerospike CE Kubernetes Operator Helm Chart

Helm chart for deploying the Aerospike Community Edition Kubernetes Operator.

## Prerequisites

### 1. cert-manager (Required for webhooks)

The operator uses cert-manager to provision TLS certificates for admission webhooks.
Install cert-manager **before** installing this chart:

```bash
# Install cert-manager with CRDs
helm repo add jetstack https://charts.jetstack.io
helm repo update
helm install cert-manager jetstack/cert-manager \
  --namespace cert-manager --create-namespace \
  --set crds.enabled=true
```

Verify cert-manager is running:
```bash
kubectl -n cert-manager get pods
```

> **Alternative**: If you don't want cert-manager, disable it and provide a TLS secret manually:
> ```bash
> helm install acko ./charts/acko \
>   --set certManager.enabled=false \
>   --set webhookTlsSecret=my-webhook-tls
> ```

### 2. Prometheus Operator (Optional — for monitoring)

If you want to use `ServiceMonitor`, `PrometheusRule`, or Grafana dashboards,
install Prometheus Operator (kube-prometheus-stack) first:

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install kube-prometheus-stack prometheus-community/kube-prometheus-stack \
  --namespace monitoring --create-namespace \
  --set crds.enabled=true
```

## Installation

### Method 1: Single chart (Recommended for most users)

CRDs and the operator are installed together. `crds.install=true` is the default.

```bash
helm install acko oci://ghcr.io/kimsoungryoul/charts/acko \
  --version 0.1.0 \
  --namespace aerospike-operator --create-namespace
```

### Method 2: Separate CRD chart (GitOps / ArgoCD / Flux)

Install `acko-crds` once, then the operator separately. This is the recommended
approach for GitOps workflows to control CRD lifecycle independently.

```bash
# Step 1: Install CRDs (once per cluster)
helm install acko-crds oci://ghcr.io/kimsoungryoul/charts/acko-crds \
  --version 0.1.0

# Step 2: Install operator (skip CRD installation)
helm install acko oci://ghcr.io/kimsoungryoul/charts/acko \
  --version 0.1.0 \
  --set crds.install=false \
  --namespace aerospike-operator --create-namespace
```

### From Local Chart

```bash
helm install acko ./charts/acko \
  --namespace aerospike-operator --create-namespace
```

### With monitoring enabled

```bash
helm install acko oci://ghcr.io/kimsoungryoul/charts/acko \
  --version 0.1.0 \
  --namespace aerospike-operator --create-namespace \
  --set serviceMonitor.enabled=true \
  --set prometheusRule.enabled=true \
  --set grafanaDashboard.enabled=true
```

### With Cilium network policy

```bash
helm install acko oci://ghcr.io/kimsoungryoul/charts/acko \
  --version 0.1.0 \
  --namespace aerospike-operator --create-namespace \
  --set cilium.enabled=true
```

### With Cluster Manager UI

```bash
helm install acko oci://ghcr.io/kimsoungryoul/charts/acko \
  --version 0.1.0 \
  --namespace aerospike-operator --create-namespace \
  --set ui.enabled=true
```

Access the UI:
```bash
kubectl port-forward svc/<release-name>-acko-ui 3000:3000 -n aerospike-operator
# Open http://localhost:3000
```

### Full example

```bash
helm install acko oci://ghcr.io/kimsoungryoul/charts/acko \
  --version 0.1.0 \
  --namespace aerospike-operator --create-namespace \
  --set serviceMonitor.enabled=true \
  --set prometheusRule.enabled=true \
  --set grafanaDashboard.enabled=true \
  --set podDisruptionBudget.enabled=true \
  --set cilium.enabled=true \
  --set ui.enabled=true
```

### GitOps — ArgoCD example

```yaml
# Application 1: CRDs (sync-wave 0, prune disabled)
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: acko-crds
  annotations:
    argocd.argoproj.io/sync-options: Replace=true
spec:
  source:
    repoURL: ghcr.io/kimsoungryoul/charts
    chart: acko-crds
    targetRevision: "0.1.0"
  syncPolicy:
    automated:
      prune: false   # Never auto-delete CRDs
      selfHeal: true
---
# Application 2: Operator (sync-wave 1, depends on CRDs)
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: acko
spec:
  source:
    repoURL: ghcr.io/kimsoungryoul/charts
    chart: acko
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

### GitOps — Flux example

```yaml
# HelmRepository (OCI) — shared by both HelmReleases
apiVersion: source.toolkit.fluxcd.io/v1
kind: HelmRepository
metadata:
  name: acko
  namespace: flux-system
spec:
  type: oci
  url: oci://ghcr.io/kimsoungryoul/charts
---
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: acko-crds
  namespace: flux-system
spec:
  chart:
    spec:
      chart: acko-crds
      version: "0.1.0"
      sourceRef:
        kind: HelmRepository
        name: acko
  install:
    crds: CreateReplace
  upgrade:
    crds: CreateReplace
---
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: acko
  namespace: flux-system
spec:
  dependsOn:
    - name: acko-crds
  targetNamespace: aerospike-operator
  chart:
    spec:
      chart: acko
      version: "0.1.0"
      sourceRef:
        kind: HelmRepository
        name: acko
  values:
    crds:
      install: false
```

## Deploy an Aerospike cluster

After the operator is running, create an Aerospike CE cluster:

```bash
kubectl create namespace aerospike

cat <<EOF | kubectl apply -f -
apiVersion: acko.io/v1alpha1
kind: AerospikeCluster
metadata:
  name: aerospike-basic
  namespace: aerospike
spec:
  size: 1
  image: aerospike:ce-8.1.1.1
  aerospikeConfig:
    namespaces:
      - name: test
        replication-factor: 1
        storage-engine:
          type: memory
          data-size: 1073741824
EOF
```

Check the cluster status:
```bash
kubectl -n aerospike get asc
```

More sample CRs are in `config/samples/`.

## Configuration

See [values.yaml](values.yaml) for all available configuration options with descriptions.

### Key configuration sections

| Section | Description |
|---------|-------------|
| `certManager` | cert-manager integration for webhook TLS |
| `serviceMonitor` | Prometheus ServiceMonitor |
| `prometheusRule` | Prometheus alerting rules |
| `grafanaDashboard` | Grafana dashboard ConfigMap |
| `networkPolicy` | Standard Kubernetes NetworkPolicy |
| `cilium` | CiliumNetworkPolicy (alternative to networkPolicy) |
| `podDisruptionBudget` | PDB for operator pods |
| `autoscaling` | HPA for operator pods |
| `ui` | Aerospike Cluster Manager web UI |

## Uninstall

```bash
# Delete all Aerospike clusters first to avoid orphaned StatefulSets/PVCs
kubectl delete asc --all --all-namespaces

# Uninstall the operator
helm uninstall acko -n aerospike-operator
```

> **Note:** CRDs are protected with `helm.sh/resource-policy: keep` — they are
> **not** removed on `helm uninstall`. To remove CRDs explicitly (this deletes
> **all** AerospikeCluster resources and their data):
>
> ```bash
> kubectl delete crd aerospikeclusters.acko.io aerospikeclustertemplates.acko.io
> ```
