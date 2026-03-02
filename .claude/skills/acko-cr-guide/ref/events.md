# Kubernetes Events 전체 참조

## 롤링 재시작

| Event Reason | 타입 | 발생 시점 |
|---|---|---|
| `RollingRestartStarted` | Normal | 롤링 재시작 배치 시작 |
| `RollingRestartCompleted` | Normal | 모든 재시작 대상 파드 완료 |
| `RestartFailed` | Warning | 파드 재시작 실패 |
| `PodWarmRestarted` | Normal | SIGUSR1 warm restart 완료 |
| `PodColdRestarted` | Normal | 파드 삭제 후 cold restart 완료 |

## Quiesce

| Event Reason | 타입 | 발생 시점 |
|---|---|---|
| `NodeQuiesceStarted` | Normal | 노드 quiesce 시작 |
| `NodeQuiesced` | Normal | 노드 quiesce 완료 |
| `NodeQuiesceFailed` | Warning | 노드 quiesce 실패 |

## 설정 관리

| Event Reason | 타입 | 발생 시점 |
|---|---|---|
| `ConfigMapCreated` | Normal | 새 rack용 ConfigMap 생성 |
| `ConfigMapUpdated` | Normal | ConfigMap 내용 변경 |
| `DynamicConfigApplied` | Normal | 동적 설정(set-config) 성공 |
| `DynamicConfigStatusFailed` | Warning | 동적 설정 변경 실패 |

## StatefulSet / Rack

| Event Reason | 타입 | 발생 시점 |
|---|---|---|
| `StatefulSetCreated` | Normal | 새 rack용 StatefulSet 생성 |
| `StatefulSetUpdated` | Normal | StatefulSet 스펙 변경 |
| `RackScaled` | Normal | rack 파드 수 변경 |
| `ScaleDownDeferred` | Warning | 데이터 마이그레이션으로 스케일다운 보류 |
| `PVCCleanedUp` | Normal | 삭제된 파드의 PVC 정리 완료 |
| `PVCCleanupFailed` | Warning | PVC 정리 실패 |

## ACL

| Event Reason | 타입 | 발생 시점 |
|---|---|---|
| `ACLSyncStarted` | Normal | ACL 동기화 시작 |
| `ACLSyncCompleted` | Normal | ACL 동기화 완료 |
| `ACLSyncError` | Warning | ACL 동기화 실패 |

## PDB / Service

| Event Reason | 타입 | 발생 시점 |
|---|---|---|
| `PDBCreated` | Normal | PDB 생성 |
| `PDBUpdated` | Normal | PDB 업데이트 |
| `ServiceCreated` | Normal | 서비스 생성 |
| `ServiceUpdated` | Normal | 서비스 업데이트 |

## 클러스터 생명주기

| Event Reason | 타입 | 발생 시점 |
|---|---|---|
| `ClusterDeletionStarted` | Normal | 삭제 처리 시작 |
| `FinalizerRemoved` | Normal | finalizer 제거 직전 |

## 템플릿

| Event Reason | 타입 | 발생 시점 |
|---|---|---|
| `TemplateApplied` | Normal | 템플릿 스냅샷 적용 성공 |
| `TemplateResolutionError` | Warning | 템플릿 해석 실패 |
| `TemplateDrifted` | Warning | 참조 템플릿이 스냅샷 이후 변경됨 |

## Readiness Gate

| Event Reason | 타입 | 발생 시점 |
|---|---|---|
| `ReadinessGateSatisfied` | Normal | readiness gate 조건 충족 |
| `ReadinessGateBlocking` | Warning | gate 미충족으로 롤링 재시작 차단 |

## Circuit Breaker

| Event Reason | 타입 | 발생 시점 |
|---|---|---|
| `CircuitBreakerActive` | Warning | 연속 실패 10회 초과, 백오프 적용 |
| `CircuitBreakerReset` | Normal | 성공적 조정으로 해제 |

## 기타

| Event Reason | 타입 | 발생 시점 |
|---|---|---|
| `ValidationWarning` | Warning | 검증 경고 |
| `ReconcileError` | Warning | 조정 오류 |
| `Operation` | Normal | 온디맨드 오퍼레이션 처리 |
