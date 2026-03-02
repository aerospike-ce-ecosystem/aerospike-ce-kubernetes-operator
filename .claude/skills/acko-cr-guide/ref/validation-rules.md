# Validation Rules 전체 참조

## Validation Errors (거부 규칙)

### 크기/이미지

| 규칙 | 에러 메시지 |
|------|------------|
| `spec.size > 8` | `"spec.size N exceeds CE maximum of 8"` |
| `spec.size == 0` + templateRef 없음 | `"spec.size must be set (1–8) when spec.templateRef is not specified"` |
| `spec.image` 비어있음 + templateRef 없음 | `"spec.image must not be empty when spec.templateRef is not specified"` |
| 이미지명에 `enterprise`/`ee-`/`ent-` | `"spec.image \"...\" is an Enterprise Edition image; only Community Edition images are allowed"` |

### Aerospike Config

| 규칙 | 에러 메시지 |
|------|------------|
| `xdr` 섹션 존재 | `"aerospikeConfig must not contain 'xdr' section (XDR is Enterprise-only)"` |
| `tls` 섹션 존재 | `"aerospikeConfig must not contain 'tls' section (TLS is Enterprise-only)"` |
| namespaces > 2 | `"aerospikeConfig.namespaces count N exceeds CE maximum of 2"` |
| `heartbeat.mode != "mesh"` | `"aerospikeConfig.network.heartbeat.mode must be 'mesh' for CE"` |

### Enterprise 전용 Namespace 키 (10개)

`compression`, `compression-level`, `durable-delete`, `fast-restart`, `index-type`, `sindex-type`, `rack-id`, `strong-consistency`, `tomb-raider-eligible-age`, `tomb-raider-period`

에러: `"namespace[N] \"name\": 'key' is not allowed (reason)"`

### Enterprise 전용 Security 키 (4개)

`tls`, `ldap`, `log`, `syslog` — CE 허용 키: `enable-security`, `default-password-file`

에러: `"aerospikeConfig.security.KEY is not allowed in CE edition (reason)"`

### Namespace 검증

| 규칙 | 에러 메시지 |
|------|------------|
| replication-factor < 1 또는 > 4 | `"namespace[N] \"name\": replication-factor must be between 1 and 4"` |
| replication-factor > spec.size | `"namespace \"name\": replication-factor N exceeds cluster size M"` |

### ACL 검증

| 규칙 | 에러 메시지 |
|------|------------|
| admin 유저 없음 (sys-admin + user-admin) | `"aerospikeAccessControl must have at least one user with both 'sys-admin' and 'user-admin' roles"` |
| secretName 비어있음 | `"user \"name\" must have a secretName for password"` |
| 중복 유저 이름 | `"accessControl.users: duplicate user name \"name\""` |
| 중복 역할 이름 | `"accessControl.roles: duplicate role name \"name\""` |
| 정의되지 않은 역할 참조 | `"user \"name\" references undefined role \"role\""` |
| 유효하지 않은 권한 코드 | `"role \"name\" has invalid privilege code \"code\""` |
| 권한 앞뒤 공백 | `"role \"name\" privileges[N]: privilege string \"...\" must not have leading or trailing whitespace"` |

유효 권한: `read`, `write`, `read-write`, `read-write-udf`, `sys-admin`, `user-admin`, `data-admin`, `truncate`
형식: `"<code>"` / `"<code>.<namespace>"` / `"<code>.<namespace>.<set>"`

### Rack Config 검증

| 규칙 | 에러 메시지 |
|------|------------|
| Rack ID ≤ 0 | `"rack ID must be > 0 (rack ID 0 is reserved)"` |
| 중복 Rack ID | `"duplicate rack ID N"` |
| 중복 rackLabel | `"duplicate rackLabel \"label\""` |
| 중복 nodeName | `"racks[N] and racks[M] both constrained to node \"name\""` |
| 유효하지 않은 IntOrString | `"rackConfig.scaleDownBatchSize must be a positive integer or percentage"` |
| Update: Rack ID 변경 | `"rackConfig rack IDs cannot be changed"` |

### Storage 검증

| 규칙 | 에러 메시지 |
|------|------------|
| 중복 볼륨 이름 | `"storage.volumes: duplicate volume name \"name\""` |
| 볼륨 소스 0개 또는 2개+ | `"exactly one volume source must be specified"` |
| PV size 비어있음/무효/음수 | `"persistentVolume.size must not be empty"` / `"is not a valid Kubernetes quantity"` |
| path 절대경로 아님 | `"aerospike.path must be an absolute path"` |
| subPath + subPathExpr 동시 | `"subPath and subPathExpr are mutually exclusive"` |
| deleteLocalStorageOnRestart + localStorageClasses 비어있음 | `"deleteLocalStorageOnRestart is true but localStorageClasses is empty"` |

### Monitoring 검증

| 규칙 | 에러 메시지 |
|------|------------|
| port 범위 벗어남 | `"monitoring.port must be in range 1-65535"` |
| port가 3000-3003 충돌 | `"monitoring.port N conflicts with Aerospike service port"` |
| exporterImage 비어있음 | `"monitoring.exporterImage must not be empty when monitoring is enabled"` |
| metricLabels에 `=` 또는 `,` | `"monitoring.metricLabels key/value must not contain '=' or ','"` |
| customRules name/rules 누락 | `"customRules[N]: missing required field 'name'/'rules'"` |

### Operations 검증

| 규칙 | 에러 메시지 |
|------|------------|
| Operation 2개+ | `"only one operation can be specified at a time"` |
| ID 길이 1-20자 벗어남 | `"operation id must be 1-20 characters"` |
| InProgress 중 변경 | `"cannot change operations while operation \"ID\" is InProgress"` |

### Update 전용

| 규칙 | 에러 메시지 |
|------|------------|
| overrides + templateRef 없음 | `"spec.overrides can only be set when spec.templateRef is specified"` |

---

## Validation Warnings (비차단 경고)

| 경고 조건 | 메시지 요약 |
|-----------|-----------|
| image 태그 없음 / `latest` | 명시적 버전 태그 권장 |
| exporter image `latest` / 태그 없음 | 명시적 버전 태그 권장 |
| `data-in-memory=true` | 메모리 사용량 2배 경고 |
| `rollingUpdateBatchSize > spec.size` | 모든 파드 동시 재시작 가능 |
| `maxUnavailable >= spec.size` / `100%` | PDB가 보호 기능 상실 |
| hostPath 볼륨 | 프로덕션 비권장, 노드 종속 |
| 비PV에 cascadeDelete | 효과 없음 |
| work-directory PV 없음 | 재시작 시 데이터 손실 가능 |
| hostNetwork + multiPodPerHost | 포트 충돌 가능 |
| hostNetwork + dnsPolicy 불일치 | DNS 해석 문제 가능 |
| serviceMonitor.enabled + monitoring.disabled | ServiceMonitor 생성 안됨 |
| prometheusRule.enabled + monitoring.disabled | PrometheusRule 생성 안됨 |
| localStorageClasses + deleteLocalStorageOnRestart 미설정 | 로컬 PVC 재시작 시 미삭제 |
