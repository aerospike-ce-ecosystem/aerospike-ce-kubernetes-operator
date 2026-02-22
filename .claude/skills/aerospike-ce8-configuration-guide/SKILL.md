---
name: aerospike-ce8-configuration-guide
description: Aerospike CE 8.1 operational guide — full config examples, K8s deployment checklist, CRD mapping, eviction flow, best practices
disable-model-invocation: false
---

# Aerospike CE 8.1 Configuration & Operations Guide

Aerospike CE 8.1 전체 설정 예제, K8s 배포 체크리스트, CRD 매핑, 운영 Best Practices를 제공합니다.
파라미터 레퍼런스는 `aerospike-ce-8-guide` skill을 참조하세요.

## Arguments

선택적 인자: 섹션 이름 (예: `examples`, `k8s-checklist`, `crd-mapping`, `eviction`, `best-practices`, `logging-examples`, `security-example`)
인자 없으면 전체 운영 가이드를 출력합니다.

## Output

아래 전체 내용을 마크다운으로 출력하세요.

---

# Aerospike CE 8.1 — Configuration & Operations Guide

> 파라미터 상세, CE 제약사항, 7.x→8.1 변경사항은 `aerospike-ce-8-guide` skill 참조.

---

## 1. 전체 설정 예제

### 1.1 최소 단일 노드 인메모리

```
service {
    cluster-name aerospike-ce-basic
    proto-fd-max 15000
}

logging {
    console {
        context any info
    }
}

network {
    service {
        address any
        port 3000
    }
    heartbeat {
        mode mesh
        port 3002
        interval 150
        timeout 10
    }
    fabric {
        address any
        port 3001
    }
    admin {
        address any
        port 3008
    }
}

namespace test {
    replication-factor 1
    max-record-size 1M
    storage-engine memory {
        data-size 1G
    }
}
```

### 1.2 3노드 클러스터 (파일 영속)

```
service {
    cluster-name aerospike-ce-3node
    proto-fd-max 15000
    service-threads 20
}

logging {
    console {
        context any info
    }
}

network {
    service {
        address any
        port 3000
        access-address 10.0.0.101
    }
    heartbeat {
        mode mesh
        port 3002
        mesh-seed-address-port 10.0.0.101 3002
        mesh-seed-address-port 10.0.0.102 3002
        mesh-seed-address-port 10.0.0.103 3002
        interval 150
        timeout 10
    }
    fabric {
        port 3001
    }
    admin {
        address any
        port 3008
    }
}

namespace testns {
    replication-factor 2
    stop-writes-sys-memory-pct 90
    max-record-size 1M

    storage-engine device {
        file /opt/aerospike/data/testns.dat
        filesize 4G
        data-in-memory true
        flush-size 128K
        defrag-lwm-pct 50
        evict-used-pct 70
    }
}
```

### 1.3 보안(ACL) 활성화

```
service {
    cluster-name aerospike-ce-secure
    proto-fd-max 15000
}

logging {
    console {
        context any info
    }
}

security {
}

network {
    service {
        address any
        port 3000
    }
    heartbeat {
        mode mesh
        port 3002
        mesh-seed-address-port 10.0.0.101 3002
        mesh-seed-address-port 10.0.0.102 3002
        mesh-seed-address-port 10.0.0.103 3002
        interval 150
        timeout 10
    }
    fabric {
        port 3001
    }
    admin {
        address any
        port 3008
    }
}

namespace testns {
    replication-factor 2
    max-record-size 1M
    storage-engine device {
        file /opt/aerospike/data/testns.dat
        filesize 4G
        flush-size 1M
    }
}
```

### 1.4 고성능 Raw SSD

```
service {
    cluster-name aerospike-ce-perf
    proto-fd-max 15000
    service-threads 20
}

logging {
    console {
        context any info
    }
}

network {
    service {
        address any
        port 3000
    }
    heartbeat {
        mode mesh
        port 3002
        mesh-seed-address-port 10.0.0.101 3002
        mesh-seed-address-port 10.0.0.102 3002
        mesh-seed-address-port 10.0.0.103 3002
        interval 150
        timeout 10
    }
    fabric {
        port 3001
    }
    admin {
        address any
        port 3008
    }
}

namespace bar {
    max-record-size 1M
    stop-writes-sys-memory-pct 90

    storage-engine device {
        device /dev/nvme0n1p1
        device /dev/nvme0n1p2
        device /dev/nvme0n1p3
        device /dev/nvme0n1p4
        flush-size 128K
        defrag-lwm-pct 50
        evict-used-pct 70
    }
}
```

---

## 2. 고급 로깅 예제

```
logging {
    # 기본: 모든 모듈 info
    file /var/log/aerospike/aerospike.log {
        context any info
    }

    # 보안 디버깅: 모든 모듈 critical + security만 detail
    file /var/log/aerospike/security.log {
        context any critical
        context security detail
    }

    # 감사 로그 (보안 이벤트 추적)
    file /var/log/aerospike/audit.log {
        context audit info
    }

    # syslog 전송
    syslog {
        facility local0
        tag aerospike-audit
        context audit info
    }

    # 컨테이너 환경 (stderr 출력)
    console {
        context any info
    }
}
```

---

## 3. ACL 설정 예제 (Operator CRD)

```yaml
# aerospike.conf에서 security {} 블록 필수
aerospikeConfig:
  security: {}

# ACL 사용자/역할 정의
aerospikeAccessControl:
  roles:
    - name: readwrite-role
      privileges:
        - read-write
    - name: readonly-role
      privileges:
        - read
  users:
    - name: admin                    # admin 필수
      secretName: aerospike-admin-secret
      roles:
        - sys-admin
        - user-admin                 # 필수 역할
    - name: app-user
      secretName: aerospike-appuser-secret
      roles:
        - readwrite-role
```

Secret 생성:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: aerospike-admin-secret
type: Opaque
data:
  password: YWRtaW4xMjM=            # base64(admin123)
```

---

## 4. K8s 배포 체크리스트

| # | 체크 항목 | 설명 |
|---|----------|------|
| 1 | `mode mesh` + `mesh-seed-address-port` | StatefulSet headless DNS 지정. Operator가 자동 주입 |
| 2 | `access-address` 설정 | Pod IP 또는 Service IP 명시. Smart Client 파티션 라우팅에 필수 |
| 3 | `proto-fd-max < OS ulimit -n` | K8s에서는 securityContext로 ulimit 관리 |
| 4 | `cluster-name` 명시 | 의도치 않은 클러스터 합류 방지 |
| 5 | `info { port 3003 }` 절대 사용 금지 | **8.1에서 파싱 오류**. `admin { port 3008 }` 사용 |
| 6 | `data-size` 최소 512MiB | 8 stripes * 8 write-blocks * 8MiB |
| 7 | `flush-size` 사용 | `write-block-size` 대신 사용 (7.1+) |
| 8 | console 로깅 | `kubectl logs` 연동을 위해 console 로그 싱크 권장 |
| 9 | `default-ttl` + `nsup-period` 정합성 | `nsup-period=0`이고 `default-ttl!=0`이면 시작 실패 |
| 10 | CE 이미지 사용 | `aerospike:ce-8.1.1.1` (enterprise/ee- 이미지 불가) |

### Operator 자동 처리 항목

| 항목 | 동작 |
|------|------|
| `cluster-name` | 미설정 시 CR 이름으로 자동 설정 |
| `network 기본 포트` | service: 3000, fabric: 3001, heartbeat: 3002 |
| `heartbeat mode` | 미설정 시 `mesh`로 자동 설정 |
| `proto-fd-max` | 미설정 시 15000으로 자동 설정 |
| `mesh-seed-address-port` | 모든 Pod FQDN 자동 주입 (`pod.svc.ns.svc.cluster.local`) |
| `access-address` | aerospikeNetworkPolicy 기반 Pod/Node IP 자동 주입 |

### CRD Webhook 검증 항목

- `size` > 8 → 에러 (CE 최대 8노드)
- `namespaces` 수 > 2 → 에러 (CE 최대 2개)
- 이미지에 `enterprise` 또는 `ee-` 포함 → 에러
- `xdr` 섹션 존재 → 에러 (CE 미지원)
- `tls` 섹션 존재 → 에러 (CE 미지원)
- `security` 있는데 `aerospikeAccessControl` 미정의 → 에러
- admin 사용자에 `sys-admin` + `user-admin` 역할 미부여 → 에러

---

## 5. CRD YAML → aerospike.conf 매핑

### 변환 규칙

| YAML 구조 | aerospike.conf 출력 |
|-----------|-------------------|
| `key: value` | `key value` |
| `key: { sub: val }` | `key { sub val }` |
| `namespaces: [{ name: ns1, ... }]` | `namespace ns1 { ... }` |
| `logging: [{ name: /path }]` | `logging { file /path { ... } }` |
| `storage-engine: { type: memory }` | `storage-engine memory` |
| `storage-engine: { type: device, file: ... }` | `storage-engine device { file ... }` |

### CRD YAML 예제

```yaml
apiVersion: acko.io/v1alpha1
kind: AerospikeCECluster
metadata:
  name: aerospike-cluster
spec:
  size: 3
  image: aerospike:ce-8.1.1.1

  aerospikeConfig:
    service:
      cluster-name: my-cluster
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
      - name: cache
        replication-factor: 2
        storage-engine:
          type: memory
          data-size: 4294967296      # 4G (바이트 단위)
      - name: data
        replication-factor: 2
        max-record-size: 1048576     # 1M (바이트 단위)
        storage-engine:
          type: device
          file: /opt/aerospike/data/data.dat
          filesize: 17179869184      # 16G (바이트 단위)
          flush-size: 1048576        # 1M (바이트 단위)
    logging:
      - name: /var/log/aerospike/aerospike.log
        context: any info
```

---

## 6. Eviction / Stop-writes 동작 흐름

```
     0%  ┌─────────────────────────────────┐
         │         정상 운영                │
         ├─────────────────────────────────┤
  ~50%   │  defrag-lwm-pct (defrag 시작)    │
         ├─────────────────────────────────┤
  ~70%   │  evict-used-pct (eviction 시작)  │  ← storage 사용률 기준
         ├─────────────────────────────────┤
  ~90%   │  stop-writes-sys-memory-pct     │  ← 시스템 메모리 기준 쓰기 중지
         ├─────────────────────────────────┤
 100%    │  가용 공간 없음                   │
         └─────────────────────────────────┘
```

- `default-ttl 0` 레코드는 eviction 대상이 아님 → 디스크 가득 참 가능
- 프로덕션에서는 `default-ttl` 설정 또는 충분한 스토리지 확보 필수

---

## 7. 운영 Best Practices

### 메모리 설정

- memory 모드: `data-size`로 인메모리 데이터 크기 지정 (최소 512MiB)
- device 모드: `indexes-memory-budget`으로 인덱스 메모리 예산 지정 (8.0+ 권장)
- `stop-writes-sys-memory-pct`로 시스템 전체 메모리 보호

### 스토리지 최적화

- SSD/NVMe: `flush-size 128K` 권장 (7.1+, `write-block-size` 대체)
- 파일 기반: `flush-size 1M` 기본값 유지
- `filesize` 최대 2TiB
- 복수 파일/디바이스로 I/O 분산

### 서비스 스레드 튜닝

- `service-threads`: SSD namespace → vCPU * 5, 메모리 전용 → vCPU * 1
- `batch-threads`: 배치 요청 처리 (기본: 6)
- `migrate-threads`: 리밸런싱 속도 vs 클라이언트 성능 트레이드오프

### K8s 환경 특화

- 컨테이너 이미지: `aerospike:ce-8.1.1.1`
- PersistentVolume으로 데이터 영속성 보장
- `console` 로깅으로 `kubectl logs` 연동
- Headless Service로 Pod DNS 기반 mesh-seed 자동 구성
