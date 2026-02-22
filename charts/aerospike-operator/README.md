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
> helm install aerospike-operator ./charts/aerospike-operator \
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

### Basic (minimal)

```bash
helm install aerospike-operator ./charts/aerospike-operator \
  --namespace aerospike-operator --create-namespace
```

### With monitoring enabled

```bash
helm install aerospike-operator ./charts/aerospike-operator \
  --namespace aerospike-operator --create-namespace \
  --set serviceMonitor.enabled=true \
  --set prometheusRule.enabled=true \
  --set grafanaDashboard.enabled=true
```

### With Cilium network policy

```bash
helm install aerospike-operator ./charts/aerospike-operator \
  --namespace aerospike-operator --create-namespace \
  --set cilium.enabled=true
```

### Full example

```bash
helm install aerospike-operator ./charts/aerospike-operator \
  --namespace aerospike-operator --create-namespace \
  --set serviceMonitor.enabled=true \
  --set prometheusRule.enabled=true \
  --set grafanaDashboard.enabled=true \
  --set podDisruptionBudget.enabled=true \
  --set cilium.enabled=true
```

## Deploy an Aerospike cluster

After the operator is running, create an Aerospike CE cluster:

```bash
kubectl create namespace aerospike

cat <<EOF | kubectl apply -f -
apiVersion: acko.io/v1alpha1
kind: AerospikeCECluster
metadata:
  name: aerospike-ce-basic
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
kubectl -n aerospike get asce
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

## Uninstall

```bash
helm uninstall aerospike-operator -n aerospike-operator
```
