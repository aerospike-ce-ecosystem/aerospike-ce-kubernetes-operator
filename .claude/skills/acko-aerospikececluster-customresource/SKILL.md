---
name: acko-aerospikececluster-customresource
description: Generate a full AerospikeCECluster CR YAML example with comprehensive comments explaining every field
disable-model-invocation: false
---

# AerospikeCECluster Custom Resource Guide

Generate a complete, annotated AerospikeCECluster CR YAML with all available fields documented.

## Output

Print the full example CR YAML below to the user. This is the comprehensive reference for all CRD fields.

```yaml
##############################################################################
# AerospikeCECluster - Full Configuration Reference
# API Group: acko.io/v1alpha1
# Short names: asce, ascecluster
#
# CE Constraints:
#   - Max 8 nodes per cluster
#   - Max 2 namespaces
#   - No XDR, TLS, Strong Consistency, All Flash
#   - Image must be community edition (e.g., aerospike:ce-8.1.1.1)
##############################################################################
apiVersion: acko.io/v1alpha1
kind: AerospikeCECluster
metadata:
  name: aerospike-ce-full-example
  namespace: aerospike
spec:
  ##########################################################################
  # [REQUIRED] size: Number of Aerospike pods (min: 1, max: 8 for CE)
  ##########################################################################
  size: 3

  ##########################################################################
  # [REQUIRED] image: Aerospike CE container image
  # Available images: https://hub.docker.com/_/aerospike
  # Examples: aerospike:ce-8.1.1.1, aerospike:ce-7.2.0.6
  ##########################################################################
  image: aerospike:ce-8.1.1.1

  ##########################################################################
  # [OPTIONAL] paused: Stop reconciliation when true
  # Useful for maintenance or troubleshooting
  ##########################################################################
  # paused: false

  ##########################################################################
  # [OPTIONAL] enableDynamicConfigUpdate: Enable runtime config changes
  # without pod restart using Aerospike's set-config command
  ##########################################################################
  # enableDynamicConfigUpdate: true

  ##########################################################################
  # [OPTIONAL] rollingUpdateBatchSize: Pods to restart in parallel
  # during a rolling restart. Defaults to 1 (sequential).
  # Minimum: 1
  ##########################################################################
  # rollingUpdateBatchSize: 1

  ##########################################################################
  # [OPTIONAL] disablePDB: Disable PodDisruptionBudget creation
  ##########################################################################
  # disablePDB: false

  ##########################################################################
  # [OPTIONAL] maxUnavailable: Max pods unavailable during disruption
  # Used for PodDisruptionBudget. Can be integer or percentage string.
  # Default: 1
  ##########################################################################
  # maxUnavailable: 1

  ##########################################################################
  # [OPTIONAL] enableRackIDOverride: Enable dynamic rack ID assignment
  # via pod annotations. When true, the operator reads rack IDs from
  # pod annotations instead of static rack config.
  ##########################################################################
  # enableRackIDOverride: false

  ##########################################################################
  # [OPTIONAL] k8sNodeBlockList: Nodes that should NOT run Aerospike pods
  ##########################################################################
  # k8sNodeBlockList:
  #   - node-maintenance-1
  #   - node-maintenance-2

  ##########################################################################
  # [OPTIONAL] validationPolicy: Control validation behavior
  ##########################################################################
  # validationPolicy:
  #   skipWorkDirValidate: false    # Skip work directory persistence check

  ##########################################################################
  # [OPTIONAL] operations: On-demand operations
  # Only one operation can be active at a time (maxItems: 1).
  #
  # Kinds:
  #   - WarmRestart: Sends SIGUSR1 to the Aerospike process
  #   - PodRestart: Deletes and recreates the pod
  #
  # ID: Unique tracking identifier (1-20 chars)
  # podList: Optional list of specific pods to target (empty = all pods)
  ##########################################################################
  # operations:
  #   - kind: WarmRestart
  #     id: "restart-001"
  #     podList:                      # Optional: omit to target all pods
  #       - aerospike-ce-full-example-0-0
  #       - aerospike-ce-full-example-0-1

  ##########################################################################
  # [OPTIONAL] headlessService: Custom metadata for the headless service
  ##########################################################################
  # headlessService:
  #   metadata:
  #     annotations:
  #       example.com/annotation: value
  #     labels:
  #       example.com/label: value

  ##########################################################################
  # [OPTIONAL] podService: Per-pod individual service creation
  # When set, the operator creates an individual Service for each pod.
  ##########################################################################
  # podService:
  #   metadata:
  #     annotations:
  #       example.com/annotation: value
  #     labels:
  #       example.com/label: value

  ##########################################################################
  # aerospikeConfig: Raw Aerospike configuration (converted to aerospike.conf)
  #
  # The operator converts this YAML map into aerospike.conf format.
  # Special handling:
  #   - namespaces: array of namespace objects (each must have "name")
  #   - storage-engine: requires "type" field (memory/device)
  #   - network.heartbeat: mesh-seed-address-port is auto-injected
  #   - logging: array of log file configs (each must have "name")
  #   - security: empty object enables ACL (requires aerospikeAccessControl)
  ##########################################################################
  aerospikeConfig:
    # -- service stanza --
    service:
      cluster-name: aerospike-ce-full-example
      proto-fd-max: 15000         # Max client file descriptors (default: 15000)

    # -- network stanza --
    # Ports 3000 (service), 3001 (fabric), 3002 (heartbeat) are defaults
    network:
      service:
        address: any
        port: 3000                # Client service port
      heartbeat:
        mode: mesh                # mesh mode (auto-configured by operator)
        port: 3002                # Heartbeat port
      fabric:
        address: any
        port: 3001                # Inter-node fabric port

    # -- security stanza --
    # Uncomment to enable ACL (requires aerospikeAccessControl section)
    # security: {}

    # -- namespace stanza --
    # CE limit: max 2 namespaces
    namespaces:
      # Memory storage namespace
      - name: test
        replication-factor: 2
        storage-engine:
          type: memory            # "memory" or "device"
          data-size: 1073741824   # 1GB in bytes

      # Device/file storage namespace (uncomment to use)
      # - name: persistent-ns
      #   replication-factor: 2
      #   storage-engine:
      #     type: device
      #     file: /opt/aerospike/data/persistent-ns.dat
      #     filesize: 4294967296    # 4GB in bytes
      #     data-in-memory: true    # Cache data in memory

    # -- logging stanza --
    logging:
      - name: /var/log/aerospike/aerospike.log
        context: any info         # Log level: detail, debug, info, warning, critical

  ##########################################################################
  # storage: Volume definitions for Aerospike pods
  #
  # Volume sources:
  #   - persistentVolume: Creates PVC (for data persistence)
  #   - emptyDir: Ephemeral storage (lost on pod restart)
  #   - secret: Mount a Kubernetes Secret
  #   - configMap: Mount a Kubernetes ConfigMap
  #   - hostPath: Mount a path from the host node
  #
  # cascadeDelete: Delete PVC when CR is deleted (default: false)
  # initMethod: Volume init on first use
  #   - none (default), deleteFiles, dd, blkdiscard, headerCleanup
  # wipeMethod: Volume wipe when marked dirty
  #   - none, deleteFiles, dd, blkdiscard, headerCleanup,
  #     blkdiscardWithHeaderCleanup
  ##########################################################################
  storage:
    # cleanupThreads: Max threads for volume cleanup/init (default: 1)
    # cleanupThreads: 1

    # filesystemVolumePolicy: Default policy for filesystem-mode PVs
    # Per-volume settings override this policy.
    # filesystemVolumePolicy:
    #   initMethod: deleteFiles
    #   wipeMethod: deleteFiles
    #   cascadeDelete: true

    # blockVolumePolicy: Default policy for block-mode PVs
    # Per-volume settings override this policy.
    # blockVolumePolicy:
    #   initMethod: blkdiscard
    #   wipeMethod: blkdiscard
    #   cascadeDelete: true

    # localStorageClasses: StorageClasses that use local storage
    # (e.g., local-path, openebs-hostpath). Requires special handling on pod restart.
    # localStorageClasses:
    #   - local-path
    #   - openebs-hostpath

    # deleteLocalStorageOnRestart: Delete PVCs backed by local storage
    # classes before pod restart, forcing re-provisioning on the new node.
    # deleteLocalStorageOnRestart: false

    volumes:
      # Persistent data volume (PVC)
      - name: data-vol
        source:
          persistentVolume:
            storageClass: standard    # Your StorageClass name
            size: 10Gi
            volumeMode: Filesystem    # Filesystem (default) or Block
            # accessModes:
            #   - ReadWriteOnce
            # selector:               # Label selector for binding to specific PV
            #   matchLabels:
            #     type: aerospike-data
            # metadata:               # Custom labels/annotations for PVC
            #   labels:
            #     app: aerospike
            #   annotations:
            #     volume.beta.kubernetes.io/storage-class: standard
        aerospike:
          path: /opt/aerospike/data   # Mount path in Aerospike container
          # readOnly: false
          # subPath: ""               # Mount only a sub-path of the volume
          # subPathExpr: ""           # Expanded path using env vars (mutually exclusive with subPath)
          # mountPropagation: None    # None, HostToContainer, Bidirectional
        cascadeDelete: true           # Delete PVC on CR deletion
        # initMethod: none            # Per-volume init method override
        # wipeMethod: none            # Per-volume wipe method override

      # Working directory (emptyDir)
      - name: workdir
        source:
          emptyDir: {}
        aerospike:
          path: /opt/aerospike/work

      # HostPath volume example (uncomment to use)
      # - name: host-data
      #   source:
      #     hostPath:
      #       path: /mnt/aerospike-data
      #       type: DirectoryOrCreate   # DirectoryOrCreate, Directory, File, etc.
      #   aerospike:
      #     path: /opt/aerospike/host-data

      # Mount to sidecar and init containers (uncomment to use)
      # - name: shared-logs
      #   source:
      #     emptyDir: {}
      #   aerospike:
      #     path: /var/log/aerospike
      #   sidecars:
      #     - containerName: log-collector
      #       path: /logs
      #       # readOnly: false
      #       # subPath: ""
      #       # subPathExpr: ""
      #       # mountPropagation: None
      #   initContainers:
      #     - containerName: custom-init
      #       path: /init-logs

  ##########################################################################
  # podSpec: Pod-level customization
  ##########################################################################
  podSpec:
    # -- Main Aerospike container settings --
    aerospikeContainer:
      resources:
        requests:
          memory: "2Gi"
          cpu: "1"
        limits:
          memory: "4Gi"
          cpu: "2"
      # securityContext:
      #   readOnlyRootFilesystem: true

    # -- Sidecar containers --
    # sidecars:
    #   - name: log-collector
    #     image: busybox:latest
    #     command: ["sh", "-c", "tail -f /logs/aerospike.log"]
    #     volumeMounts:
    #       - name: shared-logs
    #         mountPath: /logs

    # -- Additional init containers (run after operator's built-in init) --
    # initContainers:
    #   - name: custom-init
    #     image: busybox:latest
    #     command: ["sh", "-c", "echo initializing"]

    # -- Image pull secrets --
    # imagePullSecrets:
    #   - name: my-registry-secret

    # -- Scheduling constraints --
    # nodeSelector:
    #   kubernetes.io/arch: amd64
    #   node-type: aerospike

    # tolerations:
    #   - key: "dedicated"
    #     operator: "Equal"
    #     value: "aerospike"
    #     effect: "NoSchedule"

    # affinity:
    #   podAntiAffinity:
    #     requiredDuringSchedulingIgnoredDuringExecution:
    #       - labelSelector:
    #           matchLabels:
    #             app.kubernetes.io/name: aerospike-ce-full-example
    #         topologyKey: kubernetes.io/hostname

    # -- multiPodPerHost: Allow multiple Aerospike pods on same node --
    # When false (or nil with hostNetwork=true), anti-affinity is auto-injected
    # multiPodPerHost: false

    # -- Host networking --
    # hostNetwork: false

    # -- Pod security context --
    # securityContext:
    #   fsGroup: 1000
    #   runAsUser: 1000

    # -- Service account --
    # serviceAccountName: aerospike-sa

    # -- DNS policy --
    # dnsPolicy: ClusterFirst

    # -- Termination grace period --
    # terminationGracePeriodSeconds: 30

    # -- Additional pod metadata --
    # metadata:
    #   labels:
    #     team: platform
    #   annotations:
    #     prometheus.io/scrape: "true"

  ##########################################################################
  # rackConfig: Rack-aware deployment
  #
  # Each rack gets its own StatefulSet (<cluster-name>-<rackID>).
  # Pods are distributed evenly across racks.
  # Each rack can override aerospikeConfig, storage, and podSpec.
  ##########################################################################
  # rackConfig:
  #   # namespaces: Aerospike namespace names that are rack-aware
  #   namespaces:
  #     - test
  #
  #   # scaleDownBatchSize: Pods to scale down simultaneously per rack
  #   # Can be absolute number or percentage string (e.g., "25%"). Default: 1
  #   # scaleDownBatchSize: 1
  #
  #   # maxIgnorablePods: Max pending/failed pods to ignore during reconcile
  #   # Useful when pods are stuck due to scheduling issues.
  #   # Can be absolute number or percentage string.
  #   # maxIgnorablePods: 0
  #
  #   # rollingUpdateBatchSize: Pods to restart simultaneously per rack
  #   # Takes precedence over spec.rollingUpdateBatchSize when set.
  #   # Can be absolute number or percentage string (e.g., "25%"). Default: 1
  #   # rollingUpdateBatchSize: 1
  #
  #   racks:
  #     - id: 1
  #       zone: us-east-1a          # topology.kubernetes.io/zone
  #       # region: us-east-1       # topology.kubernetes.io/region
  #       # nodeName: specific-node # Pin to specific node
  #       # rackLabel: rack-a       # Custom label: acko.io/rack=rack-a
  #       # revision: "v1"          # Version identifier for controlled migrations
  #       # aerospikeConfig:        # Override cluster-level config
  #       #   namespaces:
  #       #     - name: test
  #       #       replication-factor: 2
  #       # storage:                # Override cluster-level storage
  #       # podSpec:                # Override scheduling (affinity, tolerations, nodeSelector)
  #       #   affinity: ...
  #       #   tolerations: [...]
  #       #   nodeSelector:
  #       #     node-type: aerospike
  #     - id: 2
  #       zone: us-east-1b
  #     - id: 3
  #       zone: us-east-1c

  ##########################################################################
  # aerospikeAccessControl: ACL configuration (requires security: {} in config)
  #
  # Requires:
  #   - aerospikeConfig.security: {} must be set
  #   - An admin user with sys-admin and user-admin roles is REQUIRED
  #   - Passwords stored in K8s Secrets with "password" key
  ##########################################################################
  # aerospikeAccessControl:
  #   adminPolicy:
  #     timeout: 2000               # Admin operation timeout in ms
  #   roles:
  #     - name: readwrite-role
  #       privileges:
  #         - read-write
  #       # whitelist:              # Allowed CIDR ranges
  #       #   - 10.0.0.0/8
  #     - name: readonly-role
  #       privileges:
  #         - read
  #   users:
  #     - name: admin               # Admin user (REQUIRED when ACL enabled)
  #       secretName: aerospike-admin-secret
  #       roles:
  #         - sys-admin
  #         - user-admin
  #     - name: app-user
  #       secretName: aerospike-appuser-secret
  #       roles:
  #         - readwrite-role

  ##########################################################################
  # aerospikeNetworkPolicy: Client access network configuration
  #
  # Access types:
  #   - pod (default): Clients use pod IP (cluster-internal)
  #   - hostInternal: Clients use node IP (same network)
  #   - hostExternal: Clients use node IP (external access)
  #   - configuredIP: Use custom network names
  ##########################################################################
  # aerospikeNetworkPolicy:
  #   accessType: pod
  #   alternateAccessType: pod
  #   fabricType: pod
  #   # customAccessNetworkNames:
  #   #   - my-custom-network
  #   # customAlternateAccessNetworkNames:
  #   #   - my-alt-network
  #   # customFabricNetworkNames:
  #   #   - my-fabric-network

  ##########################################################################
  # monitoring: Prometheus exporter sidecar and alerting
  ##########################################################################
  # monitoring:
  #   enabled: true
  #   exporterImage: aerospike/aerospike-prometheus-exporter:v1.16.1
  #   port: 9145                    # Metrics port
  #   resources:
  #     requests:
  #       memory: "64Mi"
  #       cpu: "50m"
  #     limits:
  #       memory: "128Mi"
  #       cpu: "100m"
  #
  #   # env: Additional environment variables for the exporter container
  #   # Useful for metric filtering, custom configuration, etc.
  #   # env:
  #   #   - name: AS_EXPORTER_LOG_LEVEL
  #   #     value: "debug"
  #   #   - name: AS_EXPORTER_NAMESPACE_METRICS_ALLOWLIST
  #   #     value: "client_read_success,client_write_success"
  #
  #   # metricLabels: Custom labels added to all exported metrics
  #   # Passed via METRIC_LABELS env var as sorted key=value pairs
  #   # metricLabels:
  #   #   cluster: production
  #   #   team: platform
  #
  #   serviceMonitor:
  #     enabled: true               # Create ServiceMonitor for Prometheus Operator
  #     interval: "30s"             # Scrape interval
  #     labels:
  #       release: prometheus       # Label for Prometheus discovery
  #
  #   # prometheusRule: Create PrometheusRule for Aerospike cluster alerts
  #   # When customRules is empty, built-in alerts are generated:
  #   #   NodeDown, StopWrites, HighDiskUsage, HighMemoryUsage
  #   # prometheusRule:
  #   #   enabled: true
  #   #   labels:
  #   #     release: prometheus
  #   #   # customRules: Completely replaces default alerts when provided
  #   #   # Each entry must be a complete Prometheus rule group object.
  #   #   # customRules:
  #   #   #   - name: aerospike-custom-alerts
  #   #   #     rules:
  #   #   #       - alert: AerospikeHighLatency
  #   #   #         expr: aerospike_latencies_read_ms_bucket{le="1"} < 0.9
  #   #   #         for: 5m
  #   #   #         labels:
  #   #   #           severity: warning
  #   #   #         annotations:
  #   #   #           summary: "High read latency on {{ $labels.instance }}"

  ##########################################################################
  # networkPolicyConfig: Automatic NetworkPolicy creation
  ##########################################################################
  # networkPolicyConfig:
  #   enabled: true
  #   type: kubernetes              # "kubernetes" or "cilium"

  ##########################################################################
  # bandwidthConfig: CNI traffic shaping annotations
  ##########################################################################
  # bandwidthConfig:
  #   ingress: "1Gbps"
  #   egress: "500Mbps"

  ##########################################################################
  # seedsFinderServices: LoadBalancer for external seed discovery
  ##########################################################################
  # seedsFinderServices:
  #   loadBalancer:
  #     port: 3000
  #     targetPort: 3000
  #     externalTrafficPolicy: Local
  #     annotations:
  #       service.beta.kubernetes.io/aws-load-balancer-type: nlb
  #     # labels: {}
  #     # loadBalancerSourceRanges:
  #     #   - 10.0.0.0/8

---
# [OPTIONAL] Secrets for ACL (required when aerospikeAccessControl is used)
# apiVersion: v1
# kind: Secret
# metadata:
#   name: aerospike-admin-secret
#   namespace: aerospike
# type: Opaque
# data:
#   password: YWRtaW4xMjM=        # base64 encoded password
# ---
# apiVersion: v1
# kind: Secret
# metadata:
#   name: aerospike-appuser-secret
#   namespace: aerospike
# type: Opaque
# data:
#   password: YXBwdXNlcjEyMw==    # base64 encoded password
```

## Usage

```bash
# Create namespace
kubectl create namespace aerospike

# Apply the CR
kubectl apply -f aerospike-ce-full-example.yaml

# Check status
kubectl get asce -n aerospike
kubectl get pods -n aerospike

# View cluster details
kubectl describe asce -n aerospike aerospike-ce-full-example

# Check aerospike logs
kubectl logs -n aerospike <pod-name> -c aerospike-server

# Trigger an on-demand operation (e.g., warm restart)
kubectl patch asce -n aerospike aerospike-ce-full-example --type=merge -p '
  {"spec":{"operations":[{"kind":"WarmRestart","id":"restart-001"}]}}'

# Check operation status
kubectl get asce -n aerospike aerospike-ce-full-example -o jsonpath='{.status.operationStatus}'

# Delete cluster
kubectl delete asce -n aerospike aerospike-ce-full-example
```
