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
├── policydoc/
│   └── policydoc.go             # Compare JSON policy documents by meaning
├── provider/
│   ├── provider.go              # Schema, Configure, Resources(), DataSources()
│   ├── resource_*_test.go       # Acceptance tests, one per resource
│   ├── sweep_test.go            # Sweepers with dependency ordering
│   └── provider_test.go         # PreCheck, ProtoV6ProviderFactories
├── validators/
│   └── validators.go            # BucketName, IAMName, JSONDocument, ...
├── datasources/
│   └── <name>/datasource.go     # 11 data sources
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
        workflow_replication/                               # S3 replication configuration
```

Each resource is a package with:
- `model.go` — struct with `tfsdk` tags
- `resource.go` — schema, Configure, CRUD, import
- `schema_test.go` — schema validation test (unit)

## Four Clients, Four Auth Models

| Client | Auth | Wire Format | Used By |
|--------|------|-------------|---------|
| `ManagementClient` | OIDC bearer (`X-Authentication-Token` header) | JSON/REST | account create/delete/key generation, locations, endpoints, replication, bucket_workflows data source |
| `IAMClient` | SigV4 (service `iam`); unsigned with `WebIdentityToken` for account lookup | XML / form-encoded; JSON for `GetRolesForWebIdentity` | account read + account data sources, users, user_access_key, user_policy, user_policy_attachment, group, group_membership, group_policy, group_policy_attachment, policy, role, role_policy_attachment |
| `S3Client` | SigV4 (service `s3`) | XML / REST | bucket, bucket_encryption, bucket_policy, bucket_tagging, workflow_expiration, workflow_transition, workflow_replication |
| `STSClient` | SigV4 (service `sts`); unsigned with `WebIdentityToken` for `AssumeRoleWithWebIdentity` | XML | per-account credentials for every account-scoped resource (via `AccountCredentialSource`), caller_identity data source, assumed_role_credentials ephemeral |

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
- **STS**: `s3.<host>` → `sts.<host>`, or `management.<host>` → `sts.<host>` when no S3 endpoint is configured

Only the management endpoint and (optionally) the S3 endpoint are configured directly. This mirrors ARTESCA's DNS convention for its four public surfaces.

### Per-account credentials

IAM and S3 operations must be signed with credentials of the owning account. Account-scoped resources name that account with `account_name`; `client.AccountCredentialSource` turns it into temporary credentials:

1. Resolve the account ID with IAM `GetRolesForWebIdentity` (OIDC token, unsigned).
2. Call STS `AssumeRoleWithWebIdentity` on `arn:aws:iam::<id>:role/scality-internal/storage-manager-role` with the OIDC token.
3. Cache the resulting access key / secret key / session token per account until 5 minutes before expiry (1-hour credentials).

IAM and S3 requests carry the session token as the signed `X-Amz-Security-Token` header. No account keys are stored in configuration or state. The `artesca_account` resource drops an account's cached credentials when it deletes the account.

The `internal/creds` package holds the shared `account_name` attribute and the `<account_name>/<id>` import ID parsing.

## Overlay reads

Almost all *infrastructure* reads — locations, endpoints, replication streams — go through a **single** management API call: `GET /config/overlay/view/{instanceId}`. Resources don't each have their own GET-by-id endpoint. When a resource's `Read()` runs, it fetches the whole overlay view and finds itself by name/id within the response.

Implication: the management client batches (and where appropriate caches) that overlay fetch. Don't add a "GET this one resource" call expecting an endpoint to exist — it usually doesn't on the management side.

## Workflow resource reads

All three bucket workflow resources are S3 bucket configuration under the hood: `artesca_bucket_workflow_expiration` and `_transition` manage lifecycle rules (`Get/PutBucketLifecycle`), and `_replication` manages replication rules (`Get/Put/DeleteBucketReplication`). Each resource owns one rule; Create/Update/Delete read the bucket's configuration, merge the change, and write it back under a per-bucket lock (`S3Client.LockBucket`, shared by every bucket config write). Rules the provider doesn't own are written back exactly as read. Read finds the rule by ID, so deletion and out-of-band changes are detected.

The management API's workflow endpoints show the same rules (a replication rule's ID is its workflow `streamId`), but they don't return anything the S3 configuration lacks, and the `name`/`version` a workflow is created with are not stored.

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
| `SSEAlgorithm()` | `AES256` | sse_algorithm on bucket_encryption |
| `LocationName()` | 3+ chars, lowercase + numbers + hyphens, starts with a letter | location name, endpoint location_name |
| `RuleID()` | 1-255 chars | rule_id on lifecycle and replication workflows |
| `RoleSessionName()` | Letters, numbers, `_=,.@-` | role_session_name on assumed_role_credentials |
| `MapSizeAtMost(max)` | At most `max` entries | tags on bucket_tagging (50) |
| `Int64AtLeast(min)` | ≥ `min` | lifecycle day counts |
| `Int64Between(min, max)` | `min`–`max` inclusive | duration_seconds on assumed_role_credentials |

## Testing

Unit tests (`*_test.go`) cover the client layer, validators, and per-resource schema. Acceptance tests (`resource_*_test.go` under `internal/provider/`) are gated behind `TF_ACC=1` and require a live cluster. Sweepers with dependency ordering live in `sweep_test.go`.

## Resource Patterns

### Atomic Create

`artesca_account` saves state immediately after creation, before generating access keys. If key generation fails, the account is still tracked in state and can be destroyed or retried.

### RequiresReplace

Fields the API cannot update in-place use `stringplanmodifier.RequiresReplace()`. Terraform destroys and recreates the resource when these change. Notable case: `assume_role_policy_document` on `artesca_role` is immutable (ARTESCA does not implement `UpdateAssumeRolePolicy`).

### State-preserved Read

Resources where the API cannot return secrets after creation (`access_key`, `secret_key`) preserve those fields from prior state in `Read`. Don't overwrite from the (empty) API response.

### Policy documents

Resources that hold a JSON policy (`bucket_policy`, `policy`, `user_policy`, `group_policy`, the `role` trust policy) read it on every refresh and compare it with state using `policydoc.Refresh`, which keeps the state value when the two differ only in whitespace or key order. `policydoc.EquivalenceModifier` does the same for the plan, ahead of any `RequiresReplace`.

### Bucket sub-resources

Bucket features (policy, encryption, tagging) are separate resources rather than inline attributes on `artesca_bucket`. This keeps each resource focused and allows independent lifecycle management. The `bucket_name` attribute on each sub-resource uses `RequiresReplace`.

## Retry behavior

`S3Client.doSignedRequest` retries 502/503/504 with exponential backoff (500 ms → 8 s, 4 attempts). 500 is **not** retried — treated as a real server error. `CreateBucket` has a separate longer (5 min) retry loop for `InvalidLocationConstraint` to handle location propagation across the cluster. Both loops co-exist.
