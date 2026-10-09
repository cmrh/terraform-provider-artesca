---
page_title: "artesca_group Data Source - artesca"
subcategory: "Identity"
description: |-
  Looks up an existing IAM group within an ARTESCA account.
---

# Data Source: artesca_group

Looks up an existing IAM group within an ARTESCA account. Useful for referencing a group that exists outside Terraform without re-creating it.

IAM operations are account-scoped: the provider gets temporary credentials for `account_name` from its OIDC login.

## Example

```hcl
data "artesca_group" "admins" {
  account_name = artesca_account.ops.name
  name         = "platform-admins"
}

output "admins_arn" {
  value = data.artesca_group.admins.arn
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_name` | String | Yes | Name of the account to read from. |
| `name` | String | Yes | The name of the IAM group to look up. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `group_id` | The unique ID of the group. |
| `arn` | The ARN of the group. |
| `path` | The path of the group. |
