---
page_title: "artesca_endpoints Data Source - artesca"
subcategory: "Infrastructure"
description: |-
  Lists all ARTESCA S3 endpoints on the cluster.
---

# Data Source: artesca_endpoints

Lists every S3 endpoint (a hostname mapped to a location) on the cluster. Includes built-in cluster endpoints (e.g. `s3.<cluster>`) alongside user-created ones.

## Example — List all non-builtin endpoints

```hcl
data "artesca_endpoints" "all" {}

output "user_endpoints" {
  value = [
    for e in data.artesca_endpoints.all.endpoints : e.hostname
    if !e.is_builtin
  ]
}
```

## Example — Group endpoints by storage location

```hcl
data "artesca_endpoints" "all" {}

locals {
  endpoints_by_location = {
    for e in data.artesca_endpoints.all.endpoints :
    e.location_name => e.hostname...
  }
}
```

## Argument Reference

No arguments.

## Attributes Exported

| Name | Description |
|------|-------------|
| `endpoints` | List of endpoint summaries. Each element has: `hostname`, `location_name`, `is_builtin`. |
