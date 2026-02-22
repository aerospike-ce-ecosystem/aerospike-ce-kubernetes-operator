---
name: aerospike-ce8-configuration-guide
description: Aerospike CE 8.1 K8s operator guide — deployment checklist, CRD YAML mapping, webhook validation, config example
disable-model-invocation: false
---

# Aerospike CE 8.1 K8s Operator Configuration Guide

이 Operator 프로젝트에 특화된 K8s 배포 체크리스트, CRD 매핑 규칙, Webhook 검증 항목, 설정 예제를 제공합니다.
일반적인 aerospike.conf 파라미터 정보는 `aerospike-ce-8-guide` skill을 참조하세요.

## Output

아래 내용을 마크다운으로 출력하세요.

---

# Aerospike CE 8.1 — K8s Operator Configuration Guide

---

## K8s 배포 체크리스트

| # | 항목 | 설명 |
|---|------|------|
| 1 | `mode mesh` + headless DNS | Operator가 mesh-seed-address-port 자동 주입 |
| 2 | `access-address` | Smart Client 파티션 라우팅에 필수 (Pod/Service IP) |
| 3 | `proto-fd-max < ulimit -n` | securityContext로 ulimit 관리 |
| 4 | `cluster-name` 명시 | 의도치 않은 클러스터 합류 방지 |
| 5 | `info { port 3003 }` 금지 | **8.1 파싱 오류**. `admin { port 3008 }` 사용 |
| 6 | `data-size >= 512MiB` | 8 stripes * 8 write-blocks * 8MiB |
| 7 | `flush-size` 사용 | `write-block-size` 대신 (7.1+) |
| 8 | console 로깅 | `kubectl logs` 연동 |
| 9 | `nsup-period` + `default-ttl` | `nsup-period=0` + `default-ttl!=0` → 시작 실패 |
| 10 | CE 이미지 | `aerospike:ce-8.1.1.1` (enterprise/ee- 불가) |

---

## Operator 자동 처리

| 항목 | 미설정 시 동작 |
|------|--------------|
| `cluster-name` | CR 이름으로 설정 |
| `network 포트` | service:3000, fabric:3001, heartbeat:3002 |
| `heartbeat mode` | `mesh` |
| `proto-fd-max` | 15000 |
| `mesh-seed-address-port` | 모든 Pod FQDN 자동 주입 |
| `access-address` | aerospikeNetworkPolicy 기반 Pod/Node IP 주입 |

---

## Webhook 검증

- `size > 8` → 에러
- `namespaces > 2` → 에러
- 이미지에 `enterprise`/`ee-` → 에러
- `xdr`/`tls` 섹션 존재 → 에러
- `security` 있는데 `aerospikeAccessControl` 없음 → 에러
- admin에 `sys-admin` + `user-admin` 미부여 → 에러

---

## CRD YAML → aerospike.conf 변환 규칙

| CRD YAML | aerospike.conf |
|----------|---------------|
| `namespaces: [{ name: ns1, ... }]` | `namespace ns1 { ... }` |
| `logging: [{ name: /path }]` | `logging { file /path { ... } }` |
| `storage-engine: { type: memory, data-size: N }` | `storage-engine memory { data-size N }` |
| `storage-engine: { type: device, file: ... }` | `storage-engine device { file ... }` |
| `security: {}` | `security { }` |

> CRD에서 크기 값은 **바이트 단위 정수**로 지정: `4294967296` (= 4G)

---

## CRD 설정 예제 (3노드, 2 namespace)

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
      cluster-name: aerospike-cluster
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
          data-size: 4294967296      # 4G
      - name: data
        replication-factor: 2
        storage-engine:
          type: device
          file: /opt/aerospike/data/data.dat
          filesize: 17179869184      # 16G
          flush-size: 1048576        # 1M
    logging:
      - name: /var/log/aerospike/aerospike.log
        context: any info

  storage:
    volumes:
      - name: data-vol
        source:
          persistentVolume:
            storageClass: standard
            size: 20Gi
            volumeMode: Filesystem
        aerospike:
          path: /opt/aerospike/data
        cascadeDelete: true
```

---

## ACL CRD 예제

```yaml
aerospikeConfig:
  security: {}                       # ACL 활성화

aerospikeAccessControl:
  users:
    - name: admin                    # 필수
      secretName: aerospike-admin-secret
      roles: [sys-admin, user-admin] # 둘 다 필수
    - name: app-user
      secretName: aerospike-appuser-secret
      roles: [read-write]
```

Secret (`password` 키 필수):
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: aerospike-admin-secret
type: Opaque
data:
  password: YWRtaW4xMjM=            # base64
```
