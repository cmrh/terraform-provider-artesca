# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **`internal/creds` package**: env-var fallback (`ARTESCA_ACCOUNT_ACCESS_KEY` / `ARTESCA_ACCOUNT_SECRET_KEY`) for per-account credentials, used during `tofu import` when state attributes are empty.
- **Import support** for 17 previously-unimportable resources: `artesca_bucket`, all bucket sub-resources (`_policy`, `_tagging`, `_encryption`), all IAM resources (users, groups, roles, policies, attachments, memberships), and all workflow resources (expiration, transition, replication).

### Changed

- **Release pipeline** moved from a self-hosted runner to `ubuntu-latest`. Build artifacts are handed to the signing job via immutable within-run workflow artifacts, and every checksum is re-verified before signing.
- **Release binary naming**: the binary inside each zip now uses the `v`-prefixed version (`terraform-provider-artesca_v0.4.0`) — the archive filename remains unprefixed (`terraform-provider-artesca_0.4.0_linux_amd64.zip`). This matches the OpenTofu / Terraform Registry conventions.

### Added (CI / security)

- **CodeQL** analysis on push, pull-request, and a weekly schedule.
- **gitleaks** secret-detection scan on push and pull-request.
- **gosec** Go security scan on push and pull-request.
- **`-race`** detector added to unit test runs.

## [0.4.0] - unreleased

### Added

#### IAM resources
- **artesca_group**: IAM group management within an account.
- **artesca_group_membership**: Attach an IAM user to one or more groups.
- **artesca_group_policy**: Inline IAM policy attached to a group.
- **artesca_group_policy_attachment**: Attach a managed policy to a group.
- **artesca_policy**: Managed IAM policy (account-scoped).
- **artesca_role**: IAM role with trust policy document.
- **artesca_role_policy_attachment**: Attach a managed policy to a role.
- **artesca_user_policy_attachment**: Attach a managed policy to a user.

#### Bucket sub-resources
- **artesca_bucket_policy**: Bucket policy via S3 `PutBucketPolicy` / `GetBucketPolicy`.
- **artesca_bucket_tagging**: Bucket tag set management.
- **artesca_bucket_encryption**: Server-side encryption configuration (SSE-S3 / `AES256`) via `PutBucketEncryption` / `GetBucketEncryption`.

#### Data sources
- **data.artesca_account** / **data.artesca_accounts**: Look up a single account by name or list all accounts.
- **data.artesca_location** / **data.artesca_locations**: Look up a single location by name or list all locations.
- **data.artesca_endpoints**: List all data-service endpoints.
- **data.artesca_user**, **data.artesca_group**, **data.artesca_role**, **data.artesca_policy**: Look up existing IAM objects without managing them.
- **data.artesca_caller_identity**: Resolve the identity (`account`, `user_id`, `arn`) associated with an access key via STS `GetCallerIdentity`.
- **data.artesca_bucket_workflows**: List the workflows (replication / expiration / transition) configured on a bucket via management-API workflow search.
- **data.artesca_instance**: Connected ARTESCA instance metadata + health status.

#### Ephemeral resources
- **ephemeral.artesca_assumed_role_credentials**: Mint short-lived role credentials via STS `AssumeRole`. Session tokens are not persisted to state.

### Changed

- **artesca_bucket_workflow_replication**: `Read()` now uses workflow search to detect drift (deletion, `enabled` flips, source/destination changes). Previously it preserved state as-is. `name` and `version` are preserved from state because the workflow-search endpoint returns them as `null` for replication workflows.
- **S3 client**: 502 / 503 / 504 responses are retried with exponential backoff (4 attempts, 500 ms → 8 s). 500 is not retried (treated as a real server error). `CreateBucket` has a separate longer retry loop for `InvalidLocationConstraint` to handle location propagation.

### Added (infrastructure)

- STS client (`internal/client/sts.go`) with `AssumeRole` and `GetCallerIdentity`. STS endpoint is derived from the configured S3 endpoint (`s3.` → `sts.`).
- Provider-level `EphemeralResources()` wiring.

## [0.3.0] - 2026-05-04

### Added

- **Provider**: OIDC + management API authentication with auto-discovered instance ID.
- **artesca_account**: Account management via management API with credential generation.
- **artesca_bucket**: S3 bucket creation with versioning and location constraints.
- **artesca_location**: Storage location management (AWS S3, Azure, GCP, Scality RING, and other S3-compatible backends).
- **artesca_endpoint**: S3 data service endpoint mapping.
- **artesca_replication**: Config-scoped overlay replication streams with server-managed versioning.
- **artesca_user**: IAM user management within accounts.
- **artesca_user_access_key**: IAM access key pair generation.
- **artesca_user_policy**: Inline IAM policy attachment.
- **artesca_bucket_workflow_expiration**: Object expiration lifecycle workflows.
- **artesca_bucket_workflow_transition**: Object transition lifecycle workflows.
- **artesca_bucket_workflow_replication**: Bucket-scoped replication workflows.
- Import support for account, endpoint, location, and replication resources.
- Input validators for account/bucket/IAM names, email, hostname, JSON documents.
- Acceptance tests for all 11 resources with `CheckDestroy` verification.
- GPG-signed release artifacts with SHA256SUMS.
