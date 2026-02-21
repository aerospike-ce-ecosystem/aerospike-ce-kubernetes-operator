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
  # [OPTIONAL] k8sNodeBlockList: Nodes that should NOT run Aerospike pods
  ##########################################################################
  # k8sNodeBlockList:
  #   - node-maintenance-1
  #   - node-maintenance-2

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
  #
  # cascadeDelete: Delete PVC when CR is deleted (default: false)
  # initMethod: Volume init on first use
  #   - none (default), deleteFiles, dd, blkdiscard, headerCleanup
  ##########################################################################
  storage:
    # cleanupThreads: Max threads for volume cleanup/init (default: 1)
    # cleanupThreads: 1
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
        aerospike:
          path: /opt/aerospike/data   # Mount path in Aerospike container
        cascadeDelete: true           # Delete PVC on CR deletion
        # initMethod: none

      # Working directory (emptyDir)
      - name: workdir
        source:
          emptyDir: {}
        aerospike:
          path: /opt/aerospike/work

      # Mount sidecar volumes example (uncomment to use)
      # - name: shared-logs
      #   source:
      #     emptyDir: {}
      #   aerospike:
      #     path: /var/log/aerospike
      #   sidecars:
      #     - containerName: log-collector
      #       path: /logs

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
  #   racks:
  #     - id: 1
  #       zone: us-east-1a          # topology.kubernetes.io/zone
  #       # region: us-east-1       # topology.kubernetes.io/region
  #       # nodeName: specific-node # Pin to specific node
  #       # aerospikeConfig:        # Override cluster-level config
  #       #   namespaces:
  #       #     - name: test
  #       #       replication-factor: 2
  #       # storage:                # Override cluster-level storage
  #       # podSpec:                # Override scheduling (affinity, tolerations, nodeSelector)
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
  # monitoring: Prometheus exporter sidecar
  ##########################################################################
  # monitoring:
  #   enabled: true
  #   exporterImage: aerospike/aerospike-prometheus-exporter:latest
  #   port: 9145                    # Metrics port
  #   resources:
  #     requests:
  #       memory: "64Mi"
  #       cpu: "50m"
  #     limits:
  #       memory: "128Mi"
  #       cpu: "100m"
  #   serviceMonitor:
  #     enabled: true               # Create ServiceMonitor for Prometheus Operator
  #     interval: "30s"             # Scrape interval
  #     labels:
  #       release: prometheus       # Label for Prometheus discovery

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

# Delete cluster
kubectl delete asce -n aerospike aerospike-ce-full-example
```
