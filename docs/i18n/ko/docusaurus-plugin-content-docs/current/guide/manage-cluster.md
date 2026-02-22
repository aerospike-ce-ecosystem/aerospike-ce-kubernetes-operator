---
sidebar_position: 3
title: 클러스터 관리
---

# Aerospike 클러스터 관리

이 가이드는 Day-2 운영을 다룹니다: 스케일링, 업데이트, 설정 변경, 트러블슈팅.

## 스케일링

`spec.size`를 변경하여 클러스터를 스케일 업/다운합니다.

```bash
kubectl -n aerospike patch asce aerospike-ce-3node --type merge -p '{"spec":{"size":5}}'
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

## 재조정 일시 중지

오퍼레이터의 재조정을 임시로 중지합니다:

```yaml
spec:
  paused: true
```

일시 중지된 동안 오퍼레이터는 모든 재조정을 건너뜁니다. 다시 `false`로 설정하거나 필드를 제거하면 재개됩니다.

```bash
# 일시 중지
kubectl -n aerospike patch asce aerospike-ce-3node --type merge -p '{"spec":{"paused":true}}'

# 재개
kubectl -n aerospike patch asce aerospike-ce-3node --type merge -p '{"spec":{"paused":null}}'
```

## 스토리지

### 볼륨 타입

| 소스 | 사용 사례 |
|---|---|
| `persistentVolume` | 영구 데이터 (파드 재시작 시 유지) |
| `emptyDir` | 임시 작업 공간 |
| `secret` | 자격 증명, TLS 인증서 |
| `configMap` | 커스텀 설정 파일 |

### Cascade Delete

`cascadeDelete: true`이면 AerospikeCECluster CR 삭제 시 PVC가 자동으로 삭제됩니다.

```yaml
storage:
  volumes:
    - name: data-vol
      cascadeDelete: true       # CR과 함께 PVC 삭제
      source:
        persistentVolume:
          storageClass: standard
          size: 10Gi
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
kubectl -n aerospike get asce
```

| Phase | 의미 |
|---|---|
| `InProgress` | 재조정 진행 중 |
| `Completed` | 클러스터 정상, 최신 상태 |
| `Error` | 재조정 중 오류 발생 |

### Conditions 확인

```bash
kubectl -n aerospike get asce aerospike-ce-3node -o jsonpath='{.status.conditions}' | jq .
```

### 파드 상태 확인

```bash
kubectl -n aerospike get asce aerospike-ce-3node -o jsonpath='{.status.pods}' | jq .
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
