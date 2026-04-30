---
page_title: "artesca_replication Resource - artesca"
subcategory: "Replication"
description: |-
  Manages a config-scoped ARTESCA replication stream between two buckets at the instance level.
---

# artesca_replication

Manages a config-scoped ARTESCA replication stream between two buckets. This is a global replication configuration at the instance level, as opposed to the account-scoped `artesca_bucket_workflow_replication` resource.

## Example

```hcl
resource "artesca_replication" "backup" {
  name    = "primary-to-backup"
  enabled = true

  source {
    bucket_name = "my-source-bucket"
    prefix      = ""
  }

  destination {
    bucket_name = "my-backup-bucket"
    location    = artesca_location.backup.name
  }
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | String | Yes | Replication stream name. |
| `enabled` | Boolean | Yes | Whether the replication stream is active. |
| `source` | Block | Yes | Source bucket configuration. See below. |
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
| `locations` | Block List | No | Destination locations with storage class. See below. |
| `preferred_read_location` | String | No | Preferred location for reads. |
| `role` | String | No | IAM role ARN for replication. |

### Destination Locations Block

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | String | Yes | Destination location name. |
| `storage_class` | String | No | Storage class at the destination location. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `stream_id` | Unique replication stream identifier. |
| `version` | Configuration version. Auto-incremented by the server on each update. |

## Import

```bash
tofu import artesca_replication.backup <stream-id>
```

## Notes

- `version` is computed and managed by the server. It starts at 1 on creation and is auto-incremented on each update.
- For account-scoped, per-bucket replication, use `artesca_bucket_workflow_replication` instead.
- The replication stream is read via the config overlay -- there is no dedicated read endpoint.
