---
name: operator-local-deploy
description: Build and deploy the operator to a local Kind cluster for testing
disable-model-invocation: false
---

# Local Operator Deployment

Build and deploy the Aerospike CE Kubernetes Operator to a local Kind cluster.

## Arguments

Optional arguments:
- `setup` - Create Kind cluster + build + deploy (full setup)
- `build` - Build image and load into existing Kind cluster
- `deploy` - Deploy operator to existing Kind cluster (assumes image loaded)
- `apply <sample>` - Apply a sample CR after deployment
- `status` - Check current deployment status
- `cleanup` - Tear down everything
- No argument defaults to `setup`

## Environment Variables

```bash
IMG=ghcr.io/kimsoungryoul/aerospike-ce-operator:latest
KIND_CLUSTER=aerospike-ce-operator-test-e2e
KIND_PROVIDER=podman   # or podman
```

## Full Setup Workflow (`setup`)

### 1. Create Kind Cluster
```bash
make setup-test-e2e
```
Creates `aerospike-ce-operator-test-e2e` cluster if not exists.

### 2. Build Container Image
```bash
make docker-build
```
Builds image as `$IMG`.

### 3. Load Image into Kind
```bash
kind load docker-image $IMG --name $KIND_CLUSTER
```

### 4. Install CRDs
```bash
make install
```

### 4.1 install cert-manager
```bash
helm install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --set crds.enabled=true \
  --set global.privateKeyRotationPolicy=Always
```

### 5. Deploy Operator
```bash
make deploy IMG=$IMG
```
Deploys to `aerospike-operator` namespace with webhook, cert-manager, RBAC.

### 6. Verify Deployment
```bash
kubectl -n aerospike-operator get deployment
kubectl -n aerospike-operator get pods
kubectl -n aerospike-operator wait --for=condition=Available deployment/aerospike-ce-operator-controller-manager --timeout=120s
```

## Apply Sample CR (`apply`)

Available samples in `config/samples/`:
- `acko_v1alpha1_aerospikececluster.yaml` - Minimal single-node in-memory
- `aerospike-ce-cluster-3node.yaml` - 3-node with PV storage
- `aerospike-ce-cluster-multirack.yaml` - 6-node multi-rack
- `aerospike-ce-cluster-acl.yaml` - 3-node with ACL

```bash
kubectl create namespace aerospike --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f config/samples/<sample-file>.yaml
kubectl -n aerospike get aerospikececlusters.acko.io
kubectl -n aerospike get pods -w
```

## Status Check (`status`)

```bash
kubectl -n aerospike-operator get deployment
kubectl -n aerospike-operator get pods
kubectl -n aerospike-operator logs deployment/aerospike-ce-operator-controller-manager --tail=30
kubectl -n aerospike get asce
kubectl -n aerospike get pods
kubectl -n aerospike get statefulsets
```

## Cleanup (`cleanup`)

```bash
make undeploy
make uninstall
make cleanup-test-e2e
```

## Troubleshooting

If deployment fails:
1. Check operator logs: `kubectl -n aerospike-operator logs deployment/aerospike-ce-operator-controller-manager`
2. Check events: `kubectl -n aerospike-operator get events --sort-by=.lastTimestamp`
3. Check cert-manager: `kubectl get certificates -n aerospike-operator`
4. Check webhook: `kubectl get validatingwebhookconfigurations`

If pods are pending:
1. Check PVC status: `kubectl -n aerospike get pvc`
2. Check node resources: `kubectl describe nodes`
3. For Kind: storage provisioner may need time to create PVs
