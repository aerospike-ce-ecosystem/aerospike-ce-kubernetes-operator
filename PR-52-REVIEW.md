# PR #52 Review: "improve: fix nil dereference, conflict handling, status tests, docs"

**Reviewer**: Claude (automated)
**Verdict**: **Request Changes** — Critical regressions in template persistence, resync workflow, and merge safety must be fixed before merging.

---

## Critical Issues

### 1. Template snapshot is no longer persisted to the API server

**File**: `internal/controller/reconciler.go:127-150`

The PR removes the `r.Status().Update(ctx, cluster)` call that saved the template snapshot after `Resolve()`. The `Resolve()` function in `resolver.go` only sets `cluster.Status.TemplateSnapshot` **in-memory** (via `FetchAndSnapshot`). Without the explicit `Status().Update()`, the snapshot is never written to the API server. On the next reconcile (or after any status re-fetch), the snapshot will be lost.

The comment says _"Resolve() signals this via AnnotationNeedsCleanup to avoid a stale resourceVersion issue"_ but `AnnotationNeedsCleanup` only controls annotation removal — it doesn't persist the snapshot.

**Impact**: Template-based clusters will re-fetch the template on every reconcile. If the template is deleted, the cluster will fail to reconcile entirely.

**Suggestion**: Keep the `Status().Update()` call. If the concern is a stale resourceVersion, re-fetch the cluster after the update.

### 2. `AnnotationChangedPredicate` removed — annotation-based resync is broken

**File**: `internal/controller/reconciler.go:382-392`

```go
// BEFORE (main)
builder.WithPredicates(predicate.Or(
    predicate.GenerationChangedPredicate{},
    predicate.AnnotationChangedPredicate{},
)),

// AFTER (PR)
builder.WithPredicates(predicate.GenerationChangedPredicate{}),
```

The template resync workflow relies on setting `acko.io/resync-template=true` as an annotation. Annotation-only changes do **not** increment `.metadata.generation`, so without `AnnotationChangedPredicate`, this annotation change will never trigger a reconcile, making the resync feature silently broken.

The event recording and `mapTemplateToCluster` still reference this annotation, so the mechanism is half-removed.

### 3. `merge.go` — `DeepCopy()` replaced with shallow struct copies

**File**: `internal/template/merge.go` (all functions)

Every `DeepCopy()` call has been replaced with `result := *base` (shallow struct copy). This is only safe if the struct contains no pointer/slice/map fields, but `AerospikeCEClusterTemplateSpec` contains:
- `*AerospikeConfigSpec` (wraps `map[string]interface{}`)
- `*AerospikeStorageSpec` (with `[]VolumeSpec` slices)
- `*RackConfig` (with `[]Rack` slices)

A shallow copy shares the same underlying pointer values. Mutating the returned merged spec will **mutate the original base/override objects**, violating the function contract:
> _"Returns a new spec; neither input is modified."_

For example, `storageCopy := *override.Storage` still shares `Volumes []VolumeSpec` with the override. Any append/modify on the result will corrupt the original.

**Suggestion**: Revert to `DeepCopy()`.

### 4. Deleted tests for functions that still exist

| Deleted test file | Function tested | Function still exists? |
|---|---|---|
| `aero_client_test.go` | `getServicePort()` (6 cases) | Yes, in `aero_client.go` |
| `reconciler_services_test.go` → `TestServicePortsChanged` | `servicePortsChanged()` (6 cases) | Yes, in `reconciler_services.go` |

These are test coverage regressions with no justification.

---

## Medium Issues

### 5. Removed PV size validation from webhook

**File**: `api/v1alpha1/aerospikececluster_webhook.go:834-850`

The entire `persistentVolume.size` validation block was removed:
- No empty-string check
- No `resource.ParseQuantity` validation
- No positive-value check

Invalid PV sizes will now only be caught at StatefulSet creation time, producing cryptic Kubernetes errors instead of clear webhook rejection messages.

### 6. Removed `MinLength=1` CRD markers on ACL fields

**File**: `api/v1alpha1/types_rack.go`

`MinLength=1` removed from `AerospikeRoleSpec.Name`, `AerospikeUserSpec.Name`, and `AerospikeUserSpec.SecretName`. This allows empty strings to pass CRD-level validation. CRD-level validation is the first line of defense and provides better error messages.

### 7. Removed CRD template resource from kustomization

**File**: `config/crd/kustomization.yaml`

`bases/acko.io_aerospikececlustertemplates.yaml` removed. If the `AerospikeCEClusterTemplate` CRD is still used (and the template feature is still active), `make install` will no longer install the template CRD.

### 8. `reconciler_operations.go` — Conflict handling silently dropped

**File**: `internal/controller/reconciler_operations.go:108-121`

Switching from `Status().Update()` to `Status().Patch()` is fine in principle, but the conflict error handling (`IsConflict` → requeue with log) was removed. If a `Patch` conflict occurs, it now returns a raw error instead of cleanly requeueing.

### 9. ACLSynced condition semantics

**File**: `internal/controller/reconciler_status.go:287-290`

When ACL sync is skipped (no ready pods), the condition is set to `Status: False, Reason: ACLSyncPending`. `ConditionFalse` reads as "ACL is **not** synced" which could trigger alerts. Consider `ConditionUnknown` with reason `ACLSyncSkipped` for "we haven't tried yet" semantics.

---

## Low / Nits

### 10. Inline StatefulSet name instead of utility

**File**: `internal/controller/reconciler_statefulset.go:246`

`utils.StatefulSetName(cluster.Name, rack.ID)` replaced with `cluster.Name + "-" + fmt.Sprintf("%d", rack.ID)`. Functionally equivalent, but the utility still exists and is used elsewhere — this creates inconsistency.

### 11. Helm chart breaking changes

- **`existingSecret`** support removed — users with `ui.postgresql.existingSecret` configured will break on upgrade
- **`runAsUser: 70` / `runAsGroup: 70`** removed from PostgreSQL sidecar — container may run as root
- **autoscaling + postgresql mutual exclusion** check removed — users can enable both, risking data corruption

### 12. NOTES.txt heavily simplified

Lost helpful diagnostic instructions (backend health check, pod status commands, log viewing, storage info).

---

## Positive Aspects

- Nil dereference fix for `nodeHost` in `collectAerospikeInfo` — good catch
- Event recording for PDB, Service, StatefulSet — improves observability
- Conflict error handling in `mapTemplateToCluster` (log only non-conflicts)
- `reconcileACL` returning `(bool, error)` — cleaner API
- Additional edge case tests for `parseServiceEndpoints`
- Resource limit increase (128Mi → 512Mi) prevents OOMKill
- Test reorganization for `conditionsSnapshot`/`conditionsChanged` is cleaner
