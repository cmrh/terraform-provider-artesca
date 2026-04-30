---
page_title: "artesca_bucket_workflow_replication Resource - artesca"
subcategory: "Bucket Workflows"
description: |-
  Manages a bucket-scoped replication workflow for per-account replication in ARTESCA.
---

# artesca_bucket_workflow_replication

Manages a bucket-scoped replication workflow in ARTESCA. This is an account-scoped resource, as opposed to the config-scoped `artesca_replication` resource which operates at the instance level.

~> **Note:** Per-bucket replication only supports `destination.bucket_name`. The `destination.location` and `destination.locations` attributes are not supported for this resource. For location-based replication, use `artesca_replication` instead.

## Example

```hcl
resource "artesca_bucket_workflow_replication" "backup" {
  account_id  = artesca_account.app.id
  bucket_name = "my-source-bucket"
  name        = "replicate-to-backup"
  version     = 1
  enabled     = true

  source {
    bucket_name = "my-source-bucket"
    prefix      = ""
  }

  destination {
    bucket_name = "my-backup-bucket"
  }
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `instance_id` | String | No | Instance UUID. Defaults to the provider's auto-discovered instance ID. Forces replacement. |
| `account_id` | String | Yes | Account ID (from `artesca_account.id`). Forces replacement. |
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

### Destination Block

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `bucket_name` | String | No | Destination bucket name. |
| `preferred_read_location` | String | No | Preferred location for reads. |
| `role` | String | No | IAM role ARN for replication. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `workflow_id` | Unique workflow identifier assigned by the server. |
| `instance_id` | The resolved instance ID (useful when auto-discovered). |

## Import

Import is planned for a future release.

## Notes

- `instance_id`, `account_id`, and `bucket_name` force replacement -- the workflow cannot be moved.
- `destination.location` and `destination.locations` are rejected by validation. Use `destination.bucket_name` only.
- For instance-level, location-based replication, use `artesca_replication` instead.
- Referenced buckets must exist before creating the workflow.
