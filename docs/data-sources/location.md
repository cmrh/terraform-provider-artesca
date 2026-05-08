---
page_title: "artesca_location Data Source - artesca"
subcategory: "Infrastructure"
description: |-
  Looks up an existing ARTESCA storage location by name.
---

# Data Source: artesca_location

Looks up an existing ARTESCA storage location by name. Useful for referencing a built-in or out-of-band-managed location without redefining it.

Sensitive `details` (`secret_key`, `password`) are **not** returned -- the overlay view masks them.

## Example

```hcl
data "artesca_location" "primary" {
  name = "us-east-1"
}

resource "artesca_bucket" "example" {
  name                = "my-bucket"
  location_constraint = data.artesca_location.primary.name
  account_access_key  = artesca_account.example.access_key
  account_secret_key  = artesca_account.example.secret_key
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | String | Yes | The name of the location to look up. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `location_type` | Backend type (e.g. `location-aws-s3-v1`). |
| `is_transient` | Whether the location is transient. |
| `is_builtin` | Whether the location is a built-in default. |
| `legacy_aws_behavior` | Whether legacy AWS behavior is enabled. |
| `size_limit_gb` | Storage size limit in gigabytes. |
| `object_id` | Internal object identifier. |
| `details` | Backend-specific configuration block. Field set varies by `location_type`. Sensitive fields (`secret_key`, `password`) are blank. |
