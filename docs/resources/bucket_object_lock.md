---
page_title: "artesca_bucket_object_lock Resource - artesca"
subcategory: "Storage"
description: |-
  Manages the default retention of an object-lock bucket on ARTESCA.
---

# artesca_bucket_object_lock

Manages the default retention of an object-lock bucket. Every new object version is locked for the configured period. Immutable backup repositories, such as Veeam immutability, use this.

The bucket must be created with `object_lock_enabled = true` on [`artesca_bucket`](bucket.md). Object lock can't be enabled on an existing bucket.

## Example

```hcl
resource "artesca_bucket" "backups" {
  account_name        = artesca_account.app.name
  name                = "veeam-backups"
  object_lock_enabled = true
}

resource "artesca_bucket_object_lock" "backups" {
  account_name   = artesca_account.app.name
  bucket_name    = artesca_bucket.backups.name
  retention_mode = "GOVERNANCE"
  retention_days = 30
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_name` | String | Yes | Name of the account that owns the bucket. Forces replacement. |
| `bucket_name` | String | Yes | Name of the object-lock bucket. Forces replacement. |
| `retention_mode` | String | Yes | `GOVERNANCE`: users with the `s3:BypassGovernanceRetention` permission can delete a locked version or shorten its retention. `COMPLIANCE`: nobody can delete a locked version until its retention ends. |
| `retention_days` | Number | No | Default retention period in days, at least 1. Exactly one of `retention_days` and `retention_years` must be set. |
| `retention_years` | Number | No | Default retention period in years, at least 1. Exactly one of `retention_days` and `retention_years` must be set. |

## Attributes Exported

All arguments are also exported.

## Import

```bash
tofu import artesca_bucket_object_lock.backups <account_name>/<bucket_name>
```

## Notes

- The default retention applies to object versions written after it is set. Existing versions keep their retention.
- Destroying this resource removes the default retention. Object lock stays enabled on the bucket, and locked versions stay locked.
- Applying this resource to a bucket without object lock fails; set `object_lock_enabled = true` on the bucket, which replaces it.
- `artesca_bucket.force_destroy` can delete versions under GOVERNANCE retention, but not versions under COMPLIANCE retention.
- Cross-region replication isn't supported on object-lock buckets.
- Runs as the owning account, with temporary credentials the provider obtains from its OIDC login.
