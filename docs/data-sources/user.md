---
page_title: "artesca_user Data Source - artesca"
subcategory: "Identity"
description: |-
  Looks up an existing IAM user within an ARTESCA account.
---

# Data Source: artesca_user

Looks up an existing IAM user within an ARTESCA account. Useful for referencing a user that exists outside Terraform without re-creating it.

IAM operations are account-scoped: the provider gets temporary credentials for `account_name` from its OIDC login.

## Example

```hcl
data "artesca_user" "ops" {
  account_name = artesca_account.ops.name
  username     = "alice"
}

output "alice_arn" {
  value = data.artesca_user.ops.arn
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_name` | String | Yes | Name of the account to read from. |
| `username` | String | Yes | The name of the IAM user to look up. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `user_id` | The unique ID of the user. |
| `arn` | The ARN of the user. |
| `path` | The path of the user. |
