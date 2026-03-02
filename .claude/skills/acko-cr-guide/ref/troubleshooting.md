# 트러블슈팅 가이드

## 증상별 진단

| 증상 | 확인 명령 | 원인 | 해결 |
|------|----------|------|------|
| Phase = `Error` | `kubectl get asc <name> -o jsonpath='{.status.lastReconcileError}'` | 잘못된 config, 이미지 풀 실패, 리소스 부족 | 에러 메시지 기반 수정 후 apply |
| Phase = `WaitingForMigration` | `kubectl exec <pod> -- asinfo -v 'statistics' \| grep migrate` | 데이터 마이그레이션 진행 중 | 완료 대기 (자동 재시도) |
| `InProgress`에서 멈춤 | `kubectl get pvc -n <ns> -l aerospike.io/cr-name=<name>` | PVC Pending, ImagePull 실패, 스케줄링 실패 | StorageClass/이미지/리소스 확인 |
| `CircuitBreakerActive` 이벤트 | `kubectl get asc <name> -o jsonpath='{.status.failedReconcileCount}'` | 연속 10회+ 실패 | lastReconcileError 확인 → 근본 원인 수정 |
| Pod `CrashLoopBackOff` | `kubectl logs <pod> -c aerospike-server --previous` | config 파싱 오류, 메모리 부족 | 서버 로그 확인 → config 수정 |
| Webhook이 CR 거부 | `kubectl apply` 에러 메시지 확인 | CE 제약 위반 | `/acko-webhook-validation` 참조 |
| `dynamicConfigStatus=Failed` | `kubectl get asc <name> -o jsonpath='{.status.pods}' \| jq '.[].dynamicConfigStatus'` | 동적 변경 불가 파라미터 | `enableDynamicConfigUpdate: false`로 변경하여 롤링 재시작 유도 |
| `ReadinessGateBlocking` | `kubectl get pod <pod> -o jsonpath='{.status.conditions}' \| jq '.[]'` | readiness gate 미충족 | 파드 Aerospike 상태 점검 |

## 유용한 kubectl 명령

```bash
# 클러스터 상태
kubectl get asc -n <ns>                                          # 목록 + PHASE
kubectl get asc <name> -o jsonpath='{.status.phase}'            # Phase
kubectl get asc <name> -o jsonpath='{.status.phaseReason}'      # Phase 이유
kubectl get asc <name> -o jsonpath='{.status.conditions}' | jq . # Conditions
kubectl get asc <name> -o jsonpath='{.status.failedReconcileCount}' # Circuit breaker
kubectl get asc <name> -o jsonpath='{.status.lastReconcileError}'   # 마지막 에러

# Pod 상태
kubectl get asc <name> -o jsonpath='{.status.pods}' | jq .
kubectl get asc <name> -o jsonpath='{.status.size}'             # Ready 파드 수
kubectl get asc <name> -o jsonpath='{.status.pendingRestartPods}'

# 이벤트
kubectl get events -n <ns> --field-selector involvedObject.name=<name> --sort-by='.lastTimestamp'
kubectl get events -n <ns> -w

# 로그
kubectl -n aerospike-operator logs -l control-plane=controller-manager -f  # Operator
kubectl -n <ns> logs <pod> -c aerospike-server -f                          # Aerospike
kubectl -n <ns> logs <pod> -c aerospike-server --previous                  # 이전 크래시

# Template
kubectl get asc <name> -o jsonpath='{.status.templateSnapshot.synced}'
```
