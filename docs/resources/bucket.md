---
page_title: "artesca_bucket Resource - artesca"
subcategory: "Storage"
description: |-
  Manages an S3 bucket on the ARTESCA S3 endpoint with optional versioning and location constraint.
---

# artesca_bucket

Manages an S3 bucket on the ARTESCA S3 endpoint. Supports versioning and location constraints for controlling which storage backend the bucket uses.

## Example

```hcl
resource "artesca_bucket" "data" {
  account_name        = artesca_account.app.name
  name                = "app-data"
  location_constraint = artesca_location.ring_s3.name
  versioning_enabled  = true
}
```

## Example (minimal)

```hcl
resource "artesca_bucket" "logs" {
  account_name = artesca_account.app.name
  name         = "app-logs"
}
```

## Example (object lock)

```hcl
resource "artesca_bucket" "backups" {
  account_name        = artesca_account.app.name
  name                = "veeam-backups"
  object_lock_enabled = true
}

resource "artesca_bucket_object_lock" "backups" {
  account_name   = artesca_account.app.name
  bucket_name    = artesca_bucket.backups.name
  retention_mode = "COMPLIANCE"
  retention_days = 30
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | String | Yes | Bucket name: 3–63 characters, lowercase letters, numbers, hyphens, and periods, starting and ending with a letter or number. Forces replacement. |
| `account_name` | String | Yes | Name of the account that owns the resource. Forces replacement. |
| `location_constraint` | String | No | ARTESCA location name to use as the storage backend. Unset, `""` and `us-east-1` all mean the default location. Forces replacement when the location changes. |
| `versioning_enabled` | Boolean | No | Whether versioning is enabled. Required for replication workflows. When unset, versioning is not managed and the attribute reports the bucket's state; set `false` to suspend versioning. |
| `object_lock_enabled` | Boolean | No | Create the bucket with object lock (WORM). Object lock can only be enabled when the bucket is created; forces replacement. It enables versioning, which can't be suspended afterwards, so `versioning_enabled = false` is rejected with it. Set the default retention with [`artesca_bucket_object_lock`](bucket_object_lock.md). When unset, the attribute reports the bucket's state. |
| `force_destroy` | Boolean | No | Delete every object version and delete marker when the bucket is destroyed. On an object-lock bucket, it bypasses GOVERNANCE retention; a version still under COMPLIANCE retention stops the destroy until its retention ends. Defaults to `false`. Set it and apply before destroying. |

## Attributes Exported

All arguments are also exported.

## Import

```bash
tofu import artesca_bucket.data <account_name>/<name>
```

The import ID starts with the name of the account that owns the resource.

## Notes

- Destroying a bucket that still has objects or object versions fails unless `force_destroy = true`.
- `name`, `location_constraint` and `object_lock_enabled` force replacement -- buckets cannot be renamed or relocated, and object lock can't be added or removed.
- `versioning_enabled` can be toggled in-place, except on an object-lock bucket.
- Cross-region replication isn't supported on object-lock buckets.
- Enable versioning before configuring replication workflows on a bucket.
- Runs as the owning account, with temporary credentials the provider obtains from its OIDC login.
