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

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | String | Yes | Bucket name: 3–63 characters, lowercase letters, numbers, hyphens, and periods, starting and ending with a letter or number. Forces replacement. |
| `account_name` | String | Yes | Name of the account that owns the resource. Forces replacement. |
| `location_constraint` | String | No | ARTESCA location name to use as the storage backend. Forces replacement. |
| `versioning_enabled` | Boolean | No | Whether versioning is enabled. Default: `false`. Required for replication workflows. |

## Attributes Exported

All arguments are also exported.

## Import

```bash
tofu import artesca_bucket.data <account_name>/<name>
```

The import ID starts with the name of the account that owns the resource.

## Notes

- `name` and `location_constraint` force replacement -- buckets cannot be renamed or relocated.
- `versioning_enabled` can be toggled in-place.
- A bucket must be empty to be deleted, including every object version when versioning is enabled.
- Enable versioning before configuring replication workflows on a bucket.
- Runs as the owning account, with temporary credentials the provider obtains from its OIDC login.
