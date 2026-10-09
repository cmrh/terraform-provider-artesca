---
page_title: "artesca_role Data Source - artesca"
subcategory: "Identity"
description: |-
  Looks up an existing IAM role within an ARTESCA account.
---

# Data Source: artesca_role

Looks up an existing IAM role within an ARTESCA account, including its trust policy. Useful for referencing a role that exists outside Terraform without re-creating it.

IAM operations are account-scoped: the provider gets temporary credentials for `account_name` from its OIDC login.

## Example

```hcl
data "artesca_role" "writer" {
  account_name = artesca_account.ops.name
  name         = "object-writer"
}

variable "ci_access_key" {
  type      = string
  sensitive = true
}

variable "ci_secret_key" {
  type      = string
  sensitive = true
}

ephemeral "artesca_assumed_role_credentials" "writer" {
  access_key        = var.ci_access_key
  secret_key        = var.ci_secret_key
  role_arn          = data.artesca_role.writer.arn
  role_session_name = "tf-session"
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_name` | String | Yes | Name of the account to read from. |
| `name` | String | Yes | The name of the IAM role to look up. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `role_id` | The unique ID of the role. |
| `arn` | The ARN of the role. |
| `path` | The path of the role. |
| `assume_role_policy_document` | The JSON trust policy document for the role. |
| `description` | The description of the role. |
