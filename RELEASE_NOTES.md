# v0.4.0

First public release of the ARTESCA Terraform provider on `cmrh/artesca`.

## Provider surface

22 resources, 12 data sources, and 1 ephemeral resource covering three ARTESCA API surfaces (Management, IAM, S3), plus STS for the ephemeral role-credential resource. See [README.md](README.md) for the full inventory.

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
- **data.artesca_instance** — connected instance metadata + status.
- **data.artesca_bucket_workflows** — list workflows configured on a bucket.
- Look-up data sources for **account**, **accounts**, **location**, **locations**, **endpoints**, **user**, **group**, **role**, **policy**.

### Ephemeral
- **ephemeral.artesca_assumed_role_credentials** — mint short-lived role credentials via STS `AssumeRole`. Session tokens are never persisted to state.

### Brownfield import
- Import support for 17 resources previously unimportable: buckets, bucket sub-resources, IAM users/groups/roles/policies and their attachments, and workflows.
- New `internal/creds` helper with an env-var fallback (`ARTESCA_ACCOUNT_ACCESS_KEY` / `ARTESCA_ACCOUNT_SECRET_KEY`) so `tofu import` works without ak/sk baked into state.

## Fixed

- **`workflow_replication` drift detection.** `Read()` now uses workflow search to detect deletion, `enabled` flips, and source/destination changes. `name` and `version` are preserved from state — the workflow-search endpoint returns them as `null` for replication entries (tracked upstream).

## CI / tooling

- Release pipeline builds on `ubuntu-latest`, hands artifacts to the signing job via immutable within-run artifacts, and re-verifies every checksum before signing.
- CodeQL, gitleaks, and gosec security scanners run on every push and pull request.

See [CHANGELOG.md](CHANGELOG.md) for the full history.
