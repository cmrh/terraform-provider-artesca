---
page_title: "artesca_locations Data Source - artesca"
subcategory: "Storage"
description: |-
  Lists all ARTESCA storage locations on the cluster.
---

# Data Source: artesca_locations

Lists every storage location on the management overlay (built-in defaults plus user-created). Returns summary metadata only — for full backend-specific details (endpoint, region, bucket_name, etc.) use [`artesca_location`](location.md) (singular) on a specific name.

## Example — Names of all non-builtin locations

```hcl
data "artesca_locations" "all" {}

output "user_locations" {
  value = [
    for l in data.artesca_locations.all.locations : l.name
    if !l.is_builtin
  ]
}
```

## Example — Group by backend type

```hcl
data "artesca_locations" "all" {}

locals {
  by_type = {
    for l in data.artesca_locations.all.locations :
    l.location_type => l.name...
  }
}
```

## Argument Reference

No arguments.

## Attributes Exported

| Name | Description |
|------|-------------|
| `locations` | List of location summaries. Each element has: `name`, `location_type`, `is_builtin`, `is_transient`, `legacy_aws_behavior`, `size_limit_gb`, `object_id`. |

Use `data.artesca_location` with a specific `name` to fetch the nested `details` block (endpoint, credentials are masked, bucket settings, etc.).
