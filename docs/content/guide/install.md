---
sidebar_position: 1
title: Installation
---

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

# Installation

This guide covers three methods to install the ACKO operator.

## Prerequisites

- Kubernetes cluster v1.26+
- kubectl configured to access the cluster
- [cert-manager](https://cert-manager.io/) installed (required for webhook TLS)

### Install cert-manager

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
helm install aerospike-operator oci://ghcr.io/kimsoungryoul/aerospike-operator \
  --version 0.1.0 \
  -n aerospike-operator --create-namespace
```

### Customizing Helm Values

You can override default values:

```bash
helm install aerospike-operator oci://ghcr.io/kimsoungryoul/aerospike-operator \
  --version 0.1.0 \
  -n aerospike-operator --create-namespace \
  --set replicaCount=2 \
  --set resources.limits.memory=256Mi
```

To see all available values:

```bash
helm show values oci://ghcr.io/kimsoungryoul/aerospike-operator --version 0.1.0
```

</TabItem>
<TabItem value="helm-local" label="Helm Local Chart">

Use the chart bundled in the repository.

```bash
git clone https://github.com/KimSoungRyoul/aerospike-ce-kubernetes-operator.git
cd aerospike-ce-kubernetes-operator

helm install aerospike-operator ./charts/aerospike-operator \
  -n aerospike-operator --create-namespace
```

</TabItem>
<TabItem value="kustomize" label="Kustomize">

For users who prefer Kustomize-based deployment.

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

## Verify Installation

Check the operator pod is running:

```bash
kubectl -n aerospike-operator get pods
```

Expected output:

```
NAME                                                READY   STATUS    RESTARTS   AGE
aerospike-operator-controller-manager-xxxxx-yyyyy   1/1     Running   0          30s
```

Check the CRD is registered:

```bash
kubectl get crd aerospikececlusters.acko.io
```

## Uninstall

:::warning
Always delete AerospikeCECluster resources before uninstalling the operator. Removing the operator first will leave orphaned StatefulSets and PVCs.
:::

<Tabs groupId="install-method">
<TabItem value="helm-oci" label="Helm" default>

```bash
# Delete all Aerospike clusters first
kubectl delete asce --all --all-namespaces

# Uninstall the operator
helm uninstall aerospike-operator -n aerospike-operator

# (Optional) Delete the namespace
kubectl delete namespace aerospike-operator
```

</TabItem>
<TabItem value="helm-local" label="Helm (Local)">

```bash
# Delete all Aerospike clusters first
kubectl delete asce --all --all-namespaces

# Uninstall the operator
helm uninstall aerospike-operator -n aerospike-operator

# (Optional) Delete the namespace
kubectl delete namespace aerospike-operator
```

</TabItem>
<TabItem value="kustomize" label="Kustomize">

```bash
# Delete all Aerospike clusters first
kubectl delete asce --all --all-namespaces

# Remove the operator
make undeploy

# Remove CRDs
make uninstall
```

</TabItem>
</Tabs>
