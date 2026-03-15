---
sidebar_position: 3
title: 클러스터 관리
---

# Aerospike 클러스터 관리

이 가이드는 Day-2 운영을 다룹니다: 스케일링, 업데이트, 설정 변경, 트러블슈팅.

## 스케일링

`spec.size`를 변경하여 클러스터를 스케일 업/다운합니다.

```bash
kubectl -n aerospike patch asc aerospike-ce-3node --type merge -p '{"spec":{"size":5}}'
```

오퍼레이터가 원하는 크기에 맞게 파드를 생성하거나 제거합니다. 멀티랙 배포의 경우, 파드가 랙 간에 균등하게 분배됩니다.

:::warning
`spec.size`는 8을 초과할 수 없습니다 (CE 제한). `replication-factor`는 새 클러스터 크기를 초과할 수 없습니다.
:::

## 롤링 업데이트

### 이미지 업데이트

`spec.image`를 변경하여 새 이미지로 롤링 재시작을 트리거합니다:

```yaml
spec:
  image: aerospike:ce-8.1.1.1   # 새 버전으로 변경
```

오퍼레이터는 `OnDelete` 업데이트 전략을 사용합니다 — 파드를 하나씩(또는 배치 단위로) 삭제하고 새 파드가 준비될 때까지 기다린 후 다음으로 진행합니다.

### 배치 크기

동시에 재시작하는 파드 수를 제어합니다:

```yaml
spec:
  rollingUpdateBatchSize: 2   # 한 번에 2개 파드 재시작 (기본값: 1)
```

랙 인식 배포의 경우, `rackConfig`에서 랙별 배치 크기를 설정할 수 있습니다. 이 값은 `spec.rollingUpdateBatchSize`보다 우선합니다:

```yaml
spec:
  rackConfig:
    rollingUpdateBatchSize: "50%"   # 랙당 파드의 50%를 동시에 재시작
```

#### 배치 크기: 정수 vs 퍼센트

배치 크기는 정수 또는 퍼센트 문자열을 허용합니다:

| 형식 | 예시 | 동작 (size=6) |
|--------|---------|-------------------|
| 정수 | `2` | 한 번에 정확히 2개 파드 |
| 퍼센트 | `"50%"` | 6의 50% = 3개 파드 |
| 퍼센트 | `"25%"` | 6의 25% = 2개 파드 (올림) |

:::tip
퍼센트 문자열은 `%` 접미사를 포함해야 합니다 (예: `"50%"`). 퍼센트는 랙당 총 파드 수에 대해 계산되며, 최소 1로 올림됩니다.
:::

### 스케일 다운 배치 크기

스케일 다운 시 랙당 동시에 제거하는 파드 수를 제어합니다:

```yaml
spec:
  rackConfig:
    scaleDownBatchSize: 2            # 랙당 한 번에 2개 파드 제거
    # scaleDownBatchSize: "25%"      # 퍼센트도 사용 가능
```

`scaleDownBatchSize`는 스케일 다운 오퍼레이션 중 **랙당** 적용됩니다. 한 번에 너무 많은 파드를 제거하여 데이터 비가용성을 초래하는 것을 방지합니다.

### 무시 가능한 파드 수

일부 파드가 Pending/Failed 상태일 때도 재조정을 계속 진행할 수 있습니다:

```yaml
spec:
  rackConfig:
    maxIgnorablePods: 1   # 멈춘 파드 1개까지 무시하고 재조정 계속
```

스케줄링 문제로 파드가 멈췄을 때 전체 재조정이 차단되지 않도록 하는 데 유용합니다.

#### 배치 크기 요약

| 필드 | 범위 | 기본값 | 설명 |
|-------|-------|---------|-------------|
| `spec.rollingUpdateBatchSize` | 클러스터 전체 | 1 | 롤링 업데이트 시 동시 재시작 파드 수 |
| `rackConfig.rollingUpdateBatchSize` | 랙당 | spec에서 상속 | 랙당 클러스터 레벨 배치 크기 오버라이드 |
| `rackConfig.scaleDownBatchSize` | 랙당 | 전체 한 번에 | 스케일 다운 시 동시 제거 파드 수 |
| `rackConfig.maxIgnorablePods` | 랙당 | 0 | 재조정 중 무시할 멈춘 파드 수 |

## 설정 변경

### 정적 설정 변경

`spec.aerospikeConfig`의 모든 변경은 롤링 재시작을 트리거하여 새 설정을 적용합니다. 오퍼레이터가 각 파드의 ConfigMap에서 `aerospike.conf`를 재생성합니다.

### 동적 설정 변경

파드 재시작 없이 런타임에 설정을 변경합니다:

```yaml
spec:
  enableDynamicConfigUpdate: true
```

활성화하면, 오퍼레이터가 Aerospike의 `set-config` 명령을 사용하여 가능한 경우 런타임에 설정 변경을 적용합니다. 동적으로 적용할 수 없는 변경만 롤링 재시작을 트리거합니다.

## Pod Readiness Gates

기본적으로 파드는 Kubernetes가 `PodReady=True`를 보고할 때 "준비됨"으로 간주됩니다. 이는 Aerospike가 클러스터 mesh에 완전히 참여하고 데이터 마이그레이션을 완료하기 전에 파드가 Service 엔드포인트에 추가될 수 있음을 의미합니다.

커스텀 readiness gate `acko.io/aerospike-ready`를 활성화하여 Aerospike가 진정으로 준비될 때까지 각 파드를 Service 엔드포인트에서 제외합니다:

```yaml
spec:
  podSpec:
    readinessGateEnabled: true
```

활성화하면 오퍼레이터가:
1. 모든 파드의 `spec.readinessGates`에 `acko.io/aerospike-ready`를 주입합니다
2. 다음 조건이 충족된 후에만 파드의 `status.conditions`를 `True`로 패치합니다:
   - 파드의 Aerospike 프로세스가 클러스터 mesh에 참여했을 때, **그리고**
   - 모든 데이터 마이그레이션이 완료되었을 때 (`cluster-stable: true`)
3. 롤링 리스타트를 보류합니다 — 이전 파드의 게이트가 충족될 때까지 다음 파드를 삭제하지 않습니다

:::info
`readinessGateEnabled` 변경은 `ReadinessGates`가 파드 생성 후 불변이므로 롤링 리스타트를 트리거합니다. 오퍼레이터가 이를 자동으로 처리합니다.
:::

:::note
이것은 **옵트인** 기능입니다. `readinessGateEnabled`가 설정되지 않은(또는 `false`인) 기존 클러스터는 이전과 동일하게 동작합니다.
:::

### 게이트 상태 확인

파드별 게이트 조건을 확인합니다:

```bash
kubectl -n aerospike get pod aerospike-ce-3node-0 \
  -o jsonpath='{.status.conditions}' | jq '.[] | select(.type=="acko.io/aerospike-ready")'
```

오퍼레이터는 클러스터의 파드 상태에 게이트 상태를 반영합니다:

```bash
kubectl -n aerospike get asc aerospike-ce-3node \
  -o jsonpath='{.status.pods}' | jq 'to_entries[] | {pod: .key, gateOk: .value.readinessGateSatisfied}'
```

### Readiness Gates와 롤링 리스타트 동작

Readiness gates 없이:
```
Pod-2 삭제 → Pod-2 Running → Pod-1 삭제 → ...   (K8s Ready = 충분)
```

`readinessGateEnabled: true`일 때:
```
Pod-2 삭제 → Pod-2 Running → Gate=False (마이그레이션 중) → Gate=True → Pod-1 삭제 → ...
```

게이트 대기로 리스타트가 차단되면 `ReadinessGateBlocking` 경고 이벤트가 발생합니다:

```bash
kubectl -n aerospike get events --field-selector reason=ReadinessGateBlocking
```

## 마이그레이션 상태 모니터링

오퍼레이터는 `status.migrationStatus`에서 데이터 마이그레이션 진행 상태를 추적합니다. 각 재조정 시 모든 Aerospike 노드의 `migrate_partitions_remaining` 통계를 조회하여 결과를 집계합니다.

**마이그레이션이 발생하는 시점:**

- **스케일 업/다운** — 노드 추가 또는 제거 시 파티션 리밸런싱 발생.
- **롤링 리스타트** — 재시작된 각 노드가 파티션 사본을 다시 수신해야 함.
- **랙 변경** — 랙 간 파드 이동 시 데이터 재분배.
- **Replication factor 변경** — RF 증가 시 새 복제본 생성.

**마이그레이션 상태 확인:**

```bash
# 클러스터 레벨 마이그레이션 상태
kubectl -n aerospike get asc aerospike-ce-3node \
  -o jsonpath='{.status.migrationStatus}' | jq .
```

출력 예시:
```json
{
  "inProgress": true,
  "remainingPartitions": 142857,
  "lastChecked": "2026-03-13T10:30:00Z"
}
```

**파드별 마이그레이션 레코드:**

```bash
kubectl -n aerospike get asc aerospike-ce-3node \
  -o jsonpath='{.status.pods}' | jq 'to_entries[] | {pod: .key, migrating: .value.migratingPartitions}'
```

**jsonpath로 빠르게 확인:**

```bash
# 마이그레이션 진행 중인지 확인 (true/false 반환)
kubectl -n aerospike get asc aerospike-ce-3node \
  -o jsonpath='InProgress={.status.migrationStatus.inProgress} Remaining={.status.migrationStatus.remainingPartitions}'

# MigrationComplete 조건 직접 확인
kubectl -n aerospike get asc aerospike-ce-3node \
  -o jsonpath='{range .status.conditions[?(@.type=="MigrationComplete")]}{.status}{end}'
```

**Prometheus 메트릭:**

오퍼레이터는 `acko_cluster_migrating_records`를 `namespace`와 `name` 레이블이 있는 Prometheus 게이지 메트릭으로 노출합니다. 장기 마이그레이션에 대한 알림 설정이 가능합니다:

```promql
# 30분 이상 마이그레이션이 진행 중일 때 알림
acko_cluster_migrating_records{namespace="aerospike", name="aerospike-ce-3node"} > 0

# 마이그레이션 진행 속도 추적 (초당 마이그레이션된 레코드)
deriv(acko_cluster_migrating_records[5m])

# 멈춘 마이그레이션 알림 (남은 레코드가 줄어들지 않음)
deriv(acko_cluster_migrating_records[10m]) >= 0
  and acko_cluster_migrating_records > 0
```

:::tip
`status.conditions`의 `MigrationComplete` 조건은 `migrationStatus.remainingPartitions`가 0에 도달하면 `True`로 설정됩니다. 전체 마이그레이션 상태를 파싱하지 않고 간단한 상태 확인에 활용하세요.
:::

## 재조정 일시 중지

오퍼레이터의 재조정을 임시로 중지합니다:

```yaml
spec:
  paused: true
```

일시 중지된 동안 오퍼레이터는 모든 재조정을 건너뜁니다. 다시 `false`로 설정하거나 필드를 제거하면 재개됩니다.

```bash
# 일시 중지
kubectl -n aerospike patch asc aerospike-ce-3node --type merge -p '{"spec":{"paused":true}}'

# 재개
kubectl -n aerospike patch asc aerospike-ce-3node --type merge -p '{"spec":{"paused":null}}'
```

## 온디맨드 오퍼레이션

`spec.operations`를 통해 파드 재시작을 선언적으로 트리거합니다. 한 번에 하나의 오퍼레이션만 활성화할 수 있습니다.

### WarmRestart

Aerospike 프로세스에 SIGUSR1을 보내 파드 삭제 없이 graceful restart합니다:

```yaml
spec:
  operations:
    - kind: WarmRestart
      id: "config-reload-v2"       # 고유 ID (1-20자)
      podList:                      # 선택사항: 비우면 전체 파드 대상
        - aerospike-ce-3node-0
        - aerospike-ce-3node-1
```

### PodRestart

파드를 삭제하고 재생성합니다 (cold restart):

```yaml
spec:
  operations:
    - kind: PodRestart
      id: "cold-restart-01"
      podList:
        - aerospike-ce-3node-2
```

### 오퍼레이션 상태 확인

```bash
kubectl -n aerospike get asc aerospike-ce-3node -o jsonpath='{.status.operationStatus}' | jq .
```

상태에는 `phase` (`InProgress`, `Completed`, `Error`), `completedPods`, `failedPods`가 포함됩니다.

:::warning
- InProgress 중에는 오퍼레이션을 변경할 수 없습니다
- 오퍼레이션 `id`는 고유해야 합니다 (1-20자)
- 완료 후 spec에서 오퍼레이션을 제거하세요
:::

## 서비스 커스터마이징

### Headless 서비스 메타데이터

Headless 서비스(파드 디스커버리용)에 커스텀 어노테이션과 레이블을 추가합니다:

```yaml
spec:
  headlessService:
    metadata:
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8687"
      labels:
        monitoring: enabled
```

### Pod별 서비스

`podService`를 설정하면 오퍼레이터가 각 파드에 대해 개별 ClusterIP Service를 생성하여 직접적인 파드 레벨 접근을 가능하게 합니다:

```yaml
spec:
  podService:
    metadata:
      annotations:
        external-dns.alpha.kubernetes.io/hostname: "aero.example.com"
      labels:
        service-type: pod-local
```

**사용 사례:** 외부 DNS 연동, 파드 레벨 로드밸런싱, 특정 파드에 대한 직접 클라이언트 접근.

## 검증 정책

웹훅 검증 동작을 제어합니다:

```yaml
spec:
  validationPolicy:
    skipWorkDirValidate: true   # 작업 디렉토리 PV 검증 건너뛰기
```

| 필드 | 기본값 | 설명 |
|---|---|---|
| `skipWorkDirValidate` | `false` | 작업 디렉토리가 영구 스토리지에 있는지 검증을 건너뜁니다 |

개발 환경이나 영구 작업 디렉토리가 필요 없는 인메모리 배포에 유용합니다.

## 랙 ID 오버라이드

파드 어노테이션을 통한 동적 랙 ID 할당을 활성화합니다:

```yaml
spec:
  enableRackIDOverride: true
```

활성화하면 오퍼레이터가 엄격하게 관리하는 대신 파드 어노테이션으로 랙 ID를 오버라이드할 수 있습니다. 수동 랙 관리 시나리오에 유용합니다.

## 스토리지

### 볼륨 타입

| 소스 | 사용 사례 |
|---|---|
| `persistentVolume` | 영구 데이터 (파드 재시작 시 유지) |
| `emptyDir` | 임시 작업 공간 |
| `secret` | 자격 증명, TLS 인증서 |
| `configMap` | 커스텀 설정 파일 |
| `hostPath` | 노드 로컬 경로 (개발/테스트 전용) |

### 글로벌 볼륨 정책

카테고리별(파일시스템 또는 블록) 모든 영구 볼륨의 기본 정책을 설정합니다. 볼륨별 설정이 항상 글로벌 정책보다 우선합니다.

```yaml
storage:
  filesystemVolumePolicy:
    initMethod: deleteFiles
    wipeMethod: deleteFiles
    cascadeDelete: true
  blockVolumePolicy:
    initMethod: blkdiscard
    wipeMethod: blkdiscardWithHeaderCleanup
```

오퍼레이터는 다음 우선순위로 설정을 결정합니다:
1. **볼륨별** `initMethod` / `wipeMethod` / `cascadeDelete`
2. **글로벌 정책** (`volumeMode` 기반: `filesystemVolumePolicy` 또는 `blockVolumePolicy`)
3. **기본값**: `none` / `none` / `false`

### Cascade Delete

`cascadeDelete: true`이면 AerospikeCluster CR 삭제 시 PVC가 자동으로 삭제됩니다. 볼륨별 또는 글로벌 정책으로 설정할 수 있습니다.

```yaml
storage:
  # 글로벌: CR 삭제 시 모든 파일시스템 PVC 삭제
  filesystemVolumePolicy:
    cascadeDelete: true
  volumes:
    - name: data-vol
      source:
        persistentVolume:
          storageClass: standard
          size: 10Gi
      # filesystemVolumePolicy에서 cascadeDelete: true 상속
```

### 볼륨 초기화 방법

| 방법 | 설명 |
|---|---|
| `none` | 초기화 없음 (기본값) |
| `deleteFiles` | 볼륨의 모든 파일 삭제 |
| `dd` | `dd`로 볼륨 제로 채우기 |
| `blkdiscard` | 블록 디스카드 (블록 디바이스 전용) |
| `headerCleanup` | Aerospike 파일 헤더만 정리 |

```yaml
storage:
  volumes:
    - name: data-vol
      initMethod: deleteFiles
```

### 와이프 방법

와이프 방법은 초기화 방법과 유사하지만 **더티 볼륨** (비정상 종료 후 정리가 필요한 볼륨)에 적용됩니다. `wipeMethod` 필드는 다음 값을 지원합니다:

| 방법 | 설명 |
|---|---|
| `none` | 와이프 없음 (기본값) |
| `deleteFiles` | 볼륨의 모든 파일 삭제 |
| `dd` | `dd`를 사용하여 디바이스를 제로 필 |
| `blkdiscard` | 디바이스의 모든 블록 디스카드 |
| `headerCleanup` | Aerospike 파일 헤더만 정리 |
| `blkdiscardWithHeaderCleanup` | 블록 디스카드 후 Aerospike 헤더 정리 |

```yaml
storage:
  volumes:
    - name: data-vol
      wipeMethod: headerCleanup
```

### HostPath 볼륨

:::warning
HostPath 볼륨은 **프로덕션에 권장되지 않습니다**. 데이터가 특정 노드에 종속되어 파드 재스케줄링 시 이동이 불가능합니다.
:::

```yaml
storage:
  volumes:
    - name: host-logs
      source:
        hostPath:
          path: /var/log/aerospike
          type: DirectoryOrCreate
      aerospike:
        path: /opt/aerospike/logs
```

### PVC 커스텀 메타데이터

PersistentVolumeClaim에 커스텀 라벨과 어노테이션을 추가합니다:

```yaml
storage:
  volumes:
    - name: data-vol
      source:
        persistentVolume:
          storageClass: standard
          size: 50Gi
          metadata:
            labels:
              backup-policy: "daily"
            annotations:
              volume.kubernetes.io/storage-provisioner: "ebs.csi.aws.com"
```

### 볼륨 마운트 옵션

Aerospike 및 사이드카 컨테이너의 고급 마운트 옵션:

```yaml
storage:
  volumes:
    - name: shared-data
      source:
        emptyDir: {}
      aerospike:
        path: /opt/aerospike/shared
        readOnly: false
        subPath: "aerospike-data"       # 하위 디렉토리 마운트
        mountPropagation: HostToContainer
      sidecars:
        - containerName: exporter
          path: /shared
          readOnly: true
```

| 옵션 | 설명 |
|---|---|
| `readOnly` | 볼륨을 읽기 전용으로 마운트 |
| `subPath` | 볼륨의 특정 하위 디렉토리 마운트 |
| `subPathExpr` | subPath와 유사하나 환경 변수 확장 지원 |
| `mountPropagation` | 마운트 전파 제어 (`None`, `HostToContainer`, `Bidirectional`) |

:::note
`subPath`와 `subPathExpr`는 상호 배타적입니다.
:::

### 로컬 스토리지

파드 재시작 시 특수 처리를 위해 스토리지 클래스를 로컬로 표시합니다:

```yaml
storage:
  localStorageClasses:
    - local-path
    - openebs-hostpath
  deleteLocalStorageOnRestart: true
```

`deleteLocalStorageOnRestart: true`이면 오퍼레이터는 콜드 리스타트 시 파드 삭제 **전에** 로컬 스토리지 클래스로 지원되는 PVC를 삭제합니다. 로컬 스토리지는 노드에 종속되므로 새 노드에서 재프로비저닝이 필요합니다.

## 네트워크 설정

### 접근 타입

클라이언트가 Aerospike 노드를 발견하고 연결하는 방법을 제어합니다:

| 타입 | 설명 |
|---|---|
| `pod` | 파드 IP (기본값, 클러스터 내부 클라이언트) |
| `hostInternal` | 노드 내부 IP |
| `hostExternal` | 노드 외부 IP |
| `configuredIP` | 파드 어노테이션의 커스텀 IP |

```yaml
spec:
  aerospikeNetworkPolicy:
    accessType: pod
    alternateAccessType: hostExternal
    fabricType: pod
```

### LoadBalancer

LoadBalancer 서비스로 클러스터를 노출합니다:

```yaml
spec:
  seedsFinderServices:
    loadBalancer:
      port: 3000
      targetPort: 3000
      annotations:
        service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
      loadBalancerSourceRanges:
        - "10.0.0.0/8"
```

### NetworkPolicy

자동 NetworkPolicy 생성을 활성화합니다:

```yaml
spec:
  networkPolicyConfig:
    enabled: true
    type: kubernetes    # 또는 CiliumNetworkPolicy용 "cilium"
```

### 대역폭 제한

CNI 어노테이션을 통한 대역폭 제한 설정 (예: Cilium):

```yaml
spec:
  bandwidthConfig:
    ingress: "1Gbps"
    egress: "1Gbps"
```

## Pod Disruption Budget

기본적으로 오퍼레이터는 유지보수 중 클러스터를 보호하기 위해 PodDisruptionBudget을 생성합니다.

### PDB 비활성화

```yaml
spec:
  disablePDB: true
```

### 커스텀 MaxUnavailable

```yaml
spec:
  maxUnavailable: 1         # 정수 또는 "25%" 같은 퍼센트 문자열 가능
```

## 호스트 네트워크

직접 노드 포트 접근을 위한 호스트 네트워킹 활성화:

```yaml
spec:
  podSpec:
    hostNetwork: true
    # 자동으로 적용되는 기본값:
    #   multiPodPerHost: false
    #   dnsPolicy: ClusterFirstWithHostNet
```

## 랙 라벨 스케줄링

`rackLabel`을 사용하여 특정 라벨이 있는 노드에 파드를 스케줄링합니다. 오퍼레이터가 `acko.io/rack=<rackLabel>`에 대한 노드 어피니티를 설정합니다:

```yaml
spec:
  rackConfig:
    racks:
      - id: 1
        rackLabel: zone-a    # acko.io/rack=zone-a 노드에 스케줄링
      - id: 2
        rackLabel: zone-b
      - id: 3
        rackLabel: zone-c
```

:::warning
`rackLabel` 값은 랙 간에 고유해야 합니다.
:::

제어된 마이그레이션을 위해 각 랙에 `revision`을 부여할 수도 있습니다:

```yaml
spec:
  rackConfig:
    racks:
      - id: 1
        rackLabel: zone-a
        revision: "v1.0"     # 마이그레이션 추적용 버전 식별자
```

## 노드 스케줄링

### 노드 셀렉터

```yaml
spec:
  podSpec:
    nodeSelector:
      node-type: aerospike
```

### Tolerations

```yaml
spec:
  podSpec:
    tolerations:
      - key: "dedicated"
        operator: "Equal"
        value: "aerospike"
        effect: "NoSchedule"
```

### 노드 블록 리스트

특정 노드에서 스케줄링을 방지합니다:

```yaml
spec:
  k8sNodeBlockList:
    - node-maintenance-01
    - node-maintenance-02
```

## 트러블슈팅

### 클러스터 Phase 확인

```bash
kubectl -n aerospike get asc
```

| Phase | 의미 |
|---|---|
| `InProgress` | 재조정 진행 중 (일반) |
| `Completed` | 클러스터 정상, 최신 상태 |
| `Error` | 재조정 중 복구 불가능한 오류 발생 |
| `ScalingUp` | 클러스터 스케일 업 중 (파드 추가) |
| `ScalingDown` | 클러스터 스케일 다운 중 (파드 제거) |
| `RollingRestart` | 롤링 리스타트 진행 중 |
| `ACLSync` | ACL 역할 및 사용자 동기화 중 |
| `Paused` | 사용자에 의해 재조정 일시 중지됨 |
| `Deleting` | 클러스터 삭제 중 |

### Conditions 확인

```bash
kubectl -n aerospike get asc aerospike-ce-3node -o jsonpath='{.status.conditions}' | jq .
```

### 파드 상태 확인

```bash
kubectl -n aerospike get asc aerospike-ce-3node -o jsonpath='{.status.pods}' | jq .
```

각 파드 상태에 포함된 정보:

| 필드 | 설명 |
|---|---|
| `podIP` | 파드 IP 주소 |
| `hostIP` | 노드 IP 주소 |
| `image` | 실행 중인 컨테이너 이미지 |
| `rack` | 랙 ID |
| `isRunningAndReady` | 파드 정상 여부 |
| `configHash` | 적용된 설정의 SHA256 해시 |
| `dynamicConfigStatus` | `Applied`, `Failed`, `Pending`, 또는 빈 문자열 |
| `nodeID` | Aerospike가 할당한 노드 식별자 (예: `BB9020012AC4202`) |
| `clusterName` | 노드가 보고한 Aerospike 클러스터 이름 |
| `accessEndpoints` | 직접 클라이언트 접근용 네트워크 엔드포인트 |
| `readinessGateSatisfied` | `acko.io/aerospike-ready` 게이트 충족 시 `true` (`readinessGateEnabled: true` 필요) |
| `lastRestartReason` | 파드가 마지막으로 재시작된 이유: `ConfigChanged`, `ImageChanged`, `PodSpecChanged`, `ManualRestart`, `WarmRestart` |
| `lastRestartTime` | 오퍼레이터에 의한 마지막 재시작 타임스탬프 |
| `unstableSince` | 이 파드가 처음 NotReady가 된 시점; Ready 시 초기화 |
| `migratingPartitions` | 이 파드가 현재 마이그레이션 중인 파티션 수; 접근 불가 시 `nil` |

### 오퍼레이터 로그 확인

```bash
kubectl -n aerospike-operator logs -l control-plane=controller-manager -f
```

### 일반적인 문제

**Phase가 InProgress에서 멈춤:**
- 오퍼레이터 로그에서 오류 세부사항 확인
- 스토리지 클래스 존재 여부 확인: `kubectl get sc`
- 이미지 풀 가능 여부 확인: `kubectl -n aerospike describe pod <파드-이름>`

**파드 CrashLoopBackOff:**
- Aerospike 로그 확인: `kubectl -n aerospike logs <파드-이름> -c aerospike`
- `aerospikeConfig`가 유효한지 확인 (네임스페이스 이름, 스토리지 경로)

**웹훅 거부:**
- 오류 메시지를 읽으세요 — 웹훅이 CE 제약 조건을 검증합니다
- [CE 검증 규칙](./create-cluster#ce-검증-규칙)을 확인하세요

## Kubernetes 이벤트

오퍼레이터는 모든 중요한 라이프사이클 전환에 대해 Kubernetes Event를 발생시킵니다.
`kubectl get events`를 사용하여 클러스터 활동을 실시간으로 관찰할 수 있습니다:

```bash
# 특정 클러스터의 이벤트 감시
kubectl get events --field-selector involvedObject.name=my-cluster -w

# 네임스페이스의 모든 AerospikeCluster 이벤트 조회
kubectl get events --field-selector involvedObject.kind=AerospikeCluster -n aerospike
```

### 이벤트 레퍼런스

| Reason | Type | 설명 |
|--------|------|------|
| `RollingRestartStarted` | Normal | 롤링 리스타트 루프 시작; 랙 ID와 파드 수 표시 |
| `RollingRestartCompleted` | Normal | 모든 대상 파드에 대해 롤링 리스타트 완료 |
| `PodWarmRestarted` | Normal | 파드가 SIGUSR1 수신 (무중단 설정 리로드) |
| `PodColdRestarted` | Normal | 파드가 삭제 후 재생성됨 (풀 리스타트) |
| `RestartFailed` | Warning | 롤링 리스타트 중 파드 재시작 실패 |
| `LocalPVCDeleteFailed` | Warning | 콜드 리스타트 전 로컬 PVC 삭제 실패 |
| `ConfigMapCreated` | Normal | 랙 ConfigMap 최초 생성 |
| `ConfigMapUpdated` | Normal | 랙 ConfigMap 새 설정으로 업데이트 |
| `DynamicConfigApplied` | Normal | 재시작 없이 파드에 설정 변경 적용 |
| `DynamicConfigStatusFailed` | Warning | 동적 설정 상태 업데이트 실패 |
| `StatefulSetCreated` | Normal | 랙 StatefulSet 최초 생성 |
| `StatefulSetUpdated` | Normal | 랙 StatefulSet 스펙 업데이트 |
| `RackScaled` | Normal | 랙 레플리카 수 변경; 이전/새 수 표시 |
| `ACLSyncStarted` | Normal | ACL 역할/사용자 동기화 시작 |
| `ACLSyncCompleted` | Normal | ACL 역할과 사용자 동기화 성공 |
| `ACLSyncError` | Warning | ACL 동기화 중 오류 발생 |
| `PDBCreated` | Normal | PodDisruptionBudget 생성 |
| `PDBUpdated` | Normal | PodDisruptionBudget 업데이트 |
| `ServiceCreated` | Normal | Headless 서비스 생성 |
| `ServiceUpdated` | Normal | Headless 서비스 업데이트 |
| `ClusterDeletionStarted` | Normal | 클러스터 삭제 시작 (finalizer 활성) |
| `FinalizerRemoved` | Normal | 스토리지 finalizer 제거; 객체 삭제 예정 |
| `ReadinessGateSatisfied` | Normal | 파드 readiness gate `acko.io/aerospike-ready`가 True로 설정됨 |
| `ReadinessGateBlocking` | Warning | Readiness gate 대기 중 롤링 리스타트 차단됨 |
| `TemplateApplied` | Normal | ClusterTemplate 스펙이 이 클러스터에 적용됨 |
| `TemplateDrifted` | Warning | 클러스터 스펙이 템플릿에서 드리프트됨 |
| `TemplateResolutionError` | Warning | ClusterTemplate 해결 또는 적용 실패 |
| `ValidationWarning` | Warning | 비차단 검증 경고 감지 |
| `ReconcileError` | Warning | 재조정 루프에서 복구 불가능한 오류 발생 |
| `Operation` | Normal | 온디맨드 오퍼레이션 이벤트 |
