---
sidebar_position: 1
title: 설치
---

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

# 설치

이 가이드는 ACKO 오퍼레이터를 설치하는 세 가지 방법을 다룹니다.

## 사전 준비

- Kubernetes 클러스터 v1.26+
- 클러스터에 접근 가능하도록 설정된 kubectl
- [cert-manager](https://cert-manager.io/) 설치 완료 (웹훅 TLS에 필요)

### cert-manager 설치

```bash
helm repo add jetstack https://charts.jetstack.io
helm repo update
helm install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --set crds.enabled=true
```

cert-manager 준비 대기:

```bash
kubectl -n cert-manager wait --for=condition=Available deployment/cert-manager --timeout=60s
kubectl -n cert-manager wait --for=condition=Available deployment/cert-manager-webhook --timeout=60s
```

## 오퍼레이터 설치

<Tabs groupId="install-method">
<TabItem value="helm-oci" label="Helm OCI (권장)" default>

게시된 OCI Helm 차트를 사용하는 가장 간단한 설치 방법입니다.

```bash
helm install aerospike-operator oci://ghcr.io/kimsoungryoul/aerospike-operator \
  --version 0.1.0 \  # 최신 버전으로 교체
  -n aerospike-operator --create-namespace
```

### Helm 값 커스터마이징

기본값을 재정의할 수 있습니다:

```bash
helm install aerospike-operator oci://ghcr.io/kimsoungryoul/aerospike-operator \
  --version 0.1.0 \  # 최신 버전으로 교체
  -n aerospike-operator --create-namespace \
  --set replicaCount=2 \
  --set resources.limits.memory=256Mi
```

사용 가능한 모든 값 조회:

```bash
helm show values oci://ghcr.io/kimsoungryoul/aerospike-operator --version 0.1.0  # 최신 버전으로 교체
```

</TabItem>
<TabItem value="helm-local" label="Helm 로컬 차트">

리포지토리에 포함된 차트를 사용합니다.

```bash
git clone https://github.com/KimSoungRyoul/aerospike-ce-kubernetes-operator.git
cd aerospike-ce-kubernetes-operator

helm install aerospike-operator ./charts/aerospike-operator \
  -n aerospike-operator --create-namespace
```

</TabItem>
<TabItem value="kustomize" label="Kustomize">

Kustomize 기반 배포를 선호하는 사용자를 위한 방법입니다.

```bash
git clone https://github.com/KimSoungRyoul/aerospike-ce-kubernetes-operator.git
cd aerospike-ce-kubernetes-operator

# 오퍼레이터 이미지를 빌드하고 레지스트리에 푸시
make docker-build docker-push IMG=<your-registry>/aerospike-ce-operator:latest

# CRD 설치
make install

# 오퍼레이터 배포
make deploy IMG=<your-registry>/aerospike-ce-operator:latest
```

</TabItem>
</Tabs>

## 설치 확인

오퍼레이터 파드가 실행 중인지 확인:

```bash
kubectl -n aerospike-operator get pods
```

예상 출력:

```
NAME                                                READY   STATUS    RESTARTS   AGE
aerospike-operator-controller-manager-xxxxx-yyyyy   1/1     Running   0          30s
```

CRD가 등록되었는지 확인:

```bash
kubectl get crd aerospikececlusters.acko.io
```

## 제거

:::warning
오퍼레이터를 제거하기 전에 반드시 AerospikeCECluster 리소스를 먼저 삭제하세요. 오퍼레이터를 먼저 제거하면 고아 상태의 StatefulSet과 PVC가 남게 됩니다.
:::

<Tabs groupId="install-method">
<TabItem value="helm-oci" label="Helm" default>

```bash
# 먼저 모든 Aerospike 클러스터를 삭제
kubectl delete asce --all --all-namespaces

# 오퍼레이터 제거
helm uninstall aerospike-operator -n aerospike-operator

# (선택) 네임스페이스 삭제
kubectl delete namespace aerospike-operator
```

</TabItem>
<TabItem value="helm-local" label="Helm (로컬)">

```bash
# 먼저 모든 Aerospike 클러스터를 삭제
kubectl delete asce --all --all-namespaces

# 오퍼레이터 제거
helm uninstall aerospike-operator -n aerospike-operator

# (선택) 네임스페이스 삭제
kubectl delete namespace aerospike-operator
```

</TabItem>
<TabItem value="kustomize" label="Kustomize">

```bash
# 먼저 모든 Aerospike 클러스터를 삭제
kubectl delete asce --all --all-namespaces

# 오퍼레이터 제거
make undeploy

# CRD 제거
make uninstall
```

</TabItem>
</Tabs>
