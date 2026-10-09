---
page_title: "ARTESCA Provider"
subcategory: ""
description: |-
  Terraform/OpenTofu provider for managing ARTESCA storage infrastructure.
---

# ARTESCA Provider

Terraform/OpenTofu provider for managing ARTESCA storage infrastructure. Supports account management, storage locations, endpoints, IAM (users, groups, roles, policies, access keys), STS, S3 buckets and bucket sub-resources (policy, tagging, encryption), and bucket lifecycle workflows.

The provider authenticates via three API surfaces:
- **Management API** -- OIDC bearer token for infrastructure operations (account creation and deletion, locations, endpoints, replication, workflows).
- **IAM API** -- AWS Signature V4 with per-account credentials (users, groups, roles, policies, access keys). Account lookups use the provider's OIDC token.
- **S3 / STS API** -- AWS Signature V4 with per-account credentials (buckets and sub-resources; assume-role and caller-identity).

Per-account credentials are temporary: for each `account_name`, the provider exchanges its OIDC token for credentials on the account's storage-manager role (STS `AssumeRoleWithWebIdentity`) and refreshes them before they expire. No account keys are configured or stored.

The IAM endpoint is automatically derived from the management endpoint (`management.` → `iam.`). The STS endpoint is derived from the S3 endpoint (`s3.` → `sts.`), or from the management endpoint (`management.` → `sts.`) when `s3_endpoint` is not set.

## Provider Configuration

```hcl
provider "artesca" {
  management_endpoint  = "https://management.artesca.example.com" # or ARTESCA_MANAGEMENT_ENDPOINT
  oidc_url             = "https://ui.artesca.example.com"         # or ARTESCA_OIDC_URL
  username             = var.artesca_username                     # or ARTESCA_USERNAME
  password             = var.artesca_password                     # or ARTESCA_PASSWORD
  insecure_skip_verify = true                                     # or ARTESCA_INSECURE_SKIP_VERIFY

  # iam_region defaults to "us-east-1"
  # iam_region = "us-east-1"                                      # or ARTESCA_IAM_REGION

  # Required for bucket resources and STS
  # s3_endpoint = "https://s3.artesca.example.com"                # or ARTESCA_S3_ENDPOINT
}
```

`management_endpoint`, `oidc_url`, `username`, and `password` are required. All other attributes have defaults.

## Resources

### Accounts & Infrastructure

| Resource | Description |
|----------|-------------|
| [artesca_account](resources/account.md) | Account (S3 user) via management API |
| [artesca_bucket](resources/bucket.md) | S3 bucket with versioning and location constraint |
| [artesca_bucket_encryption](resources/bucket_encryption.md) | Server-side encryption configuration (SSE-S3) |
| [artesca_bucket_policy](resources/bucket_policy.md) | Bucket policy via S3 `PutBucketPolicy` |
| [artesca_bucket_tagging](resources/bucket_tagging.md) | Bucket tag set management |
| [artesca_location](resources/location.md) | Storage location (AWS S3, Azure, GCP, Scality RING, etc.) |
| [artesca_endpoint](resources/endpoint.md) | S3 data service endpoint |
| [artesca_replication](resources/replication.md) | Config-scoped replication stream |

### IAM

| Resource | Description |
|----------|-------------|
| [artesca_user](resources/user.md) | IAM user within an account |
| [artesca_user_access_key](resources/user_access_key.md) | Access key for a user |
| [artesca_user_policy](resources/user_policy.md) | Inline policy attached to a user |
| [artesca_user_policy_attachment](resources/user_policy_attachment.md) | Attach a managed policy to a user |
| [artesca_group](resources/group.md) | IAM group within an account |
| [artesca_group_membership](resources/group_membership.md) | Attach a user to one or more groups |
| [artesca_group_policy](resources/group_policy.md) | Inline policy attached to a group |
| [artesca_group_policy_attachment](resources/group_policy_attachment.md) | Attach a managed policy to a group |
| [artesca_role](resources/role.md) | IAM role with trust policy |
| [artesca_role_policy_attachment](resources/role_policy_attachment.md) | Attach a managed policy to a role |
| [artesca_policy](resources/policy.md) | Managed policy (account-scoped) |

### Bucket Workflows

| Resource | Description |
|----------|-------------|
| [artesca_bucket_workflow_expiration](resources/bucket_workflow_expiration.md) | Object expiration lifecycle workflow |
| [artesca_bucket_workflow_transition](resources/bucket_workflow_transition.md) | Object transition lifecycle workflow |
| [artesca_bucket_workflow_replication](resources/bucket_workflow_replication.md) | Bucket replication rule (S3 replication configuration) |

## Data Sources

| Data source | Description |
|----------|-------------|
| [data.artesca_account](data-sources/account.md) / [data.artesca_accounts](data-sources/accounts.md) | Look up an account by name, or list all accounts |
| [data.artesca_location](data-sources/location.md) / [data.artesca_locations](data-sources/locations.md) | Look up a location by name, or list all locations |
| [data.artesca_endpoints](data-sources/endpoints.md) | List all data-service endpoints |
| [data.artesca_user](data-sources/user.md) / [data.artesca_group](data-sources/group.md) / [data.artesca_role](data-sources/role.md) / [data.artesca_policy](data-sources/policy.md) | Look up existing IAM objects |
| [data.artesca_caller_identity](data-sources/caller_identity.md) | Resolve identity for an access key via STS `GetCallerIdentity` |
| [data.artesca_bucket_workflows](data-sources/bucket_workflows.md) | List workflows (replication / expiration / transition) on a bucket |

## Ephemeral Resources

| Ephemeral resource | Description |
|----------|-------------|
| [ephemeral.artesca_assumed_role_credentials](ephemeral-resources/assumed_role_credentials.md) | Mint short-lived role credentials via STS `AssumeRole`; session tokens never persist to state |

## Account-Scoped Resources

IAM and S3 resources name the account they belong to with `account_name`. The provider obtains credentials for that account itself:

```hcl
resource "artesca_account" "app" {
  name  = "my-app"
  email = "app@example.com"
}

resource "artesca_user" "operator" {
  account_name = artesca_account.app.name
  username     = "bucket-operator"
}

resource "artesca_user_access_key" "operator_key" {
  account_name = artesca_account.app.name
  username     = artesca_user.operator.username
}
```

## Upgrading From Account Keys

Earlier versions configured account-scoped resources with `account_access_key` / `account_secret_key`. After replacing them with `account_name` in your configuration, run `tofu plan`: existing state is migrated automatically using the keys still recorded in it, and the plan shows no changes. If those keys no longer work, the plan fails with instructions to `tofu state rm` the resource and import it again (see below).

## Importing Account-Scoped Resources

Import IDs for account-scoped resources start with the account name:

```bash
tofu import artesca_user.operator my-app/bucket-operator
```

See each resource's Import section for its ID format. `artesca_user_access_key` cannot be imported.
