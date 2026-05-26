---
page_title: "artesca_policy Data Source - artesca"
subcategory: "Identity"
description: |-
  Looks up an existing IAM managed policy within an ARTESCA account.
---

# Data Source: artesca_policy

Looks up an existing IAM managed policy within an ARTESCA account by ARN, including its current default-version document. Useful for referencing a managed policy that exists outside Terraform without re-creating it.

IAM operations are account-scoped, so you must supply the account's `access_key` and `secret_key` to authenticate the lookup. Policies are looked up by ARN rather than by name to match the IAM API.

## Example

```hcl
data "artesca_policy" "readonly" {
  account_access_key = artesca_account.ops.access_key
  account_secret_key = artesca_account.ops.secret_key
  arn                = "arn:aws:iam::511865208010:policy/readonly"
}

output "readonly_document" {
  value = data.artesca_policy.readonly.policy_document
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | The access key of the account this policy belongs to. Sensitive. |
| `account_secret_key` | String | Yes | The secret key of the account this policy belongs to. Sensitive. |
| `arn` | String | Yes | The ARN of the IAM managed policy to look up. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `name` | The name of the policy. |
| `policy_id` | The unique ID of the policy. |
| `path` | The path of the policy. |
| `default_version_id` | The default version ID of the policy. |
| `description` | The description of the policy. |
| `policy_document` | The JSON policy document for the default version. |
