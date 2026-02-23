# Aerospike CE Kubernetes Operator - Feature Implementation Plan

> AKO(Aerospike Kubernetes Operator) 공식 문서 분석 기반, 현재 구현 상태 대비 향후 구현해야 할 기능 TODO 리스트
>
> 작성일: 2026-02-22 (P0 구현 완료: 2026-02-23, P1 구현 완료: 2026-02-23)
> 참고: https://aerospike.com/docs/kubernetes/
> PR(P0): https://github.com/KimSoungRyoul/aerospike-ce-kubernetes-operator/pull/7
> PR(P1): https://github.com/KimSoungRyoul/aerospike-ce-kubernetes-operator/pull/10

---

총 27개 TODO 항목을 6단계 우선순위로 분류했습니다:

우선순위: P0 ✅ 완료 (PR #7)
카테고리: Configure Cluster
항목 수: 6개 (6/6 완료)
핵심 내용: On-demand Operations, ScaleDownBatchSize, MaxIgnorablePods, ValidationPolicy, Headless/Pod Service 커스터마이징, Rack 추가 필드
────────────────────────────────────────
우선순위: P1 ✅ 완료 (PR #10)
카테고리: Storage 고급
항목 수: 6개 (6/6 완료)
핵심 내용: Global Volume Policy, WipeMethod, HostPath, Local Storage 인식, Mount 옵션, PVC Metadata
────────────────────────────────────────
우선순위: P2
카테고리: Observability
항목 수: 4개
핵심 내용: Operator Self-Monitoring, Grafana Dashboard, Alert Rules, Logging 검증
────────────────────────────────────────
우선순위: P3
카테고리: Security
항목 수: 2개
핵심 내용: ACL Quota, Non-Root 가이드
────────────────────────────────────────
우선순위: P4
카테고리: Network 고급
항목 수: 2개
핵심 내용: CustomInterface (Multus CNI), Per-Pod Service
────────────────────────────────────────
우선순위: P5
카테고리: 운영 편의
항목 수: 3개
핵심 내용: HPA, Init Container 커스터마이징, Multi-Namespace Watch
────────────────────────────────────────
우선순위: P6
카테고리: EE 호환 대비
항목 수: 4개
핵심 내용: TLS, LDAP, XDR, Strong Consistency

각 항목에는 AKO 공식 문서 URL, CRD YAML 예시, 구체적인 체크리스트가 포함되어 있습니다.

## 현재 구현 완료된 기능 (Already Implemented)

| 카테고리 | 기능 | 상태 |
|---------|------|------|
| **Core** | Size (1-8 CE제한), Image, AerospikeConfig, Paused | Done |
| **Core** | RollingUpdateBatchSize | Done |
| **Storage** | PersistentVolume, EmptyDir, Secret, ConfigMap 소스 | Done |
| **Storage** | InitMethod (none, deleteFiles, dd, blkdiscard, headerCleanup) | Done |
| **Storage** | CascadeDelete, CleanupThreads | Done |
| **Storage** | Per-rack storage override | Done |
| **Network** | AccessType (pod, hostInternal, hostExternal, configuredIP) | Done |
| **Network** | AlternateAccessType, FabricType | Done |
| **Network** | Mesh seed 자동 주입 (heartbeat config) | Done |
| **Network** | LoadBalancer SeedsFinderServices | Done |
| **Network** | K8s NetworkPolicy / Cilium NetworkPolicy 자동 생성 | Done |
| **Network** | Bandwidth throttling (Ingress/Egress) | Done |
| **Rack** | Multi-rack (zone, region, nodeName 기반) | Done |
| **Rack** | Per-rack AerospikeConfig, Storage, PodSpec override | Done |
| **Pod** | Resources, SecurityContext, Sidecars, InitContainers | Done |
| **Pod** | Affinity, Tolerations, NodeSelector | Done |
| **Pod** | HostNetwork, MultiPodPerHost, DNSPolicy | Done |
| **Pod** | ImagePullSecrets, ServiceAccountName | Done |
| **Pod** | TerminationGracePeriodSeconds, Metadata (labels/annotations) | Done |
| **ACL** | Role CRUD (privileges, whitelist) | Done |
| **ACL** | User CRUD (K8s Secret 기반 password) | Done |
| **ACL** | Admin user 필수 검증 | Done |
| **Monitoring** | Prometheus exporter sidecar | Done |
| **Monitoring** | ServiceMonitor (Prometheus Operator) | Done |
| **Monitoring** | Metrics Service (ClusterIP) | Done |
| **PDB** | PodDisruptionBudget (MaxUnavailable, DisablePDB) | Done |
| **Dynamic Config** | EnableDynamicConfigUpdate, asinfo set-config | Done |
| **Restart** | Cold Restart (pod delete), Warm Restart (SIGUSR1) | Done |
| **Restart** | Config hash / PodSpec hash 기반 변경 감지 | Done |
| **Node** | K8sNodeBlockList | Done |
| **Status** | Phase (InProgress/Completed/Error), Conditions | Done |
| **Status** | Per-pod status (IP, Image, Rack, Hashes) | Done |
| **Status** | DynamicConfigStatus per pod | Done |
| **Webhook** | Defaulter (ports, cluster-name, heartbeat mode) | Done |
| **Webhook** | Validator (CE제한: size<=8, ns<=2, no xdr/tls/ee-image) | Done |
| **Metrics** | Operator 내부 Prometheus metrics (reconcile duration, phase 등) | Done |
| **Operations** | On-Demand Operations (WarmRestart, PodRestart) with status tracking | Done (P0, PR #7) |
| **Operations** | Operation 진행 중 변경 차단 (ValidateUpdate) | Done (P0, PR #7) |
| **Rack** | ScaleDownBatchSize (IntOrString: 절대값/퍼센트) | Done (P0, PR #7) |
| **Rack** | MaxIgnorablePods (pending/failed pod 무시) | Done (P0, PR #7) |
| **Rack** | RollingUpdateBatchSize at RackConfig level (IntOrString) | Done (P0, PR #7) |
| **Rack** | RackLabel (custom node affinity label) | Done (P0, PR #7) |
| **Rack** | Rack Revision (버전 식별자) | Done (P0, PR #7) |
| **Validation** | ValidationPolicy (skipWorkDirValidate) | Done (P0, PR #7) |
| **Service** | Headless Service custom annotations/labels | Done (P0, PR #7) |
| **Service** | Per-pod Service 생성 (spec.podService) | Done (P0, PR #7) |
| **Config** | EnableRackIDOverride (pod annotation 기반 rack ID) | Done (P0, PR #7) |
| **Storage** | Global Volume Policy (filesystemVolumePolicy / blockVolumePolicy) | Done (P1, PR #10) |
| **Storage** | WipeMethod (6가지: none, deleteFiles, dd, blkdiscard, headerCleanup, blkdiscardWithHeaderCleanup) | Done (P1, PR #10) |
| **Storage** | HostPath 볼륨 소스 (webhook warning 포함) | Done (P1, PR #10) |
| **Storage** | Local Storage 인식 (localStorageClasses, deleteLocalStorageOnRestart) | Done (P1, PR #10) |
| **Storage** | Volume Mount 고급 옵션 (readOnly, subPath, subPathExpr, mountPropagation) | Done (P1, PR #10) |
| **Storage** | PVC Metadata (custom labels/annotations) | Done (P1, PR #10) |

---

## TODO: 구현해야 할 기능 (Priority Order)

### P0: Configure Aerospike Cluster (가장 중요) ✅ 완료

> **구현 완료**: 2026-02-23 | PR: [#7](https://github.com/KimSoungRyoul/aerospike-ce-kubernetes-operator/pull/7) (`feature/p0-implementation`)

#### 1. On-Demand Operations (spec.operations) ✅
> AKO 문서: https://aerospike.com/docs/kubernetes/manage/node-maintenance/

현재는 config/image 변경 시에만 자동 restart가 발생하지만, AKO는 사용자가 직접 특정 pod에 대해 restart를 트리거할 수 있음.

```yaml
spec:
  operations:
    - kind: WarmRestart    # WarmRestart | PodRestart
      id: "restart-001"    # 고유 ID (1-20자)
      podList:             # 선택: 특정 pod만 대상 (생략 시 전체)
        - aerospike-0-0
        - aerospike-0-1
```

- [x] `OperationSpec` 타입 정의 (`kind`, `id`, `podList`)
  - `kind`: `WarmRestart` (SIGUSR1) / `PodRestart` (pod 삭제+재생성)
  - `id`: 1-20자 고유 문자열 (추적용)
  - `podList`: 선택적 pod 이름 목록 (없으면 전체 대상)
  - `maxItems: 1` (한 번에 하나의 operation만)
- [x] `OperationStatus` 상태 추적 (`status.operationStatus`)
- [x] Reconciler에서 operations 필드 처리 로직 (`reconciler_operations.go`)
- [x] Operation 완료 후 status 업데이트
- [x] Webhook validation: id 유효성, podList 존재 여부 검증
- [x] ValidateUpdate: InProgress 상태에서 operation 변경 차단

**구현 파일**:
- `api/v1alpha1/aerospikececluster_types.go` — `OperationSpec`, `OperationStatus`, `OperationKind` 타입
- `internal/controller/reconciler_operations.go` — `reconcileOperations`, `getOperationTargetPods`
- `api/v1alpha1/aerospikececluster_webhook.go` — `validateOperations`

#### 2. ScaleDownBatchSize / MaxIgnorablePods ✅
> AKO 문서: https://aerospike.com/docs/kubernetes/manage/configure/rack-awareness/

```yaml
spec:
  rackConfig:
    rollingUpdateBatchSize: 2    # RackConfig 레벨 (IntOrString)
    scaleDownBatchSize: 1        # NEW: scale-down 시 배치 크기
    maxIgnorablePods: 1          # NEW: 무시 가능한 pending/failed pod 수
```

- [x] `RackConfig`에 `ScaleDownBatchSize` 필드 추가 (`*intstr.IntOrString`)
- [x] Scale-down reconciliation에서 batch 처리 로직 (`getScaleDownBatchSize`, `resolveIntOrPercent`)
- [x] `RackConfig`에 `MaxIgnorablePods` 필드 추가 (`*intstr.IntOrString`)
- [x] Reconciler에서 pending/failed pod 무시 로직 (cluster stability 판단 시)
- [x] `RackConfig`에 `RollingUpdateBatchSize` 추가 (`*intstr.IntOrString`, spec-level 보다 우선)

**구현 파일**:
- `api/v1alpha1/types_rack.go` — `ScaleDownBatchSize`, `MaxIgnorablePods`, `RollingUpdateBatchSize` 필드
- `internal/controller/reconciler_statefulset.go` — `getScaleDownBatchSize`, `resolveIntOrPercent`
- `internal/controller/reconciler_restart.go` — `getRollingUpdateBatchSize`, `getMaxIgnorablePods`

#### 3. ValidationPolicy ✅
> AKO CRD Reference

```yaml
spec:
  validationPolicy:
    skipWorkDirValidate: false    # work directory PV 마운트 검증 스킵
```

- [x] `ValidationPolicySpec` 타입 정의
- [x] Webhook에서 work directory 볼륨 마운트 검증 추가 (`validateWorkDirectory`)
- [x] `skipWorkDirValidate` 플래그로 검증 우회 옵션

**구현 파일**:
- `api/v1alpha1/aerospikececluster_types.go` — `ValidationPolicySpec`
- `api/v1alpha1/aerospikececluster_webhook.go` — `validateWorkDirectory`

#### 4. Headless Service / Pod Service 커스터마이징 ✅
> AKO 문서: https://aerospike.com/docs/kubernetes/manage/configure/network-policy/

```yaml
spec:
  headlessService:
    metadata:
      annotations:
        service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
      labels:
        custom-label: value
  podService:
    metadata:
      annotations: {}
      labels: {}
```

- [x] `spec.headlessService` 필드 추가 (`AerospikeServiceSpec` with metadata)
- [x] `spec.podService` 필드 추가 (per-pod Service 생성)
- [x] Headless Service reconciler에서 custom annotations/labels 적용
- [x] Per-pod Service reconciler 구현 (각 pod마다 `<pod-name>-pod` 형태의 ClusterIP Service)

**구현 파일**:
- `api/v1alpha1/aerospikececluster_types.go` — `AerospikeServiceSpec`, `AerospikeObjectMeta`
- `internal/controller/reconciler_services.go` — headless service에 custom metadata 적용
- `internal/controller/reconciler_pod_service.go` — `reconcilePodServices` (NEW)

#### 5. Rack 추가 필드 ✅
> AKO 문서: https://aerospike.com/docs/kubernetes/manage/configure/rack-awareness/

```yaml
spec:
  rackConfig:
    racks:
      - id: 1
        rackLabel: "rack-a"     # NEW: custom label 기반 affinity
        revision: "v2"          # NEW: 버전 관리용 식별자
```

- [x] `Rack` 구조체에 `RackLabel` 필드 추가
- [x] `Rack` 구조체에 `Revision` 필드 추가
- [x] RackLabel 기반 node affinity 스케줄링 로직 (`acko.io/rack=<rackLabel>` NodeSelectorRequirement)
- [x] Webhook에서 RackLabel 유니크 검증

**구현 파일**:
- `api/v1alpha1/types_rack.go` — `RackLabel`, `Revision` 필드
- `internal/podutil/pod.go` — RackLabel node affinity 주입
- `api/v1alpha1/aerospikececluster_webhook.go` — RackLabel uniqueness 검증

#### 6. EnableRackIDOverride ✅
> AKO CRD Reference

- [x] `spec.enableRackIDOverride` 필드 추가
- [x] Pod annotation 기반 동적 rack ID 할당 로직

**구현 파일**:
- `api/v1alpha1/aerospikececluster_types.go` — `EnableRackIDOverride` 필드

---

### P1: Storage 고급 기능 ✅ 완료

> **구현 완료**: 2026-02-23 | PR: [#10](https://github.com/KimSoungRyoul/aerospike-ce-kubernetes-operator/pull/10) (`feature/p1-storage-advanced`)

#### 7. Global Volume Policy (filesystemVolumePolicy / blockVolumePolicy) ✅
> AKO 문서: https://aerospike.com/docs/kubernetes/manage/storage/overview/

```yaml
spec:
  storage:
    filesystemVolumePolicy:
      initMethod: deleteFiles
      wipeMethod: deleteFiles
      cascadeDelete: true
    blockVolumePolicy:
      initMethod: blkdiscard
      wipeMethod: blkdiscardWithHeaderCleanup
```

- [x] `AerospikeStorageSpec`에 `FilesystemVolumePolicy` 추가
- [x] `AerospikeStorageSpec`에 `BlockVolumePolicy` 추가
- [x] Policy 해석 로직: per-volume > global > default 우선순위 (`internal/storage/policy.go`)
- [x] `CascadeDelete` `*bool` 타입으로 per-volume explicit false가 global policy를 override 가능

**구현 파일**:
- `api/v1alpha1/types_storage.go` — `AerospikeVolumePolicy` 타입, `FilesystemVolumePolicy`, `BlockVolumePolicy` 필드
- `internal/storage/policy.go` — `ResolveInitMethod`, `ResolveWipeMethod`, `ResolveCascadeDelete`
- `internal/storage/policy_test.go` — 정책 해석 테스트 (274줄)

#### 8. WipeMethod (InitMethod과 별도) ✅
> AKO 문서: https://aerospike.com/docs/kubernetes/manage/storage/persistent-volume/

```yaml
storage:
  blockVolumePolicy:
    initMethod: blkdiscard
    wipeMethod: blkdiscardWithHeaderCleanup
  volumes:
    - name: data
      wipeMethod: headerCleanup  # per-volume override
```

- [x] `VolumeSpec`에 `WipeMethod` 필드 추가 (6가지: none, deleteFiles, dd, blkdiscard, headerCleanup, blkdiscardWithHeaderCleanup)
- [x] Init container에서 wipe vs init 분기 로직 (`WIPE_VOLUMES` env → INIT 전에 실행)
- [x] `blkdiscardWithHeaderCleanup` 메소드 추가

**구현 파일**:
- `api/v1alpha1/types_storage.go` — `VolumeWipeMethod` 타입
- `internal/initcontainer/scripts/aerospike-init.sh` — `process_volumes()` 헬퍼, WIPE→INIT 순서
- `internal/podutil/container.go` — `buildWipeVolumesEnv()`

#### 9. HostPath 볼륨 소스 ✅
> AKO CRD Reference

```yaml
volumes:
  - name: host-logs
    source:
      hostPath:
        path: /var/log/aerospike
        type: DirectoryOrCreate
```

- [x] `VolumeSource`에 `HostPath` 필드 추가 (`*corev1.HostPathVolumeSource`)
- [x] StatefulSet 빌더에서 hostPath 볼륨 처리
- [x] Webhook warning: 프로덕션 환경에서 hostPath 사용 시 경고

**구현 파일**:
- `api/v1alpha1/types_storage.go` — `VolumeSource.HostPath`
- `internal/storage/volume.go` — `volumeForSpec()` hostPath case
- `api/v1alpha1/aerospikececluster_webhook.go` — `validateStorage()` hostPath 경고

#### 10. Local Storage 인식 ✅
> AKO 문서: https://aerospike.com/docs/kubernetes/manage/storage/local-volume/

```yaml
spec:
  storage:
    localStorageClasses:
      - local-path
      - openebs-hostpath
    deleteLocalStorageOnRestart: true
```

- [x] `AerospikeStorageSpec`에 `LocalStorageClasses` 필드 추가
- [x] `AerospikeStorageSpec`에 `DeleteLocalStorageOnRestart` 필드 추가
- [x] Pod cold restart 시 local PVC 삭제 로직 (NotFound 에러 방어 포함)
- [x] Webhook: `deleteLocalStorageOnRestart: true` + `localStorageClasses` 비어있으면 에러

**구현 파일**:
- `internal/storage/local.go` — `DeleteLocalPVCsForPod`, `GetLocalPVCsForPod`, `ParsePodName`
- `internal/controller/reconciler_restart.go` — `coldRestartPod`에서 local PVC 삭제 호출
- `internal/storage/local_test.go` — 테스트 (77줄)

#### 11. Volume Mount 고급 옵션 ✅
> AKO CRD Reference

```yaml
volumes:
  - name: data
    aerospike:
      path: /opt/aerospike/data
      readOnly: false
      subPath: "subdir"
      subPathExpr: "$(POD_NAME)"  # mutually exclusive with subPath
      mountPropagation: HostToContainer
```

- [x] `AerospikeVolumeAttachment`에 `ReadOnly`, `SubPath`, `SubPathExpr`, `MountPropagation` 추가
- [x] `VolumeAttachment`에 동일 필드 추가 (sidecar/init containers)
- [x] StatefulSet 빌더에서 mount option 적용 (`buildVolumeMount()` 헬퍼)
- [x] Webhook: `SubPath`와 `SubPathExpr` 상호 배타 검증

**구현 파일**:
- `api/v1alpha1/types_storage.go` — `AerospikeVolumeAttachment`, `VolumeAttachment` 확장
- `internal/storage/volume.go` — `buildVolumeMount()`

#### 12. PVC Metadata (Annotations/Labels) ✅
> AKO CRD Reference

```yaml
volumes:
  - name: data
    source:
      persistentVolume:
        size: 50Gi
        metadata:
          labels:
            backup-policy: "daily"
          annotations:
            volume.kubernetes.io/storage-provisioner: "ebs.csi.aws.com"
```

- [x] `PersistentVolumeSpec`에 `Metadata` (`AerospikeObjectMeta`) 필드 추가
- [x] PVC template에 custom annotations/labels 적용 (`maps.Clone`)

**구현 파일**:
- `api/v1alpha1/types_storage.go` — `PersistentVolumeSpec.Metadata`
- `internal/storage/volume.go` — `BuildVolumeClaimTemplates()` metadata 적용

---

### P2: Observability 강화

#### 13. Operator Self-Monitoring 강화
> AKO 문서: https://aerospike.com/docs/kubernetes/observe/operator-monitoring/

현재 operator 내부 metrics는 있지만, 더 풍부한 모니터링 지원 필요.

- [ ] `aerospike_acko_aerospikececluster_phase` 전용 Prometheus metric 추가
- [ ] Operator /metrics 엔드포인트 TLS 지원 (cert-manager 연동)
- [ ] Operator 전용 ServiceMonitor YAML 생성 (config/monitoring/)

#### 14. Grafana Dashboard 템플릿
> AKO 문서: https://aerospike.com/docs/kubernetes/observe/clusters/

- [ ] Operator dashboard JSON 템플릿 (controller-runtime, workqueue metrics)
- [ ] Aerospike cluster dashboard JSON 템플릿 (APE exporter metrics)
- [ ] config/monitoring/grafana/ 디렉토리에 배치

#### 15. Prometheus Alert Rules 기본 제공
> AKO 문서: https://aerospike.com/docs/kubernetes/observe/clusters/

```yaml
# config/monitoring/prometheus/alert-rules.yaml
groups:
  - name: aerospike-alerts
    rules:
      - alert: AerospikeHighMemoryUtilization
        expr: aerospike_node_stats_system_free_mem_pct < 5
        for: 5m
        severity: critical
      - alert: AerospikeHighDiskUtilization
        expr: aerospike_namespace_device_available_pct < 5
        for: 5m
        severity: warning
```

- [ ] PrometheusRule CRD YAML 템플릿 작성
- [ ] Memory, Disk, Namespace 사용률 alert rule
- [ ] config/monitoring/prometheus/ 디렉토리에 배치

#### 16. Logging 설정 가이드/검증
> AKO 문서: https://aerospike.com/docs/kubernetes/observe/logs/

```yaml
aerospikeConfig:
  logging:
    - name: console
      any: info
      tls: debug
    - name: /var/log/aerospike/aerospike.log
      any: info
```

- [ ] Webhook에서 logging 설정 validation (유효한 sink type, log level 검증)
- [ ] Log file 경로에 대한 volume mount 검증 (PV가 마운트되어 있는지)
- [ ] Sample CR에 logging 설정 예제 추가

---

### P3: Security 강화

#### 17. ACL 고급 기능 (ReadQuota/WriteQuota)
> AKO 문서: https://aerospike.com/docs/kubernetes/manage/security/access-control/

```yaml
aerospikeAccessControl:
  roles:
    - name: rate-limited-reader
      privileges:
        - read.my-namespace
      readQuota: 10000     # NEW: 초당 최대 read 수
      writeQuota: 5000     # NEW: 초당 최대 write 수
```

**참고**: readQuota/writeQuota는 Enterprise Edition 전용 기능이므로 CE에서는 CRD 필드만 정의하고 webhook에서 CE 사용 시 경고/차단.

- [ ] `AerospikeRoleSpec`에 `ReadQuota`, `WriteQuota` 필드 추가
- [ ] Webhook: CE image일 때 quota 설정 시 경고 메시지
- [ ] (선택) EE image 지원 시 ACL reconciler에서 quota 설정 로직

#### 18. Non-Root Cluster 실행 가이드
> AKO 문서: https://aerospike.com/docs/kubernetes/manage/security/nonroot-cluster/

- [ ] Sample CR에 non-root 실행 예제 추가
  ```yaml
  podSpec:
    securityContext:
      runAsNonRoot: true
      runAsUser: 1000
      fsGroup: 1000
  ```
- [ ] Webhook에서 runAsNonRoot 설정 시 필요한 volume permission 검증

---

### P4: Network 고급 기능

#### 19. CustomInterface 네트워크 타입 (Multus CNI)
> AKO 문서: https://aerospike.com/docs/kubernetes/manage/configure/network-policy/

```yaml
spec:
  aerospikeNetworkPolicy:
    accessType: customInterface
    customAccessNetworkNames:
      - default/my-network-attachment
    fabricType: customInterface
    customFabricNetworkNames:
      - default/my-fabric-network
```

- [ ] NetworkType에 `customInterface` 추가
- [ ] `CustomFabricNetworkNames`, `CustomTLSAccessNetworkNames` 등 추가 필드
- [ ] Pod annotation에 `k8s.v1.cni.cncf.io/networks` 주입 로직
- [ ] Config generation에서 custom interface IP 해석 로직

#### 20. Per-Pod Service 생성 ✅ (P0에서 구현 완료)
> AKO CRD Reference

AKO는 각 pod마다 개별 Service를 생성하여 외부에서 특정 pod에 직접 접근 가능.

- [x] `spec.podService` 설정 기반 per-pod Service reconciler (`reconciler_pod_service.go`)
- [x] 각 pod에 대해 `<pod-name>-pod` 형태의 ClusterIP Service 생성
- [ ] NodePort / LoadBalancer 타입 지원 (현재 ClusterIP만)
- [x] Custom annotations/labels 적용

---

### P5: 운영 편의 기능

#### 21. HPA (Horizontal Pod Autoscaler) 지원
> AKO 문서: https://aerospike.com/docs/kubernetes/manage/configure/hpa/

- [ ] Status에 `spec.selector` (label selector string) 정확한 HPA 호환 형태 노출
- [ ] Scale subresource 등록 확인 (kubebuilder marker)
- [ ] KEDA + Prometheus 기반 autoscaling 예제 문서
- [ ] `minReplicaCount >= replication-factor` 검증 가이드

#### 22. AerospikeInitContainer 커스터마이징
> AKO CRD Reference

AKO는 init container의 이미지를 별도로 커스터마이징 가능.

```yaml
podSpec:
  aerospikeInitContainerSpec:
    imageRegistry: "my-registry.com"
    imageRegistryNamespace: "aerospike"
    imageNameAndTag: "aerospike-kubernetes-init:2.2.1"
    securityContext: ...
    resources: ...
```

- [ ] `AerospikeInitContainerSpec` 타입 정의
- [ ] Init container 이미지 커스터마이징 지원
- [ ] Init container resources/securityContext 설정

#### 23. Multi-Namespace Operator Watch
> AKO 문서: https://aerospike.com/docs/kubernetes/manage/configure/multiple-aerospike-clusters/

- [ ] `WATCH_NAMESPACE` 환경변수 기반 multi-namespace watch 지원
- [ ] Per-namespace RBAC 설정 가이드/스크립트
- [ ] 여러 namespace의 AerospikeCECluster 동시 관리

---

### P6: Enterprise Edition 호환 (향후 EE 지원 시)

> 아래 기능들은 Aerospike Enterprise Edition 전용. CE operator에서는 CRD 필드만 정의해두고 webhook에서 차단하거나, EE 이미지 감지 시에만 활성화하는 방식으로 대비.

#### 24. TLS 지원
- [ ] `spec.operatorClientCertSpec` 타입 정의
- [ ] Network config에 TLS access type 필드 추가 (tlsAccessType, tlsAlternateAccessType, tlsFabricType)
- [ ] TLS cert volume 자동 마운트
- [ ] Webhook: CE image 사용 시 TLS 설정 차단

#### 25. LDAP 외부 인증
- [ ] `aerospikeConfig.security.ldap` 설정 지원
- [ ] LDAP 관련 Secret 마운트 (query-user-password-file)
- [ ] Webhook: CE image 사용 시 LDAP 설정 차단

#### 26. XDR (Cross-Datacenter Replication)
- [ ] `aerospikeConfig.xdr` 설정 지원
- [ ] XDR Proxy 연동 가이드
- [ ] Webhook: CE image 사용 시 XDR 설정 차단 (이미 구현됨)

#### 27. Strong Consistency
- [ ] `spec.rosterNodeBlockList` 필드 추가
- [ ] `Rack.forceBlockFromRoster` 필드 추가
- [ ] Roster 관리 reconciler
- [ ] Webhook: CE image 사용 시 strong-consistency 설정 차단 (이미 구현됨)

---

## 구현 우선순위 요약

| 우선순위 | 카테고리 | 항목 수 | 상태 | 핵심 이유 |
|---------|---------|--------|------|----------|
| **P0** | Configure Cluster | 6개 | ✅ 완료 (PR #7) | AKO 핵심 기능, 운영 필수 |
| **P1** | Storage 고급 | 6개 | ✅ 완료 (PR #10) | 프로덕션 스토리지 관리 |
| **P2** | Observability | 4개 | 미착수 | 모니터링/알림 강화 |
| **P3** | Security | 2개 | 미착수 | 보안 강화 |
| **P4** | Network 고급 | 2개 | 1/2 완료 | 고급 네트워킹 |
| **P5** | 운영 편의 | 3개 | 미착수 | 운영 편의성 |
| **P6** | EE 호환 | 4개 | 미착수 | 향후 EE 지원 대비 |

---

## 기능별 AKO 공식 문서 참조

| 기능 | AKO 문서 URL |
|------|-------------|
| Cluster Configuration | https://aerospike.com/docs/kubernetes/manage/configure/overview/ |
| Aerospike Config | https://aerospike.com/docs/kubernetes/manage/configure/aerospike-cluster/ |
| Rack Awareness | https://aerospike.com/docs/kubernetes/manage/configure/rack-awareness/ |
| Network Policy | https://aerospike.com/docs/kubernetes/manage/configure/network-policy/ |
| Pod Spec | https://aerospike.com/docs/kubernetes/manage/configure/pod-spec/ |
| Multiple Clusters | https://aerospike.com/docs/kubernetes/manage/configure/multiple-aerospike-clusters/ |
| Dynamic Config | https://aerospike.com/docs/kubernetes/manage/configure/dynamic-config/ |
| HPA | https://aerospike.com/docs/kubernetes/manage/configure/hpa/ |
| Storage Overview | https://aerospike.com/docs/kubernetes/manage/storage/overview/ |
| Persistent Volume | https://aerospike.com/docs/kubernetes/manage/storage/persistent-volume/ |
| Local Volume | https://aerospike.com/docs/kubernetes/manage/storage/local-volume/ |
| K8s Secrets | https://aerospike.com/docs/kubernetes/manage/security/kubernetes-secrets/ |
| Access Control | https://aerospike.com/docs/kubernetes/manage/security/access-control/ |
| Non-Root | https://aerospike.com/docs/kubernetes/manage/security/nonroot-cluster/ |
| Node Maintenance | https://aerospike.com/docs/kubernetes/manage/node-maintenance/ |
| Monitor Clusters | https://aerospike.com/docs/kubernetes/observe/clusters/ |
| Monitor Operator | https://aerospike.com/docs/kubernetes/observe/operator-monitoring/ |
| Logging | https://aerospike.com/docs/kubernetes/observe/logs/ |
| Config Reference | https://aerospike.com/docs/kubernetes/reference/config-reference/ |

---

## 참고: AKO vs CE Operator 아키텍처 비교

| 항목 | AKO (Enterprise) | CE Operator (이 프로젝트) |
|------|-------------------|--------------------------|
| CRD Group | `asdb.aerospike.com/v1` | `acko.io/v1alpha1` |
| Kind | `AerospikeCluster` | `AerospikeCECluster` |
| Size 제한 | 무제한 | 8노드 |
| Namespace 제한 | 무제한 | 2개 |
| TLS | 지원 | 미지원 (CE 제한) |
| XDR | 지원 | 미지원 (CE 제한) |
| LDAP | 지원 | 미지원 (CE 제한) |
| Strong Consistency | 지원 | 미지원 (CE 제한) |
| Heartbeat Mode | mesh + multicast | mesh만 (CE 제한) |
| Init Container | 커스텀 이미지 | 내장 |
| Volume Sources | PV, EmptyDir, Secret, ConfigMap, HostPath | PV, EmptyDir, Secret, ConfigMap, HostPath ✅ |
| Volume Policy | Global + Per-volume + Wipe 분리 | Global + Per-volume + Wipe 분리 ✅ |
| Operations | WarmRestart, PodRestart (on-demand) | WarmRestart, PodRestart (on-demand) ✅ |
| Rack 배치 | rollingUpdateBatchSize + scaleDownBatchSize + maxIgnorablePods | rollingUpdateBatchSize + scaleDownBatchSize + maxIgnorablePods ✅ |
| ValidationPolicy | skipWorkDirValidate | skipWorkDirValidate ✅ |
| Service 커스터마이징 | headlessService + podService metadata | headlessService + podService metadata ✅ |
| Rack 추가 필드 | rackLabel, revision | rackLabel, revision ✅ |
