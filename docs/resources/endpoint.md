---
page_title: "artesca_endpoint Resource - artesca"
subcategory: "Infrastructure"
description: |-
  Manages an ARTESCA S3 data service endpoint that maps a hostname to a storage location.
---

# artesca_endpoint

Manages an ARTESCA S3 data service endpoint. Endpoints map a hostname to a storage location, allowing S3 clients to access data at a specific location via the assigned hostname.

## Example

```hcl
resource "artesca_endpoint" "data" {
  hostname      = "data.s3.example.com"
  location_name = artesca_location.aws_s3.name
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `hostname` | String | Yes | Endpoint hostname. Forces replacement. |
| `location_name` | String | Yes | Name of the storage location this endpoint serves. Forces replacement. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `is_builtin` | Whether this is a built-in endpoint. |

## Import

```bash
tofu import artesca_endpoint.data data.s3.example.com
```

## Notes

- Both `hostname` and `location_name` force replacement -- endpoints cannot be renamed or reassigned.
- The referenced location must exist before creating the endpoint.
