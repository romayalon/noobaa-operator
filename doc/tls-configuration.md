# TLS Configuration for NooBaa Endpoints

NooBaa supports configuring TLS version, cipher suites, and key exchange group preferences for endpoint HTTPS servers. This enables TLS 1.3 enforcement and prepares the platform for Post-Quantum Cryptography (PQC) readiness.

## NooBaa CR Interface

TLS configuration is set under **`spec.security.tlsConfig`** on the NooBaa custom resource. The field type is OCS [`TLSConfig`](https://github.com/red-hat-storage/ocs-tls-profiles/blob/main/api/v1/tlsprofile_types.go) (`github.com/red-hat-storage/ocs-tls-profiles/api/v1`). The StorageCluster propagates the platform API Server TLS profile here and NooBaa applies it to endpoint HTTPS servers. **`version`**, **`ciphers`** (at least one), and **`groups`** (at least one) are all required together when `tlsConfig` is set—the same rules as OCS. To use defaults, omit `tlsConfig` entirely (do not set `tlsConfig: {}`).

```yaml
apiVersion: noobaa.io/v1alpha1
kind: NooBaa
metadata:
  name: noobaa
  namespace: openshift-storage
spec:
  security:
    tlsConfig:
      version: "TLSv1.3"
      ciphers:
        - "TLS_AES_128_GCM_SHA256"
        - "TLS_AES_256_GCM_SHA384"
      groups:
        - "X25519MLKEM768"
        - "X25519"
        - "secp256r1"
```

### Field Reference

| Field | Type | Values | Description |
|---|---|---|---|
| `version` | string **(required)** | `TLSv1.2`, `TLSv1.3` | TLS protocol version (OCS `TLSProtocolVersion`); propagated as `TLS_MIN_VERSION` for endpoints. |
| `ciphers` | []string **(required, min 1)** | IANA TLS cipher suite names | Cipher algorithms (OCS `TLSCipherSuite`); the operator maps them to OpenSSL names for `TLS_CIPHERS`. |
| `groups` | []string **(required, min 1)** | `X25519`, `secp256r1`, `secp384r1`, `secp521r1`, `X25519MLKEM768`, `SecP256r1MLKEM768`, `SecP384r1MLKEM1024` | Key exchange groups (OCS `TLSGroupName`); the operator maps them to OpenSSL curve names for `TLS_GROUPS`. |

### Defaults

When no fields are set, NooBaa uses Node.js / OpenSSL defaults:
- TLS 1.2 minimum (Node.js default)
- All OpenSSL-supported ciphers
- OpenSSL default group negotiation (includes ML-KEM on OpenSSL 3.5+)

### PQC Readiness Out of the Box

Since Node.js v24.7.0, TLS 1.3 and X25519MLKEM768 are negotiated by default when the underlying OpenSSL supports them (see [release notes](https://github.com/nodejs/node/releases/tag/v24.7.0)). NooBaa has upgraded to Node.js v24.13.0 (see [PR #9458](https://github.com/noobaa/noobaa-core/pull/9458)), which includes this behavior. This means that even without setting any custom TLS properties on the NooBaa CR, NooBaa is PQC-ready — clients that support post-quantum key exchange will automatically negotiate X25519MLKEM768 with the endpoint.

## Applying Configuration

### Set TLS 1.3 with PQC groups

```bash
kubectl patch noobaa noobaa -n openshift-storage --type merge -p '{
  "spec": {
    "security": {
      "tlsConfig": {
        "version": "TLSv1.3",
        "ciphers": ["TLS_AES_128_GCM_SHA256", "TLS_AES_256_GCM_SHA384"],
        "groups": ["X25519MLKEM768", "X25519", "secp256r1"]
      }
    }
  }
}'
```

### Revert to defaults (remove TLS configuration)

```bash
kubectl patch noobaa noobaa -n openshift-storage --type json -p '[
  {"op": "remove", "path": "/spec/security/tlsConfig"}
]'
```

When the configuration is removed, the operator sets the corresponding environment variables to empty strings on the next reconciliation, reverting the endpoint to Node.js defaults.

## How It Works

The operator reconciler (`SetDesiredDeploymentEndpoint`) maps the CR fields to endpoint pod environment variables:

| CR Field | Environment Variable | Example value |
|---|---|---|
| `version: TLSv1.3` | `TLS_MIN_VERSION` | `TLSv1.3` |
| `ciphers: [...]` | `TLS_CIPHERS` | `TLS_AES_256_GCM_SHA384:TLS_AES_128_GCM_SHA256` (OpenSSL-style) |
| `groups: [...]` | `TLS_GROUPS` | `X25519MLKEM768:x25519:prime256v1` (OpenSSL-style curve names) |

These env vars are defined in the endpoint deployment template and updated in-place on every reconciliation cycle. Updating the CR triggers a deployment rollout with the new TLS settings.

## Verification

### Check environment variables in the endpoint pod

```bash
kubectl exec -it noobaa-endpoint-<suffix> -n openshift-storage -- bash -c 'env | grep TLS'
```

Expected output when TLS is configured:

```
TLS_MIN_VERSION=TLSv1.3
TLS_CIPHERS=TLS_AES_128_GCM_SHA256:TLS_AES_256_GCM_SHA384
TLS_GROUPS=X25519MLKEM768:x25519:prime256v1
```

When TLS configuration is removed from the CR, the variables will be present but empty.

## Manual Testing

### Using openssl

Verify TLS 1.3 with PQC group negotiation (requires OpenSSL 3.5+ with ML-KEM support):

```bash
echo Q | openssl s_client -connect <endpoint-address>:6443 -tls1_3 -groups X25519MLKEM768 2>&1 | grep "Negotiated TLS1.3 group: X25519MLKEM768"
```

A successful match confirms the endpoint is negotiating TLS 1.3 with the X25519MLKEM768 key exchange group.

> **Note:** The `echo Q |` pipes a quit command so `s_client` terminates cleanly after the handshake. Without it, the server may close the connection causing an `unexpected eof` error.

### Using testssl.sh

Run a TLS scan against the endpoint using the `testssl.sh` Docker image:

```bash
docker run --rm -it \
  -v <path-local>:<path-in-container> \
  ghcr.io/testssl/testssl.sh \
  --forward-secrecy --protocols --server-preference --client-simulation \
  --quiet --wide \
  --jsonfile <path-in-container>/testssl_output.json \
  <endpoint-address>
```

Key things to look for in the output:
- **Protocols**: Only TLS 1.3 should be listed as offered when `TLSv1.3` is set
- **Forward secrecy / key exchange groups**: Should reflect configured `groups` (CR uses OCS names; the endpoint sees OpenSSL names in `TLS_GROUPS`)
- **Server preference / ciphers**: Should match the configured `ciphers`
- **Client simulation**: Shows how various clients negotiate with the server

Parse the JSON output to verify TLS 1.3 and PQC key exchange:

```bash
jq -r '["TLS1_3", "FS_KEMs"][] as $id | (map(select(.id == $id))[0] // {id: $id, severity: "not found", finding: ""}) | "\(.id): \(.severity) \(.finding)"' <path-local>/testssl_output.json
```

Expected output when TLS 1.3 with PQC is configured:

```
TLS1_3: OK offered
FS_KEMs: OK X25519MLKEM768
```


## TODO
- Add PQC scanner info

### Testing the admission webhook server

Pre-requisites - make sure that `ENABLE_NOOBAA_ADMISSION == "true"` or on CLI installation `--admission` in order to start the admission webhook server.

The admission webhook runs on port 8080 inside the operator pod.

#### Using openssl (with port-forward)

In one terminal, start a port-forward:

```bash
kubectl port-forward deploy/noobaa-operator -n openshift-storage 8080:8080
```

In another terminal, verify TLS 1.3 with PQC group negotiation:

```bash
echo Q | openssl s_client -connect localhost:8080 -tls1_3 -groups X25519MLKEM768 2>&1 | grep "Negotiated TLS1.3 group: X25519MLKEM768"
```

#### Using testssl.sh (from within the cluster)

Run testssl.sh as a pod inside the cluster to connect directly to the operator service without port-forwarding:

```bash
kubectl run testssl-webhook -it --restart=Never \
  --image=ghcr.io/testssl/testssl.sh -- \
  --forward-secrecy --protocols --server-preference --client-simulation \
  --quiet --wide \
  https://noobaa-operator.openshift-storage.svc.cluster.local:8080
```

#### Using testssl.sh with JSON output (from within the cluster)

```bash
kubectl run testssl-webhook -it --restart=Never \
  --image=ghcr.io/testssl/testssl.sh -- \
  --forward-secrecy --protocols --server-preference --client-simulation \
  --quiet --wide \
  --jsonfile /tmp/testssl_webhook_output.json \
  https://noobaa-operator.openshift-storage.svc.cluster.local:8080
```

To extract the JSON results, copy them from the pod before it terminates, or parse the console output directly.
