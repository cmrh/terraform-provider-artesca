---
page_title: "artesca_caller_identity Data Source - artesca"
subcategory: "Identity"
description: |-
  Returns the identity (account, user ID, ARN) associated with the supplied access key via STS GetCallerIdentity.
---

# Data Source: artesca_caller_identity

Calls STS `GetCallerIdentity` to resolve the identity associated with a given access key / secret key pair. Useful for asserting "these credentials belong to the expected account/user" inside a Terraform configuration.

The STS endpoint is derived from the configured S3 endpoint (`s3.` → `sts.`) — no additional provider configuration is needed.

> **Note:** session credentials (those produced by `ephemeral.artesca_assumed_role_credentials`, which carry a `session_token`) are not supported here yet. Pass static account or IAM user credentials only.

## Example

```hcl
resource "artesca_account" "ops" {
  name  = "ops"
  email = "ops@example.com"
}

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

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `access_key` | String | Yes | The access key whose identity should be looked up. Sensitive. |
| `secret_key` | String | Yes | The secret key paired with `access_key`. Sensitive. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `user_id` | The unique identifier of the calling principal. |
| `account` | The account ID the calling principal belongs to. |
| `arn` | The ARN of the calling principal. |
