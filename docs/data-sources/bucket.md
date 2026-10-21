---
page_title: "artesca_bucket Data Source - artesca"
subcategory: "Storage"
description: |-
  Looks up an existing bucket in an ARTESCA account.
---

# Data Source: artesca_bucket

Looks up an existing bucket in an ARTESCA account and returns its location, versioning, object lock and tags. Useful for referencing a bucket that exists outside Terraform without managing it. Matches `scality_bucket` in the `scality` provider.

The provider gets temporary credentials for `account_name` from its OIDC login.

## Example

```hcl
data "artesca_bucket" "backups" {
  account_name = artesca_account.app.name
  name         = "veeam-backups"
}

check "backups_are_immutable" {
  assert {
    condition     = data.artesca_bucket.backups.object_lock_enabled
    error_message = "The backup bucket must have object lock enabled."
  }
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_name` | String | Yes | Name of the account that owns the bucket. |
| `name` | String | Yes | Name of the bucket to look up. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `arn` | ARN of the bucket (`arn:aws:s3:::<name>`). |
| `location_constraint` | ARTESCA location the bucket stores data in; `us-east-1` for the default location. |
| `versioning_enabled` | Whether versioning is enabled. |
| `object_lock_enabled` | Whether the bucket was created with object lock. |
| `tags` | Map of the bucket's tags; empty when it has none. |

## Notes

- Reading a bucket that doesn't exist in the account is an error.
