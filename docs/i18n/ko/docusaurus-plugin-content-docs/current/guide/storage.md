---
sidebar_position: 4
title: 스토리지 설정
---

# 스토리지 설정

Aerospike 클러스터는 네임스페이스 데이터, 보조 인덱스, 로그를 위한 영구 스토리지를 사용합니다. `spec.storage` 필드는 볼륨의 프로비저닝, 초기화, 정리 방법을 제어합니다.

---

## 글로벌 볼륨 정책

글로벌 정책은 개별 볼륨 수준에서 오버라이드되지 않는 한 모든 볼륨에 적용됩니다.

### filesystemVolumePolicy

`volumeMode: Filesystem`(기본값)인 PVC 볼륨에 적용됩니다.

| 필드 | 옵션 | 설명 |
|------|------|------|
| `initMethod` | `none`, `dd`, `deleteFiles`, `blkdiscard` | 처음 사용 시 볼륨 초기화 방법 |
| `wipeMethod` | `none`, `dd`, `deleteFiles`, `blkdiscard`, `blkdiscardWithHeaderCleanup`, `headerCleanup` | 파드 삭제 전 볼륨 초기화 방법 |
| `cascadeDelete` | `true` / `false` | 파드 삭제 시 PVC도 삭제할지 여부 |

### blockVolumePolicy

`volumeMode: Block`인 PVC 볼륨에 적용됩니다.

`filesystemVolumePolicy`와 동일한 필드를 사용합니다. 블록 디바이스에는 `initMethod: blkdiscard`(빠른 하드웨어 가속)와 `wipeMethod: blkdiscardWithHeaderCleanup`을 권장합니다.

### localStorageClasses

"로컬" 스토리지로 인식할 StorageClass 이름 목록 (예: `local-path`, `openebs-hostpath`). 로컬 PVC는 파드 재시작 전에 삭제되어 클린 상태를 보장합니다.

```yaml
storage:
  localStorageClasses:
    - local-path
    - openebs-hostpath
  deleteLocalStorageOnRestart: true
  cleanupThreads: 2
```

---

## 볼륨 타입

### PVC — Filesystem (기본)

블록 기반 파일시스템. 데이터, 로그, WAL 파일에 적합합니다.

```yaml
volumes:
  - name: data-vol
    source:
      persistentVolume:
        storageClass: standard
        size: 50Gi
        volumeMode: Filesystem   # 기본값
    aerospike:
      path: /opt/aerospike/data
```

### PVC — Block Device

파일시스템이 없는 원시 블록 볼륨. 스토리지에 직접 접근하는 Aerospike device 모드 스토리지 엔진에 사용합니다.

```yaml
volumes:
  - name: sindex-vol
    source:
      persistentVolume:
        storageClass: fast-ssd
        size: 20Gi
        volumeMode: Block
    aerospike:
      path: /opt/aerospike/sindex
```

### EmptyDir

임시 파드 내 스토리지. Aerospike 컨테이너와 사이드카(예: exporter) 간 데이터 공유에 유용합니다. 파드 재시작 시 데이터가 소실됩니다.

```yaml
volumes:
  - name: shared-data
    source:
      emptyDir: {}
    aerospike:
      path: /opt/aerospike/shared
      subPath: "aerospike-data"
    sidecars:
      - containerName: aerospike-prometheus-exporter
        path: /shared
        readOnly: true
```

### HostPath

호스트 노드의 디렉토리를 마운트합니다. 개발/테스트 전용 — 데이터가 특정 노드에 종속됩니다.

```yaml
volumes:
  - name: host-logs
    source:
      hostPath:
        path: /var/log/aerospike
        type: DirectoryOrCreate
    aerospike:
      path: /opt/aerospike/logs
```

### CSI 볼륨 (mountPropagation)

`HostToContainer` 전파가 필요한 CSI 드라이버(예: FUSE 기반 스토리지)에 사용합니다.

```yaml
volumes:
  - name: csi-data
    source:
      persistentVolume:
        storageClass: csi-driver
        size: 100Gi
    aerospike:
      path: /opt/aerospike/csi
      mountPropagation: HostToContainer
```

---

## 볼륨별 오버라이드

각 볼륨은 `initMethod`, `wipeMethod`, `cascadeDelete`로 글로벌 정책을 오버라이드할 수 있습니다:

```yaml
storage:
  filesystemVolumePolicy:
    initMethod: deleteFiles   # 글로벌 기본값
  volumes:
    - name: data-vol
      source:
        persistentVolume:
          storageClass: standard
          size: 50Gi
      aerospike:
        path: /opt/aerospike/data
      initMethod: dd          # 오버라이드: 이 볼륨에만 dd 사용
```

우선순위: **볼륨별** > **글로벌 정책** > **오퍼레이터 기본값**

---

## PVC 메타데이터

PVC에 커스텀 레이블과 어노테이션을 추가할 수 있습니다 — 백업 도구, 스토리지 정책, 비용 추적에 유용합니다:

```yaml
volumes:
  - name: data-vol
    source:
      persistentVolume:
        storageClass: standard
        size: 50Gi
        metadata:
          labels:
            app.kubernetes.io/data-tier: "aerospike"
            backup-policy: "daily"
          annotations:
            volume.kubernetes.io/storage-provisioner: "ebs.csi.aws.com"
    aerospike:
      path: /opt/aerospike/data
```

---

## 완전한 예제

다음 예제(`config/samples/aerospike-cluster-storage-advanced.yaml`)는 모든 볼륨 타입과 정책 옵션을 단일 클러스터에서 보여줍니다:

```yaml
apiVersion: acko.io/v1alpha1
kind: AerospikeCluster
metadata:
  name: aerospike-storage-advanced
  namespace: aerospike
spec:
  size: 3
  image: aerospike:ce-8.1.1.1

  storage:
    filesystemVolumePolicy:
      initMethod: deleteFiles
      wipeMethod: deleteFiles
      cascadeDelete: true
    blockVolumePolicy:
      initMethod: blkdiscard
      wipeMethod: blkdiscardWithHeaderCleanup

    localStorageClasses:
      - local-path
      - openebs-hostpath
    deleteLocalStorageOnRestart: true

    cleanupThreads: 2

    volumes:
      - name: data-vol
        source:
          persistentVolume:
            storageClass: standard
            size: 50Gi
            volumeMode: Filesystem
            metadata:
              labels:
                app.kubernetes.io/data-tier: "aerospike"
                backup-policy: "daily"
              annotations:
                volume.kubernetes.io/storage-provisioner: "ebs.csi.aws.com"
        aerospike:
          path: /opt/aerospike/data
        initMethod: dd

      - name: sindex-vol
        source:
          persistentVolume:
            storageClass: fast-ssd
            size: 20Gi
            volumeMode: Block
        aerospike:
          path: /opt/aerospike/sindex

      - name: shared-data
        source:
          emptyDir: {}
        aerospike:
          path: /opt/aerospike/shared
          subPath: "aerospike-data"
        sidecars:
          - containerName: aerospike-prometheus-exporter
            path: /shared
            readOnly: true

      - name: host-logs
        source:
          hostPath:
            path: /var/log/aerospike
            type: DirectoryOrCreate
        aerospike:
          path: /opt/aerospike/logs

      - name: csi-data
        source:
          persistentVolume:
            storageClass: csi-driver
            size: 100Gi
        aerospike:
          path: /opt/aerospike/csi
          mountPropagation: HostToContainer

      - name: local-data
        source:
          persistentVolume:
            storageClass: local-path
            size: 10Gi
        aerospike:
          path: /opt/aerospike/local
        initMethod: deleteFiles
        wipeMethod: deleteFiles

  aerospikeConfig:
    service:
      cluster-name: storage-advanced-demo
    namespaces:
      - name: test
        memory-size: 1073741824
        replication-factor: 2
        storage-engine:
          type: device
          devices:
            - /opt/aerospike/data/test.dat
          filesize: 4294967296
          data-in-memory: true
```

적용:

```bash
kubectl apply -f config/samples/aerospike-cluster-storage-advanced.yaml
```
