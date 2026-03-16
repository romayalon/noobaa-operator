[NooBaa Operator](../README.md) /
# Vector Buckets

Vector buckets provide a vector database interface backed by NooBaa's namespace filesystem storage.
They are provisioned through the standard OBC (ObjectBucketClaim) flow with additional configuration
that specifies the vector database engine and its parameters.

## Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│                        OBC (ObjectBucketClaim)                       │
│                                                                      │
│  spec.additionalConfig:                                              │
│    bucketType: "vector"                                              │
│    vectorDBType: "lance"          ← vector DB engine                 │
│    lanceConfig: '{"subPath":"embeddings"}'  ← optional, JSON config  │
│                                                                      │
│  spec.storageClassName: <storageclass>                               │
│    └─ parameters.bucketclass: <bucketclass-name>                     │
└──────────────────────┬───────────────────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────────────────────┐
│               BucketClass (NamespacePolicy: Single)                  │
│                                                                      │
│  spec.namespacePolicy:                                               │
│    type: Single              ← MUST be Single for vector buckets     │
│    single:                                                           │
│      resource: <nsstore>     ← references a NamespaceStore           │
└──────────────────────┬───────────────────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────────────────────┐
│                   NamespaceStore (type: nsfs)                         │
│                                                                      │
│  spec.type: nsfs             ← MUST be nsfs (currently)              │
│  spec.nsfs:                                                          │
│    pvcName: <pvc>            ← underlying filesystem PVC             │
│    fsBackend: <backend>      ← e.g. GPFS, CEPH_FS                   │
│                                                                      │
│  NOTE: S3 and other NamespaceStore types may be supported            │
│        for vector buckets in the future.                             │
└──────────────────────┬───────────────────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────────────────────┐
│                    NooBaa Core (Vector Service)                       │
│                                                                      │
│  RPC: create_vector_bucket / delete_vector_bucket                    │
│  Endpoint: HTTPS port 14443                                          │
│                                                                      │
│  Exposed via:                                                        │
│    • K8s Service  → "vector" (port 443 → 14443)                      │
│    • OCP Route    → "vector" (TLS reencrypt)                         │
└──────────────────────────────────────────────────────────────────────┘
```

## Client Setup Flow

End-to-end steps a client (cluster admin / app developer) needs to follow
to create a vector bucket, from scratch to a working vector endpoint:

```
 ┌─────────────────────────────────────────────────────────────────────┐
 │ STEP 0: Prerequisites                                              │
 │   • NooBaa system is deployed (operator + core)                     │
 │   • Vector service + route are created automatically by the         │
 │     operator during system reconciliation                           │
 │   • A PVC exists for the filesystem storage                         │
 └──────────────────────────┬──────────────────────────────────────────┘
                            │
                            ▼
 ┌─────────────────────────────────────────────────────────────────────┐
 │ STEP 1: Create a NamespaceStore (NSFS)                    [Admin]  │
 │                                                                     │
 │   apiVersion: noobaa.io/v1alpha1                                    │
 │   kind: NamespaceStore                                              │
 │   metadata:                                                         │
 │     name: vector-fs                                                 │
 │   spec:                                                             │
 │     type: nsfs                                                      │
 │     nsfs:                                                           │
 │       pvcName: vector-data-pvc                                      │
 │       fsBackend: ""           ← optional (GPFS, CEPH_FS, etc.)     │
 │                                                                     │
 │   ⏳ Wait for NamespaceStore Phase = Ready                          │
 └──────────────────────────┬──────────────────────────────────────────┘
                            │
                            ▼
 ┌─────────────────────────────────────────────────────────────────────┐
 │ STEP 2: Create a BucketClass (Single policy)              [Admin]  │
 │                                                                     │
 │   apiVersion: noobaa.io/v1alpha1                                    │
 │   kind: BucketClass                                                 │
 │   metadata:                                                         │
 │     name: vector-bucket-class                                       │
 │   spec:                                                             │
 │     namespacePolicy:                                                │
 │       type: Single           ← MUST be Single                       │
 │       single:                                                       │
 │         resource: vector-fs  ← points to the NamespaceStore         │
 │                                                                     │
 │   ⏳ Wait for BucketClass Phase = Ready                             │
 └──────────────────────────┬──────────────────────────────────────────┘
                            │
                            ▼
 ┌─────────────────────────────────────────────────────────────────────┐
 │ STEP 3: Create a StorageClass                             [Admin]  │
 │                                                                     │
 │   apiVersion: storage.k8s.io/v1                                     │
 │   kind: StorageClass                                                │
 │   metadata:                                                         │
 │     name: noobaa-vector                                             │
 │   provisioner: <namespace>.noobaa.io/obc                            │
 │   reclaimPolicy: Delete                                             │
 │   parameters:                                                       │
 │     bucketclass: vector-bucket-class                                │
 │                                                                     │
 │   (StorageClass is cluster-scoped, create once)                     │
 └──────────────────────────┬──────────────────────────────────────────┘
                            │
                            ▼
 ┌─────────────────────────────────────────────────────────────────────┐
 │ STEP 4: Create an OBC (vector bucket)                  [App Dev]   │
 │                                                                     │
 │   apiVersion: objectbucket.io/v1alpha1                              │
 │   kind: ObjectBucketClaim                                           │
 │   metadata:                                                         │
 │     name: my-vector-bucket                                          │
 │   spec:                                                             │
 │     storageClassName: noobaa-vector                                 │
 │     generateBucketName: my-vector                                   │
 │     additionalConfig:                                               │
 │       bucketType: vector                                            │
 │       vectorDBType: lance        ← lance | davinci                  │
 │       subPath: embeddings        ← optional                         │
 │       lanceConfig: '{"subPath":"..."}' ← optional, lance-specific   │
 │                                                                     │
 │   ⏳ Wait for OBC Phase = Bound                                     │
 └──────────────────────────┬──────────────────────────────────────────┘
                            │
                            ▼
 ┌─────────────────────────────────────────────────────────────────────┐
 │ STEP 5: Use the vector bucket                          [App Dev]   │
 │                                                                     │
 │   Get the vector service endpoint:                                  │
 │     kubectl get noobaa -o jsonpath=                                 │
 │       '{.items[0].status.services.serviceVector}'                   │
 │                                                                     │
 │   Access via:                                                       │
 │     • Internal: vector.<namespace>.svc.cluster.local:443            │
 │     • External: OCP Route (if routes enabled)                       │
 │                                                                     │
 │   The vector DB (LanceDB/Davinci) is ready to accept               │
 │   vector operations on the provisioned bucket.                      │
 └─────────────────────────────────────────────────────────────────────┘
```

## Internal Provisioning Flow

What happens inside the operator when the OBC is created (Step 4):

```
OBC created by user
        │
        ▼
  ┌─────────────┐
  │ ValidateOBC │  validates bucketType, vectorDBType values
  └──────┬──────┘  rejects vectorDBType on bucketType=data
         │
         ▼
  ┌──────────────────┐
  │ isVectorBucket() │  checks additionalConfig:
  │                  │    bucketType == "vector"
  │                  │    OR vectorDBType == "lance" | "davinci"
  └──────┬───────────┘
         │ yes
         ▼
  ┌────────────────────────────────────────────────┐
  │ createVectorBucket()                           │
  │   • Validate: NamespacePolicy is Single        │
  │   • Load NamespaceStore from BucketClass       │
  │   • Validate: NamespaceStore is NSFS           │
  │   • Build VectorDBConfig per engine type       │
  │   • Call CreateVectorBucketAPI (RPC)            │
  │   • Store bucketType + vectorDBType in OB      │
  └────────────────────────────────────────────────┘
```

## Constraints

| Constraint | Current | Future |
|---|---|---|
| BucketClass NamespacePolicy type | **Single only** | Single only |
| NamespaceStore type | **NSFS only** | S3, and others |
| Supported vectorDBType values | **lance**, **davinci** | opensearch, others |
| Vector service port | 14443 (HTTPS) | — |

## OBC Configuration

Vector bucket behavior is controlled via `spec.additionalConfig` on the OBC:

| Key | Required | Values | Description |
|---|---|---|---|
| `bucketType` | No | `data`, `vector` | Bucket type. Defaults to `data`. |
| `vectorDBType` | No | `lance`, `davinci` | Vector DB engine. Defaults to `lance` when `bucketType=vector`. |
| `lanceConfig` | No | JSON string | LanceDB-specific config. JSON with fields: `subPath` (sub-path within the NamespaceStore filesystem). Extensible for future LanceDB settings. |

A vector bucket is detected if **either** `bucketType=vector` **or** `vectorDBType` is set to a supported engine (`lance`, `davinci`).

Setting `vectorDBType` on a `bucketType=data` OBC is rejected by validation.

## RPC Parameters

The `CreateVectorBucketParams` sent to NooBaa Core:

```go
type CreateVectorBucketParams struct {
    Name           string                       // bucket name
    VectorDBType   string                       // "lance", "davinci"
    Resource       *NamespaceResourceFullConfig  // nsStore name + path
    VectorDBConfig *VectorDBConfig               // engine-specific config
    BucketClaim    *BucketClaimInfo              // OBC metadata
}
```

`VectorDBConfig` is extensible per engine:

```go
type VectorDBConfig struct {
    LanceDBConfig *LanceDBConfigParam  // optional
    // future: DavinciConfig, OpenSearchConfig
}
```

## Kubernetes Resources

The vector service is exposed through:

- **Service** `vector` — LoadBalancer (or ClusterIP if `DisableLoadBalancerService` is set), port 443 → targetPort `vector-https` (14443)
- **Route** `vector` — OpenShift Route with TLS reencrypt termination (skipped if `DisableRoutes` is set)
- **Endpoint Deployment** — container port `vector-https` (14443) added to the endpoint pods

Service status is reported in `NooBaa.Status.Services.ServiceVector`.

## Example

### 1. Create a NamespaceStore (NSFS)

```yaml
apiVersion: noobaa.io/v1alpha1
kind: NamespaceStore
metadata:
  name: vector-fs
spec:
  type: nsfs
  nsfs:
    pvcName: vector-data-pvc
```

### 2. Create a BucketClass (Single namespace policy)

```yaml
apiVersion: noobaa.io/v1alpha1
kind: BucketClass
metadata:
  name: vector-bucket-class
spec:
  namespacePolicy:
    type: Single
    single:
      resource: vector-fs
```

### 3. Create a StorageClass

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: noobaa-vector
provisioner: noobaa.noobaa.io/obc
reclaimPolicy: Delete
parameters:
  bucketclass: vector-bucket-class
```

### 4. Create an OBC (vector bucket)

```yaml
apiVersion: objectbucket.io/v1alpha1
kind: ObjectBucketClaim
metadata:
  name: my-vector-bucket
spec:
  storageClassName: noobaa-vector
  generateBucketName: my-vector
  additionalConfig:
    bucketType: vector
    vectorDBType: lance
    lanceConfig: '{"subPath":"embeddings"}'
```

## Deletion

When an OBC with `bucketType=vector` (or a recognized `vectorDBType`) is deleted,
the provisioner calls `DeleteVectorBucketAPI` instead of the standard bucket deletion path.

> **TODO**: Vector bucket deletion currently removes only the bucket metadata.
> Deletion of the underlying vector data on the filesystem is not yet implemented.

## CLI

The NooBaa CLI (`noobaa obc`) supports vector bucket operations.

### Create a vector bucket

```shell
noobaa obc create <name> --bucketclass <bucketclass> --bucket-type vector [flags]
```

| Flag | Default | Description |
|---|---|---|
| `--bucket-type` | `data` | Bucket type: `data` or `vector` |
| `--vector-db-type` | `lance` | Vector DB engine: `lance`, `davinci` |
| `--lance-config` | _(none)_ | LanceDB configuration as JSON (e.g. `'{"subPath":"embeddings"}'`). Only valid with `--vector-db-type=lance`. |

Examples:

```shell
# Create a vector bucket with default LanceDB engine
noobaa obc create my-vector-obc --bucketclass vector-bc --bucket-type vector

# Create with explicit DB type and sub-path
noobaa obc create my-vector-obc --bucketclass vector-bc \
  --bucket-type vector --vector-db-type lance --lance-config '{"subPath":"embeddings"}'

# Vector bucket is also detected when only --vector-db-type is specified
noobaa obc create my-vector-obc --bucketclass vector-bc --vector-db-type lance
```

### Status

```shell
noobaa obc status <name>
```

For vector buckets, the status output includes:
- **Bucket Type**: `vector`
- **Vector DB Type**: the engine used (e.g., `lance`)
- **Vector service info**: internal and external DNS endpoints for the vector service

### List

```shell
noobaa obc list
```

The list output includes a `BUCKET-TYPE` column showing `object` or `vector` for each OBC.

Example output:

```
NAMESPACE  NAME             BUCKET-NAME       STORAGE-CLASS      BUCKET-CLASS         BUCKET-TYPE  PHASE
default    my-data-obc      my-data-obc-xxx   noobaa.noobaa.io   noobaa-default-bc    object       Bound
default    my-vector-obc    my-vector-xxx     noobaa-vector      vector-bucket-class  vector       Bound
```

### Delete

```shell
noobaa obc delete <name>
```

Deletion works the same as for data buckets. The provisioner automatically detects the
bucket type from the ObjectBucket's `AdditionalState` and calls the appropriate delete API.

## Validation Summary

| Layer | What is validated |
|---|---|
| `validateBucketType` (OBC validation) | `bucketType` ∈ {data, vector}, `vectorDBType` ∈ {lance, davinci, opensearch}, opensearch rejected as unsupported, `vectorDBType` rejected when `bucketType=data`, `lanceConfig` must be valid JSON and only allowed with lance |
| `ValidateVectorBucketBucketClass` (provisioner runtime) | BucketClass has NamespacePolicy of type Single with a resource reference |
| `ValidateVectorBucketNamespaceStore` (provisioner runtime) | NamespaceStore is of type NSFS with valid config |
