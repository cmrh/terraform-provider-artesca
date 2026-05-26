---
page_title: "artesca_user Data Source - artesca"
subcategory: "Identity"
description: |-
  Looks up an existing IAM user within an ARTESCA account.
---

# Data Source: artesca_user

Looks up an existing IAM user within an ARTESCA account. Useful for referencing a user that exists outside Terraform without re-creating it.

IAM operations are account-scoped, so you must supply the account's `access_key` and `secret_key` to authenticate the lookup.

## Example

```hcl
data "artesca_user" "ops" {
  account_access_key = artesca_account.ops.access_key
  account_secret_key = artesca_account.ops.secret_key
  username           = "alice"
}

output "alice_arn" {
  value = data.artesca_user.ops.arn
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | The access key of the account this user belongs to. Sensitive. |
| `account_secret_key` | String | Yes | The secret key of the account this user belongs to. Sensitive. |
| `username` | String | Yes | The name of the IAM user to look up. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `user_id` | The unique ID of the user. |
| `arn` | The ARN of the user. |
| `path` | The path of the user. |
