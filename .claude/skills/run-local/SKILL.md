---
name: run-local
description: Set up a local Kind dev environment and run the operator on the host via `make run`
disable-model-invocation: false
---

# Local Dev Environment (make run)

Kind 클러스터를 생성하고 operator를 로컬 호스트 프로세스로 실행하는 개발 환경을 구성합니다.
Webhook은 비활성 상태이므로 cert-manager가 불필요하며, 코드 수정 후 Ctrl+C → `make run`으로 즉시 반영됩니다.

## Setup Steps

### 1. Kind 클러스터 생성

기존 클러스터를 삭제하고 재생성합니다 (3-worker 노드, zone label 포함).

```bash
make setup-kind
```

클러스터 생성을 확인합니다:
```bash
kubectl get nodes
```

### 2. CRD 설치

CRD만 설치합니다 (webhook config 제외). `make install`이 annotation 크기 제한으로 실패하면 `kubectl replace`를 사용합니다.

```bash
make install
```

실패 시 대안:
```bash
make manifests && bin/kustomize build config/crd | kubectl replace -f -
```

### 3. Operator 로컬 실행

`make run`은 manifests 생성, fmt, vet을 포함하므로 별도의 `make build`가 불필요합니다.

```bash
make run
```

foreground 프로세스로 실행됩니다. 로그가 터미널에 출력됩니다.
코드 수정 후: **Ctrl+C → `make run`** 으로 즉시 반영.

### 4. Sample CR 배포 (선택)

다른 터미널에서 실행:

```bash
kubectl create namespace aerospike --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f config/samples/acko_v1alpha1_aerospikecluster.yaml
```

상태 확인:
```bash
kubectl -n aerospike get asc
kubectl -n aerospike get pods -w
```

## Available Samples

| 파일 | 설명 |
|------|------|
| `acko_v1alpha1_aerospikecluster.yaml` | 단일 노드 in-memory |
| `aerospike-cluster-3node.yaml` | 3노드 PV storage |
| `aerospike-cluster-multirack.yaml` | 6노드 multi-rack |
| `aerospike-cluster-acl.yaml` | 3노드 ACL |

## Cleanup

```bash
kind delete cluster --name kind
```

## Troubleshooting

- **make run 시 webhook cert 에러**: 무시 가능. webhook 서버는 자체 self-signed cert를 생성하지만, 클러스터에 webhook config가 없으므로 apiserver가 호출하지 않습니다.
- **CRD 설치 실패 (annotation too long)**: `bin/kustomize build config/crd | kubectl replace -f -` 사용.
- **Pod pending**: Kind에서 PVC를 사용하는 샘플은 storage provisioner가 필요합니다. in-memory 샘플(`acko_v1alpha1_aerospikecluster.yaml`)부터 시작하세요.
- **컨텍스트 확인**: `kubectl config current-context`가 `kind-kind`인지 확인.
