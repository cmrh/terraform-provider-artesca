---
page_title: "artesca_account Data Source - artesca"
subcategory: "Identity"
description: |-
  Looks up an existing ARTESCA account by name.
---

# Data Source: artesca_account

Looks up an existing ARTESCA account by name. Useful for referencing an account that exists outside Terraform without recreating it.

The account's `secret_key` is **not** returned -- the API does not expose it after creation. Only the `access_key` is enumerable from the management view.

## Example

```hcl
data "artesca_account" "ops" {
  name = "operations"
}

output "ops_access_key" {
  value     = data.artesca_account.ops.access_key
  sensitive = true
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | String | Yes | The name of the account to look up. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `id` | The unique account ID. |
| `canonical_id` | The canonical ID of the account. |
| `arn` | The ARN of the account. |
| `email` | The email associated with the account. |
| `access_key` | The account's access key. Sensitive. |
