---
name: aerospike-ce-8-guide
description: Aerospike CE 8.1 parameter reference — CE constraints, all stanza parameters, 7.x→8.1 migration changes, config syntax
disable-model-invocation: false
---

# Aerospike CE 8.1 Parameter Reference

Aerospike CE 8.1 aerospike.conf 파라미터 레퍼런스. 섹션별 파라미터 테이블, CE 제약사항, 7.x→8.1 변경사항을 제공합니다.
인자가 없으면 전체 레퍼런스를 출력하고, 섹션 이름이 주어지면 해당 섹션만 출력하세요.

## Arguments

선택적 인자: 섹션 이름 (예: `service`, `network`, `namespace`, `logging`, `security`, `migration`, `constraints`)
인자 없으면 전체 레퍼런스를 출력합니다.

## Output

아래 전체 내용을 마크다운으로 출력하세요.

---

# Aerospike CE 8.1 — Parameter Reference

> **대상 버전**: Aerospike CE 8.1.x (`aerospike:ce-8.1.1.1`)
> **기본 설정 파일 위치**: `/etc/aerospike/aerospike.conf`

---

## 1. CE 제약사항

| 항목 | Community Edition (CE) | Enterprise Edition (EE) |
|------|----------------------|------------------------|
| **최대 클러스터 노드 수** | **8개** | 256개 |
| **최대 namespace 수** | **2개** | 32개 |
| **최대 레코드 수/namespace/node** | 4,294,967,296 (2^32) | 549,755,813,888 (2^39) |
| **index-stage-size 범위** | 128M ~ 1G (2^27 ~ 2^30) | 128M ~ 16G (2^27 ~ 2^34) |
| **XDR (Cross-DC Replication)** | 사용 불가 | 사용 가능 |
| **TLS 암호화** | 사용 불가 | 사용 가능 |
| **Strong Consistency** | 사용 불가 | 사용 가능 |
| **Encryption at Rest** | 사용 불가 | 사용 가능 |
| **LDAP 인증** | 사용 불가 | 사용 가능 |
| **Fast Restart** | 사용 불가 | 사용 가능 |
| **Compression** | 사용 불가 | 사용 가능 |
| **Durable Deletes** | 사용 불가 | 사용 가능 |
| **Multi-record Transactions** | 사용 불가 | 사용 가능 |
| **Data Masking** | 사용 불가 | 사용 가능 (8.1.1+) |
| **Feature Key File** | 불필요 | 필수 |

> CE 설정에서 `xdr`, `tls` 섹션을 포함하면 서버 시작이 실패합니다.

---

## 2. 설정 파일 구조 및 문법

설정 파일 위치: `/etc/aerospike/aerospike.conf`

```
service { }
logging { }
network {
    service { }
    heartbeat { }
    fabric { }
    admin { }       # [8.1] info { } 제거됨
}
namespace <이름> {
    storage-engine <타입> { }
}
security { }        # 선택
```

### 문법 규칙

- 중괄호 `{ }` 로 섹션 구분
- 키-값은 공백으로 구분 (등호 없음): `proto-fd-max 15000`
- 주석은 `#`으로 시작
- 크기 표기: `4G`, `128K`, `1M` 등 단위 접미사 지원
- boolean: `true` / `false`
- 대소문자 구분

---

## 3. 7.x → 8.1 변경사항 요약

| 항목 | 이전 (CE 7.x) | CE 8.1 | 변경 시점 |
|------|--------------|--------|----------|
| **info port** | `network { info { port 3003 } }` | **완전 제거**. `network { admin { port 3008 } }` 사용 | 8.1.0 |
| **네임스페이스 데이터 크기 (memory)** | `memory-size 4G` | **`data-size 4G`** (storage-engine memory 블록 내) | 7.0+ deprecated, 7.1+ 제거 |
| **쓰기 블록 크기** | `write-block-size 128K` | **`flush-size 128K`** | 7.1+ (write-block-size는 내부적으로 8M 고정) |
| **최대 레코드 크기** | 명시 필요 (기본값 = write-block-size) | **기본값 1M** (최대 8M) | 7.1+ |
| **MRT write-block** | 없음 | MRT write-block이 data-size 계산에 영향 | 8.0+ |
| **admin port** | 없음 (info port 3003 사용) | `network { admin { port 3008 } }` 신규 추가 | 8.1.0 |
| **memory-size (device 모드)** | `memory-size 64G` (인덱스+데이터 예산) | **제거됨**. `indexes-memory-budget` 사용 권장 | 7.0+ deprecated |
| **stop-writes-pct** | `stop-writes-pct 90` | **제거됨**. `stop-writes-sys-memory-pct 90` 사용 | 7.0 제거 |
| **data-in-memory** | namespace 레벨 설정 | storage-engine device 블록 내에서만 설정 | 7.0 이동 |
| **high-water-memory-pct** | namespace 레벨 eviction 제어 | **제거됨**. evict-used-pct 및 evict-tenths-pct 사용 | 7.0 제거 |

### 마이그레이션 diff

```diff
# === 7.x (구 방식) ===
- namespace cache {
-     memory-size 4G              # 7.0에서 deprecated
-     storage-engine memory
- }
- namespace data {
-     memory-size 64G             # 7.0에서 deprecated
-     storage-engine device {
-         write-block-size 128K   # 7.1에서 deprecated
-     }
- }
- network {
-     info { port 3003 }          # 8.1에서 제거
- }

# === 8.1 (신 방식) ===
+ namespace cache {
+     storage-engine memory {
+         data-size 4G            # memory 블록 내에서 지정
+     }
+ }
+ namespace data {
+     storage-engine device {
+         flush-size 128K         # write-block-size 대체
+     }
+ }
+ network {
+     admin { port 3008 }         # info 대체
+ }
```

---

## 4. service 섹션

| 파라미터 | 타입 | 기본값 | 동적변경 | 설명 |
|---------|------|--------|---------|------|
| `cluster-name` | string | (없음) | No | 클러스터 식별. 동일 이름 노드끼리만 합류. **8.0+ 필수 권장** |
| `proto-fd-max` | int | **15000** | Yes | 최대 클라이언트 FD. OS ulimit -n 보다 작아야 함 |
| `proto-fd-idle-ms` | int | 60000 | Yes | idle 클라이언트 FD 타임아웃 (ms) |
| `service-threads` | int | vCPU 수 | Yes | 요청 처리 스레드. SSD: vCPU*5, 메모리: vCPU*1. 범위: 1~4096 |
| `batch-threads` | int | 6 | Yes | 배치 요청 처리 스레드 |
| `batch-index-threads` | int | CPU 수 | Yes | 배치 인덱스 응답 워커 스레드. 0=배치 비활성화 |
| `migrate-threads` | int | 1 | Yes | 마이그레이션 스레드. 리밸런싱 속도 vs 성능 트레이드오프 |
| `migrate-fill-delay` | int | 0 | Yes | 마이그레이션 시작 지연 시간 (초) |
| `node-id-interface` | string | (없음) | No | 노드 ID 생성에 사용할 네트워크 인터페이스 (예: `eth0`) |
| `pidfile` | string | (없음) | No | PID 파일 경로 |
| `user` | string | (없음) | No | 프로세스 실행 사용자 |
| `group` | string | (없음) | No | 프로세스 실행 그룹 |
| `paxos-single-replica-limit` | int | 1 | No | 노드 수 이하일 때 replication-factor 자동 1 |

### 동적 변경 예제

```bash
# asinfo
asinfo -v "set-config:context=service;batch-index-threads=16"

# asadm
asadm -e 'enable; manage config service param proto-fd-max to 100000'
```

---

## 5. network 섹션

### 5.1 service (클라이언트 연결)

| 파라미터 | 타입 | 기본값 | 설명 |
|---------|------|--------|------|
| `address` | string | `any` | 서비스 바인드 주소. `any` = 모든 인터페이스 |
| `port` | int | **3000** | 클라이언트 연결 포트 |
| `access-address` | string | (address) | 클라이언트에 공개되는 IP. IPv4/IPv6/DNS 지원. 복수 가능 |
| `alternate-access-address` | string | (없음) | 대체 접근 주소 (NAT, K8s NodePort). 복수 가능 |

> `address`가 `any`이고 `access-address` 미설정이면 모든 IP가 공개됩니다.

### 5.2 heartbeat (노드 감시)

| 파라미터 | 타입 | 기본값 | 설명 |
|---------|------|--------|------|
| `mode` | string | - | CE에서는 **`mesh`** 만 사용 |
| `port` | int | **3002** | 하트비트 통신 포트 |
| `address` | string | (없음) | 하트비트 바인드 주소 |
| `mesh-seed-address-port` | string int | - | 시드 노드 IP/호스트명과 포트. 복수 가능. 자기 자신 포함 허용 |
| `interval` | int | **150** | 하트비트 전송 간격 (ms) |
| `timeout` | int | **10** | 연속 실패 횟수 초과 시 Dead 판정 |

> 클러스터 모든 노드를 시드로 등록하면 가장 안정적. 자기 자신 포함 허용.

### 5.3 fabric (데이터 복제)

| 파라미터 | 타입 | 기본값 | 설명 |
|---------|------|--------|------|
| `address` | string | `any` | fabric 바인드 주소 |
| `port` | int | **3001** | 노드 간 데이터 복제/마이그레이션 포트 |

### 5.4 admin (관리 인터페이스) — 8.1 신규

> **[8.1 변경]** `info { port 3003 }` 완전 제거. `info` 블록 잔존 시 **파싱 오류**.

| 파라미터 | 타입 | 기본값 | 설명 |
|---------|------|--------|------|
| `address` | string | (없음) | 관리 인터페이스 바인드 주소 |
| `port` | int | **3008** | asadm 등 관리 도구 포트. info port 3003 대체 |

### 네트워크 포트 요약

| 서브스탠자 | 포트 | 용도 |
|-----------|------|------|
| `service` | 3000 | 클라이언트 연결 |
| `fabric` | 3001 | 노드 간 데이터 복제/마이그레이션 |
| `heartbeat` | 3002 | 노드 상태 감시 (Paxos) |
| `admin` | 3008 | **[8.1 신규]** 관리 도구 |
| ~~`info`~~ | ~~3003~~ | **[8.1 제거]** 사용 금지 |

---

## 6. namespace 섹션

CE 최대 2개 namespace.

### namespace 파라미터

| 파라미터 | 타입 | 기본값 | 설명 |
|---------|------|--------|------|
| `replication-factor` | int | 2 | 복제본 수 (Primary + Replica). 단일 노드: 1 |
| `default-ttl` | int | 0 | 기본 TTL (초). 0 = 만료 없음 |
| `max-ttl` | int | (없음) | 최대 허용 TTL (초) |
| `max-record-size` | size | **1M** | **[7.1+]** 최대 레코드 크기. 최대 8M |
| `nsup-period` | int | 0 | TTL 만료 체크 주기 (초). 0 = 비활성화 |
| `evict-tenths-pct` | int | 5 | eviction 사이클당 제거 비율 (1/10 % 단위) |
| `stop-writes-sys-memory-pct` | int | 90 | **[7.0+]** 시스템 메모리 임계값 초과 시 쓰기 중지 |

### storage-engine 파라미터

| 파라미터 | 타입 | 기본값 | 적용 | 설명 |
|---------|------|--------|------|------|
| `data-size` | size | - | memory | **[7.0+]** 인메모리 데이터 크기. 최소 512MiB. `memory-size` 대체 |
| `file` | string | - | device | 데이터 파일 경로 (복수 가능, 최대 128개) |
| `filesize` | size | - | device (file) | 파일당 최대 크기 (최대 2TiB) |
| `device` | string | - | device | Raw 디바이스 경로 (복수 가능). 파일시스템 마운트 불가 |
| `flush-size` | size | 1M | device | **[7.1+]** 쓰기 블록 크기. NVMe: 128K, 파일: 1M. `write-block-size` 대체 |
| `data-in-memory` | bool | false | device | 데이터를 메모리에도 유지 (Hybrid Memory) |
| `defrag-lwm-pct` | int | 50 | device | 이 % 미만 채워진 블록은 defrag 대상 |
| `defrag-sleep` | int | 1000 | device | defrag 스레드 sleep (us) |
| `evict-used-pct` | int | 70 | device | 스토리지 사용률 eviction 임계값 (%) |

### 스토리지 패턴 선택

| 패턴 | storage-engine | 영속성 | 읽기 성능 | 용도 |
|------|---------------|--------|----------|------|
| 메모리 전용 | `memory { data-size }` | 없음 | 최고 | 캐시, 세션 |
| 파일 영속 | `device { file }` | 있음 | 중간 | K8s PV 일반 영속 |
| Raw SSD | `device { device }` | 있음 | 높음 | 베어메탈 고성능 |
| Hybrid (HMA) | `device { file, data-in-memory true }` | 있음 | 최고 | 읽기 성능 + 영속성 |

---

## 7. logging 섹션

### 로그 싱크 유형

| 싱크 타입 | 설명 |
|----------|------|
| `console` | stderr 출력. 컨테이너/K8s 환경 권장 |
| `file <경로>` | 파일에 로그 기록 |
| `syslog` | syslog 데몬 전송 |

### 로그 심각도 (높은 → 낮은)

| 수준 | 설명 |
|------|------|
| `critical` | 치명적 오류 |
| `warning` | 경고 |
| `info` | 일반 운영 정보 (**기본 권장**) |
| `debug` | 디버깅 |
| `detail` | 최대 상세 |

### 로그 컨텍스트 (모듈)

| 컨텍스트 | 설명 |
|---------|------|
| `any` | 모든 모듈 (기본 수준 설정) |
| `fabric` | 노드 간 fabric 통신 |
| `heartbeat` | 클러스터 멤버십 |
| `drv_ssd` | SSD/디바이스 드라이버 |
| `migrate` | 데이터 마이그레이션 |
| `namespace` | namespace 동작 |
| `security` | 보안/ACL |
| `audit` | 감사 로그 |

### 동적 로그 수준 변경

```bash
asadm -e 'enable; manage config logging file /var/log/aerospike/aerospike.log param security to detail'
```

---

## 8. security 섹션

빈 `security {}` 블록만으로 ACL 활성화. CE에서도 사용 가능.

### 기본 권한 목록

| 권한 | 설명 |
|------|------|
| `sys-admin` | 시스템 관리 (클러스터 설정 변경) |
| `user-admin` | 사용자/역할 관리 |
| `read` | 데이터 읽기 |
| `read-write` | 데이터 읽기/쓰기 |
| `read-write-udf` | 읽기/쓰기 + UDF 실행 |
| `data-admin` | 인덱스/UDF 관리 |
| `write` | 데이터 쓰기 |

### ACL 규칙

- `security` 섹션 존재 시 반드시 admin 사용자 필요
- admin 사용자는 `sys-admin` + `user-admin` 역할 필수
- Operator CRD에서는 `aerospikeAccessControl`로 관리

---

## 참고 자료

- [Aerospike Configuration Reference](https://aerospike.com/docs/database/reference/config/)
- [Aerospike 8.1.0 Release Notes](https://aerospike.com/docs/database/release/8-1)
- [Aerospike Storage Configuration](https://aerospike.com/docs/database/manage/namespace/storage/config/)
- [Aerospike Network Configuration](https://aerospike.com/docs/database/manage/network/)
- [Aerospike Log Management](https://aerospike.com/docs/database/manage/logging/logs/)
- [Aerospike System Limits](https://aerospike.com/docs/database/reference/limitations)
- [Aerospike CE vs EE](https://aerospike.com/products/features-and-editions/)
- [Aerospike GitHub - Default Config](https://github.com/aerospike/aerospike-server/blob/master/as/etc/aerospike.conf)
- [Aerospike Dynamic Runtime Config](https://aerospike.com/docs/database/tools/runtime-config/)
- [Docker Hub: aerospike](https://hub.docker.com/_/aerospike)
