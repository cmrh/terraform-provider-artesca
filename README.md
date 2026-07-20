> **This repository has been archived and is no longer maintained.**
> Development has moved to **[github.com/cmrh/terraform-provider-artesca](https://github.com/cmrh/terraform-provider-artesca)**.
> Issues and pull requests here are closed — please open them at the new repository.

---

# Terraform Provider for ARTESCA

A Terraform/OpenTofu provider for managing ARTESCA infrastructure, including accounts, storage locations, endpoints, replication, IAM users, and lifecycle workflows.

## Requirements

- [Go](https://go.dev/dl/) 1.26+
- [Terraform](https://www.terraform.io/downloads) or [OpenTofu](https://opentofu.org/docs/intro/install/)

## Provider Configuration

```hcl
terraform {
  required_providers {
    artesca = {
      source = "registry.opentofu.org/cmrh/artesca"
    }
  }
}

provider "artesca" {
  management_endpoint = "https://management.artesca.example.com"
  oidc_url            = "https://10.0.0.1:8443"  # see note below
  username            = var.artesca_username
  password            = var.artesca_password

  # Optional
  instance_id          = "auto-discovered-if-omitted"
  oidc_realm           = "artesca"          # default
  client_id            = "zenko-ui"         # default
  iam_region           = "us-east-1"        # default
  insecure_skip_verify = false              # default
  s3_endpoint          = "https://s3.artesca.example.com"  # required for bucket resources
}
```

All attributes can also be set via environment variables:

| Attribute | Environment Variable |
|---|---|
| `management_endpoint` | `ARTESCA_MANAGEMENT_ENDPOINT` |
| `oidc_url` | `ARTESCA_OIDC_URL` |
| `username` | `ARTESCA_USERNAME` |
| `password` | `ARTESCA_PASSWORD` |
| `instance_id` | `ARTESCA_INSTANCE_ID` |
| `oidc_realm` | `ARTESCA_OIDC_REALM` |
| `client_id` | `ARTESCA_CLIENT_ID` |
| `iam_region` | `ARTESCA_IAM_REGION` |
| `insecure_skip_verify` | `ARTESCA_INSECURE_SKIP_VERIFY` |
| `s3_endpoint` | `ARTESCA_S3_ENDPOINT` |
| _(scope only)_ | `ARTESCA_OIDC_SCOPE` (default: `openid`) |

> **Finding the OIDC URL:** The `oidc_url` must be the control plane ingress endpoint (typically an IP-based URL like `https://10.0.0.1:8443`). Using a DNS alias with a non-standard port may produce OIDC tokens whose issuer doesn't match the internal trust policies, causing account deletion to fail. Retrieve the correct URL from an ARTESCA node:
>
> ```bash
> sudo salt-call metalk8s_network.get_control_plane_ingress_endpoint --out=json | jq -cr '.local'
> ```

## Resources

### Infrastructure

| Resource | Description |
|---|---|
| `artesca_account` | Manage ARTESCA accounts (S3 users) |
| `artesca_bucket` | Manage S3 buckets on the ARTESCA S3 endpoint |
| `artesca_bucket_encryption` | Server-side encryption configuration (SSE-S3 / `AES256`) |
| `artesca_bucket_policy` | Bucket policy via S3 `PutBucketPolicy` |
| `artesca_bucket_tagging` | Bucket tag set management |
| `artesca_location` | Manage storage locations (AWS S3, Azure, GCP, Scality RING, etc.) |
| `artesca_endpoint` | Manage data service endpoints |
| `artesca_replication` | Manage cross-region replication streams |

### IAM

| Resource | Description |
|---|---|
| `artesca_user` | Manage IAM users within accounts |
| `artesca_user_access_key` | Create access keys for IAM users |
| `artesca_user_policy` | Attach inline policies to IAM users |
| `artesca_user_policy_attachment` | Attach a managed policy to a user |
| `artesca_group` | Manage IAM groups within an account |
| `artesca_group_membership` | Attach a user to one or more groups |
| `artesca_group_policy` | Inline IAM policy attached to a group |
| `artesca_group_policy_attachment` | Attach a managed policy to a group |
| `artesca_role` | Manage IAM roles with a trust policy |
| `artesca_role_policy_attachment` | Attach a managed policy to a role |
| `artesca_policy` | Manage account-scoped IAM managed policies |

### Lifecycle Workflows

| Resource | Description |
|---|---|
| `artesca_bucket_workflow_expiration` | Manage object expiration lifecycle rules |
| `artesca_bucket_workflow_transition` | Manage object transition lifecycle rules |
| `artesca_bucket_workflow_replication` | Manage bucket-scoped replication workflows |

## Data Sources

| Data source | Description |
|---|---|
| `data.artesca_account` / `data.artesca_accounts` | Look up an account by name, or list all accounts |
| `data.artesca_location` / `data.artesca_locations` | Look up a location by name, or list all locations |
| `data.artesca_endpoints` | List all data-service endpoints |
| `data.artesca_user` / `data.artesca_group` / `data.artesca_role` / `data.artesca_policy` | Look up existing IAM objects without managing them |
| `data.artesca_caller_identity` | Resolve `account` / `user_id` / `arn` for an access key via STS `GetCallerIdentity` |
| `data.artesca_bucket_workflows` | List the workflows (replication / expiration / transition) configured on a bucket |

## Ephemeral Resources

| Ephemeral resource | Description |
|---|---|
| `ephemeral.artesca_assumed_role_credentials` | Mint short-lived role credentials via STS `AssumeRole` (session tokens never persisted to state) |

## Usage Examples

### Create an account

```hcl
resource "artesca_account" "example" {
  name  = "my-account"
  email = "admin@example.com"
}

output "access_key" {
  value = artesca_account.example.access_key
}

output "secret_key" {
  value     = artesca_account.example.secret_key
  sensitive = true
}
```

### Create a storage location

```hcl
resource "artesca_location" "s3" {
  name          = "aws-us-east-1"
  location_type = "location-aws-s3-v1"

  details {
    access_key              = var.aws_access_key
    secret_key              = var.aws_secret_key
    bucket_name             = "my-target-bucket"
    bucket_match            = true
    region                  = "us-east-1"
    server_side_encryption  = true
  }
}
```

### Create an endpoint

```hcl
resource "artesca_endpoint" "data" {
  hostname      = "data.artesca.example.com"
  location_name = artesca_location.s3.name
}
```

### Create an IAM user with an access key

```hcl
resource "artesca_user" "app" {
  account_access_key = artesca_account.example.access_key
  account_secret_key = artesca_account.example.secret_key
  username           = "app-user"
}

resource "artesca_user_access_key" "app" {
  account_access_key = artesca_account.example.access_key
  account_secret_key = artesca_account.example.secret_key
  username           = artesca_user.app.username
}
```

## Local Development

### Build

```bash
make build
```

### Install locally

This installs the provider binary into the OpenTofu plugin directory so it can be used by local Terraform/OpenTofu configurations:

```bash
make install
```

### Run tests

```bash
# Unit tests
make test

# Acceptance tests (requires a running ARTESCA instance)
make testacc
```

### Lint and format

```bash
make lint
make fmt
```

### Using the local provider

After `make install`, configure your Terraform/OpenTofu to use the local build:

```hcl
terraform {
  required_providers {
    artesca = {
      source  = "registry.opentofu.org/cmrh/artesca"
      version = "0.1.0"
    }
  }
}
```

Then create a [dev overrides](https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides-for-provider-developers) file at `~/.terraformrc` to skip version checks during development:

```hcl
provider_installation {
  dev_overrides {
    "registry.opentofu.org/cmrh/artesca" = "/path/to/go/bin"
  }
  direct {}
}
```

## Releasing

Releases are built automatically by GitHub Actions when a version tag is pushed:

```bash
git tag v0.1.0
git push origin v0.1.0
```

This triggers the release workflow which:

1. Runs tests, `go vet`, and format checks
2. Builds binaries for linux, darwin, and windows (amd64/arm64)
3. Generates SHA256 checksums and GPG-signs them
4. Uploads all artifacts as a GitHub release

## License

See [LICENSE](LICENSE) for details.
