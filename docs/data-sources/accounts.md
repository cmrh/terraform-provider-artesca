---
page_title: "artesca_accounts Data Source - artesca"
subcategory: "Identity"
description: |-
  Lists all ARTESCA accounts on the cluster.
---

# Data Source: artesca_accounts

Lists every account on the cluster. Useful for inventory, iteration, or filtering. For looking up a single account by name, use [`artesca_account`](account.md) (singular).

## Example — Output every account name

```hcl
data "artesca_accounts" "all" {}

output "account_names" {
  value = [for a in data.artesca_accounts.all.accounts : a.name]
}
```

## Example — Filter by name prefix

```hcl
data "artesca_accounts" "all" {}

locals {
  ops_accounts = [
    for a in data.artesca_accounts.all.accounts : a
    if startswith(a.name, "ops-")
  ]
}
```

## Argument Reference

No arguments.

## Attributes Exported

| Name | Description |
|------|-------------|
| `accounts` | List of account summaries. Each element has: `name`, `id`, `canonical_id`, `arn` (root ARN, `arn:aws:iam::<id>:root`). |

Note: this data source does not return `email`, `access_key`, or `secret_key` — the account listing does not expose them.
