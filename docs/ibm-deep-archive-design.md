<!-- ---
marp: true
theme: default
paginate: true
header: 'Glacier Integration'
footer: NooBaa 2026
size: 16:9
style: |
  section {
    overflow: hidden;
    padding-bottom: 60px;
  }
  section p, section ul, section ol {
    font-size: 22px;
  }
  section li {
    margin: 0.15em 0;
  }
  section pre {
    font-size: 18px;
    line-height: 1.35;
    margin: 0.4em 0;
    overflow-x: auto;
    overflow-y: hidden;
  }
  section code {
    font-size: 0.85em;
  }
  section pre code {
    font-size: 1em;
  }
  section table {
    font-size: 20px;
    margin: 0.4em 0;
  }
  section.columns ol {
    display: block;
    column-count: 2;
    column-gap: 80px;
    height: 420px;
    column-rule: 1px solid #eee;
  }
---

<style>
/*
@theme my-custom-theme
@auto-scaling true
*/

/* You can also add custom CSS here */
h1 {
  color: #007acc;
}
</style> -->

# Glacier Integration
## Feature Design Document

---

## Table of Contents

1. Introduction
2. Glossary
3. Goals
4. In Scope
5. Out of Scope
6. Feature Technical Details
   - AWS S3 Transition Rules background
   - NooBaa implementation
7. Affected Components
8. Bottlenecks
9. Performance, Scalability & Fault Tolerance
10. Concurrency & Race Conditions
11. DB Schema Changes
12. RPC API Changes
13. CRD Changes
14. Limitations
14. Dependencies
15. Effort Estimation
16. Open Questions

---

## Introduction

AI workloads are driving rapid data growth. Customers currently pay **standard storage rates even for cold/infrequently accessed data**, leading to unnecessary cost.

IBM Deep Archive provides **tape-based, ultra-low-cost long-term storage** accessible via an S3-compatible API, using `DEEP_ARCHIVE` as the storage class designation.

This feature integrates IBM Deep Archive into NooBaa as a first-class storage tier, enabling applications to **write directly to archive storage**, **transition data automatically** via lifecycle rules, and **restore archived objects** on demand — all through the standard S3 API.

---

## Glossary

| Term | Definition |
|------|-----------|
| **IBM Deep Archive** | IBM's tape-based cold storage, accessed via S3-compatible API using `DEEP_ARCHIVE` storage class |
| **NamespaceStore** | NooBaa CRD representing a passthrough remote object store (data is stored directly at the target endpoint) |
| **BucketClass** | NooBaa CRD defining storage policy for buckets (placement, namespace, or archive policy) |
| **ArchivePolicy** | New BucketClass type pointing to the IBM Deep Archive NamespaceStore resource |
| **PlacementPolicy** | Existing BucketClass field defining hot/standard data backing stores |
| **Lifecycle Transition** | S3 lifecycle rule action that moves objects to a different storage class after N days |
| **Restore** | Operation to temporarily copy an archived object back to standard storage for access |
| **object\_md** | NooBaa internal DB entity storing object metadata and location |
| **BG Worker** | Background service in NooBaa core running periodic maintenance tasks |

---

## Goals

- Allow applications to write directly to IBM Deep Archive via the S3 `DEEP_ARCHIVE` storage class header
- Support automatic **lifecycle transition** of objects from standard backing stores to IBM Deep Archive based on S3 lifecycle rules defined at the bucket level
- Support **Restore** of archived objects back to standard storage with a configurable expiry duration
- Expose IBM Deep Archive as a new `ibm-deep-archive` NamespaceStore type in the NooBaa operator

---

## In Scope

**Configuration -**
- **New `ibm-deep-archive` NamespaceStore type** in the noobaa-operator - CRD, reconciler, validation additions.
- **New `archivePolicy` BucketClass type** - referencing the IBM Deep Archive NamespaceStore.

**Logic -**
- **Write** directly to archive: `PutObject` / `CompleteMultipartUpload` with `StorageClass=GLACIER/DEEP_ARCHIVE` routed to archive namespace resource.
- **Lifecycle Transition**: The existing lifecycle bg worker reads S3 `Transition` rules per bucket, moves objects from standard to archive, apply changes source metadata and deletes the object's blocks/chunks metadata from db.
- **Restore**: Copy object from archive to standard bucket; set `object_md.restore_status.expiry_time = now + Days` (existing field).
- **New Expiry BG Worker**: daily job to delete expired restored copies from standard storage


---

## Out of Scope

- BackingStore variant of `ibm-deep-archive` (NamespaceStore only)
- `TransitionAfterDays` as a BucketClass-level field (configured per-bucket via S3 lifecycle API)
- Support for non-IBM Deep Archive glacier endpoints (AWS Glacier, GCS Archive, etc.) via this feature

---

## Feature Technical Details — Architecture

```
Application (S3 client)
  │
  ├─ Write (STANDARD) ──────────────────► PlacementPolicy BackingStore
  │                                       object_md.storage_class = STANDARD
  │
  ├─ Write (GLACIER/DEEP_ARCHIVE) ──────► NB calls S3 Put + storage_class = DEEP_ARCHIVE NamespaceStore ( DB: object_md.storage_class = DEEP_ARCHIVE) ──────► NooBaa with IBM Deep Archive 
  │                                       
  │
  ├─ Restore ────────► NB calls S3 Restore on Deep Archive NamespaceStore ( DB: transition_status = "in_progress") ────────────► NooBaa with IBM Deep Archive 
  |
  └─ List / Head / Get ─────────────────► object_md DB (storage_class + restore_status)

Lifecycle BG Worker 2 different loops(batched, per-object retry):
first loop - iterate over transition rules and sets
  transition_status = "in_progress"
Second loop - iterate over in_progress object_mds → read from backingstore + S3 Put to archive → DB storage_class = DEEP_ARCHIVE
  → delete data from placement BackingStore (object_md retained)

Expiry BG Worker (daily):
  restore_status.expiry_time expired → delete standard copy data → reset restore_status
  IBM Deep Archive object untouched
```

---

## Feature Technical Details — New NamespaceStore Type

New `type: ibm-deep-archive` added to the `NamespaceStore` CRD.

`IBMDeepArchiveSpec` mirrors `S3CompatibleSpec`:

```go
type IBMDeepArchiveSpec struct {
    TargetBucket     string                 `json:"targetBucket"`
    Endpoint         string                 `json:"endpoint"`
    Secret           corev1.SecretReference `json:"secret"`
    Region           string                 `json:"region,omitempty"`
    SSLDisabled      bool                   `json:"sslDisabled,omitempty"`
    SignatureVersion S3SignatureVersion      `json:"signatureVersion,omitempty"`
}
```

---

## Feature Technical Details — New NamespaceStore Type

Example CR:
```yaml
apiVersion: noobaa.io/v1alpha1
kind: NamespaceStore
metadata:
  name: ibm-deep-archive-store
spec:
  type: ibm-deep-archive
  ibmDeepArchive:
    targetBucket: my-archive-bucket
    endpoint: https://s3.us-south.cloud-object-storage.appdomain.cloud
    secret:
      name: ibm-deep-archive-credentials
      namespace: openshift-storage
```

---

## Feature Technical Details — New BucketClass Archive Policy

New optional `archivePolicy` field added to `BucketClassSpec`:

```go
type ArchivePolicy struct {
    // Resources is a list of ibm-deep-archive NamespaceStore names (same namespace)
    Resources []string `json:"resources"`
}
```

Example BucketClass CR:
```yaml
apiVersion: noobaa.io/v1alpha1
kind: BucketClass
metadata:
  name: archive-bucketclass
spec:
  placementPolicy:
    tiers:
      - backingStores:
          - standard-backing-store
  archivePolicy:
    resources:
      - ibm-deep-archive-store
```

---

## Feature Technical Details — New BucketClass Archive Policy

- **Validation**: all entries in `resources` must be `ibm-deep-archive` NamespaceStores in the same namespace; `placementPolicy` must also be present
- **Core registration**: BucketClass reconciler extends the existing `create_bucket()` / `update_bucket()` RPC to pass `archive_resources: [namespace_resource._id, ...]` — core stores this on the bucket document; lifecycle worker reads it at runtime
- **Immutability guard**: uses the existing `can_delete_bucket` RPC pattern — objects-exist check is applied before allowing `archivePolicy` to be modified or removed
- **Lifecycle transition days**: configured **at the bucket level** via S3 lifecycle rules — not in the BucketClass

---

## Feature Technical Details — Write Routing

**Applies to:** `PutObject`, `CompleteMultipartUpload`

**Standard write** (`StorageClass` absent or `STANDARD`):
- Normal path → placement backing store

**Archive write** (`StorageClass: GLACIER` or `DEEP_ARCHIVE`):
- Check bucket's `archive_resources` (populated from BucketClass `archivePolicy`)
- If present → write to IBM Deep Archive namespace resource; `object_md.storage_class = 'DEEP_ARCHIVE'`
- `object_md` is discoverable via `bucket_id → bucket.namespace.write_resource → archive endpoint + target bucket`
- On write success but `object_md` write failure → **compensating delete** on IBM Deep Archive object to prevent orphaned data
- If no `archive_resources` → return `InvalidStorageClass`

---

## Feature Technical Details — S3 Operations on Archived Objects

| Operation | Behavior |
|-----------|----------|
| **HeadObject** | Returns `x-amz-storage-class: DEEP_ARCHIVE` + `x-amz-restore` header from `restore_status` |
| **GetObject** | Blocks with `InvalidObjectState` if not restored; reads from standard copy when `restore_status.expiry_time` is set and not expired |
| **CopyObject** (source archived) | `InvalidObjectState` if not restored; supports `StorageClass=GLACIER` on target (routes to archive) |
| **DeleteObject** | Passthrough delete to IBM Deep Archive endpoint to avoid orphaned data; then deletes `object_md` |
| **GetObjectAttributes** | Returns `StorageClass` from `object_md` |
| **ListObjects** | Served from `object_md` DB — no direct IBM Deep Archive query |


---

## Feature Technical Details — AWS S3 Lifecycle Transition Rules

AWS S3 Lifecycle rules allow automatic movement of objects between storage classes over time.  
Users configure them via `PutBucketLifecycleConfiguration`:

```xml
<LifecycleConfiguration>
  <Rule>
    <ID>move-to-deep-archive</ID>
    <Status>Enabled</Status>
    <Filter>
      <Prefix>logs/</Prefix>
    </Filter>
    <Transition>
      <Days>90</Days>
      <StorageClass>DEEP_ARCHIVE</StorageClass>
    </Transition>
    <NoncurrentVersionTransition>
      <NoncurrentDays>30</NoncurrentDays>
      <StorageClass>DEEP_ARCHIVE</StorageClass>
    </NoncurrentVersionTransition>
  </Rule>
</LifecycleConfiguration>
```

Key fields: **`Days`** (relative from object creation) or **`Date`** (absolute), **`StorageClass`**, optional **`Filter`** (prefix/tag/size).

---

## Feature Technical Details — Transition Rules: Current NooBaa State

| Behavior | Current State |
|----------|--------------|
| `PutBucketLifecycleConfiguration` | Accepted and stored in DB |
| `GetBucketLifecycleConfiguration` | Returned correctly (including `Transition` elements) |
| `Transition` rule **execution** | **Skipped** by the lifecycle bg worker |
| `NoncurrentVersionTransition` execution | **Skipped** |
| Expiration / MPU abort execution | Implemented and running |

From `src/server/bg_services/lifecycle.js`:
```javascript
if (
    rule.expiration === undefined &&
    rule.abort_incomplete_multipart_upload === undefined &&
    rule.noncurrent_version_expiration === undefined
) {
    // SKIP — rule contains no expiration parameters
    return;
}
```

`Transition`-only rules are silently skipped today.  
`Transition` fields are accepted in PUT but **transition validation is also skipped**.

---

## Feature Technical Details — Lifecycle Transition

The existing lifecycle bg worker currently **skips** `Transition` / `NoncurrentVersionTransition` rules.

**New behavior — processes objects in batches:**
1. Read `Transition` rules per bucket; gate on bucket having `archive_resources`
2. For each eligible object:
   - Set `object_md.transition_status = "in_progress"` (idempotency marker — skip if already `in_progress` or `done`)
   - Write object data to IBM Deep Archive (S3-compatible write, per-request timeout)
   - Set `object_md.storage_class = 'DEEP_ARCHIVE'`, `transition_status = "done"`
   - **Delete data from the placement backing store** (NOT `object_md` — retained for List/Head/Get)
3. Per-object errors: log and **continue** — do not abort the bucket's transition run; retry on next cycle
4. Checkpoint progress — crash/restart resumes cleanly from where it left off
5. `object_md` **is never deleted**: `storage_class = DEEP_ARCHIVE` signals archive location to all S3 operations


---

## Feature Technical Details — Restore & Expiry

**Restore is async** — IBM Deep Archive (tape) retrieval takes up to 12 hours; inline copy is not viable.

**`PostObjectRestore` flow:**
1. Set `object_md.restore_status = { ongoing: true }` immediately; return **202 Accepted**
2. Background restore worker copies data from IBM Deep Archive to standard placement backing store
3. On copy complete: `restore_status = { ongoing: false, expiry_time: now + Days }`
4. Client polls via `HeadObject` (`x-amz-restore` header) to detect completion
5. Concurrent restore guard: if `restore_status.ongoing = true` or `expiry_time` not expired, update expiry only

**New Expiry BG Worker** (`src/server/bg_services/restore_expiry.js`):
- Runs once per day
- Queries `object_md` where `restore_status.expiry_time <= now` AND `storage_class` in `GLACIER_STORAGE_CLASSES`
- Deletes the **standard copy data** (blocks/chunks from placement backing store), resets `restore_status`
- IBM Deep Archive object is **not deleted** — `object_md` remains with `storage_class = DEEP_ARCHIVE`

**NamespaceStore Mode**: surface IBM Deep Archive endpoint health via the existing `Mode` field (same mechanism as `s3-compatible` stores)

---

## Affected Components

### noobaa-operator

| File / Package | Change |
|----------------|--------|
| `pkg/apis/noobaa/v1alpha1/namespacestore_types.go` | New `NSType`, `IBMDeepArchiveSpec` |
| `pkg/apis/noobaa/v1alpha1/bucketclass_types.go` | New `ArchivePolicy`, field in `BucketClassSpec` |
| `pkg/apis/noobaa/v1alpha1/zz_generated.deepcopy.go` | Regenerated |
| `deploy/crds/noobaa.io_namespacestores.yaml` | New type enum + spec block |
| `deploy/crds/noobaa.io_bucketclasses.yaml` | New `archivePolicy` block |
| `pkg/bundle/deploy.go` | Regenerated embedded bundle |
| `pkg/namespacestore/reconciler.go` | New type case in `MakeExternalConnectionParams` |
| `pkg/bucketclass/reconciler.go` | Archive policy validation + status reporting |
| `pkg/validations/` | Validate new NS type + archive policy constraints |
| `pkg/admission/validate_namespacestore.go` | Target-bucket change guard |
| `pkg/admission/validate_bucketclass.go` | Immutability guard when objects exist |
| `pkg/util/util.go` | NamespaceStore helper switch cases |

---

## Affected Components

### noobaa-core

| File | Change |
|------|--------|
| `src/endpoint/s3/ops/s3_put_object.js` | Route to archive NS when `StorageClass=GLACIER/DEEP_ARCHIVE`; compensating delete on failure |
| `src/endpoint/s3/ops/s3_post_object_uploads.js` | Same routing for CompleteMultipartUpload |
| `src/endpoint/s3/ops/s3_head_object.js` | Return `x-amz-storage-class` + `x-amz-restore` for archived objects |
| `src/endpoint/s3/ops/s3_delete_object.js` | Passthrough delete to IBM Deep Archive endpoint |
| `src/endpoint/s3/ops/s3_copy_object.js` | Block on non-restored archived source; support GLACIER target routing |
| `src/endpoint/s3/ops/s3_post_object_restore.js` | Restore-as-copy; update `restore_status` on `object_md` |
| `src/server/bg_services/lifecycle.js` | Execute `Transition` rules in batches with per-object retry |
| `src/server/bg_services/restore_expiry.js` | New daily expiry worker (new file) |
| `src/server/bucket_api.js` | Extend `create_bucket` / `update_bucket` with `archive_resources` |
| `src/api/` | `object_md` schema: add `transition_status`; add `storage_class` index |

---

## Bottlenecks

| Area | Bottleneck | Mitigation |
|------|-----------|------------|
| **Lifecycle worker** | 3 serial remote I/O ops per object (read → write → delete); single instance; tape latency | Parallel batch concurrency (configurable); per-object HTTP timeout; per-object error isolation |
| **object_md queries** | Finding transition candidates without compound index = full collection scan per bucket | Compound index `(bucket_id, storage_class, create_time)` |
| **Restore** | IBM Deep Archive tape retrieval up to 12h — synchronous copy blocks the S3 client indefinitely | Async restore: 202 Accepted immediately; background worker performs copy; client polls HeadObject |
| **Expiry worker** | Daily scan for expired restore copies without index | Compound index `(storage_class, restore_status.expiry_time)` |
| **Stuck transitions** | Worker crash leaves `transition_status = "in_progress"` forever with no recovery | `transition_started_at` timestamp; on restart, reset objects stuck > `MAX_TRANSITION_DURATION` |
| **DeleteObject** | Synchronous round-trip to IBM Deep Archive on every delete of an archived object | Accept as-is for v1; async delete queue if latency becomes a concern |

---

## Performance & Scalability

**Performance targets:**
- Lifecycle worker: process N objects in parallel per bucket (configurable `LIFECYCLE_TRANSITION_CONCURRENCY` in NooBaa config)
- Each object write to IBM Deep Archive uses the S3 client's default per-request timeout — no special throttling needed (tape systems use S3-layer buffers)
- Async restore: S3 client is never blocked waiting for tape retrieval

**Scalability:**
- `object_md` collection: all queries must hit indexes — full collection scans are unacceptable at millions of objects
- Lifecycle worker is a **single instance** for v1 (leader election via existing BG worker pattern); future: distributed with bucket-level lease/lock to parallelize across buckets
- IBM Deep Archive endpoint itself handles tape scaling — NooBaa's responsibility is to avoid unnecessary duplicate requests

**DB query plan (per operation):**

| Worker / Operation | Query filter | Required index |
|---------------------|-------------|----------------|
| Lifecycle transition candidates | `bucket_id + storage_class = STANDARD + create_time < threshold + transition_status != done` | `{ bucket_id, storage_class, create_time }` |
| Stuck transition recovery | `transition_status = in_progress + transition_started_at < (now - MAX_DURATION)` | `{ transition_status, transition_started_at }` (sparse) |
| Expiry worker | `storage_class in GLACIER_STORAGE_CLASSES + restore_status.expiry_time <= now` | `{ storage_class, restore_status.expiry_time }` |

---

## High Availability & Fault Tolerance

**Lifecycle worker crash recovery:**
- Each object sets `transition_status = "in_progress"` + `transition_started_at = now` before starting
- On worker restart: scan for objects with `transition_status = "in_progress"` AND `transition_started_at < now - MAX_TRANSITION_DURATION`; reset to `null` for retry
- Idempotent archive write: check if object already exists in IBM Deep Archive before writing to prevent duplicate tape writes on retry

**Partial transition failure scenarios:**

| Failure point | State | Recovery |
|---------------|-------|----------|
| Write to archive fails | `in_progress`; no data in archive | Retry on next cycle; clean |
| Archive write OK, `object_md` update fails | `in_progress`; data in archive + standard | Retry: idempotent archive write is a no-op; re-update `object_md` |
| `object_md` updated, placement delete fails | `done`; data in both places | Next cycle: placement delete can be retried independently |

**IBM Deep Archive unavailability:**
- Worker catches the error, logs it, skips the bucket for this cycle
- NamespaceStore `Mode` surfaces endpoint degradation to operators
- Exponential backoff is not needed (BG worker runs on its own schedule)

**Restore partial failure:**
- Restore worker sets `restore_status.ongoing = true` before starting copy
- If copy fails: reset `restore_status` (no `expiry_time`) — HeadObject shows not-restored; client can retry
- If copy succeeds but `restore_status` update fails: copy exists in standard store but not tracked; on retry, copy is idempotent (overwrite same key)

---

## Concurrency & Race Conditions

### Transition vs. Concurrent PutObject (overwrite)

**Scenario:** Lifecycle worker reads object A from standard and starts writing to archive. Simultaneously a client overwrites object A with a new PutObject (new `etag` / `version_id`).

**Risk:** Worker completes, deletes the standard copy → the client's newly written data is destroyed.

**Fix:** Before deleting from standard, the worker performs an **optimistic concurrency check** — re-reads `object_md.etag` (or `upload_id` / `version_id`) and aborts the deletion if it has changed since the worker started the transition. Reset `transition_status = null` and retry on the next cycle.

---

## Concurrency & Race Conditions

### Transition vs. Concurrent DeleteObject

**Scenario:** Lifecycle worker has written object A to IBM Deep Archive. Before the worker updates `object_md` and deletes from standard, a client calls `DeleteObject` — `object_md` is removed and a passthrough delete to IBM Deep Archive is issued.

**Risk:** Worker then tries to update `object_md` (not found) → archive write is completed but the delete to IBM Deep Archive from the `DeleteObject` path races with the worker's write. If the archive delete arrives first, the worker re-creates the object in archive.

**Fix:** Worker checks `object_md` existence after completing the archive write (before updating `storage_class`). If `object_md` is gone, the worker issues a **compensating delete** to IBM Deep Archive and aborts the transition cleanly.

---

## Concurrency & Race Conditions

### Concurrent PostObjectRestore

**Scenario:** Two clients simultaneously call `PostObjectRestore` on the same object. Both read `restore_status = null` and both attempt to set `restore_status.ongoing = true`.

**Risk:** Two background restore copies are started, resulting in double data written to the standard placement backing store and two competing `restore_status` updates.

**Fix:** Use a **MongoDB atomic findOneAndUpdate with condition** — only update if `restore_status.ongoing != true` AND `restore_status.expiry_time` is unset or expired. Only the writer that wins the atomic update starts the restore job. All others return 202 immediately without launching a copy.

---

## Concurrency & Race Conditions

### Restore Worker vs. Concurrent DeleteObject

**Scenario:** Background restore worker is copying object A from IBM Deep Archive to standard storage. Client calls `DeleteObject` — `object_md` is removed, passthrough delete to IBM Deep Archive.

**Risk:** Restore copy completes and tries to update `restore_status` on `object_md` (not found). The standard copy data is now written but completely untracked — orphaned data in the placement backing store.

**Fix:** Restore worker checks `object_md` existence before updating `restore_status`. If `object_md` is gone, it issues a **compensating delete** of the standard copy data and exits cleanly.

---

## Concurrency & Race Conditions

### Lifecycle Transition vs. Active Restore

**Scenario:** Lifecycle worker selects object A for transition (it is `storage_class = STANDARD` and past the transition threshold). Simultaneously, a restore is active on object A (`restore_status.ongoing = true` — background restore copy is in flight from archive to standard).

**Risk:** Lifecycle worker deletes the standard copy while it is being used as a restore target. Or the worker transitions the object back to archive while a client is about to read the freshly restored copy.

**Fix:** Lifecycle worker **skips** any object where `restore_status.ongoing = true` or `restore_status.expiry_time` is set and not expired. These objects are picked up in a future cycle once the restore has completed and expired.

---

## Concurrency & Race Conditions

### Transition Update Ordering (Critical)

The order of writes during a lifecycle transition determines the failure mode visible to clients:

| Order | Failure if crash between steps | Client experience |
|-------|-------------------------------|-------------------|
| 1. Write to archive → 2. Delete from standard → **3. Update object_md** | standard gone, object_md says STANDARD | Confusing 404 from backing store |
| 1. Write to archive → **2. Update object_md** → 3. Delete from standard | object_md says DEEP_ARCHIVE, standard still exists | Clean `InvalidObjectState` — correct S3 behavior |

**Correct order:** write to archive → **atomically update `object_md.storage_class = DEEP_ARCHIVE` + `transition_status = "done"`** → delete from standard (best-effort, retried by the next cycle for any `transition_status = "done"` objects with placement data still present).

---

## Concurrency & Race Conditions

### Expiry Worker vs. Concurrent GetObject

**Scenario:** Expiry worker identifies object A as having `restore_status.expiry_time <= now` and begins deleting the standard copy. Simultaneously a client is mid-read of object A's restored standard copy.

**Risk:** Client's read fails mid-stream with a partial or missing data error.

**Mitigation:** The expiry worker runs once daily; the practical overlap window is small. The expiry time should be checked **atomically at deletion time** using a `findOneAndUpdate` with condition `restore_status.expiry_time <= now` — this ensures the deletion is not applied if the expiry time was updated (e.g., a fresh restore request extended the expiry). In-flight reads are handled by the S3 layer's existing error propagation (read errors surface to the client naturally).

---

## Concurrency & Race Conditions

### Dual Lifecycle Worker Instances

**Scenario:** A bug or split-brain causes two lifecycle worker instances to run simultaneously and process the same bucket.

**Risk:** Two workers select the same object, both set `transition_status = "in_progress"`, both write to IBM Deep Archive simultaneously.

**Fix:**
- `transition_status` acts as a **distributed lock**: the second writer to atomically update from `null → "in_progress"` will see the field is already set and skip the object
- Use MongoDB **`findOneAndUpdate` with condition `transition_status = null`** to atomically claim an object
- **Idempotent archive write**: even if both workers write to IBM Deep Archive, the second write is a no-op (same key, same data)
- Existing NooBaa BG worker leader election prevents this in normal operation; this is a defense-in-depth measure

---

## DB Schema Changes

### `object_md` collection

| Field | Type | Status | Notes |
|-------|------|--------|-------|
| `storage_class` | string | **Exists** | `STANDARD \| GLACIER \| GLACIER_IR \| DEEP_ARCHIVE` |
| `restore_status` | object | **Exists** | `{ ongoing: bool, expiry_time: Date }` |
| `transition_status` | string | **New** | `"in_progress" \| "done" \| null` |
| `transition_started_at` | Date | **New** | Set when `transition_status = "in_progress"`; used for stuck recovery |

**New indexes on `object_md`:**
```javascript
// Lifecycle worker — find transition candidates
{ bucket_id: 1, storage_class: 1, create_time: 1 }

// Expiry worker — find expired restore copies
{ storage_class: 1, "restore_status.expiry_time": 1 }

// Stuck recovery — sparse (most objects null)
{ transition_status: 1, transition_started_at: 1 }, { sparse: true }
```

---

## DB Schema Changes (continued)

### `buckets` collection

| Field | Type | Status | Notes |
|-------|------|--------|-------|
| `archive_resources` | `ObjectId[]` | **New** | Array of `namespace_resources._id`; undefined = no archive policy |

### `namespace_resources` collection

No new fields. The existing `endpoint_type` field (value `S3_COMPATIBLE`) is used during `create_bucket` / `update_bucket` validation to confirm that each `archive_resources` entry is of archive type.

---

## RPC API Changes

### noobaa-operator — `pkg/nb/types.go`

```go
// CreateBucketParams — extend with archive_resources
type CreateBucketParams struct {
    Name             string   `json:"name"`
    BucketClass      string   `json:"bucket_class,omitempty"`
    // ... existing fields unchanged ...
    ArchiveResources []string `json:"archive_resources,omitempty"`
}

// UpdateBucketParams — extend with archive_resources
type UpdateBucketParams struct {
    Name             string   `json:"name"`
    // ... existing fields unchanged ...
    ArchiveResources []string `json:"archive_resources,omitempty"`
}

// BucketInfo — add archive_resources to read reply
type BucketInfo struct {
    Name             string   `json:"name"`
    // ... existing fields unchanged ...
    ArchiveResources []string `json:"archive_resources,omitempty"`
}
```

### noobaa-core — `src/api/bucket_api.js`

```javascript
create_bucket: {
    method: 'POST',
    params: {
        // ... existing params ...
        archive_resources: {
            type: 'array',
            items: { objectid: true },
        },
    },
},
update_bucket: { /* same addition */ },
read_bucket: {
    reply: {
        // ... existing reply fields ...
        archive_resources: { type: 'array', items: { objectid: true } },
    },
},
```

---

## CRD Changes — NamespaceStore

Add to `deploy/crds/noobaa.io_namespacestores.yaml`:

```yaml
spec:
  type:
    enum:
      # ... existing values ...
      - ibm-deep-archive      # NEW

  ibmDeepArchive:             # NEW — required when type = ibm-deep-archive
    type: object
    required:
      - targetBucket
      - endpoint
      - secret
    properties:
      targetBucket:
        type: string
      endpoint:
        type: string
      secret:
        type: object
        required: [name]
        properties:
          name:        { type: string }
          namespace:   { type: string }
      region:
        type: string
      sslDisabled:
        type: boolean
      signatureVersion:
        type: string
        enum: [v2, v4]
```

---

## CRD Changes — BucketClass

Add to `deploy/crds/noobaa.io_bucketclasses.yaml`:

```yaml
spec:
  archivePolicy:              # NEW — optional
    type: object
    required:
      - resources
    properties:
      resources:
        type: array
        minItems: 1
        items:
          type: string
        description: >
          Names of ibm-deep-archive NamespaceStores in the same namespace.
          All referenced stores must exist and be of type ibm-deep-archive.
          archivePolicy cannot be modified or removed while objects exist
          in the archive.
```

**Validation enforced by admission webhook (not CRD schema alone):**
- Each name in `resources` must resolve to a `Ready` NamespaceStore of type `ibm-deep-archive` in the same namespace
- `placementPolicy` must be present when `archivePolicy` is set

---

## Limitations

- **Archive policy is immutable with data** — once objects exist in the archive, `archivePolicy` cannot be modified or removed; a new BucketClass must be created
- **Restore latency is IBM-controlled** — tape retrieval is async and may take up to 12 hours; NooBaa has no way to accelerate this
- **No cross-endpoint migration** — if the IBM Deep Archive endpoint or target bucket changes, archived objects cannot be automatically migrated
- **List reflects DB state** — object listings come from `object_md` DB; out-of-band operations on the IBM Deep Archive bucket are invisible to NooBaa
- **Single lifecycle worker instance for v1** — horizontal scaling of the lifecycle worker requires future coordination (distributed bucket-level lease)

---

## Dependencies

- IBM Deep Archive endpoint - Must be S3-compatible and accessible from the cluster

---

## Effort Estimation

XL

---

## Open Questions

1. **EndpointType in core** — should IBM Deep Archive use the existing `S3_COMPATIBLE` endpoint type or a new `IBM_DEEP_ARCHIVE` endpoint type? Matters for connection routing and future auth differences.

2. **Restore latency UX** — how should NooBaa surface IBM Deep Archive retrieval latency to the user? Should `PostObjectRestore` return an estimated availability time?

3. **Versioned object transitions** — can one version of an object be in standard while another is in archive? Is each version transitioned independently per the lifecycle rule? Must `RestoreObject` accept a `VersionId`?

4. **Archive policy and OBC/COSI** — when a bucket is provisioned via OBC or COSI against an archive BucketClass, are there blockers or special behaviors to handle at provisioning time?

5. **Lifecycle batch size** — what is the target batch size for transition processing? Should it be configurable via a NooBaa system config parameter?

6. **`x-amz-transition-default-minimum-object-size`** — AWS introduced this header to prevent lifecycle rules from transitioning objects below a minimum size (since storing small objects in Glacier is cost-inefficient; the overhead per-object can exceed the storage savings). Should NooBaa support this header on `PutBucketLifecycleConfiguration`? If yes: the lifecycle worker must skip objects smaller than the configured threshold when evaluating `Transition` rules. If no: document that all objects matching the filter and age criteria are transitioned regardless of size.

---

## Open Questions (Resolved)

| Question | Resolution |
|----------|-----------|
| How does core learn about archive policy? | Extend existing `create_bucket()` / `update_bucket()` RPC with `archive_resources` array |
| Should `object_md` be deleted on transition? | No — update `storage_class = DEEP_ARCHIVE`, delete placement backing store data only |
| How does immutability guard work? | Reuse existing `can_delete_bucket` objects-exist check pattern |
| Should lifecycle worker abort on single object error? | No — log, continue, retry on next cycle |
| Should IBM Deep Archive surface health status? | Yes — via existing `Mode` field on NamespaceStore (S3-compatible) |
| Is `resource` a single string or array? | Array (`resources []string`) to allow future multi-target archive |
