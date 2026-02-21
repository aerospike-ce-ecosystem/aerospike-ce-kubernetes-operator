# Aerospike CE Kubernetes Operator (ACKO)

Kubernetes Operator for managing [Aerospike Community Edition](https://aerospike.com/) clusters. Built with [Kubebuilder](https://book.kubebuilder.io/) and [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime).

Deploy, scale, and perform rolling updates of Aerospike CE clusters via a custom `AerospikeCECluster` CRD.

## Features

- Declarative cluster lifecycle management (create, scale, rolling update)
- Rack-aware deployment with zone/region affinity
- Persistent volume storage with cascade delete support
- Aerospike configuration management (auto-generates `aerospike.conf`)
- Access control (ACL) with Kubernetes Secrets integration
- Pod Disruption Budget for safe maintenance
- Mesh heartbeat auto-configuration

### CE Limitations

| Constraint | Limit |
|---|---|
| Max cluster size | 8 nodes |
| Max namespaces | 2 |
| XDR | Not supported |
| TLS | Not supported |
| Strong Consistency | Not supported |

## Quick Start

### Prerequisites

- Kubernetes cluster v1.26+
- kubectl configured to access the cluster
- cert-manager installed (for webhook TLS)

Install required tools (macOS):

```sh
brew install kind kustomize kubectl
brew install helm
```

Install cert-manager via Helm:

```sh
helm repo add jetstack https://charts.jetstack.io --force-update
helm install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --set crds.enabled=true \
  --set global.privateKeyRotationPolicy=Always
```

### Option A: Install with Helm (Recommended)

```sh
helm repo add acko https://kimsoungryoul.github.io/aerospike-ce-kubernetes-operator
helm install -n aerospike-system --create-namespace aerospike-operator acko/aerospike-operator
```

Or install from local chart:

```sh
helm install -n aerospike-system --create-namespace aerospike-operator charts/aerospike-operator
```

### Option B: Install with Kustomize

#### 1. Install CRDs

```
kind create cluster
```


```sh
make install
```

#### 2. Deploy the Operator

```sh
# Build and push the operator image
make docker-build docker-push IMG=<your-registry>/aerospike-ce-operator:latest

# Deploy to the cluster
make deploy IMG=<your-registry>/aerospike-ce-operator:latest
```

### 3. Create an Aerospike Cluster

Create the target namespace:

```sh
kubectl create namespace aerospike
```

#### Minimal single-node (in-memory)

```yaml
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
```

```sh
kubectl apply -f config/samples/acko_v1alpha1_aerospikececluster.yaml
```

#### 3-node cluster with persistent storage

```yaml
apiVersion: acko.io/v1alpha1
kind: AerospikeCECluster
metadata:
  name: aerospike-ce-3node
  namespace: aerospike
spec:
  size: 3
  image: aerospike:ce-8.1.1.1

  podSpec:
    aerospikeContainer:
      resources:
        requests:
          memory: "2Gi"
          cpu: "1"
        limits:
          memory: "4Gi"
          cpu: "2"

  storage:
    volumes:
      - name: data-vol
        source:
          persistentVolume:
            storageClass: standard
            size: 10Gi
            volumeMode: Filesystem
        aerospike:
          path: /opt/aerospike/data
        cascadeDelete: true

  aerospikeConfig:
    namespaces:
      - name: testns
        replication-factor: 2
        storage-engine:
          type: device
          file: /opt/aerospike/data/testns.dat
          filesize: 4294967296
          data-in-memory: true
```

```sh
kubectl apply -f config/samples/aerospike-ce-cluster-3node.yaml
```

### 4. Verify the Cluster

```sh
# Check the CR status
kubectl -n aerospike get asce

# Check pods
kubectl -n aerospike get pods

# Check logs
kubectl -n aerospike logs -l app.kubernetes.io/name=aerospike-ce-operator
```

### 5. Connect to Aerospike

```sh
# Port-forward to a pod
kubectl -n aerospike port-forward aerospike-ce-basic-0-0 3000:3000

# Connect using aql or asadm
aql -h 127.0.0.1 -p 3000
```

## More Examples

| Sample | Description |
|---|---|
| [`acko_v1alpha1_aerospikececluster.yaml`](config/samples/acko_v1alpha1_aerospikececluster.yaml) | Minimal single-node in-memory |
| [`aerospike-ce-cluster-3node.yaml`](config/samples/aerospike-ce-cluster-3node.yaml) | 3-node with persistent volume storage |
| [`aerospike-ce-cluster-multirack.yaml`](config/samples/aerospike-ce-cluster-multirack.yaml) | 6-node multi-rack with zone affinity |
| [`aerospike-ce-cluster-acl.yaml`](config/samples/aerospike-ce-cluster-acl.yaml) | 3-node with ACL (roles, users, K8s secrets) |

## Uninstall

```sh
# Delete the cluster
kubectl -n aerospike delete asce aerospike-ce-basic

# Remove the operator
make undeploy

# Remove CRDs
make uninstall
```

## Codebase Stats

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 Language              Files        Lines         Code     Comments       Blanks
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 Go                       61        11598         8678         1287         1633
 YAML                     26          793          634          121           38
 Makefile                  1          260          167           46           47
 Dockerfile                2           37           23            8            6
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 Total                    96        14331         9736         2528         2067
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

> Measured with [tokei](https://github.com/XAMPPRocky/tokei) (`tokei -C`). Auto-generated code, Helm chart templates, and CRD/RBAC manifests are excluded via `.tokeignore`.

## Development

```sh
make build          # Build manager binary
make test           # Run unit + integration tests
make lint           # Run golangci-lint
make run            # Run controller locally against current kubeconfig
```

### Local deployment with Kind

로컬 빌드 후 Kind 클러스터에 배포할 때:

```sh
# 1. Kind 클러스터 생성
make setup-test-e2e

# 2. 이미지 로컬 빌드
make docker-build

# 3. Kind 클러스터에 이미지 로드
kind load docker-image ghcr.io/kimsoungryoul/aerospike-ce-kubernetes-operator:latest \
  --name aerospike-ce-operator-test-e2e

# 4. CRD 설치 및 오퍼레이터 배포
make install
make deploy
```

> `config/manager/manager.yaml`에 `imagePullPolicy: IfNotPresent`가 설정되어 있어, Kind에 로드된 로컬 이미지를 그대로 사용합니다.

## License

Copyright 2026. Licensed under the Apache License, Version 2.0.
