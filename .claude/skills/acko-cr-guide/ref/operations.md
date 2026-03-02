# Day-2 Operations 상세

## 1. 스케일링

### Scale Up
```bash
kubectl patch asc <name> -n <ns> --type=merge -p '{"spec":{"size":5}}'
```
Phase: `ScalingUp` → `Completed`. CE 최대: 8.

### Scale Down
```bash
kubectl patch asc <name> -n <ns> --type=merge -p '{"spec":{"size":3}}'
```
Phase: `ScalingDown` → (`WaitingForMigration`) → `Completed`.
마이그레이션 진행 중이면 자동 보류 → 완료 후 자동 재시도.

### Batch Size
```yaml
spec:
  rackConfig:
    scaleDownBatchSize: "1"    # 1 pod per rack at a time
    maxIgnorablePods: 1        # Allow 1 stuck pod without blocking
```

---

## 2. 롤링 업데이트

### 이미지 업그레이드
```bash
kubectl patch asc <name> -n <ns> --type=merge -p '{"spec":{"image":"aerospike:ce-8.1.1.1"}}'
```

### Static Config 변경 (재시작 필요)
```bash
kubectl patch asc <name> -n <ns> --type=merge \
  -p '{"spec":{"aerospikeConfig":{"service":{"proto-fd-max":20000}}}}'
```

### Dynamic Config 변경 (재시작 없음)
```yaml
spec:
  enableDynamicConfigUpdate: true
  aerospikeConfig:
    service:
      proto-fd-max: 20000           # Dynamic
    namespaces:
      - name: test
        high-water-memory-pct: 70   # Dynamic
        stop-writes-pct: 90         # Dynamic
```
확인: `kubectl get asc <name> -o jsonpath='{.status.pods}' | jq '.[].dynamicConfigStatus'`
- `Applied`: 성공 / `Failed`: 실패 → 롤링 재시작 / `Pending`: 진행 중

### Batch Size
```yaml
spec:
  rollingUpdateBatchSize: 1         # Global (integer or "25%")
  rackConfig:
    rollingUpdateBatchSize: "50%"   # Per-rack override
```

---

## 3. 온디맨드 재시작

한 번에 1개 operation만. 완료 후 spec에서 제거 필수.

### WarmRestart (SIGUSR1)
```yaml
spec:
  operations:
    - kind: WarmRestart
      id: warm-001          # 1-20 chars, unique
      # podList: [...]      # Optional: specific pods
```

### PodRestart (Cold)
```yaml
spec:
  operations:
    - kind: PodRestart
      id: cold-001
      podList:
        - <cluster>-0-2     # Optional
```

### 상태 확인 / 정리
```bash
kubectl get asc <name> -o jsonpath='{.status.operationStatus}' | jq .
kubectl patch asc <name> -n <ns> --type=merge -p '{"spec":{"operations":null}}'
```

---

## 4. ACL 관리

### User 추가
```bash
kubectl create secret generic new-user-secret -n <ns> --from-literal=password=<pw>
kubectl patch asc <name> -n <ns> --type=json \
  -p '[{"op":"add","path":"/spec/aerospikeAccessControl/users/-","value":{"name":"new-user","roles":["reader"],"secretName":"new-user-secret"}}]'
```

### 비밀번호 변경
```bash
kubectl create secret generic <secret> -n <ns> --from-literal=password=<new-pw> --dry-run=client -o yaml | kubectl apply -f -
kubectl patch asc <name> -n <ns> --type=merge -p '{"spec":{"operations":[{"kind":"WarmRestart","id":"pw-change-001"}]}}'
```

---

## 5. 템플릿 운영

### Resync
```bash
kubectl annotate asc <name> -n <ns> acko.io/resync-template=true
```
annotation은 resync 완료 후 자동 제거.

### Sync 상태
```bash
kubectl get asc <name> -o jsonpath='{.status.templateSnapshot.synced}'  # true/false
kubectl get events -n <ns> --field-selector reason=TemplateDrifted
```

---

## 6. Pause / Resume

```bash
kubectl patch asc <name> -n <ns> --type=merge -p '{"spec":{"paused":true}}'   # Pause
kubectl patch asc <name> -n <ns> --type=merge -p '{"spec":{"paused":null}}'    # Resume
```

---

## 7. Readiness Gate

```yaml
spec:
  podSpec:
    readinessGateEnabled: true   # Triggers rolling restart when toggled
```
```bash
kubectl get pod <pod> -o jsonpath='{.status.conditions}' | jq '.[] | select(.type=="acko.io/aerospike-ready")'
```

---

## 8. 네트워크

### Access Type
```yaml
spec:
  aerospikeNetworkPolicy:
    accessType: pod          # pod | hostInternal | hostExternal | configuredIP
```

### LoadBalancer
```yaml
spec:
  seedsFinderServices:
    loadBalancer:
      port: 3000
      annotations:
        service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
```

### NetworkPolicy
```yaml
spec:
  networkPolicyConfig:
    enabled: true
    type: kubernetes   # kubernetes | cilium
```

---

## 9. PDB / 유지보수

```yaml
spec:
  disablePDB: false           # PDB enabled (default)
  maxUnavailable: 1           # Integer or "25%"
  k8sNodeBlockList:
    - node-to-drain-01        # Block before draining
```

---

## 10. 복구

### Circuit Breaker
```bash
kubectl get asc <name> -o jsonpath='{.status.failedReconcileCount}'   # Threshold: 10
kubectl get asc <name> -o jsonpath='{.status.lastReconcileError}'
# Fix root cause → operator auto-retries with backoff: min(2^n, 300) sec
```

### WaitingForMigration
```bash
kubectl exec -n <ns> <pod> -c aerospike-server -- asinfo -v 'statistics' | grep -i migrat
# Wait for completion → operator auto-proceeds
```

### InProgress 고착
```bash
kubectl get events -n <ns> --field-selector involvedObject.name=<name> --sort-by='.lastTimestamp'
kubectl get pvc -n <ns> -l aerospike.io/cr-name=<name>
kubectl -n aerospike-operator logs -l control-plane=controller-manager --tail=100
```

---

## 11. 클러스터 삭제

```bash
kubectl delete asc <name> -n <ns>
```
1. `ClusterDeletionStarted` → Phase `Deleting`
2. `cascadeDelete: true` PVC 자동 삭제
3. `cascadeDelete: false` PVC 수동 삭제: `kubectl delete pvc -n <ns> -l aerospike.io/cr-name=<name>`
4. `FinalizerRemoved` → CR 삭제
