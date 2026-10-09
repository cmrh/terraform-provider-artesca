---
page_title: "artesca_location Data Source - artesca"
subcategory: "Infrastructure"
description: |-
  Looks up an existing ARTESCA storage location by name.
---

# Data Source: artesca_location

Looks up an existing ARTESCA storage location by name. Useful for referencing a built-in or out-of-band-managed location without redefining it.

ARTESCA doesn't return `secret_key` or `password`, so they are empty.

## Example

```hcl
data "artesca_location" "primary" {
  name = "us-east-1"
}

resource "artesca_bucket" "example" {
  name                = "my-bucket"
  location_constraint = data.artesca_location.primary.name
  account_name        = artesca_account.example.name
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
| `details` | Backend-specific configuration, with the same fields as the [`artesca_location` details block](../resources/location.md#details-block); which are set depends on `location_type`. `access_key` is sensitive; `secret_key` and `password` are empty. |
