---
name: acko-cr-guide
description: AerospikeCluster CR 종합 가이드 — YAML 예제, lifecycle/status, webhook 검증, Day-2 운영
disable-model-invocation: false
---

# AerospikeCluster CR Guide

## 1. YAML 예제

사용자 요청에 맞는 시나리오를 찾아 출력. 커스텀 요청이면 조합.

| # | 시나리오 | 파일 | 핵심 |
|---|---------|------|------|
| 1 | Minimal (1-node in-memory) | [./examples/01-minimal.yaml](./examples/01-minimal.yaml) | 스토리지 없음, `replication-factor: 1` |
| 2 | 3-Node PV Storage | [./examples/02-3node-pv.yaml](./examples/02-3node-pv.yaml) | PVC, device storage, cascadeDelete |
| 3 | ACL | [./examples/03-acl.yaml](./examples/03-acl.yaml) | security 스탠자, admin 필수, Secret |
| 4 | Monitoring | [./examples/04-monitoring.yaml](./examples/04-monitoring.yaml) | exporter, ServiceMonitor, PrometheusRule |
| 5 | Multi-Rack | [./examples/05-multirack.yaml](./examples/05-multirack.yaml) | rackConfig, zone, 랙 인식 복제 |
| 6 | Advanced Storage | [./examples/06-storage-advanced.yaml](./examples/06-storage-advanced.yaml) | block/hostPath/CSI/local PV, 정책 |
| 7 | Template | [./examples/07-template.yaml](./examples/07-template.yaml) | templateRef, overrides, resync |
| 8 | Full-Featured | [./examples/08-full-featured.yaml](./examples/08-full-featured.yaml) | ACL+monitoring+rack+PV+PDB+dynamic |

## 2. Lifecycle & Status

### Phase (10개)

| Phase | 의미 |
|-------|------|
| `InProgress` | 조정 진행 중 |
| `Completed` | 안정 상태 |
| `Error` | 복구 불가 에러 |
| `ScalingUp` / `ScalingDown` | 파드 추가/제거 |
| `WaitingForMigration` | 마이그레이션 완료 대기 (스케일다운 보류) |
| `RollingRestart` | 롤링 재시작 진행 |
| `ACLSync` | ACL 동기화 |
| `Paused` | spec.paused=true |
| `Deleting` | 삭제 처리 중 |

### Conditions (6개)

| Type | True | False |
|------|------|-------|
| `Available` | 1개+ 파드 Ready | 전체 NotReady |
| `Ready` | 전체 Ready | 일부 NotReady |
| `ConfigApplied` | 설정 적용 완료 | 미적용 파드 있음 |
| `ACLSynced` | ACL 동기화 완료 | 실패/대기 |
| `MigrationComplete` | 마이그레이션 없음 | 진행 중 |
| `ReconciliationPaused` | paused=true | 활성 |

### Circuit Breaker (미문서화)
연속 실패 **10회** 초과 시 백오프: `min(2^n, 300)`초. Events: `CircuitBreakerActive` / `CircuitBreakerReset`

### 상세: [./ref/events.md](./ref/events.md) (이벤트 31개) · [./ref/troubleshooting.md](./ref/troubleshooting.md) (진단 + kubectl)

## 3. Webhook

### 자동 설정 (생략 가능)

| 필드 | 기본값 |
|------|--------|
| `service.cluster-name` | CR name |
| `service.proto-fd-max` | 15000 |
| `network.service/fabric/heartbeat.port` | 3000 / 3001 / 3002 |
| `heartbeat.mode` | mesh |
| `monitoring.exporterImage` | `aerospike/aerospike-prometheus-exporter:v1.16.1` |
| `monitoring.port` | 9145 |

### CE 제약 (위반 시 거부)
- `size` ≤ 8, `namespaces` ≤ 2, `xdr`/`tls` 금지, Enterprise 이미지 금지, `heartbeat.mode`=mesh

### 바이트 값 (정수 필수)
1 GiB=`1073741824` · 2 GiB=`2147483648` · 4 GiB=`4294967296` · 8 GiB=`8589934592`

### 상세: [./ref/validation-rules.md](./ref/validation-rules.md) (에러 53개 + 경고 15개)

## 4. Day-2 운영

| 운영 | 명령 | Phase |
|------|------|-------|
| Scale Up/Down | `kubectl patch asc <name> --type=merge -p '{"spec":{"size":N}}'` | ScalingUp/Down → Completed |
| Image 업그레이드 | `kubectl patch asc <name> --type=merge -p '{"spec":{"image":"..."}}'` | RollingRestart |
| Dynamic Config | `enableDynamicConfigUpdate: true` + config patch | Completed (재시작 없음) |
| WarmRestart | `operations: [{kind: WarmRestart, id: "..."}]` | RollingRestart |
| Template Resync | `kubectl annotate asc <name> acko.io/resync-template=true` | — |
| Pause / Resume | `spec.paused: true / null` | Paused ↔ InProgress |
| 삭제 | `kubectl delete asc <name>` | Deleting |

### 상세: [./ref/operations.md](./ref/operations.md) (11개 운영 시나리오 + kubectl 명령)

## 5. Quick Reference

| 하고 싶은 것 | 설정 필드 | 시나리오 |
|---|---|---|
| 노드 수 | `spec.size` (CE ≤ 8) | 전체 |
| 영구 스토리지 | `storage.volumes[].source.persistentVolume` | 2, 6 |
| ACL/보안 | `aerospikeAccessControl` + `aerospikeConfig.security` | 3, 8 |
| Prometheus | `monitoring.enabled: true` | 4, 8 |
| 가용 영역 분산 | `rackConfig.racks[].zone` | 5, 8 |
| 템플릿 | `templateRef.name` | 7 |
| Dynamic config | `enableDynamicConfigUpdate: true` | 8 |
| Batch size | `rollingUpdateBatchSize` | 8 |
| Pause | `paused: true` | 8 |
| PDB 비활성화 | `disablePDB: true` | 8 |
| 블록 볼륨 | `persistentVolume.volumeMode: Block` | 6 |
| 사이드카 공유 | `volumes[].sidecars[]` | 6 |
| PVC 자동 삭제 | `cascadeDelete: true` | 2, 3 |
| ServiceMonitor | `monitoring.serviceMonitor.enabled` | 4, 8 |
