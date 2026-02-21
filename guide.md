# Aerospike CE 8.1 Configuration Guide (aerospike.conf)

Aerospike Community Edition 8.1 기준 `aerospike.conf` 설정 가이드.
이 문서는 Aerospike CE 8.1의 모든 주요 설정 섹션과 파라미터를 다룹니다.

> **참고**: 이 가이드는 Community Edition(CE) 기준이며, Enterprise Edition(EE) 전용 기능은 별도로 표기합니다.

---

## 목차

1. [Community Edition 제한 사항](#1-community-edition-제한-사항)
2. [설정 파일 기본 구조](#2-설정-파일-기본-구조)
3. [service 섹션](#3-service-섹션)
4. [network 섹션](#4-network-섹션)
5. [namespace 섹션](#5-namespace-섹션)
6. [logging 섹션](#6-logging-섹션)
7. [security 섹션](#7-security-섹션)
8. [전체 설정 예제](#8-전체-설정-예제)
9. [Kubernetes Operator에서의 설정 매핑](#9-kubernetes-operator에서의-설정-매핑)
10. [운영 Best Practices](#10-운영-best-practices)

---

## 1. Community Edition 제한 사항

Aerospike CE 8.1은 EE 대비 다음 제한이 있습니다:

| 항목 | Community Edition (CE) | Enterprise Edition (EE) |
|------|----------------------|------------------------|
| **최대 클러스터 노드 수** | **8개** | 256개 |
| **최대 namespace 수** | **2개** | 32개 |
| **최대 레코드 수/namespace/node** | 4,294,967,296 (2^32) | 549,755,813,888 (2^39) |
| **index-stage-size 범위** | 128M ~ 1G (2^27 ~ 2^30) | 128M ~ 16G (2^27 ~ 2^34) |
| **XDR (Cross-DC Replication)** | 사용 불가 | 사용 가능 |
| **TLS 암호화** | 사용 불가 | 사용 가능 |
| **LDAP 인증** | 사용 불가 | 사용 가능 |
| **압축 (Compression)** | 사용 불가 | 사용 가능 |
| **Fast Restart** | 사용 불가 | 사용 가능 |
| **Durable Deletes** | 사용 불가 | 사용 가능 |
| **Multi-record Transactions** | 사용 불가 | 사용 가능 |
| **Data Masking** | 사용 불가 | 사용 가능 (8.1.1+) |
| **Feature Key File** | 불필요 | 필수 |

> CE에서는 `xdr`, `tls` 섹션을 설정 파일에 포함하면 안 됩니다.

---

## 2. 설정 파일 기본 구조

설정 파일의 기본 위치: `/etc/aerospike/aerospike.conf`

```
service {
    # 클러스터 전역 설정
}

logging {
    # 로그 설정
}

network {
    service { }
    heartbeat { }
    fabric { }
    info { }
}

namespace <이름> {
    # 데이터 저장소 설정
    storage-engine <타입> { }
}

security {
    # 보안/ACL 설정 (선택)
}
```

### 설정 파일 문법 규칙

- 중괄호 `{ }` 로 섹션을 구분
- 키-값은 공백으로 구분 (등호 사용하지 않음): `proto-fd-max 15000`
- 주석은 `#`으로 시작
- 크기 표기: `4G`, `128K`, `1M` 등 단위 접미사 지원
- boolean: `true` / `false`
- 대소문자 구분

---

## 3. service 섹션

클러스터 전역 동작을 제어하는 핵심 섹션.

```
service {
    cluster-name my-cluster
    proto-fd-max 15000
    service-threads 20
    node-id-interface eth0
}
```

### 주요 파라미터

| 파라미터 | 타입 | 기본값 | 설명 |
|---------|------|--------|------|
| `cluster-name` | string | (없음) | 클러스터 식별 이름. 동일 네트워크의 클러스터를 구분. 재시작 후에도 유지됨 |
| `proto-fd-max` | integer | **15000** | 최대 파일 디스크립터 수. 동시 클라이언트 연결 수 제한 |
| `service-threads` | integer | CPU 수 | 클라이언트 요청 처리 스레드 수. SSD namespace 사용 시 vCPU 수 × 5 권장. 범위: 1~4096 |
| `batch-index-threads` | integer | CPU 수 | 배치 인덱스 응답 처리 워커 스레드 수. 0으로 설정 시 배치 명령 비활성화. 동적 변경 가능 |
| `migrate-threads` | integer | 1 | 데이터 마이그레이션 스레드 수. 노드 추가/제거 시 리밸런싱 속도에 영향 |
| `migrate-fill-delay` | integer | 0 | 마이그레이션 시작까지의 지연 시간 (초) |
| `node-id-interface` | string | (없음) | 노드 ID 생성에 사용할 네트워크 인터페이스 (예: `eth0`) |
| `pidfile` | string | (없음) | PID 파일 경로 (예: `/var/run/aerospike/asd.pid`) |
| `user` | string | (없음) | 프로세스 실행 사용자 |
| `group` | string | (없음) | 프로세스 실행 그룹 |

### 동적 변경 가능 파라미터

일부 service 파라미터는 재시작 없이 `asadm` 또는 `asinfo`로 런타임 변경 가능:

```bash
# 예: batch-index-threads 변경
asinfo -v "set-config:context=service;batch-index-threads=16"

# 예: proto-fd-max 변경
asadm -e 'enable; manage config service param proto-fd-max to 100000'
```

---

## 4. network 섹션

노드 간 통신, 클라이언트 연결, 하트비트를 설정합니다.
4개의 하위 섹션으로 구성됩니다.

### 4.1 network > service (클라이언트 연결)

```
network {
    service {
        address any          # 수신 바인드 주소
        port 3000            # 클라이언트 연결 포트
        access-address 10.0.0.101  # 클라이언트에게 공개되는 주소
    }
}
```

| 파라미터 | 타입 | 기본값 | 설명 |
|---------|------|--------|------|
| `address` | string | `any` | 서비스 바인드 주소. `any`는 모든 인터페이스에서 수신 |
| `port` | integer | **3000** | 클라이언트 연결 포트 |
| `access-address` | string | (address) | 클라이언트에 공개되는 IP. 미설정 시 `address` 값 사용. IPv4, IPv6, DNS 이름 지원. 복수 설정 가능 |
| `alternate-access-address` | string | (없음) | 대체 접근 주소. 외부 네트워크(NAT 환경 등)에서의 접근 용. 복수 설정 가능 |

> **access-address**: `address`가 `any`이고 `access-address`가 미설정이면 모든 사용 가능한 IP가 공개됩니다. 특정 IP만 공개하려면 `access-address`를 명시적으로 설정하세요.

### 4.2 network > heartbeat (노드 탐지)

CE에서는 **mesh 모드만** 지원합니다.

```
network {
    heartbeat {
        mode mesh                              # CE에서는 mesh만 가능
        port 3002                              # 하트비트 포트
        address 10.0.0.101                     # 하트비트 바인드 주소

        # 시드 노드 목록 (클러스터 내 모든 노드 또는 일부)
        mesh-seed-address-port 10.0.0.101 3002
        mesh-seed-address-port 10.0.0.102 3002
        mesh-seed-address-port 10.0.0.103 3002

        interval 150                           # 하트비트 간격 (밀리초)
        timeout 10                             # 타임아웃 (하트비트 횟수)
    }
}
```

| 파라미터 | 타입 | 기본값 | 설명 |
|---------|------|--------|------|
| `mode` | string | - | 하트비트 모드. CE에서는 **`mesh`** 만 사용 |
| `port` | integer | **3002** | 하트비트 통신 포트 |
| `address` | string | (없음) | 하트비트 바인드 주소 |
| `mesh-seed-address-port` | string int | - | 시드 노드의 IP/호스트명과 포트. 복수 설정 가능. 자기 자신을 포함해도 무방 |
| `interval` | integer | **150** | 하트비트 전송 간격 (밀리초) |
| `timeout` | integer | **10** | 연속 하트비트 실패 횟수 초과 시 노드를 다운으로 판정 |

> **mesh-seed-address-port**: 클러스터의 모든 노드를 시드로 등록하면 가장 안정적입니다. 자기 자신의 주소를 포함하는 것도 허용되며, 클러스터 전체에 동일 설정 파일을 사용할 수 있어 편리합니다.

### 4.3 network > fabric (노드 간 데이터 통신)

```
network {
    fabric {
        address any     # 바인드 주소
        port 3001       # fabric 통신 포트
    }
}
```

| 파라미터 | 타입 | 기본값 | 설명 |
|---------|------|--------|------|
| `address` | string | `any` | fabric 바인드 주소 |
| `port` | integer | **3001** | 노드 간 데이터 복제/마이그레이션 포트 |

### 4.4 network > info (관리 인터페이스)

```
network {
    info {
        address 127.0.0.1   # 로컬만 허용
        port 3003            # 관리 포트
    }
}
```

| 파라미터 | 타입 | 기본값 | 설명 |
|---------|------|--------|------|
| `address` | string | (없음) | 관리 인터페이스 바인드 주소 |
| `port` | integer | **3003** | 관리/모니터링 포트 (asadm, asinfo 등에서 사용) |

### 4.5 네트워크 전체 예제

```
network {
    service {
        address any
        port 3000
        access-address 10.0.0.101
    }
    heartbeat {
        mode mesh
        address 10.0.0.101
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
    info {
        address 127.0.0.1
        port 3003
    }
}
```

---

## 5. namespace 섹션

데이터 저장 공간을 정의합니다. CE에서는 **최대 2개**의 namespace를 사용할 수 있습니다.

### 5.1 기본 namespace 파라미터

```
namespace <이름> {
    replication-factor 2
    memory-size 4G
    default-ttl 0
    high-water-memory-pct 60
    high-water-disk-pct 60
    stop-writes-sys-memory-pct 90
    max-record-size 128K
    storage-engine memory
}
```

| 파라미터 | 타입 | 기본값 | 설명 |
|---------|------|--------|------|
| `replication-factor` | integer | **2** | 데이터 복제본 수. 단일 노드에서는 1로 설정 |
| `memory-size` | size | **4G** | namespace에 할당되는 메모리 크기. primary/secondary 인덱스 및 데이터 저장에 사용 |
| `default-ttl` | integer | **0** (만료 없음) | 기본 TTL (초). 0이면 레코드가 만료되지 않음. 클라이언트가 TTL을 지정하지 않으면 이 값 사용 |
| `high-water-memory-pct` | integer | **60** | 메모리 사용률이 이 비율을 초과하면 TTL이 0이 아닌 데이터를 eviction(제거). `memory-size` 기준 |
| `high-water-disk-pct` | integer | **60** | 디스크 사용률이 이 비율을 초과하면 eviction 시작. device/file 크기 기준 |
| `stop-writes-sys-memory-pct` | integer | **90** | 시스템 전체 메모리 사용률이 이 비율을 초과하면 해당 namespace에 대한 쓰기 중지 |
| `max-record-size` | size | write-block-size | 최대 레코드 크기. 미설정 시 `write-block-size`가 최대값 |

### 5.2 Storage Engine: memory (인메모리)

데이터를 메모리에만 저장. 영속성 없음. 재시작 시 데이터 유실.

```
namespace test {
    replication-factor 1
    storage-engine memory {
        data-size 1G         # 데이터 저장에 사용할 메모리 크기
    }
}
```

**간단한 형식** (data-size 미지정 시):
```
namespace test {
    replication-factor 1
    memory-size 4G
    storage-engine memory
}
```

| 파라미터 | 타입 | 기본값 | 설명 |
|---------|------|--------|------|
| `data-size` | size | (없음) | 인메모리 데이터 저장에 할당할 크기 |

> 인메모리 namespace는 테스트, 캐시, 세션 저장 등 영속성이 불필요한 용도에 적합합니다.

### 5.3 Storage Engine: device (SSD/파일 영속성)

데이터를 SSD 디바이스 또는 파일에 영속 저장합니다.

#### 5.3.1 파일 기반 저장 (File-backed)

```
namespace myns {
    replication-factor 2
    memory-size 4G
    high-water-memory-pct 60
    high-water-disk-pct 60

    storage-engine device {
        file /opt/aerospike/data/myns.dat        # 데이터 파일 경로
        filesize 4G                               # 파일 최대 크기 (최대 2TiB)
        data-in-memory true                       # 데이터를 메모리에도 유지
        write-block-size 128K                     # SSD 최적화 블록 크기
    }
}
```

#### 5.3.2 Raw 디바이스 기반 저장

```
namespace myns {
    memory-size 64G
    storage-engine device {
        device /dev/nvme0n1p1                     # raw 디바이스 (최대 2TiB)
        device /dev/nvme0n1p2                     # 추가 디바이스 (선택)
        write-block-size 128K
        scheduler-mode noop                       # SSD 최적화 스케줄러
    }
}
```

#### 5.3.3 인메모리 + 파일 영속 (Hybrid)

데이터를 메모리에서 읽고, 파일에 영속 기록:

```
namespace myns {
    memory-size 64G
    high-water-memory-pct 60
    max-record-size 1M

    storage-engine device {
        write-block-size 1M
        file /opt/aerospike/ns1.dat
        file /opt/aerospike/ns2.dat              # 복수 파일 가능
        file /opt/aerospike/ns3.dat
        file /opt/aerospike/ns4.dat
        filesize 64G
        data-in-memory true                      # 읽기 시 메모리에서 서빙
        min-avail-pct 5                          # 디스크 여유 공간 stop-writes 임계값
        max-used-pct 70                          # 디스크 사용률 stop-writes 임계값
    }

    high-water-disk-pct 60
}
```

### 5.4 Storage Engine 파라미터 상세

| 파라미터 | 타입 | 기본값 | 설명 |
|---------|------|--------|------|
| `file` | string | - | 데이터 파일 경로. 복수 지정 가능 |
| `filesize` | size | - | 각 파일의 최대 크기 (최대 2TiB) |
| `device` | string | - | raw 디바이스 경로. 복수 지정 가능. **파티션은 파일시스템으로 마운트되면 안 됨** |
| `data-in-memory` | boolean | `false` | `true`이면 모든 데이터를 메모리에도 유지 (읽기 성능 향상) |
| `write-block-size` | size | **1M** | 쓰기 블록 크기. SSD에서는 `128K` 권장 |
| `scheduler-mode` | string | (없음) | 디스크 I/O 스케줄러. SSD에서는 `noop` 권장 |
| `min-avail-pct` | integer | **5** | 디스크 여유 비율이 이 값 미만이면 쓰기 중지 |
| `max-used-pct` | integer | **70** | 디스크 사용 비율이 이 값 초과 시 쓰기 중지 |

### 5.5 Storage Engine 선택 가이드

| 사용 사례 | Storage Engine | 설명 |
|----------|---------------|------|
| 테스트/캐시 | `memory` | 빠르지만 재시작 시 데이터 유실 |
| 일반 영속 저장 | `device` (file) | 파일 기반. Kubernetes PV와 함께 사용하기 적합 |
| 고성능 영속 저장 | `device` (raw) | raw SSD 디바이스 직접 사용. 최대 I/O 성능 |
| 빠른 읽기 + 영속성 | `device` + `data-in-memory true` | 메모리에서 읽기, 디스크에 영속 기록 |

---

## 6. logging 섹션

로그 출력 대상(sink)과 로그 수준을 설정합니다.

### 6.1 로그 싱크 유형

| 싱크 타입 | 설명 |
|----------|------|
| `console` | 표준 에러(stderr)로 출력. 컨테이너 환경에서 유용 |
| `file <경로>` | 지정된 파일에 로그 기록 |
| `syslog` | syslog 데몬으로 전송 |

### 6.2 기본 설정 예제

```
logging {
    # 콘솔 로그
    console {
        context any info
    }

    # 파일 로그
    file /var/log/aerospike/aerospike.log {
        context any info
    }
}
```

### 6.3 로그 컨텍스트 (모듈)

`context` 지시자로 특정 모듈의 로그 수준을 세밀하게 조절할 수 있습니다.

| 컨텍스트 | 설명 |
|---------|------|
| `any` | 모든 모듈 (기본 수준 설정에 사용) |
| `fabric` | 노드 간 fabric 통신 |
| `heartbeat` | 하트비트/클러스터 멤버십 |
| `drv_ssd` | SSD/디바이스 스토리지 드라이버 |
| `migrate` | 데이터 마이그레이션 |
| `namespace` | namespace 관련 동작 |
| `security` | 보안/ACL 관련 |
| `audit` | 감사 로그 (보안 이벤트 추적) |
| `tls` | TLS 관련 (EE 전용) |

### 6.4 로그 심각도 수준

높은 심각도에서 낮은 심각도 순서 (각 수준은 자신보다 높은 수준을 모두 포함):

| 수준 | 설명 |
|------|------|
| `critical` | 치명적 오류. 서버 중단 가능성 있는 이벤트 |
| `warning` | 경고. 잠재적 문제 |
| `info` | 일반 운영 정보 (기본 권장 수준) |
| `debug` | 디버깅 정보. 문제 해결 시 유용 |
| `detail` | 가장 상세한 로그. 대량의 세부 데이터 기록 |

### 6.5 고급 로깅 예제

```
logging {
    # 기본: 모든 모듈 info 수준
    file /var/log/aerospike/aerospike.log {
        context any info
    }

    # 보안 디버깅용: 모든 모듈 critical + security만 detail
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

### 6.6 동적 로그 수준 변경

재시작 없이 런타임에 로그 수준 변경 가능:

```bash
# security 컨텍스트를 detail로 변경
asadm -e 'enable; manage config logging file /var/log/aerospike/aerospike.log param security to detail'
```

---

## 7. security 섹션

보안/ACL (Access Control List) 설정. `security` 스탠자가 존재하면 ACL이 활성화됩니다.

> **중요**: 공식 문서에 따르면 RBAC는 EE/FE 전용이지만, CE에서도 기본적인 보안 설정은 가능합니다.
> 이 Operator에서는 CE 환경에서도 ACL을 지원하며, `security {}` 스탠자 활성화와 함께 사용합니다.

### 7.1 기본 보안 활성화

```
security {
}
```

빈 `security` 스탠자만으로 ACL 모드가 활성화됩니다.

### 7.2 ACL 설정 (Operator CRD 기준)

Kubernetes Operator에서는 `aerospikeAccessControl` 필드로 ACL을 관리합니다:

```yaml
aerospikeAccessControl:
  roles:
    - name: readwrite-role
      privileges:
        - read-write
    - name: readonly-role
      privileges:
        - read
  users:
    - name: admin
      secretName: aerospike-admin-secret    # K8s Secret (password 키 필요)
      roles:
        - sys-admin
        - user-admin                        # admin 필수 역할
    - name: app-user
      secretName: aerospike-appuser-secret
      roles:
        - readwrite-role
```

### 7.3 ACL 규칙

- `security` 섹션이 있으면 **반드시** `aerospikeAccessControl`을 정의해야 함
- 최소 1명의 admin 사용자가 `sys-admin` + `user-admin` 역할을 모두 가져야 함
- 비밀번호는 Kubernetes Secret의 `password` 키에 base64 인코딩으로 저장

### 7.4 기본 권한 목록

| 권한 | 설명 |
|------|------|
| `sys-admin` | 시스템 관리 (클러스터 설정 변경 등) |
| `user-admin` | 사용자/역할 관리 |
| `read` | 데이터 읽기 |
| `read-write` | 데이터 읽기/쓰기 |
| `read-write-udf` | 데이터 읽기/쓰기 + UDF 실행 |
| `data-admin` | 인덱스/UDF 관리 |
| `write` | 데이터 쓰기 |

---

## 8. 전체 설정 예제

### 8.1 최소 단일 노드 인메모리

```
service {
    cluster-name aerospike-ce-basic
    proto-fd-max 15000
}

logging {
    file /var/log/aerospike/aerospike.log {
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
    info {
        port 3003
    }
}

namespace test {
    replication-factor 1
    memory-size 1G
    storage-engine memory
}
```

### 8.2 3노드 클러스터 (SSD 파일 영속성)

```
service {
    cluster-name aerospike-ce-3node
    proto-fd-max 15000
    service-threads 20
}

logging {
    file /var/log/aerospike/aerospike.log {
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
    info {
        port 3003
    }
}

namespace testns {
    replication-factor 2
    memory-size 4G
    high-water-memory-pct 60
    high-water-disk-pct 60
    stop-writes-sys-memory-pct 90

    storage-engine device {
        file /opt/aerospike/data/testns.dat
        filesize 4G
        data-in-memory true
        write-block-size 128K
    }
}
```

### 8.3 보안(ACL) 활성화

```
service {
    cluster-name aerospike-ce-secure
    proto-fd-max 15000
}

logging {
    file /var/log/aerospike/aerospike.log {
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
}

namespace testns {
    replication-factor 2
    memory-size 4G
    storage-engine device {
        file /opt/aerospike/data/testns.dat
        filesize 4G
    }
}
```

### 8.4 고성능 Raw SSD 구성

```
service {
    cluster-name aerospike-ce-perf
    proto-fd-max 15000
    service-threads 20
}

logging {
    file /var/log/aerospike/aerospike.log {
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
}

namespace bar {
    memory-size 64G
    max-record-size 128K
    stop-writes-sys-memory-pct 90

    storage-engine device {
        device /dev/nvme0n1p1
        device /dev/nvme0n1p2
        device /dev/nvme0n1p3
        device /dev/nvme0n1p4
        scheduler-mode noop
        write-block-size 128K
    }
}
```

---

## 9. Kubernetes Operator에서의 설정 매핑

이 프로젝트의 Aerospike CE Kubernetes Operator에서는 `aerospikeConfig` 필드를 통해 aerospike.conf를 YAML로 구성합니다.

### 9.1 YAML → aerospike.conf 변환 규칙

| YAML 구조 | aerospike.conf 출력 |
|-----------|-------------------|
| `key: value` | `key value` |
| `key: { sub: val }` | `key { sub val }` |
| `namespaces: [{ name: ns1, ... }]` | `namespace ns1 { ... }` |
| `logging: [{ name: /path }]` | `logging { file /path { ... } }` |
| `storage-engine: { type: memory }` | `storage-engine memory` |
| `storage-engine: { type: device, file: ... }` | `storage-engine device { file ... }` |

### 9.2 CRD에서 설정 예제

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
          filesize: 4294967296       # 4G (바이트 단위)
          data-in-memory: true
    logging:
      - name: /var/log/aerospike/aerospike.log
        context: any info
```

### 9.3 Operator의 자동 처리

Operator가 자동으로 처리하는 항목:

| 항목 | 동작 |
|------|------|
| **cluster-name** | 미설정 시 CR 이름으로 자동 설정 |
| **network 기본 포트** | service: 3000, fabric: 3001, heartbeat: 3002 |
| **heartbeat mode** | 미설정 시 `mesh`로 자동 설정 |
| **proto-fd-max** | 미설정 시 15000으로 자동 설정 |
| **mesh-seed-address-port** | 모든 Pod의 FQDN으로 자동 주입 (`pod.svc.ns.svc.cluster.local`) |
| **access-address** | `aerospikeNetworkPolicy` 기반 `MY_POD_IP` 또는 `MY_NODE_IP` 플레이스홀더 주입, init 컨테이너에서 실제 값으로 치환 |

### 9.4 CRD 검증 (Webhook)

Operator의 Webhook이 자동으로 검증하는 항목:

- `size` > 8 → 에러 (CE 최대 8노드)
- `namespaces` 수 > 2 → 에러 (CE 최대 2개)
- 이미지에 `enterprise` 또는 `ee-` 포함 → 에러
- `xdr` 섹션 존재 → 에러 (CE 미지원)
- `tls` 섹션 존재 → 에러 (CE 미지원)
- `security` 있는데 `aerospikeAccessControl` 미정의 → 에러
- admin 사용자에 `sys-admin` + `user-admin` 역할 미부여 → 에러

---

## 10. 운영 Best Practices

### 10.1 메모리 설정

- `memory-size`는 primary index + data-in-memory가 사용할 메모리 예산
- device 모드에서 `data-in-memory false`이면 `memory-size`는 인덱스만 저장
- `stop-writes-sys-memory-pct`로 시스템 전체 메모리 보호

### 10.2 스토리지 최적화

- SSD 사용 시 `write-block-size 128K` 권장
- SSD에서 `scheduler-mode noop` 사용 권장
- `filesize`는 최대 2TiB까지 설정 가능
- 복수 파일/디바이스로 I/O 분산 가능

### 10.3 Eviction 및 Stop-writes 임계값

```
                  ┌─────────────────────────────┐
       0%         │     정상 운영               │
                  ├─────────────────────────────┤
      60%         │     Eviction 시작            │  ← high-water-memory-pct / high-water-disk-pct
                  │     (TTL > 0 데이터 제거)     │
                  ├─────────────────────────────┤
      70%         │     Stop-writes             │  ← max-used-pct (디스크)
                  ├─────────────────────────────┤
      90%         │     시스템 보호 Stop-writes   │  ← stop-writes-sys-memory-pct
                  ├─────────────────────────────┤
     100%         │     가용 공간 없음            │
                  └─────────────────────────────┘
```

- `high-water-*-pct` 도달 시: TTL이 0이 아닌 레코드부터 eviction
- `default-ttl 0` 레코드는 eviction 대상이 아님 → 디스크 가득 참 가능
- 프로덕션에서는 `default-ttl`을 적절히 설정하거나, 충분한 스토리지 확보 필요

### 10.4 서비스 스레드 튜닝

- `service-threads`: SSD namespace가 있으면 vCPU × 5, 없으면 vCPU 수와 동일
- `batch-index-threads`: 배치 작업이 많으면 CPU 수까지 증가 가능
- `migrate-threads`: 리밸런싱 속도 vs 클라이언트 성능 트레이드오프

### 10.5 Kubernetes 환경 특화

- 컨테이너 이미지: `aerospike:ce-8.1.1.1` 사용
- PersistentVolume으로 데이터 영속성 보장
- `console` 로깅으로 `kubectl logs` 연동
- Headless Service로 Pod DNS 기반 mesh-seed 자동 구성

---

## 참고 자료

- [Aerospike Configuration Reference](https://aerospike.com/docs/database/reference/config/)
- [Aerospike Namespace Storage Configuration](https://aerospike.com/docs/database/manage/namespace/storage/config/)
- [Aerospike Network Configuration](https://aerospike.com/docs/database/manage/network/)
- [Aerospike Log Management](https://aerospike.com/docs/database/manage/logging/logs/)
- [Aerospike System Limits](https://aerospike.com/docs/database/reference/limitations)
- [Aerospike CE vs EE Feature Comparison](https://aerospike.com/products/features-and-editions/)
- [Aerospike GitHub - Default Config](https://github.com/aerospike/aerospike-server/blob/master/as/etc/aerospike.conf)
- [Aerospike Dynamic Runtime Config](https://aerospike.com/docs/database/tools/runtime-config/)
