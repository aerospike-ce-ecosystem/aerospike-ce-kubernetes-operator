---
name: aerospike-ce-8-guide
description: Aerospike CE 8.1 parameter reference — 7.x→8.1 breaking changes, version-specific defaults, dynamic config commands
disable-model-invocation: false
---

# Aerospike CE 8.1 Parameter Reference

CE 8.1에서 변경된 파라미터, 버전별 기본값 차이, 동적 변경 명령 등 Claude가 모를 수 있는 버전 고유 정보만 제공합니다.
일반적인 Aerospike 개념(CE/EE 차이, 설정 문법, 로그 수준 등)은 Claude가 이미 알고 있으므로 생략합니다.

## Output

아래 내용을 마크다운으로 출력하세요.

---

# Aerospike CE 8.1 — 버전 고유 레퍼런스

> 대상: `aerospike:ce-8.1.1.1` / 설정 파일: `/etc/aerospike/aerospike.conf`

---

## 7.x → 8.1 Breaking Changes

| 항목 | 7.x (구) | 8.1 (신) | 시점 |
|------|---------|---------|------|
| info port | `info { port 3003 }` | **제거**. `admin { port 3008 }` | 8.1.0 |
| 메모리 데이터 | `memory-size 4G` | `storage-engine memory { data-size 4G }` | 7.0 dep → 7.1 제거 |
| 쓰기 블록 | `write-block-size 128K` | `flush-size 128K` (내부 write-block 8M 고정) | 7.1 |
| 최대 레코드 | 기본값 = write-block-size | **기본값 1M**, 최대 8M | 7.1 |
| MRT write-block | 없음 | data-size 계산에 영향 | 8.0 |
| 인덱스 메모리 | `memory-size` (device 모드) | `indexes-memory-budget` | 7.0 dep |
| stop-writes | `stop-writes-pct` | `stop-writes-sys-memory-pct` | 7.0 제거 |
| eviction | `high-water-memory-pct` | `evict-used-pct`, `evict-tenths-pct` | 7.0 제거 |
| data-in-memory | namespace 레벨 | storage-engine device 블록 내 | 7.0 이동 |

```diff
- namespace cache { memory-size 4G; storage-engine memory }
+ namespace cache { storage-engine memory { data-size 4G } }

- namespace data { storage-engine device { write-block-size 128K } }
+ namespace data { storage-engine device { flush-size 128K } }

- network { info { port 3003 } }
+ network { admin { port 3008 } }
```

> `info {}` 블록이 설정에 남아있으면 **파싱 오류로 서버 시작 실패**.
> `data-size` 최소 512MiB (8 stripes * 8 write-blocks * 8MiB).

---

## 네트워크 포트 (8.1)

| 서브스탠자 | 포트 | 비고 |
|-----------|------|------|
| service | 3000 | |
| fabric | 3001 | |
| heartbeat | 3002 | |
| **admin** | **3008** | **8.1 신규** (info 3003 대체) |

---

## 8.1 기본값이 변경/추가된 파라미터

| 파라미터 | 기본값 | 동적변경 | 비고 |
|---------|--------|---------|------|
| `max-record-size` | **1M** | Yes | 7.1+ 신규 기본값. 최대 8M |
| `flush-size` | **1M** | No | 7.1+ write-block-size 대체. NVMe: 128K 권장 |
| `data-size` | - | No | 7.0+ memory-size 대체. memory 블록 내 지정 |
| `stop-writes-sys-memory-pct` | **90** | Yes | 7.0+ stop-writes-pct 대체 |
| `evict-used-pct` | **70** | Yes | device storage 사용률 eviction |
| `evict-tenths-pct` | **5** | Yes | eviction 사이클당 비율 (0.5%) |
| `cluster-name` | (없음) | No | **8.0+ 필수 권장** |
| `nsup-period` | **0** | Yes | 0이고 default-ttl!=0이면 **시작 실패** |

---

## 동적 설정 변경 명령

```bash
# asinfo
asinfo -v "set-config:context=service;batch-index-threads=16"
asinfo -v "set-config:context=namespace;id=ns1;max-record-size=256K"

# asadm
asadm -e 'enable; manage config service param proto-fd-max to 100000'
asadm -e 'enable; manage config logging file /var/log/aerospike/aerospike.log param security to detail'
```
