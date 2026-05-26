---
page_title: "artesca_role Data Source - artesca"
subcategory: "Identity"
description: |-
  Looks up an existing IAM role within an ARTESCA account.
---

# Data Source: artesca_role

Looks up an existing IAM role within an ARTESCA account, including its trust policy. Useful for referencing a role that exists outside Terraform without re-creating it.

IAM operations are account-scoped, so you must supply the account's `access_key` and `secret_key` to authenticate the lookup.

## Example

```hcl
data "artesca_role" "writer" {
  account_access_key = artesca_account.ops.access_key
  account_secret_key = artesca_account.ops.secret_key
  name               = "object-writer"
}

ephemeral "artesca_assumed_role_credentials" "writer" {
  role_arn         = data.artesca_role.writer.arn
  role_session_name = "tf-session"
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | The access key of the account this role belongs to. Sensitive. |
| `account_secret_key` | String | Yes | The secret key of the account this role belongs to. Sensitive. |
| `name` | String | Yes | The name of the IAM role to look up. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `role_id` | The unique ID of the role. |
| `arn` | The ARN of the role. |
| `path` | The path of the role. |
| `assume_role_policy_document` | The JSON trust policy document for the role. |
| `description` | The description of the role. |
