---
sidebar_position: 5
title: Cluster Templates
---

# Cluster Templates

`AerospikeClusterTemplate` lets you define reusable configuration profiles for Aerospike clusters. Instead of repeating the same scheduling, storage, and Aerospike configuration across every cluster, you define it once in a template and reference it from multiple clusters.

---

## When to use templates

- **Multiple environments** — define dev/stage/prod templates with different resource sizes and anti-affinity levels
- **Organizational defaults** — enforce storage class, tolerations, and heartbeat settings across teams
- **Configuration standardization** — prevent configuration drift by keeping shared settings in one place

---

## Create a template

Templates can now supply the container **image**, cluster **size**, **monitoring** sidecar, and **network policy** as defaults — in addition to the existing scheduling, storage, and Aerospike configuration fields.

```yaml
apiVersion: acko.io/v1alpha1
kind: AerospikeClusterTemplate
metadata:
  name: hard-rack
  namespace: default
spec:
  # Standardize the Aerospike image and default cluster size across all hard-rack clusters
  image: aerospike:ce-8.1.1.1
  size: 6

  # Enable Prometheus monitoring sidecar by default
  monitoring:
    enabled: true
    port: 9145
    resources:
      requests:
        cpu: 50m
        memory: 64Mi
      limits:
        cpu: 200m
        memory: 128Mi
    serviceMonitor:
      enabled: true
      interval: 30s

  # Default network access policy
  aerospikeNetworkPolicy:
    accessType: pod
    alternateAccessType: pod
    fabricType: pod

  scheduling:
    podAntiAffinityLevel: required   # one Aerospike pod per node
    tolerations:
      - key: "aerospike"
        operator: "Exists"
        effect: "NoSchedule"

  storage:
    storageClassName: local-path
    localPVRequired: true
    resources:
      requests:
        storage: 100Gi

  resources:
    requests:
      cpu: "2"
      memory: "4Gi"
    limits:
      cpu: "2"
      memory: "4Gi"

  aerospikeConfig:
    service:
      proto-fd-max: 15000
    namespaceDefaults:
      replication-factor: 2
      data-size: 2147483648   # 2 GiB
```

```bash
kubectl apply -f config/samples/acko_v1alpha1_template_prod.yaml   # hard-rack
```

---

## Reference a template from a cluster

When a template supplies `image` and `size`, the cluster can omit those fields entirely:

```yaml
apiVersion: acko.io/v1alpha1
kind: AerospikeCluster
metadata:
  name: hard-rack-cluster
spec:
  # image and size are supplied by the "hard-rack" template (image: aerospike:ce-8.1.1.1, size: 6)
  templateRef:
    name: hard-rack

  aerospikeConfig:
    namespaces:
      - name: data
        storage-engine:
          type: memory
```

You can still set `spec.image` or `spec.size` explicitly on the cluster to override the template:

```yaml
spec:
  image: aerospike:ce-8.1.1.1   # override: pin a specific image
  size: 3                         # override: use 3 nodes instead of the template's 6
  templateRef:
    name: hard-rack
```

The operator resolves the template at creation time and stores the spec in `status.templateSnapshot`. From that point the cluster operates independently — changes to the template do not automatically affect this cluster.

---

## Override template fields

Use `spec.overrides` to change specific fields without creating a new template:

```yaml
spec:
  templateRef:
    name: hard-rack
  overrides:
    resources:
      requests:
        cpu: "4"
        memory: "8Gi"
      limits:
        cpu: "4"
        memory: "8Gi"
    scheduling:
      podAntiAffinityLevel: preferred   # relax anti-affinity for this cluster
```

**Merge priority:** `overrides` > `template.spec` > operator defaults

For maps (like `aerospikeConfig.service`), merging is recursive — only specified keys are overridden. For arrays (like `tolerations`) and scalar fields, the override replaces the template value entirely.

---

## Resync after template update

After updating a template, existing clusters show `status.templateSnapshot.synced: false` and emit a `TemplateDrifted` warning event. They continue operating on their snapshot.

To apply the updated template to a cluster:

```bash
kubectl annotate aerospikecluster hard-rack-cluster acko.io/resync-template=true
```

The operator will:
1. Re-fetch the template
2. Update `status.templateSnapshot`
3. Emit a `TemplateApplied` event
4. Remove the annotation

---

## Check template snapshot status

```bash
kubectl get aerospikecluster hard-rack-cluster -o jsonpath='{.status.templateSnapshot}'
```

Example output:
```json
{
  "name": "hard-rack",
  "resourceVersion": "12345",
  "snapshotTimestamp": "2026-03-01T10:00:00Z",
  "synced": true
}
```

---

## Sample templates

The `config/samples/` directory includes three ready-to-use template tiers:

| | `minimal` | `soft-rack` | `hard-rack` |
|---|-----------|-------------|-------------|
| **Purpose** | Dev / quick-start | Staging | Production |
| **size** | 1 | 3 (inherits) | 6 |
| **anti-affinity** | none | preferred | required |
| **storage** | standard 1 Gi | standard 10 Gi | local-path 100 Gi |
| **rack guarantee** | none | soft (same node OK) | hard (1 node : 1 rack) |
| **resources** | 100 m / 256 Mi | 500 m / 1 Gi | 2 / 4 Gi (Guaranteed QoS) |
| **monitoring** | disabled | disabled | enabled |
| **RF** | 1 | 2 | 2 |

Files:
- `acko_v1alpha1_template_dev.yaml` — `minimal`
- `acko_v1alpha1_template_stage.yaml` — `soft-rack`
- `acko_v1alpha1_template_prod.yaml` — `hard-rack`
- `aerospike-cluster-with-template.yaml` — example clusters using the templates

```bash
kubectl apply -f config/samples/acko_v1alpha1_template_prod.yaml   # hard-rack
kubectl apply -f config/samples/aerospike-cluster-with-template.yaml
```

---

## Install via Helm

Use the `defaultTemplates.enabled=true` option to automatically create all three template tiers in the release namespace:

```bash
helm install aerospike-ce-operator oci://ghcr.io/kimsoungryoul/charts/aerospike-ce-operator \
  -n aerospike-operator --create-namespace \
  --set certManagerSubchart.enabled=true \
  --set defaultTemplates.enabled=true
```

Verify the templates are created:

```bash
kubectl get aerospikeclustertemplate
# NAME         AGE
# minimal      10s
# soft-rack    10s
# hard-rack    10s
```

Reference a template in your cluster:

```yaml
apiVersion: acko.io/v1alpha1
kind: AerospikeCluster
metadata:
  name: my-cluster
spec:
  templateRef:
    name: soft-rack
  aerospikeConfig:
    namespaces:
      - name: data
        storage-engine:
          type: memory
```
