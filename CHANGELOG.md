# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- **Imported locations had empty `details` (#71).** Read only refreshed `details` fields already in state, and imported state has none, so `endpoint`, `sts_endpoint`, `bucket_name`, `region` and `access_key` stayed empty and the first plan showed changes. The first Read after import now takes every value ARTESCA returns (secrets excepted). `details.bucket_match` now defaults to `false`, since ARTESCA omits it when false; state written by earlier builds is upgraded to `false` so a plan without refresh doesn't replace the location (#73). The docs warn that a `bucket_match = true` location shared between buckets can lose data.
- **CRR locations could not be fully configured (#60).** ARTESCA requires an STS endpoint for `location-scality-crr-v1`; `artesca_location` and `data.artesca_location` now have `details.sts_endpoint`, required at plan time for CRR. Creating a location type with no region (such as CRR) also failed with "Provider returned invalid result object after apply"; `details.region` is now null when ARTESCA returns none.
- **Workflow resources rewrote other rules on the same bucket (#59).** When adding, changing, or removing their rule, `artesca_bucket_workflow_expiration`, `_transition`, and `_replication` wrote the bucket's other rules back with only the fields the provider models, dropping tag filters, dates, noncurrent-version actions, and replication `StorageClass`. A tag-filtered expiration rule became a rule for the whole bucket. Other rules are now written back exactly as read.
- **Transitions with `trigger_delay_days = 0` were rejected (#63).** The provider left `Days` out of the lifecycle XML when it was 0, and ARTESCA returned `MalformedXML`. `Days` is now always sent.
- **Accounts reported as deleted right after creation.** `artesca_account`, `data.artesca_account`, and `data.artesca_accounts` looked accounts up in the management overlay view, which does not list them. They now use IAM `GetRolesForWebIdentity` with the provider's OIDC token (#35). `artesca_account.arn` is now the account root ARN (`arn:aws:iam::<id>:root`).
- **Bucket configs lost when several were applied at once (#46).** ARTESCA can silently drop some of several config writes made concurrently to the same bucket, and OpenTofu creates bucket sub-resources in parallel. All bucket-config writes (tagging, encryption, policy, versioning, lifecycle, replication) and bucket deletion are now serialized per bucket inside the provider.
- **Imported buckets and lifecycle workflows lost attributes.** `artesca_bucket` only read `location_constraint` when it was already in state, and `artesca_bucket_workflow_expiration` / `_transition` only read the filter prefix when a `filter` block was already in state, so both were empty after import and out-of-band changes went undetected. Read now always reads them; an unset prefix stays null.
- **State from before `account_name` failed to refresh (#45).** Account-scoped resources are now at schema version 1 with a state upgrader: `account_name` is resolved from the access key pair in v0 state (STS `GetCallerIdentity`, then the account list), and `artesca_bucket_workflow_replication` v0 state is mapped to the S3 rule shape (`workflow_id` → `rule_id`, `destination.bucket_name` → `destination_bucket_name`, `source.prefix` → `filter.object_key_prefix`, `account_id` → `account_name`). If the keys no longer work, the error explains how to `tofu state rm` and re-import. `tests/upgrade/run.sh` creates resources with an older build and checks that planning with the current one shows no changes.

### Changed (breaking)

- **Plan-time validation aligned with ARTESCA docs (#61).** `artesca_location.name` and `artesca_endpoint.location_name` accept only lowercase letters, numbers, and hyphens, starting with a letter (at least 3 characters); periods and a leading digit were accepted before and rejected by ARTESCA. New checks: `artesca_bucket_workflow_replication.rule_id` 1–255 characters, `artesca_bucket_tagging.tags` at most 50, `ephemeral.artesca_assumed_role_credentials` `duration_seconds` 900–43200 and `role_session_name` letters, numbers and `_=,.@-`. Required `details` fields are now checked for the location types ARTESCA documents: added `location-scaleway-glacier-v1`, `location-ovh-cold-archive-v1`, `location-versity-tape-archive-v1`; dropped `location-do-spaces-v1`, `location-ceph-radosgw-s3-v1`, `location-scality-hdclient-v2`, `location-nfs-mount-v1` (passed to ARTESCA unchecked).
- **`artesca_location.details.server_side_encryption` is only accepted for `location-aws-s3-v1` (#53).** Other location types don't support it; setting it now fails at plan time.

- **`artesca_bucket_workflow_replication` manages an S3 replication rule (#2).** It reads and writes the bucket's S3 replication configuration (`Get/Put/DeleteBucketReplication`), merging its rule with others on the same bucket, like the expiration and transition resources do with lifecycle. The management API never stored a replication workflow's `name` or `version`, so they could not be read back. New shape: `account_name`, `bucket_name`, `destination_bucket_name`, `enabled`, optional `filter { object_key_prefix }`, computed `rule_id`; import ID `<account_name>/<bucket_name>/<rule_id>`. Removed `instance_id`, `account_id`, `name`, `version`, `workflow_id`, and the `source` / `destination` blocks.

### Removed

- **`email` and `access_key` from `data.artesca_account`, and `email` from `data.artesca_accounts`.** The account listing does not return them (#35).
- **`account_access_key` / `account_secret_key`** from the 17 account-scoped resources and `data.artesca_group`, `data.artesca_policy`, `data.artesca_role`, `data.artesca_user`. Use `account_name` (#36).
- **`policy` from `ephemeral.artesca_assumed_role_credentials`.** ARTESCA accepts STS session policies but does not enforce them (#64).

### Added

- **Temporary per-account credentials (#36).** Account-scoped resources and data sources take `account_name`. The provider resolves the account and obtains credentials on its storage-manager role via STS `AssumeRoleWithWebIdentity` with the provider's OIDC token, caching them per account until shortly before expiry. IAM and S3 requests sign the session token. The STS endpoint is now always configured (derived from the management endpoint when `s3_endpoint` is unset).
- **Import support** for 17 previously-unimportable resources: `artesca_bucket`, all bucket sub-resources (`_policy`, `_tagging`, `_encryption`), all IAM resources (users, groups, roles, policies, attachments, memberships), and all workflow resources (expiration, transition, replication). Import IDs for account-scoped resources start with the account name (`<account_name>/<id>`).

### Changed

- **Docs: ARTESCA constraints (#62).** The provider needs a user with the `StorageManager` role; buckets must be empty to delete; locations: at most 10, versioned target bucket for Amazon S3 and RING S3, truststore CA for TLS, no delete while holding data, CRR is replication-only, don't change `us-east-1`; accounts: deletion prerequisites and the three default roles; IAM names are case-insensitive.
- **`data.artesca_bucket_workflows` takes `account_name` (#58),** like every other account-scoped data source. `account_id` and `instance_id` are removed; the provider resolves the account ID from the name.
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
