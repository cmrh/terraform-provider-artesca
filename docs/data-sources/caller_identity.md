---
page_title: "artesca_caller_identity Data Source - artesca"
subcategory: "Identity"
description: |-
  Returns the identity (account, user ID, ARN) associated with the supplied access key via STS GetCallerIdentity.
---

# Data Source: artesca_caller_identity

Calls STS `GetCallerIdentity` to resolve the identity associated with a given access key / secret key pair. Useful for asserting "these credentials belong to the expected account/user" inside a Terraform configuration.

The STS endpoint is derived from the configured S3 endpoint (`s3.` → `sts.`) — no additional provider configuration is needed.

Supports both **static credentials** (an account's or IAM user's access key + secret key) and **temporary credentials** (those minted by `sts:AssumeRole`), which require an additional `session_token`.

## Examples

Static credentials:

```hcl
data "artesca_caller_identity" "ops" {
  access_key = artesca_account.ops.access_key
  secret_key = artesca_account.ops.secret_key
}

output "ops_account_id" {
  value = data.artesca_caller_identity.ops.account
}

output "ops_arn" {
  value = data.artesca_caller_identity.ops.arn
}
```

Temporary credentials (e.g. round-tripping a freshly assumed role to confirm the identity it resolves to):

```hcl
data "artesca_caller_identity" "as_writer" {
  access_key    = local.session_access_key
  secret_key    = local.session_secret_key
  session_token = local.session_token
}
```

> **Note:** Terraform data sources cannot reference attributes of `ephemeral.*` resources directly. To use `data.artesca_caller_identity` with credentials freshly minted by `ephemeral.artesca_assumed_role_credentials`, you must materialize the credentials through an intermediate path (e.g. a `terraform_data` resource with `write-only` attributes, or supply the credentials via input variables).

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `access_key` | String | Yes | The access key whose identity should be looked up. Sensitive. |
| `secret_key` | String | Yes | The secret key paired with `access_key`. Sensitive. |
| `session_token` | String | Optional | STS session token. Set this when introspecting temporary credentials (those minted by `sts:AssumeRole`); omit for static account / IAM user credentials. Sensitive. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `user_id` | The unique identifier of the calling principal. |
| `account` | The account ID the calling principal belongs to. |
| `arn` | The ARN of the calling principal. |
