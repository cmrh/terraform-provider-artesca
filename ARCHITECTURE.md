# Architecture

## Overview

```
OpenTofu / Terraform
        │
        │  Plugin Protocol (gRPC)
        ▼
┌─────────────────────────────────────────┐
│           ARTESCA Provider              │
│                                         │
│  provider.go ── config, env vars,       │
│                 resource registration   │
│                                         │
│  ┌────────────────────────────────────┐ │
│  │         Client Layer               │ │
│  │                                    │ │
│  │  ManagementClient (OIDC bearer)    │ │
│  │  IAMClient        (SigV4, "iam")   │ │
│  │  S3Client         (SigV4, "s3")    │ │
│  │  STSClient        (SigV4, "sts")   │ │
│  └────────────────────────────────────┘ │
│                                         │
│  ┌────────────────────────────────────┐ │
│  │      Resources (22)  +  Data (12)  │ │
│  │        +  Ephemeral (1)            │ │
│  │  Each: model.go + resource.go      │ │
│  └────────────────────────────────────┘ │
└─────────────────────────────────────────┘
        │           │           │           │
        ▼           ▼           ▼           ▼
   Management     IAM         S3         STS
```

## Directory Layout

```
internal/
├── client/
│   ├── provider_clients.go      # ProviderClients bundle (Mgmt + IAM + S3 + STS)
│   ├── oidc.go                  # OIDC token source, caches + refreshes
│   ├── sigv4.go                 # SigV4 primitives shared by IAM/S3/STS
│   ├── management.go            # ManagementClient: OIDC bearer, JSON/REST
│   ├── management_workflow.go   # /instance/{id}/account/{acct}/bucket/{name}/workflow/*
│   ├── management_replication.go # /config/{id}/replication overlay stream
│   ├── iam.go                   # IAMClient: SigV4 (service "iam"), form-encoded
│   ├── s3.go                    # S3Client: SigV4 (service "s3"), retry logic
│   ├── s3_lifecycle.go          # Get/Put/DeleteBucketLifecycle
│   ├── sts.go                   # STSClient: AssumeRole, GetCallerIdentity
│   └── ...
├── creds/
│   └── resolve.go               # Env-var fallback for per-account ak/sk on import
├── provider/
│   ├── provider.go              # Schema, Configure, Resources(), DataSources()
│   ├── resource_*_test.go       # Acceptance tests, one per resource
│   ├── sweep_test.go            # Sweepers with dependency ordering
│   └── provider_test.go         # PreCheck, ProtoV6ProviderFactories
├── validators/
│   └── validators.go            # BucketName, IAMName, JSONDocument, ...
├── datasources/
│   └── <name>/datasource.go     # 12 data sources
├── ephemeral/
│   └── assumed_role_credentials/ # 1 ephemeral resource (STS session tokens)
└── resources/
    ├── account/, endpoint/, location/, replication/       # Management API
    ├── bucket/, bucket_encryption/, bucket_policy/,
    │   bucket_tagging/                                     # S3 API
    ├── user/, user_access_key/, user_policy/,
    │   user_policy_attachment/,
    │   group/, group_membership/, group_policy/,
    │   group_policy_attachment/,
    │   policy/, role/, role_policy_attachment/            # IAM API
    └── workflow_expiration/, workflow_transition/,
        workflow_replication/                               # S3 lifecycle + Mgmt search
```

Each resource is a package with:
- `model.go` — struct with `tfsdk` tags
- `resource.go` — schema, Configure, CRUD, import
- `schema_test.go` — schema validation test (unit)

## Four Clients, Four Auth Models

| Client | Auth | Wire Format | Used By |
|--------|------|-------------|---------|
| `ManagementClient` | OIDC bearer (`X-Authentication-Token` header) | JSON/REST | accounts, locations, endpoints, replication, workflow_replication |
| `IAMClient` | SigV4 (service `iam`) | XML / form-encoded | users, user_access_key, user_policy, user_policy_attachment, group, group_membership, group_policy, group_policy_attachment, policy, role, role_policy_attachment |
| `S3Client` | SigV4 (service `s3`) | XML / REST | bucket, bucket_encryption, bucket_policy, bucket_tagging, workflow_expiration, workflow_transition |
| `STSClient` | SigV4 (service `sts`) | XML | caller_identity data source, assumed_role_credentials ephemeral |

The provider bundles all four in `ProviderClients` (`provider_clients.go`). Each resource extracts the client it needs in its `Configure` method.

### Standard library clients

Every client is built directly on the Go standard library. The provider does not depend on `aws-sdk-go`, `aws-sdk-go-v2`, or any third-party S3/IAM/STS client.

| Concern | Implementation |
|---|---|
| HTTP transport | `net/http` |
| SigV4 signing | `crypto/hmac`, `crypto/sha256`, `encoding/hex` (shared in `sigv4.go`) |
| OIDC token exchange | `net/http` + `encoding/json` (`oidc.go`) |
| Request/response bodies | `encoding/xml` for S3/STS, form-encoded for IAM, `encoding/json` for Management |

Each client is a few hundred lines, debuggable end-to-end with `TF_LOG=trace`, and tied directly to ARTESCA's control-plane shape (per-account credentials, OIDC-authenticated management, S3-derived endpoints for STS).

### Endpoint derivation

The IAM and STS endpoints are **derived** from other configured endpoints:

- **IAM**: `management.<host>` → `iam.<host>` (replaces the leading subdomain)
- **STS**: `s3.<host>` → `sts.<host>`

Only the management endpoint and (optionally) the S3 endpoint are configured directly. This mirrors ARTESCA's DNS convention for its four public surfaces.

### Per-account credentials

IAM- and S3-touching resources use per-account access keys passed as resource attributes (`account_access_key`, `account_secret_key`), not the provider-level admin OIDC token. This is because standard AWS IAM/S3 operations require the owning account's keys — the OIDC-authenticated management API creates and rotates these keys but doesn't sign S3 or IAM requests itself.

The `internal/creds` package resolves these attributes with an env-var fallback (`ARTESCA_ACCOUNT_ACCESS_KEY` / `ARTESCA_ACCOUNT_SECRET_KEY`) so `tofu import` works — during import the framework state starts empty and the Read call would otherwise fail without credentials.

## Overlay reads

Almost all *infrastructure* reads — locations, endpoints, replication streams — go through a **single** management API call: `GET /config/overlay/view/{instanceId}`. Resources don't each have their own GET-by-id endpoint. When a resource's `Read()` runs, it fetches the whole overlay view and finds itself by name/id within the response.

Implication: the management client batches (and where appropriate caches) that overlay fetch. Don't add a "GET this one resource" call expecting an endpoint to exist — it usually doesn't on the management side.

## Workflow resource reads

`artesca_bucket_workflow_expiration` and `_transition` round-trip through the S3 lifecycle API (`GetBucketLifecycle`). `_replication` round-trips through `POST /workflow/search`. All three have functional `Read()` paths that detect deletion and out-of-band changes.

One caveat: workflow_search returns `name` and `version` as `null` for replication entries, so `workflow_replication.Read()` preserves those two fields from prior state (upstream bug, tracked in the repo issue tracker).

## Input Validation

The `internal/validators` package provides reusable schema validators:

| Validator | Rules | Used On |
|-----------|-------|---------|
| `AccountName()` | 1-128 chars, alphanumeric + hyphens | account name |
| `BucketName()` | 3-63 chars, lowercase + numbers + hyphens + periods | bucket_name across all bucket resources |
| `Email()` | Standard email syntax | account email |
| `Hostname()` | RFC-1123 hostname | endpoint hostname |
| `IAMName(maxLen)` | 1-maxLen chars, alphanumeric + `_+=,.@-` | user, group, role, policy names |
| `IAMUsername()` | Same rules as IAMName, maxLen 64 | username |
| `IAMPolicyName()` | Same rules as IAMName, maxLen 128 | policy_name on user_policy, group_policy |
| `JSONDocument()` | Valid JSON | policy documents, trust policies |
| `SSEAlgorithm()` | One of the supported SSE algorithms | sse_algorithm on bucket_encryption |

## Testing

Unit tests (`*_test.go`) cover the client layer, validators, and per-resource schema. Acceptance tests (`resource_*_test.go` under `internal/provider/`) are gated behind `TF_ACC=1` and require a live cluster. Sweepers with dependency ordering live in `sweep_test.go`.

## Resource Patterns

### Atomic Create

`artesca_account` saves state immediately after creation, before generating access keys. If key generation fails, the account is still tracked in state and can be destroyed or retried.

### RequiresReplace

Fields the API cannot update in-place use `stringplanmodifier.RequiresReplace()`. Terraform destroys and recreates the resource when these change. Notable case: `assume_role_policy_document` on `artesca_role` is immutable (ARTESCA does not implement `UpdateAssumeRolePolicy`).

### State-preserved Read

Resources where the API cannot return secrets after creation (`access_key`, `secret_key`) preserve those fields from prior state in `Read`. Don't overwrite from the (empty) API response.

### Bucket sub-resources

Bucket features (policy, encryption, tagging) are separate resources rather than inline attributes on `artesca_bucket`. This keeps each resource focused and allows independent lifecycle management. The `bucket_name` attribute on each sub-resource uses `RequiresReplace`.

## Retry behavior

`S3Client.doSignedRequest` retries 502/503/504 with exponential backoff (500 ms → 8 s, 4 attempts). 500 is **not** retried — treated as a real server error. `CreateBucket` has a separate longer (5 min) retry loop for `InvalidLocationConstraint` to handle location propagation across the cluster. Both loops co-exist.
