---
page_title: "artesca_account Data Source - artesca"
subcategory: "Identity"
description: |-
  Looks up an existing ARTESCA account by name.
---

# Data Source: artesca_account

Looks up an existing ARTESCA account by name. Useful for referencing an account that exists outside Terraform without recreating it.

Email and access keys are **not** returned — the account listing does not expose them. Use the `access_key` / `secret_key` of the `artesca_account` resource that created the account.

## Example

```hcl
data "artesca_account" "ops" {
  name = "operations"
}

output "ops_arn" {
  value = data.artesca_account.ops.arn
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
| `arn` | The root ARN of the account (`arn:aws:iam::<id>:root`). |
