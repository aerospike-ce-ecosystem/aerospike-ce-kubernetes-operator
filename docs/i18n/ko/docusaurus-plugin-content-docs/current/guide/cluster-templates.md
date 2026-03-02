---
sidebar_position: 5
title: 클러스터 템플릿
---

# 클러스터 템플릿

`AerospikeClusterTemplate`을 사용하면 Aerospike 클러스터를 위한 재사용 가능한 설정 프로필을 정의할 수 있습니다. 모든 클러스터에 동일한 스케줄링, 스토리지, Aerospike 설정을 반복하는 대신, 템플릿에 한 번 정의하고 여러 클러스터에서 참조할 수 있습니다.

---

## 템플릿 사용 시기

- **다중 환경** — 서로 다른 리소스 크기와 anti-affinity 수준을 가진 dev/stage/prod 템플릿을 정의
- **조직 기본값** — 팀 간에 스토리지 클래스, 톨러레이션, heartbeat 설정을 통일
- **설정 표준화** — 공유 설정을 한 곳에 유지하여 설정 드리프트를 방지

---

## 템플릿 생성

템플릿은 이제 기존의 스케줄링, 스토리지, Aerospike 설정 필드 외에도 컨테이너 **image**, 클러스터 **size**, **monitoring** 사이드카, **network policy**를 기본값으로 제공할 수 있습니다.

```yaml
apiVersion: acko.io/v1alpha1
kind: AerospikeClusterTemplate
metadata:
  name: prod
  namespace: default
spec:
  # 모든 prod 클러스터에 걸쳐 Aerospike 이미지와 기본 클러스터 크기를 표준화
  image: aerospike:ce-8.1.1.1
  size: 6

  # 기본적으로 Prometheus 모니터링 사이드카를 활성화
  monitoring:
    enabled: true
    port: 9145
    resources:
      requests:
        cpu: 50m
        memory: 64Mi
      limits:
        cpu: 200m
        memory: 128Mi
    serviceMonitor:
      enabled: true
      interval: 30s

  # 기본 네트워크 접근 정책
  aerospikeNetworkPolicy:
    accessType: pod
    alternateAccessType: pod
    fabricType: pod

  scheduling:
    podAntiAffinityLevel: required   # 노드당 하나의 Aerospike pod
    tolerations:
      - key: "aerospike"
        operator: "Exists"
        effect: "NoSchedule"

  storage:
    storageClassName: local-path
    localPVRequired: true
    resources:
      requests:
        storage: 100Gi

  resources:
    requests:
      cpu: "2"
      memory: "4Gi"
    limits:
      cpu: "2"
      memory: "4Gi"

  aerospikeConfig:
    service:
      proto-fd-max: 15000
    namespaceDefaults:
      replication-factor: 2
      memory-size: 2147483648   # 2 GiB
```

```bash
kubectl apply -f prod-template.yaml
```

---

## 클러스터에서 템플릿 참조

템플릿이 `image`와 `size`를 제공하는 경우, 클러스터에서 해당 필드를 완전히 생략할 수 있습니다:

```yaml
apiVersion: acko.io/v1alpha1
kind: AerospikeCluster
metadata:
  name: prod-cluster
spec:
  # image와 size는 "prod" 템플릿에서 제공됨 (image: aerospike:ce-8.1.1.1, size: 6)
  templateRef:
    name: prod

  aerospikeConfig:
    namespaces:
      - name: data
        storage-engine:
          type: memory
```

클러스터에서 `spec.image` 또는 `spec.size`를 명시적으로 설정하여 템플릿을 오버라이드할 수도 있습니다:

```yaml
spec:
  image: aerospike:ce-8.1.1.1   # 오버라이드: 특정 이미지를 고정
  size: 3                         # 오버라이드: 템플릿의 6 대신 3개 노드 사용
  templateRef:
    name: prod
```

오퍼레이터는 생성 시 템플릿을 해석하고 spec을 `status.templateSnapshot`에 저장합니다. 그 시점부터 클러스터는 독립적으로 운영됩니다 — 템플릿 변경 사항은 이 클러스터에 자동으로 영향을 미치지 않습니다.

---

## 템플릿 필드 오버라이드

새 템플릿을 만들지 않고 특정 필드를 변경하려면 `spec.overrides`를 사용합니다:

```yaml
spec:
  templateRef:
    name: prod
  overrides:
    resources:
      requests:
        cpu: "4"
        memory: "8Gi"
      limits:
        cpu: "4"
        memory: "8Gi"
    scheduling:
      podAntiAffinityLevel: preferred   # 이 클러스터에 대해 anti-affinity를 완화
```

**병합 우선순위:** `overrides` > `template.spec` > 오퍼레이터 기본값

맵(예: `aerospikeConfig.service`)의 경우 병합은 재귀적으로 수행됩니다 — 지정된 키만 오버라이드됩니다. 배열(예: `tolerations`)과 스칼라 필드의 경우, 오버라이드가 템플릿 값을 완전히 대체합니다.

---

## 템플릿 업데이트 후 재동기화

템플릿을 업데이트한 후, 기존 클러스터는 `status.templateSnapshot.synced: false`를 표시하고 `TemplateDrifted` 경고 이벤트를 발행합니다. 클러스터는 스냅샷을 기반으로 계속 운영됩니다.

업데이트된 템플릿을 클러스터에 적용하려면:

```bash
kubectl annotate aerospikecluster prod-cluster acko.io/resync-template=true
```

오퍼레이터가 수행하는 작업:
1. 템플릿을 다시 가져옴
2. `status.templateSnapshot`을 업데이트
3. `TemplateApplied` 이벤트를 발행
4. 어노테이션을 제거

---

## 템플릿 스냅샷 상태 확인

```bash
kubectl get aerospikecluster prod-cluster -o jsonpath='{.status.templateSnapshot}'
```

출력 예시:
```json
{
  "name": "prod",
  "resourceVersion": "12345",
  "snapshotTimestamp": "2026-03-01T10:00:00Z",
  "synced": true
}
```

---

## 샘플 템플릿

`config/samples/` 디렉토리에 바로 사용할 수 있는 템플릿이 포함되어 있습니다:

| 파일 | 프로필 |
|------|---------|
| `acko_v1alpha1_template_dev.yaml` | 최소 리소스, anti-affinity 없음 |
| `acko_v1alpha1_template_stage.yaml` | 적정 리소스, preferred anti-affinity |
| `acko_v1alpha1_template_prod.yaml` | 전체 리소스, required anti-affinity, 로컬 PV |
| `aerospike-ce-cluster-with-template.yaml` | stage 템플릿을 사용하는 예제 클러스터 |

```bash
kubectl apply -f config/samples/acko_v1alpha1_template_prod.yaml
kubectl apply -f config/samples/aerospike-ce-cluster-with-template.yaml
```
