---
sidebar_position: 5
title: Cluster Templates
---

# Cluster Templates

`AerospikeCEClusterTemplate` lets you define reusable configuration profiles for Aerospike clusters. Instead of repeating the same scheduling, storage, and Aerospike configuration across every cluster, you define it once in a template and reference it from multiple clusters.

---

## When to use templates

- **Multiple environments** — define dev/stage/prod templates with different resource sizes and anti-affinity levels
- **Organizational defaults** — enforce storage class, tolerations, and heartbeat settings across teams
- **Configuration standardization** — prevent configuration drift by keeping shared settings in one place

---

## Create a template

```yaml
apiVersion: acko.io/v1alpha1
kind: AerospikeCEClusterTemplate
metadata:
  name: prod
  namespace: default
spec:
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
      memory-size: 2147483648   # 2 GiB
```

```bash
kubectl apply -f prod-template.yaml
```

---

## Reference a template from a cluster

```yaml
apiVersion: acko.io/v1alpha1
kind: AerospikeCECluster
metadata:
  name: prod-cluster
spec:
  size: 3
  image: aerospike:ce-8.1.1.1

  templateRef:
    name: prod    # references the "prod" AerospikeCEClusterTemplate

  aerospikeConfig:
    namespaces:
      - name: data
        storage-engine:
          type: memory
```

The operator resolves the template at creation time and stores the spec in `status.templateSnapshot`. From that point the cluster operates independently — changes to the template do not automatically affect this cluster.

---

## Override template fields

Use `spec.overrides` to change specific fields without creating a new template:

```yaml
spec:
  templateRef:
    name: prod
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
kubectl annotate aerospikececluster prod-cluster acko.io/resync-template=true
```

The operator will:
1. Re-fetch the template
2. Update `status.templateSnapshot`
3. Emit a `TemplateApplied` event
4. Remove the annotation

---

## Check template snapshot status

```bash
kubectl get aerospikececluster prod-cluster -o jsonpath='{.status.templateSnapshot}'
```

Example output:
```json
{
  "name": "prod",
  "resourceVersion": "12345",
  "snapshotTimestamp": "2026-03-01T10:00:00Z",
  "synced": true
}
```

---

## Sample templates

The `config/samples/` directory includes ready-to-use templates:

| File | Profile |
|------|---------|
| `acko_v1alpha1_template_dev.yaml` | Minimal resources, no anti-affinity |
| `acko_v1alpha1_template_stage.yaml` | Moderate resources, preferred anti-affinity |
| `acko_v1alpha1_template_prod.yaml` | Full resources, required anti-affinity, local PV |
| `aerospike-ce-cluster-with-template.yaml` | Example cluster using stage template |

```bash
kubectl apply -f config/samples/acko_v1alpha1_template_prod.yaml
kubectl apply -f config/samples/aerospike-ce-cluster-with-template.yaml
```
