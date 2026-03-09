---
sidebar_position: 8
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

사이드바의 **Create Cluster** 또는 우상단 버튼으로 클러스터 생성 마법사를 시작합니다. 총 9단계로 구성됩니다:

**Step 1 — Basic**: 클러스터 이름, K8s 네임스페이스, 노드 수(1-8), Aerospike 이미지를 설정합니다.

![Create Cluster - Step 1 Basic](/img/ui/create-cluster-basic.png)

**Step 3 — Monitoring & Options**: 아래 항목을 설정합니다:

- **Prometheus Monitoring** — 메트릭 exporter sidecar 활성화 및 포트 설정. 추가 구성: exporter 이미지, 메트릭 라벨, exporter 리소스(CPU/메모리), ServiceMonitor(enabled/interval/labels), PrometheusRule(enabled/labels)
- **Dynamic Config** — 재시작 없이 설정 변경 적용
- **Network Access** — 클라이언트 접근 방식(Pod IP, Host Internal/External, Configured IP). `configuredIP` 선택 시 custom network names 입력 필드가 표시됩니다.
- **Kubernetes NetworkPolicy** — K8s NetworkPolicy 자동 생성 (standard 또는 Cilium)
- **Seeds Finder LoadBalancer** — 외부 시드 검색용 LoadBalancer 서비스 생성 (포트, 트래픽 정책, 어노테이션, 라벨, source ranges 설정)

**Step 8 — Review**: 모든 설정을 최종 확인한 후 **Create Cluster** 버튼으로 배포합니다.

![Create Cluster - Step 8 Review](/img/ui/create-cluster-review.png)

---

## Cluster Overview

클러스터를 선택하면 Overview 탭이 표시됩니다. 클러스터 Phase, Pod Ready 수, 헬스 조건(Stable / Config Applied / Available / ACL Synced), Pod 목록을 한눈에 확인합니다.

상단 버튼으로 **Scale**, **Edit**, **Warm Restart**, **Pod Restart**, **Pause**, **Delete** 작업을 실행할 수 있습니다.

![Cluster Overview](/img/ui/cluster-overview.png)

**ACKO INFO** 탭에서는 Aerospike 노드 단위 상세 정보(Build, Edition, Uptime, Connections, Cluster Size)를 확인합니다.

![Cluster ACKO INFO](/img/ui/cluster-acko-info.png)

### Disconnected State

Aerospike 연결이 끊어진 경우 Overview 및 Browser 페이지에서 스켈레톤 로딩 대신 연결 해제 상태 화면이 표시됩니다. `WifiOff` 아이콘과 함께 재연결을 안내하는 메시지가 나타납니다.

### Events Timeline

클러스터 상세 페이지의 **Events** 탭에서 Kubernetes 이벤트를 확인합니다. 각 이벤트에는 타입, 이유, 메시지, 발생 횟수, 그리고 상대적 시간(예: "2m ago")이 표시됩니다. Transitional Phase에서는 자동으로 새로고침됩니다.

### Event Category Filtering

이벤트 타임라인에서 카테고리별 필터링이 가능합니다. 11개 카테고리로 자동 분류됩니다:

| Category | Description | Example Events |
|----------|-------------|----------------|
| **Lifecycle** | 클러스터 생성/삭제 | ClusterCreated, ClusterDeletionStarted |
| **Rolling Restart** | 롤링 리스타트 | RollingRestartStarted/Completed, PodRestarted |
| **Configuration** | 설정 변경 | ConfigMapCreated, DynamicConfigApplied |
| **ACL Security** | 접근 제어 | ACLSyncStarted/Completed/Failed |
| **Scaling** | 스케일 업/다운 | RackScaled, PVCCleanupCompleted |
| **Rack Management** | 랙 관리 | StatefulSetCreated, RackRemoved |
| **Network** | 네트워크 리소스 | ServiceCreated, PDBCreated, NetworkPolicyCreated |
| **Monitoring** | 모니터링 설정 | MonitoringConfigured |
| **Template** | 템플릿 동기화 | TemplateApplied, TemplateOutOfSync |
| **Circuit Breaker** | 서킷 브레이커 | CircuitBreakerActive/Reset |

카테고리 필터 칩을 클릭하여 특정 유형의 이벤트만 표시할 수 있습니다.

### Configuration Drift Detection

클러스터 상세 페이지에서 **Config Status** 카드가 현재 설정의 동기화 상태를 표시합니다:

- **In Sync** — 원하는 설정(spec)과 적용된 설정(appliedSpec)이 일치
- **Config Drift Detected** — spec과 appliedSpec 사이에 차이 발견

변경된 필드 목록과 Pod별 설정 해시 버전이 표시됩니다. 여러 해시 그룹이 있으면 일부 Pod가 아직 이전 설정으로 실행 중임을 의미합니다.

### Reconciliation Health & Circuit Breaker

Reconciliation 실패가 발생하면 **Reconciliation Health** 카드가 나타납니다:

- **Progress Bar** — 서킷 브레이커 임계값(10회)까지의 실패 진행도
- **Backoff Timer** — 서킷 브레이커 활성화 시 다음 재시도까지의 예상 시간
- **Error Details** — 마지막 reconciliation 에러 메시지
- **Reset Button** — 서킷 브레이커 수동 리셋 (no-op 패치로 재시도 트리거)

서킷 브레이커는 연속 10회 실패 시 자동 활성화되며, 지수 백오프(30s × 2^n, 최대 300s)로 재시도합니다.

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

## Record Browser

**Browser** 탭에서 Aerospike 레코드를 조회, 생성, 수정, 삭제할 수 있습니다.

- Namespace와 Set을 선택하여 레코드를 스캔
- 페이지네이션을 통한 대량 레코드 탐색
- 개별 레코드의 Bin 값을 인라인 편집
- PK(Primary Key)로 레코드 직접 조회
- Secondary Index 기반 필터를 추가하여 조건부 스캔

---

## AQL Terminal

**Terminal** 탭에서 Monaco Editor 기반의 AQL(Aerospike Query Language) 터미널을 제공합니다.

- AQL 명령어 직접 입력 및 실행
- 구문 하이라이팅 및 자동 완성
- 실행 결과를 테이블/JSON 형식으로 표시

---

## UDF Management

**UDFs** 탭에서 Lua User-Defined Function을 관리합니다.

- 등록된 UDF 모듈 목록 확인
- 새로운 Lua UDF 파일 업로드
- UDF 모듈 삭제

---

## User & Role Management

**Admin** 탭에서 Aerospike 접근 제어(ACL)를 관리합니다.

- 사용자 목록 조회, 생성, 삭제, 비밀번호 변경
- 역할(Role) 목록 조회, 생성, 삭제
- 역할별 권한(Privilege) 관리
- 사용자-역할 매핑

---

## K8s Cluster Management

`ui.k8s.enabled=true`일 때, **K8s Clusters** 페이지에서 `AerospikeCluster` CR을 GUI로 관리합니다.

### Cluster List

모든 네임스페이스의 AerospikeCluster를 카드 형식으로 표시합니다. 각 카드에 Phase, 노드 수, 이미지, 생성 시간이 표시됩니다.

### Create Cluster Wizard

**Scratch Mode** (9단계) 또는 **Template Mode** (3단계)로 클러스터를 생성합니다:

1. **Creation Mode** — Scratch 또는 Template 선택
2. **Basic** — 이름, 네임스페이스, 이미지, 노드 수
3. **Namespace & Storage** — Aerospike 네임스페이스 및 볼륨 구성
4. **Monitoring & Options** — Prometheus, Dynamic Config, NetworkPolicy, Seeds Finder LB
5. **Resources** — CPU/Memory requests/limits
6. **Security & ACL** — 역할 및 사용자 구성
7. **Rolling Update** — 배치 크기, PDB, Max Unavailable
8. **Rack Config** — 랙별 zone/region/storage/affinity 설정
9. **Advanced** — Node selector, tolerations, bandwidth, readiness gate, pod metadata
10. **Review** — 전체 설정 확인 및 배포

### Cluster Detail

클러스터 선택 시 다음 정보와 작업이 제공됩니다:

- **Overview** — Phase, Health, Conditions, Pod 목록
- **Events Timeline** — 11개 카테고리별 필터링 가능한 K8s 이벤트
- **Config Drift Detection** — spec vs appliedSpec 비교, Pod별 config hash 그룹핑
- **Reconciliation Health** — 서킷 브레이커 상태, 실패 횟수, 백오프 타이머
- **Pod Logs** — 개별 Pod 로그 조회
- **YAML Export** — 클러스터 CR을 클린 YAML로 내보내기
- **Operations** — Scale, Edit, Warm Restart, Pod Restart, Pause/Resume, Delete, HPA 관리, Template Resync

### Template Management

**K8s Templates** 페이지에서 cluster-scoped `AerospikeClusterTemplate` 리소스를 관리합니다:

- 템플릿 목록 조회 (참조 클러스터 수 표시)
- 새 템플릿 생성
- 템플릿 상세 정보 조회
- 참조 클러스터가 없는 템플릿 삭제

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
