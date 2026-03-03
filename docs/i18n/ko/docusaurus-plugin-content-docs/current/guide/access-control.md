---
sidebar_position: 4
title: 접근 제어 (ACL)
---

# 접근 제어 (ACL)

이 가이드는 오퍼레이터를 사용하여 Aerospike CE 클러스터의 인증 및 권한 부여를 설정하는 방법을 다룹니다.

## 개요

Aerospike CE는 클러스터에 접속하는 대상과 수행 가능한 작업을 제한하는 접근 제어(ACL)를 지원합니다. ACL을 활성화하면:

- 모든 클라이언트 연결은 사용자명과 비밀번호로 인증해야 합니다.
- 각 사용자는 권한을 정의하는 하나 이상의 **역할(role)** 을 부여받습니다.
- 오퍼레이터는 Aerospike 관리 API를 통해 역할과 사용자 생성을 관리합니다.

:::warning
Aerospike CE 8.x는 `aerospike.conf`의 `security` 스탠자를 지원하지 않습니다. ACL은 오퍼레이터의 `aerospikeAccessControl` 스펙을 통해서만 관리되며, 런타임에 Aerospike 관리 API를 사용하여 사용자와 역할을 구성합니다.
:::

## 사전 준비

### Kubernetes Secret 생성

각 사용자의 비밀번호는 Kubernetes Secret에 저장해야 합니다. Secret에는 `password` 키가 포함되어야 합니다.

```bash
# admin 사용자 Secret 생성
kubectl -n aerospike create secret generic admin-secret \
  --from-literal=password='admin-password-here'

# 애플리케이션 사용자 Secret 생성
kubectl -n aerospike create secret generic app-secret \
  --from-literal=password='app-password-here'
```

## 기본 ACL 설정

최소 ACL 구성은 `sys-admin`과 `user-admin` 역할을 모두 가진 사용자가 최소 한 명 필요합니다.

```yaml
apiVersion: acko.io/v1alpha1
kind: AerospikeCluster
metadata:
  name: aerospike-acl
  namespace: aerospike
spec:
  size: 3
  image: aerospike:ce-8.1.1.1
  aerospikeAccessControl:
    users:
      - name: admin
        secretName: admin-secret
        roles:
          - sys-admin
          - user-admin
      - name: appuser
        secretName: app-secret
        roles:
          - read-write
  aerospikeConfig:
    service:
      cluster-name: aerospike-acl
    namespaces:
      - name: test
        replication-factor: 2
        storage-engine:
          type: memory
```

## 내장 역할

Aerospike CE는 다음과 같은 사전 정의된 역할을 제공합니다. `roles` 목록에 별도로 정의하지 않고 사용자에게 직접 부여할 수 있습니다.

| 역할 | 설명 |
|------|------|
| `user-admin` | 사용자 생성/삭제, 역할 부여/회수 |
| `sys-admin` | 클러스터 관리 (truncate, config, info 명령) |
| `data-admin` | 인덱스 관리 (보조 인덱스 생성/삭제, UDF) |
| `read` | 레코드 읽기 |
| `write` | 레코드 쓰기 (삽입/수정/삭제) |
| `read-write` | 레코드 읽기 및 쓰기 |
| `read-write-udf` | 레코드 읽기, 쓰기, UDF 실행 |
| `truncate` | 네임스페이스/셋 Truncate |

:::info
`superuser` 역할은 Aerospike Enterprise Edition에서만 존재합니다. CE 클러스터는 위의 내장 역할을 사용하거나 커스텀 역할을 정의해야 합니다.
:::

## 커스텀 역할

세밀한 권한으로 커스텀 역할을 정의할 수 있습니다.

```yaml
spec:
  aerospikeAccessControl:
    roles:
      - name: inventory-reader
        privileges:
          - read.inventory           # 'inventory' 네임스페이스 읽기
      - name: orders-writer
        privileges:
          - read-write.orders        # 'orders' 네임스페이스 읽기/쓰기
          - read-write.orders.items  # 'orders.items' 셋 읽기/쓰기
    users:
      - name: admin
        secretName: admin-secret
        roles:
          - sys-admin
          - user-admin
      - name: inventory-svc
        secretName: inventory-secret
        roles:
          - inventory-reader
      - name: orders-svc
        secretName: orders-secret
        roles:
          - orders-writer
```

### 권한 형식

권한은 `<코드>[.<네임스페이스>[.<셋>]]` 형식을 따릅니다.

| 코드 | 설명 |
|------|------|
| `read` | 레코드 읽기 |
| `write` | 레코드 쓰기 |
| `read-write` | 레코드 읽기 및 쓰기 |
| `read-write-udf` | 레코드 읽기, 쓰기, UDF 실행 |
| `sys-admin` | 시스템 관리 |
| `user-admin` | 사용자 관리 |
| `data-admin` | 데이터 관리 |
| `truncate` | 데이터 Truncate |

**예시:**
- `read` — 모든 네임스페이스에 대한 전역 읽기
- `read-write.myns` — `myns` 네임스페이스에 대한 읽기/쓰기
- `write.myns.myset` — `myns` 내 `myset` 셋에 대한 쓰기

## Webhook 검증 규칙

오퍼레이터의 admission webhook은 다음 ACL 규칙을 적용합니다.

1. **Admin 사용자 필수**: 최소 한 명의 사용자가 `sys-admin`과 `user-admin` 역할을 모두 보유해야 합니다. 이 조건이 충족되지 않으면 초기 설정 이후 오퍼레이터가 ACL을 관리할 수 없습니다.

2. **역할 교차 검증**: 사용자가 참조하는 모든 역할은 내장 역할이거나 `roles` 목록에 명시적으로 정의되어 있어야 합니다. 정의되지 않은 역할 참조는 거부됩니다.

3. **권한 코드 검증**: 커스텀 역할의 각 권한은 유효한 권한 코드를 사용해야 합니다. 유효하지 않은 코드는 설명적인 오류와 함께 거부됩니다.

4. **Secret 필수**: 모든 사용자는 비밀번호가 포함된 Kubernetes Secret을 가리키는 `secretName`이 있어야 합니다.

## 문제 해결

### 일반적인 검증 오류

**"must have at least one user with both 'sys-admin' and 'user-admin' roles"**

최소 한 명의 사용자가 두 역할을 모두 보유하고 있는지 확인하세요.

```yaml
users:
  - name: admin
    secretName: admin-secret
    roles:
      - sys-admin
      - user-admin    # 동일한 사용자에게 두 역할 모두 필요
```

**"user X references undefined role Y"**

해당 역할이 내장 역할이거나 `roles` 목록에 정의되어 있어야 합니다.

```yaml
roles:
  - name: custom-role       # 커스텀 역할 정의
    privileges:
      - read.myns
users:
  - name: myuser
    secretName: myuser-secret
    roles:
      - custom-role          # 이제 이 참조가 유효합니다
```

**"role X has invalid privilege code Y"**

유효한 권한 코드(`read`, `write`, `read-write`, `read-write-udf`, `sys-admin`, `user-admin`, `data-admin`, `truncate`)만 사용하세요.

```yaml
roles:
  - name: my-role
    privileges:
      - read-write.myns     # 유효
      # - admin.myns        # 유효하지 않음 — 'sys-admin' 또는 'data-admin' 사용
```

### ACL 상태 확인

배포 후 ACL이 정상 동작하는지 확인합니다.

```bash
# ACL 동기화에 대한 오퍼레이터 로그 확인
kubectl -n aerospike-operator logs -l control-plane=controller-manager | grep -i acl

# 인증 정보로 클러스터에 접속
kubectl -n aerospike exec -it aerospike-acl-0-0 -- asadm -Uadmin -Padmin-password-here

# 사용자 목록 확인 (asadm 내부)
manage acl show users
```
