# artesca_bucket_workflow_replication

Manages a bucket-scoped replication workflow in ARTESCA. This is an account-scoped V2 resource, as opposed to the config-scoped `artesca_replication` resource which operates at the instance level.

## Example

```hcl
resource "artesca_bucket_workflow_replication" "backup" {
  account_id  = artesca_account.app.canonical_id
  bucket_name = "my-source-bucket"
  name        = "replicate-to-backup"
  version     = 1
  enabled     = true

  source {
    bucket_name = "my-source-bucket"
    prefix      = ""
  }

  destination {
    locations {
      name = artesca_location.backup.name
    }
  }
}
```

## Example (multi-destination with prefix filter)

```hcl
resource "artesca_bucket_workflow_replication" "multi" {
  account_id  = artesca_account.app.canonical_id
  bucket_name = "production-data"
  name        = "multi-site-replication"
  version     = 1
  enabled     = true

  source {
    bucket_name = "production-data"
    prefix      = "critical/"
  }

  destination {
    locations {
      name          = artesca_location.site_b.name
      storage_class = "STANDARD"
    }
    locations {
      name          = artesca_location.site_c.name
      storage_class = "STANDARD"
    }
  }
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `instance_id` | String | No | Instance UUID. Defaults to the provider's auto-discovered instance ID. Forces replacement. |
| `account_id` | String | Yes | Account canonical ID (from `artesca_account.canonical_id`). Forces replacement. |
| `bucket_name` | String | Yes | Source bucket name. Forces replacement. |
| `name` | String | Yes | Replication workflow name. |
| `version` | Int | Yes | Configuration version. |
| `enabled` | Boolean | Yes | Whether the replication workflow is active. |
| `source` | Block | Yes | Source configuration. See below. |
| `destination` | Block | Yes | Destination configuration. See below. |

### Source Block

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `bucket_name` | String | Yes | Source bucket name. |
| `prefix` | String | Yes | Object key prefix filter. Use `""` for all objects. |
| `location` | String | No | Source location name. |

### Destination Block

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `bucket_name` | String | No | Destination bucket name. |
| `location` | String | No | Destination location name. |
| `preferred_read_location` | String | No | Preferred location for reads. |
| `role` | String | No | IAM role ARN for replication. |
| `locations` | Block (list) | No | One or more destination locations. See below. |

### Locations Block

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | String | Yes | Destination location name. |
| `storage_class` | String | No | Storage class at the destination. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `workflow_id` | Unique workflow identifier assigned by the server. |
| `instance_id` | The resolved instance ID (useful when auto-discovered). |

## Notes

- `instance_id`, `account_id`, and `bucket_name` force replacement -- the workflow cannot be moved.
- There is no read API for workflows. State is preserved as-is between applies.
- For instance-level replication, use `artesca_replication` instead.
- Referenced locations must exist before creating the workflow.
- Import is not supported.
