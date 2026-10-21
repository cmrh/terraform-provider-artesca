# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Cross-region replication destinations in `artesca_bucket_workflow_replication` (#143).** The new `destination_role` argument sets the IAM role on the remote site for a `location-scality-crr-v1` destination. The provider writes the bucket's replication `Role` as the source and destination roles while any rule targets a CRR location, and drops the destination role when the last such rule is removed.
- **Object lock (#146).**
  - `artesca_bucket.object_lock_enabled` creates the bucket with object lock, which also enables versioning. It can only be set at creation, so changing it replaces the bucket; `versioning_enabled = false` with it is rejected. When unset, it reports the bucket's state.
  - New `artesca_bucket_object_lock` resource sets an object-lock bucket's default retention: `retention_mode` (`GOVERNANCE` or `COMPLIANCE`) and exactly one of `retention_days` or `retention_years`. Destroying it removes the default retention.
  - `artesca_bucket.force_destroy` on an object-lock bucket deletes versions under GOVERNANCE retention; a version still under COMPLIANCE retention stops the destroy with an error naming it and its retain-until date.

### Changed

- **OIDC token requests are retried on 502/503/504 (#150)**, like IAM, STS and S3 requests, instead of failing the plan or apply on the first gateway error.

## [1.1.0] - 2026-10-19

### Added

- **`force_destroy` on `artesca_bucket` and `artesca_user` (#130).** `artesca_bucket.force_destroy` deletes every object version and delete marker before deleting the bucket. `artesca_user.force_destroy` detaches the user's policies and removes it from its groups before deleting it. Both default to `false` and must be applied before destroying.

### Fixed

- **`artesca_account.email` (#121, #125).**
  - An email with uppercase letters no longer fails with "inconsistent result after apply"; ARTESCA stores it lowercased, and state keeps the configured form.
  - A change to an email that is unknown until apply replaces the account, as a known change does.
  - `email = ""` is rejected at plan time instead of failing with an API 422.
- **Provider configuration (#120).**
  - An argument that is unknown at plan time is an error, instead of silently using the environment variable or default.
  - An argument set to `""` falls back to its environment variable and default, like an unset one.
- **`artesca_group_membership` group-name case (#126).** ARTESCA matches group names case-insensitively; a membership naming the group in different case is no longer removed from state and recreated on every plan.
- **`artesca_bucket` import in the default location (#127).** Unset, `""` and `us-east-1` are the same location, so configuring `location_constraint = "us-east-1"` on an imported default-location bucket no longer plans a replacement.
- **`artesca_bucket` create when enabling versioning fails (#128).** The bucket is saved to state before versioning is enabled, so a failure leaves it tainted and the next apply replaces it, instead of orphaning it and failing with "already owns bucket".
- **Empty values (#122).** Empty values that ARTESCA drops or replaces are rejected at validation instead of failing after apply with "inconsistent result":
  - `artesca_location`: `""` in any `details` string and `[]` in `details.repo_id` / `details.bootstrap_list`;
  - `artesca_replication`: `""` in `source.location`, `destination.location`, `destination.preferred_read_location`, `destination.role` and `destination.locations[].storage_class`;
  - `artesca_role` / `artesca_policy`: `description = ""`, which forced replacement after import.
- **Delete errors for objects with dependents (#130).** Destroying a bucket, user, group or role that still has objects, policies or members created outside Terraform reports what remains. Destroying a non-empty account says it is not empty instead of showing an empty 409.
- **`artesca_policy` document changes (#129).** Changing `policy_document` creates a new default policy version in place, deleting the oldest non-default version at ARTESCA's 5-version limit, instead of replacing the policy (which failed while it was attached). Destroy deletes non-default versions first, and reports a policy that is still attached.
- **`artesca_location` drift and validation (#124).**
  - Fields removed outside Terraform now show as drift: `size_limit_gb`, `server_side_encryption`, and the `details` fields ARTESCA returns for RING S3 and sproxyd locations (`access_key`, `bucket_name`, `endpoint`, `region`, `proxy_path`, `chord_cos`).
  - Invalid `repo_id` / `bootstrap_list` elements are reported instead of silently dropping the list.
  - `location-scality-hdclient-v2` requires `details.bootstrap_list`.
- **Lifecycle rules created outside Terraform (#123).** `artesca_bucket_workflow_expiration` and `_transition` read the prefix from `Filter/And` and the legacy top-level `Prefix`, and an update changes only the parts the resource manages: tag filters, noncurrent-version and multipart-upload actions, and the other workflow's action on the same rule are kept.

## [1.0.1] - 2026-10-16

### Fixed

- **`artesca_account.email` (#101).** Removing it from the configuration no longer replaces the account; the value stays in state. Creating an account without it fails at plan time instead of with an API 422.
- **`artesca_bucket` no longer adopts an existing bucket (#102).** Creating a bucket the account already owns fails with an error pointing to `tofu import`, instead of writing unverified attributes to state.
- **`artesca_bucket` refresh on 403 (#103).** A 403 from HeadBucket (another account's bucket, expired credentials, a deny policy) is now an error instead of removing the bucket from state. Only 404 means the bucket is gone.
- **`artesca_replication` no longer adopts an existing stream (#104).** When the API rejects a create because another stream has an overlapping prefix on the same source bucket, the error is reported instead of taking over a same-named stream.
- **`artesca_bucket.versioning_enabled` no longer defaults to `false` (#105).** When unset, versioning is not managed: the attribute reports the bucket's state, and a versioned bucket whose configuration omits it (after import, or after removing the line) is no longer suspended. Set `versioning_enabled = false` to suspend versioning.
- **`artesca_replication` updates (#106).** Updates send the stream's current version instead of `1`, so updates after the first no longer fail with `409 Conflict`.
- **Refresh of objects changed outside Terraform (#108).**
  - `artesca_account`: an account deleted and recreated under the same name leaves state instead of being adopted.
  - `artesca_bucket_policy`, `_tagging`, `_encryption` and `artesca_bucket_workflow_replication`: a deleted bucket removes the resource on refresh and on destroy instead of failing on `NoSuchBucket`; expiration and transition deletes do the same.
  - `artesca_user_access_key`: a deleted user removes the key from state.
  - `artesca_bucket_workflow_expiration` / `_transition`: create fails when the bucket already has a rule with that ID; update fails when the rule is missing.
  - `data.artesca_locations` returns locations sorted by name.
  - `artesca_bucket_tagging.tags` requires at least one tag; ARTESCA stores no tag set for an empty map, so `tags = {}` was recreated on every plan.
- **Robustness (#109).**
  - `artesca_bucket_policy`, `_tagging` and `_encryption` report a configuration error instead of crashing when `s3_endpoint` isn't set.
  - Propagation retries (bucket create, lifecycle, replication streams) stop when the operation is cancelled instead of waiting up to 5 minutes.
  - IAM and STS requests are retried on 502/503/504, like S3 requests.
  - IAM list calls (access keys, groups for user, attached user/group/role policies) follow pagination.
  - `ARTESCA_INSECURE_SKIP_VERIFY` accepts any boolean (`TRUE`, `False`, `t`, `0`, …); an invalid value is an error instead of meaning `false`.

## [1.0.0] - 2026-10-12

### Added

- **`artesca_bucket_workflow_replication.destination_location` (#85).** Rules replicate a bucket to a storage location, such as a bucket on ARTESCA's own storage (`us-east-1`) to a RING location. `destination_location` is required; `destination_bucket_name` is optional and only names the remote bucket for a cross-region replication location. Bucket-to-bucket rules without a location are no longer supported. Writes naming a location created moments earlier are retried while it propagates.
- **Temporary per-account credentials (#36).** Account-scoped resources and data sources take `account_name`. The provider resolves the account and obtains credentials on its storage-manager role via STS `AssumeRoleWithWebIdentity` with the provider's OIDC token, caching them per account until shortly before expiry. IAM and S3 requests sign the session token. The STS endpoint is now always configured (derived from the management endpoint when `s3_endpoint` is unset).
- **Import support** for 17 previously-unimportable resources: `artesca_bucket`, all bucket sub-resources (`_policy`, `_tagging`, `_encryption`), all IAM resources (users, groups, roles, policies, attachments, memberships), and all workflow resources (expiration, transition, replication). Import IDs for account-scoped resources start with the account name (`<account_name>/<id>`).

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

### Changed (breaking)

- **Plan-time validation aligned with ARTESCA docs (#61).** `artesca_location.name` and `artesca_endpoint.location_name` accept only lowercase letters, numbers, and hyphens, starting with a letter (at least 3 characters); periods and a leading digit were accepted before and rejected by ARTESCA. New checks: `artesca_bucket_workflow_replication.rule_id` 1–255 characters, `artesca_bucket_tagging.tags` at most 50, `ephemeral.artesca_assumed_role_credentials` `duration_seconds` 900–43200 and `role_session_name` letters, numbers and `_=,.@-`. Required `details` fields are now checked for the location types ARTESCA documents: added `location-scaleway-glacier-v1`, `location-ovh-cold-archive-v1`, `location-versity-tape-archive-v1`; dropped `location-do-spaces-v1`, `location-ceph-radosgw-s3-v1`, `location-scality-hdclient-v2`, `location-nfs-mount-v1` (passed to ARTESCA unchecked).
- **`artesca_location.details.server_side_encryption` is only accepted for `location-aws-s3-v1` (#53).** Other location types don't support it; setting it now fails at plan time.

- **`artesca_bucket_workflow_replication` manages an S3 replication rule (#2).** It reads and writes the bucket's S3 replication configuration (`Get/Put/DeleteBucketReplication`), merging its rule with others on the same bucket, like the expiration and transition resources do with lifecycle. The management API never stored a replication workflow's `name` or `version`, so they could not be read back. New shape: `account_name`, `bucket_name`, `destination_location` (#85), optional `destination_bucket_name`, `enabled`, optional `filter { object_key_prefix }`, computed `rule_id`; import ID `<account_name>/<bucket_name>/<rule_id>`. Removed `instance_id`, `account_id`, `name`, `version`, `workflow_id`, and the `source` / `destination` blocks.

### Changed

- **Docs review (#80).** Corrected inaccurate and outdated statements across the resource, data source and project docs; fixed three examples that didn't validate; documented missing constraints (account and bucket name rules, endpoint hostnames, `bucket_match` replacement, four default IAM roles, user deletion with attached policies).
- **Docs: ARTESCA constraints (#62).** The provider needs a user with the `StorageManager` role; buckets must be empty to delete; locations: at most 10, versioned target bucket for Amazon S3 and RING S3, truststore CA for TLS, no delete while holding data, CRR is replication-only, don't change `us-east-1`; accounts: deletion prerequisites and the three default roles; IAM names are case-insensitive.
- **`data.artesca_bucket_workflows` takes `account_name` (#58),** like every other account-scoped data source. `account_id` and `instance_id` are removed; the provider resolves the account ID from the name.
- **Release pipeline** moved from a self-hosted runner to `ubuntu-latest`. Build artifacts are handed to the signing job via immutable within-run workflow artifacts, and every checksum is re-verified before signing.
- **Release binary naming**: the binary inside each zip now uses the `v`-prefixed version (`terraform-provider-artesca_v1.0.0`) — the archive filename remains unprefixed (`terraform-provider-artesca_1.0.0_linux_amd64.zip`). This matches the OpenTofu / Terraform Registry conventions.

- **artesca_bucket_workflow_replication**: `Read()` now uses workflow search to detect drift (deletion, `enabled` flips, source/destination changes). Previously it preserved state as-is. `name` and `version` are preserved from state because the workflow-search endpoint returns them as `null` for replication workflows.
- **S3 client**: 502 / 503 / 504 responses are retried with exponential backoff (4 attempts, 500 ms → 8 s). 500 is not retried (treated as a real server error). `CreateBucket` has a separate longer retry loop for `InvalidLocationConstraint` to handle location propagation.

### Removed

- **`email` and `access_key` from `data.artesca_account`, and `email` from `data.artesca_accounts`.** The account listing does not return them (#35).
- **`account_access_key` / `account_secret_key`** from the 17 account-scoped resources and `data.artesca_group`, `data.artesca_policy`, `data.artesca_role`, `data.artesca_user`. Use `account_name` (#36).
- **`policy` from `ephemeral.artesca_assumed_role_credentials`.** ARTESCA accepts STS session policies but does not enforce them (#64).

### Fixed

- **`artesca_replication` without `source` or `destination` passed plan (#79).** A missing block is now a plan-time error. The state-upgrade error for `artesca_user_access_key`, which can't be imported, now says to remove it from state and apply again instead of re-importing. The assumed-role credentials resource no longer carries an unreachable "s3_endpoint must be set" check.
- **Policy documents changed outside Terraform went undetected (#78).** `artesca_policy`, `artesca_user_policy`, `artesca_group_policy` and the `artesca_role` trust policy read the document only on import. Read now compares it on every refresh, ignoring whitespace and key order, and formatting-only differences in configuration no longer plan a change or a replacement.
- **Imported locations had empty `details` (#71).** Read only refreshed `details` fields already in state, and imported state has none, so `endpoint`, `sts_endpoint`, `bucket_name`, `region` and `access_key` stayed empty and the first plan showed changes. The first Read after import now takes every value ARTESCA returns (secrets excepted). `details.bucket_match` now defaults to `false`, since ARTESCA omits it when false; state written by earlier builds is upgraded to `false` so a plan without refresh doesn't replace the location (#73). The docs warn that a `bucket_match = true` location shared between buckets can lose data.
- **CRR locations could not be fully configured (#60).** ARTESCA requires an STS endpoint for `location-scality-crr-v1`; `artesca_location` and `data.artesca_location` now have `details.sts_endpoint`, required at plan time for CRR. Creating a location type with no region (such as CRR) also failed with "Provider returned invalid result object after apply"; `details.region` is now null when ARTESCA returns none.
- **Workflow resources rewrote other rules on the same bucket (#59).** When adding, changing, or removing their rule, `artesca_bucket_workflow_expiration`, `_transition`, and `_replication` wrote the bucket's other rules back with only the fields the provider models, dropping tag filters, dates, noncurrent-version actions, and replication `StorageClass`. A tag-filtered expiration rule became a rule for the whole bucket. Other rules are now written back exactly as read.
- **Transitions with `trigger_delay_days = 0` were rejected (#63).** The provider left `Days` out of the lifecycle XML when it was 0, and ARTESCA returned `MalformedXML`. `Days` is now always sent.
- **Accounts reported as deleted right after creation.** `artesca_account`, `data.artesca_account`, and `data.artesca_accounts` looked accounts up in the management overlay view, which does not list them. They now use IAM `GetRolesForWebIdentity` with the provider's OIDC token (#35). `artesca_account.arn` is now the account root ARN (`arn:aws:iam::<id>:root`).
- **Bucket configs lost when several were applied at once (#46).** ARTESCA can silently drop some of several config writes made concurrently to the same bucket, and OpenTofu creates bucket sub-resources in parallel. All bucket-config writes (tagging, encryption, policy, versioning, lifecycle, replication) and bucket deletion are now serialized per bucket inside the provider.
- **Imported buckets and lifecycle workflows lost attributes.** `artesca_bucket` only read `location_constraint` when it was already in state, and `artesca_bucket_workflow_expiration` / `_transition` only read the filter prefix when a `filter` block was already in state, so both were empty after import and out-of-band changes went undetected. Read now always reads them; an unset prefix stays null.
- **State from before `account_name` failed to refresh (#45).** Account-scoped resources are now at schema version 1 with a state upgrader: `account_name` is resolved from the access key pair in v0 state (STS `GetCallerIdentity`, then the account list), and `artesca_bucket_workflow_replication` v0 state is mapped to the S3 rule shape (`workflow_id` → `rule_id`, `destination.bucket_name` → `destination_bucket_name`, `source.prefix` → `filter.object_key_prefix`, `account_id` → `account_name`). If the keys no longer work, the error explains how to `tofu state rm` and re-import. `tests/upgrade/run.sh` creates resources with an older build and checks that planning with the current one shows no changes.

### Added (CI / security)

- **CodeQL** analysis on push to `main`, pull requests, and a weekly schedule.
- **gitleaks** secret-detection scan on push to `main` and pull requests.
- **gosec** Go security scan on push to `main` and pull requests.
- **`-race`** detector added to unit test runs.

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
