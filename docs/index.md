---
page_title: "ARTESCA Provider"
subcategory: ""
description: |-
  Terraform/OpenTofu provider for managing Scality ARTESCA storage infrastructure.
---

# ARTESCA Provider

Terraform/OpenTofu provider for managing Scality ARTESCA storage infrastructure. Supports account management, storage locations, endpoints, IAM (users, policies, access keys), replication streams, and bucket lifecycle workflows.

The provider authenticates via two API surfaces:
- **Management API** -- OIDC bearer token for infrastructure operations (accounts, locations, endpoints, replication, workflows)
- **IAM API** -- AWS Signature V4 for per-account IAM operations (users, access keys, policies)

The IAM endpoint is automatically derived from the management endpoint (`management.` -> `iam.`).

## Provider Configuration

```hcl
provider "artesca" {
  management_endpoint = "https://management.artesca.example.com"  # or ARTESCA_MANAGEMENT_ENDPOINT
  oidc_url            = "https://ui.artesca.example.com"          # or ARTESCA_OIDC_URL
  oidc_realm          = "artesca"                                 # or ARTESCA_OIDC_REALM (default: "artesca")
  client_id           = "zenko-ui"                                # or ARTESCA_CLIENT_ID (default: "zenko-ui")
  username            = var.artesca_username                      # or ARTESCA_USERNAME
  password            = var.artesca_password                      # or ARTESCA_PASSWORD
  insecure_skip_verify = true                                     # or ARTESCA_INSECURE_SKIP_VERIFY

  # instance_id is auto-discovered from the OIDC token if omitted
  # instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"          # or ARTESCA_INSTANCE_ID

  # iam_region defaults to "us-east-1"
  # iam_region = "us-east-1"                                      # or ARTESCA_IAM_REGION
}
```

`management_endpoint`, `oidc_url`, `username`, and `password` are required. All other attributes have defaults or are auto-discovered.

## Resources

### Accounts & Infrastructure

| Resource | Description |
|----------|-------------|
| [artesca_account](resources/account.md) | Account (S3 user) via management API |
| [artesca_bucket](resources/bucket.md) | S3 bucket with versioning and location constraint |
| [artesca_location](resources/location.md) | Storage location (AWS S3, Azure, GCP, Scality RING, etc.) |
| [artesca_endpoint](resources/endpoint.md) | S3 data service endpoint |
| [artesca_replication](resources/replication.md) | Config-scoped replication stream |

### IAM

| Resource | Description |
|----------|-------------|
| [artesca_user](resources/user.md) | IAM user within an account |
| [artesca_user_access_key](resources/user_access_key.md) | Access key for a user |
| [artesca_user_policy](resources/user_policy.md) | Inline policy attached to a user |

### Bucket Workflows

| Resource | Description |
|----------|-------------|
| [artesca_bucket_workflow_expiration](resources/bucket_workflow_expiration.md) | Object expiration lifecycle workflow |
| [artesca_bucket_workflow_transition](resources/bucket_workflow_transition.md) | Object transition lifecycle workflow |
| [artesca_bucket_workflow_replication](resources/bucket_workflow_replication.md) | Bucket-scoped replication workflow |

## Credential Pattern

Account credentials are generated on creation and stored in state. Use them to configure per-account IAM resources:

```hcl
resource "artesca_account" "app" {
  name  = "my-app"
  email = "app@example.com"
}

resource "artesca_user" "operator" {
  account_access_key = artesca_account.app.access_key
  account_secret_key = artesca_account.app.secret_key
  username           = "bucket-operator"
}

resource "artesca_user_access_key" "operator_key" {
  account_access_key = artesca_account.app.access_key
  account_secret_key = artesca_account.app.secret_key
  username           = artesca_user.operator.username
}
```
