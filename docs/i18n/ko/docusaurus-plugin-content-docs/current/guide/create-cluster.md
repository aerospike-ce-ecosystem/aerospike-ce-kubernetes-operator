---
sidebar_position: 2
title: 클러스터 생성
---

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

# Aerospike 클러스터 생성

이 가이드는 `AerospikeCECluster` CRD를 사용하여 Aerospike CE 클러스터를 배포하는 방법을 설명합니다.

## 샘플 설정

<Tabs>
<TabItem value="minimal" label="최소 (1노드)" default>

가장 간단한 클러스터: 단일 노드 인메모리 배포입니다.

```yaml
apiVersion: acko.io/v1alpha1
kind: AerospikeCECluster
metadata:
  name: aerospike-ce-basic
  namespace: aerospike
spec:
  size: 1
  image: aerospike:ce-8.1.1.1
  aerospikeConfig:
    namespaces:
      - name: test
        replication-factor: 1
        storage-engine:
          type: memory
          data-size: 1073741824   # 1 GiB
```

**사용 사례:** 개발, 테스트, 빠른 프로토타이핑.

```bash
kubectl create namespace aerospike
kubectl apply -f config/samples/acko_v1alpha1_aerospikececluster.yaml
```

</TabItem>
<TabItem value="3node" label="3노드 PV 스토리지">

리소스 제한과 영구 스토리지를 갖춘 프로덕션급 구성입니다.

```yaml
apiVersion: acko.io/v1alpha1
kind: AerospikeCECluster
metadata:
  name: aerospike-ce-3node
  namespace: aerospike
spec:
  size: 3
  image: aerospike:ce-8.1.1.1

  # 파드 리소스 제한
  podSpec:
    aerospikeContainer:
      resources:
        requests:
          memory: "2Gi"
          cpu: "1"
        limits:
          memory: "4Gi"
          cpu: "2"

  # 영구 스토리지
  storage:
    volumes:
      - name: data-vol
        source:
          persistentVolume:
            storageClass: standard
            size: 10Gi
            volumeMode: Filesystem
        aerospike:
          path: /opt/aerospike/data
        cascadeDelete: true       # CR 삭제 시 PVC도 삭제

      - name: workdir
        source:
          emptyDir: {}
        aerospike:
          path: /opt/aerospike/work

  aerospikeConfig:
    service:
      cluster-name: aerospike-ce-3node
      proto-fd-max: 15000

    network:
      service:
        address: any
        port: 3000
      heartbeat:
        mode: mesh
        port: 3002
      fabric:
        address: any
        port: 3001

    namespaces:
      - name: testns
        replication-factor: 2
        storage-engine:
          type: device
          file: /opt/aerospike/data/testns.dat
          filesize: 4294967296    # 4 GiB
```

**사용 사례:** 데이터 영속성과 복제가 필요한 프로덕션 워크로드.

```bash
kubectl apply -f config/samples/aerospike-ce-cluster-3node.yaml
```

</TabItem>
<TabItem value="multirack" label="6노드 멀티랙">

랙 인식 배포를 사용하여 장애 도메인에 파드를 분산합니다.

```yaml
apiVersion: acko.io/v1alpha1
kind: AerospikeCECluster
metadata:
  name: aerospike-ce-multirack
  namespace: aerospike
spec:
  size: 6
  image: aerospike:ce-8.1.1.1

  # 3개 랙 — 파드가 균등 분배 (랙당 2개)
  rackConfig:
    namespaces:
      - testns              # 랙 인식 네임스페이스
    racks:
      - id: 1
      - id: 2
      - id: 3

  podSpec:
    aerospikeContainer:
      resources:
        requests:
          memory: "512Mi"
          cpu: "250m"
        limits:
          memory: "1Gi"
          cpu: "500m"

  storage:
    volumes:
      - name: data-vol
        source:
          persistentVolume:
            storageClass: standard
            size: 5Gi
        aerospike:
          path: /opt/aerospike/data
        cascadeDelete: false      # 삭제 시 PVC 유지

  aerospikeConfig:
    namespaces:
      - name: testns
        replication-factor: 2
        storage-engine:
          type: device
          file: /opt/aerospike/data/testns.dat
          filesize: 4294967296
```

**사용 사례:** 존 간 고가용성. 각 랙 ID는 별도의 StatefulSet(`<이름>-<랙ID>`)과 ConfigMap을 생성합니다.

```bash
kubectl apply -f config/samples/aerospike-ce-cluster-multirack.yaml
```

</TabItem>
<TabItem value="monitoring" label="모니터링 포함">

Prometheus 익스포터 사이드카와 ServiceMonitor가 포함된 3노드 클러스터입니다.

```yaml
apiVersion: acko.io/v1alpha1
kind: AerospikeCECluster
metadata:
  name: aerospike-ce-monitored
  namespace: aerospike
spec:
  size: 3
  image: aerospike:ce-8.1.1.1

  monitoring:
    enabled: true
    exporterImage: aerospike/aerospike-prometheus-exporter:latest
    port: 9145
    resources:
      requests:
        cpu: "100m"
        memory: "64Mi"
    serviceMonitor:
      enabled: true
      interval: "30s"
      labels:
        release: prometheus    # Prometheus Operator 셀렉터에 맞춤

  storage:
    volumes:
      - name: data-vol
        source:
          persistentVolume:
            storageClass: standard
            size: 10Gi
        aerospike:
          path: /opt/aerospike/data
        cascadeDelete: true

  aerospikeConfig:
    namespaces:
      - name: testns
        replication-factor: 2
        storage-engine:
          type: device
          file: /opt/aerospike/data/testns.dat
          filesize: 4294967296
```

**사용 사례:** Prometheus/Grafana 관측성이 필요한 프로덕션. 오퍼레이터가 각 파드에 익스포터 사이드카를 주입하고 ServiceMonitor를 자동 생성합니다.

</TabItem>
</Tabs>

## 웹훅 기본값

오퍼레이터의 mutating 웹훅은 지정하지 않은 경우 다음 기본값을 자동으로 설정합니다:

| 필드 | 기본값 | 설명 |
|---|---|---|
| `aerospikeConfig.service.cluster-name` | `metadata.name` | Aerospike 클러스터 이름 |
| `aerospikeConfig.service.proto-fd-max` | `15000` | 최대 클라이언트 연결 수 |
| `aerospikeConfig.network.service.port` | `3000` | 클라이언트 서비스 포트 |
| `aerospikeConfig.network.heartbeat.port` | `3002` | 하트비트 포트 |
| `aerospikeConfig.network.heartbeat.mode` | `mesh` | 하트비트 모드 |
| `aerospikeConfig.network.fabric.port` | `3001` | 패브릭(노드 간 통신) 포트 |
| `monitoring.exporterImage` | `aerospike/aerospike-prometheus-exporter:latest` | 익스포터 이미지 (모니터링 활성화 시) |
| `monitoring.port` | `9145` | 익스포터 메트릭 포트 (모니터링 활성화 시) |
| `monitoring.serviceMonitor.interval` | `30s` | 스크래핑 주기 (ServiceMonitor 활성화 시) |
| `podSpec.multiPodPerHost` | `false` | 노드당 하나의 파드 (hostNetwork 활성화 시) |
| `podSpec.dnsPolicy` | `ClusterFirstWithHostNet` | DNS 정책 (hostNetwork 활성화 시) |

## CE 검증 규칙

validating 웹훅은 Community Edition 제약 조건을 적용합니다:

| 규칙 | 제약 | 오류 메시지 |
|---|---|---|
| 클러스터 크기 | `spec.size` 최대 8 | `spec.size N exceeds CE maximum of 8` |
| 네임스페이스 개수 | 최대 2개 | `namespaces count N exceeds CE maximum of 2` |
| XDR | 사용 불가 | `must not contain 'xdr' section` |
| TLS | 사용 불가 | `must not contain 'tls' section` |
| Security | 사용 불가 (CE 8.x) | `must not contain 'security' section` |
| 하트비트 모드 | 반드시 `mesh` | `must be 'mesh' for CE` |
| 이미지 | CE 이미지만 가능 | `Enterprise Edition image not allowed` |
| 복제 팩터 | `spec.size` 이하 | `replication-factor N exceeds cluster size` |
| 복제 팩터 범위 | 1~4 | `must be between 1 and 4` |
| 랙 ID | 고유해야 함 | `duplicate rack ID` |

### Enterprise 전용 네임스페이스 키

다음 네임스페이스 설정 키는 CE에서 차단됩니다:

| 키 | 이유 |
|---|---|
| `compression`, `compression-level` | 데이터 압축은 Enterprise 전용 |
| `durable-delete` | 영구 삭제는 Enterprise 전용 |
| `fast-restart` | 빠른 재시작은 Enterprise 전용 |
| `index-type` | Flash/pmem 인덱스는 Enterprise 전용 |
| `sindex-type` | Flash/pmem 보조 인덱스는 Enterprise 전용 |
| `rack-id` | 오퍼레이터 `rackConfig`를 대신 사용 |
| `strong-consistency` | Strong consistency는 Enterprise 전용 |
| `tomb-raider-eligible-age`, `tomb-raider-period` | Tomb-raider는 Enterprise 전용 |

### 경고

웹훅은 다음에 대해 경고(비차단)도 발생시킵니다:

- 태그 없는 이미지 또는 `latest` 태그 사용
- `hostNetwork=true`와 `multiPodPerHost=true` (포트 충돌 위험)
- `hostNetwork=true`와 `ClusterFirstWithHostNet`이 아닌 DNS 정책
- `data-in-memory=true` (메모리 사용량 2배)
- `rollingUpdateBatchSize`가 클러스터 크기보다 큰 경우
