---
page_title: "artesca_bucket Resource - artesca"
subcategory: "S3"
description: |-
  Manages an S3 bucket on the ARTESCA S3 endpoint with optional versioning and location constraint.
---

# artesca_bucket

Manages an S3 bucket on the ARTESCA S3 endpoint. Supports versioning and location constraints for controlling which storage backend the bucket uses.

## Example

```hcl
resource "artesca_bucket" "data" {
  account_access_key  = artesca_account.app.access_key
  account_secret_key  = artesca_account.app.secret_key
  name                = "app-data"
  location_constraint = artesca_location.ring_s3.name
  versioning_enabled  = true
}
```

## Example (minimal)

```hcl
resource "artesca_bucket" "logs" {
  account_access_key = artesca_account.app.access_key
  account_secret_key = artesca_account.app.secret_key
  name               = "app-logs"
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | String | Yes | Bucket name. Must be 3-63 characters, lowercase letters, numbers, hyphens, and periods. Forces replacement. |
| `account_access_key` | String | Yes | Access key of the owning account. Sensitive. |
| `account_secret_key` | String | Yes | Secret key of the owning account. Sensitive. |
| `location_constraint` | String | No | ARTESCA location name to use as the storage backend. Forces replacement. |
| `versioning_enabled` | Boolean | No | Whether versioning is enabled. Default: `false`. Required for replication workflows. |

## Attributes Exported

All arguments are also exported.

## Import

Import is planned for a future release.

## Notes

- `name` and `location_constraint` force replacement -- buckets cannot be renamed or relocated.
- `versioning_enabled` can be toggled in-place.
- Enable versioning before configuring replication workflows on a bucket.
- Uses per-account S3 credentials, not provider-level OIDC credentials.
