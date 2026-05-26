---
page_title: "artesca_group Data Source - artesca"
subcategory: "Identity"
description: |-
  Looks up an existing IAM group within an ARTESCA account.
---

# Data Source: artesca_group

Looks up an existing IAM group within an ARTESCA account. Useful for referencing a group that exists outside Terraform without re-creating it.

IAM operations are account-scoped, so you must supply the account's `access_key` and `secret_key` to authenticate the lookup.

## Example

```hcl
data "artesca_group" "admins" {
  account_access_key = artesca_account.ops.access_key
  account_secret_key = artesca_account.ops.secret_key
  name               = "platform-admins"
}

output "admins_arn" {
  value = data.artesca_group.admins.arn
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | The access key of the account this group belongs to. Sensitive. |
| `account_secret_key` | String | Yes | The secret key of the account this group belongs to. Sensitive. |
| `name` | String | Yes | The name of the IAM group to look up. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `group_id` | The unique ID of the group. |
| `arn` | The ARN of the group. |
| `path` | The path of the group. |
