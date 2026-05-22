---
page_title: "artesca_accounts Data Source - artesca"
subcategory: "Identity"
description: |-
  Lists all ARTESCA accounts on the cluster.
---

# Data Source: artesca_accounts

Lists every account visible on the management overlay. Useful for inventory, iteration, or filtering. For looking up a single account by name, use [`artesca_account`](account.md) (singular) — it returns the `access_key` too, which this data source does not.

## Example — Output every account name

```hcl
data "artesca_accounts" "all" {}

output "account_names" {
  value = [for a in data.artesca_accounts.all.accounts : a.name]
}
```

## Example — Filter by email domain

```hcl
data "artesca_accounts" "all" {}

locals {
  ops_accounts = [
    for a in data.artesca_accounts.all.accounts : a
    if endswith(a.email, "@ops.example.com")
  ]
}
```

## Argument Reference

No arguments.

## Attributes Exported

| Name | Description |
|------|-------------|
| `accounts` | List of account summaries. Each element has: `name`, `id`, `canonical_id`, `arn`, `email`. |

Note: this data source does not return `access_key` or `secret_key` — the management overlay does not include them. Use `data.artesca_account` for that.
