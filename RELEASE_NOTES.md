# v0.4.0

First public release of the ARTESCA Terraform provider on `cmrh/artesca`.

## Provider surface

22 resources, 11 data sources, and 1 ephemeral resource covering three ARTESCA API surfaces (Management, IAM, S3), plus STS for the ephemeral role-credential resource. See [README.md](README.md) for the full inventory.

## Upgrading from v0.3.x

Update your configuration (replace `account_access_key` / `account_secret_key` with `account_name = artesca_account.<name>.name`; see the breaking changes below), then run `tofu plan`. Existing state is migrated automatically: the provider identifies each resource's account from the access key still in state, so the plan shows no changes. If those keys no longer work (account deleted or keys rotated), the plan fails with instructions to `tofu state rm` and `tofu import` the affected resource instead.

## Breaking changes since v0.3.0

- **`account_name` replaces `account_access_key` / `account_secret_key`** on all account-scoped resources (buckets and sub-resources, IAM users/groups/roles/policies and attachments, user access keys, expiration/transition workflows) and on `data.artesca_group`, `data.artesca_policy`, `data.artesca_role`, `data.artesca_user`. The provider obtains temporary credentials for the account from its OIDC login; account keys are no longer configured or stored.
- **Import IDs** for account-scoped resources start with the account name: `tofu import artesca_user.alice my-app/alice`.
- **`data.artesca_account`** no longer exports `email` or `access_key`; **`data.artesca_accounts`** no longer exports `email`.
- **`artesca_account.arn`** is the account root ARN (`arn:aws:iam::<id>:root`).
- **`artesca_bucket_workflow_replication`** manages an S3 replication rule and uses the same shape as the other bucket workflow resources: `account_name`, `bucket_name`, `destination_bucket_name`, `enabled`, optional `filter { object_key_prefix }`, and computed `rule_id`. Removed: `instance_id`, `account_id`, `name`, `version`, `workflow_id`, and the `source` / `destination` blocks. Import IDs are `<account_name>/<bucket_name>/<rule_id>`.

## Recent additions since v0.3.0

### IAM
- **artesca_group** and companions (`_membership`, `_policy`, `_policy_attachment`) for group-based access management.
- **artesca_role** and **artesca_role_policy_attachment** — IAM roles with trust policies. `assume_role_policy_document` is immutable (ARTESCA does not implement `UpdateAssumeRolePolicy`).
- **artesca_policy** — account-scoped managed policies attachable to users, groups, and roles.
- **artesca_user_policy_attachment** — attach a managed policy to a user.

### S3 bucket sub-resources
- **artesca_bucket_policy** — bucket policy via S3 `PutBucketPolicy` / `GetBucketPolicy`.
- **artesca_bucket_tagging** — bucket tag set management.
- **artesca_bucket_encryption** — server-side encryption configuration (SSE-S3 / `AES256`).

### Data sources
- **data.artesca_caller_identity** — resolve identity for an access key via STS `GetCallerIdentity`.
- **data.artesca_bucket_workflows** — list workflows configured on a bucket.
- Look-up data sources for **account**, **accounts**, **location**, **locations**, **endpoints**, **user**, **group**, **role**, **policy**.

### Ephemeral
- **ephemeral.artesca_assumed_role_credentials** — mint short-lived role credentials via STS `AssumeRole`. Session tokens are never persisted to state.

### Brownfield import
- Import support for 17 resources previously unimportable: buckets, bucket sub-resources, IAM users/groups/roles/policies and their attachments, and workflows. No credentials are needed beyond the provider's own login.

## Fixed

- **`artesca_bucket_workflow_replication` drift detection.** The resource manages its rule in the bucket's S3 replication configuration, so `Read()` detects deletion and changes to `enabled`, the destination bucket, and the prefix.

## CI / tooling

- Release pipeline builds on `ubuntu-latest`, hands artifacts to the signing job via immutable within-run artifacts, and re-verifies every checksum before signing.
- CodeQL, gitleaks, and gosec security scanners run on every push and pull request.

See [CHANGELOG.md](CHANGELOG.md) for the full history.
