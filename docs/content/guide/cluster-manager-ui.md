---
sidebar_position: 6
title: Cluster Manager UI
---

# Aerospike Cluster Manager UI

[Aerospike Cluster Manager](https://github.com/KimSoungRyoul/aerospike-cluster-manager)는 Aerospike CE 클러스터를 관리하는 웹 기반 GUI입니다. Operator Helm 차트에 포함되어 선택적으로 배포할 수 있습니다.

---

## Installation

```bash
helm install acko oci://ghcr.io/kimsoungryoul/charts/aerospike-ce-kubernetes-operator \
  --namespace aerospike-operator --create-namespace \
  --set ui.enabled=true
```

UI 파드 확인:

```bash
kubectl -n aerospike-operator get pods -l app.kubernetes.io/component=ui
```

포트 포워딩으로 접속:

```bash
kubectl -n aerospike-operator port-forward svc/acko-aerospike-ce-kubernetes-operator-ui 3000:3000
```

브라우저에서 `http://localhost:3000` 접속.

---

## Clusters

사이드바의 연결 목록에서 클러스터를 선택하거나, 메인 화면에서 카드로 확인합니다. 각 카드에는 연결 상태, 노드 수, 네임스페이스 수, Aerospike 버전이 표시됩니다.

![Clusters 홈 화면](/img/ui/clusters-home.png)

---

## Create Cluster

사이드바의 **Create Cluster** 또는 우상단 버튼으로 클러스터 생성 마법사를 시작합니다. 총 8단계로 구성됩니다:

**Step 1 — Basic**: 클러스터 이름, K8s 네임스페이스, 노드 수(1-8), Aerospike 이미지를 설정합니다.

![Create Cluster - Step 1 Basic](/img/ui/create-cluster-basic.png)

**Step 8 — Review**: 모든 설정을 최종 확인한 후 **Create Cluster** 버튼으로 배포합니다.

![Create Cluster - Step 8 Review](/img/ui/create-cluster-review.png)

---

## Cluster Overview

클러스터를 선택하면 Overview 탭이 표시됩니다. 클러스터 Phase, Pod Ready 수, 헬스 조건(Stable / Config Applied / Available / ACL Synced), Pod 목록을 한눈에 확인합니다.

상단 버튼으로 **Scale**, **Edit**, **Warm Restart**, **Pod Restart**, **Pause**, **Delete** 작업을 실행할 수 있습니다.

![Cluster Overview](/img/ui/cluster-overview.png)

**ACKO INFO** 탭에서는 Aerospike 노드 단위 상세 정보(Build, Edition, Uptime, Connections, Cluster Size)를 확인합니다.

![Cluster ACKO INFO](/img/ui/cluster-acko-info.png)

---

## Namespaces

**Namespaces** 탭에서 네임스페이스별 오브젝트 수, 스토리지 타입, 복제 계수, 메모리/디스크 HWM, TTL 설정을 확인합니다. 각 네임스페이스 하위 Set 목록도 표시됩니다.

![Namespaces](/img/ui/namespaces.png)

Set 행을 클릭하면 레코드 브라우저로 이동합니다. **Add filter**로 Secondary Index 기반 필터를 추가할 수 있습니다.

![Namespaces Set Browser](/img/ui/namespaces-set-browser.png)

---

## Indexes

**Indexes** 탭에서 Secondary Index 목록(Name, Namespace, Set, Bin, Type, State)을 확인하고 **+ Create Index** 버튼으로 새 인덱스를 생성합니다.

![Secondary Indexes](/img/ui/indexes.png)

---

## Configuration Options

| Parameter | Description | Default |
|-----------|-------------|---------|
| `ui.enabled` | UI 활성화 | `false` |
| `ui.service.type` | 서비스 타입 | `ClusterIP` |
| `ui.ingress.enabled` | Ingress 생성 | `false` |
| `ui.persistence.enabled` | PostgreSQL PVC 사용 | `true` |
| `ui.k8s.enabled` | K8s 클러스터 관리 기능 | `true` |
| `ui.rbac.create` | ClusterRole/Binding 자동 생성 | `true` |

전체 옵션 확인:

```bash
helm show values oci://ghcr.io/kimsoungryoul/charts/aerospike-ce-kubernetes-operator | grep -A 500 "^ui:"
```

---

## Ingress (Production)

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
