---
page_title: "artesca_bucket_workflow_transition Resource - artesca"
subcategory: "Bucket Workflows"
description: |-
  Manages a bucket object transition lifecycle workflow that automatically moves objects to a different storage location.
---

# artesca_bucket_workflow_transition

Manages a bucket object transition lifecycle workflow in ARTESCA. Transition workflows automatically move objects to a different storage location based on age or date criteria.

## Example

```hcl
resource "artesca_bucket_workflow_transition" "archive" {
  account_id         = artesca_account.app.id
  bucket_name        = "app-data"
  name               = "archive-to-cold"
  enabled            = true
  location_name      = artesca_location.cold_storage.name
  apply_to_version   = "current"
  trigger_delay_days = 30

  filter {
    object_key_prefix = "archive/"
  }
}
```

## Example (noncurrent versions)

```hcl
resource "artesca_bucket_workflow_transition" "old_versions" {
  account_id         = artesca_account.app.id
  bucket_name        = "versioned-data"
  name               = "move-old-versions"
  enabled            = true
  location_name      = artesca_location.archive.name
  apply_to_version   = "noncurrent"
  trigger_delay_days = 90
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `instance_id` | String | No | Instance UUID. Defaults to the provider's auto-discovered instance ID. Forces replacement. |
| `account_id` | String | Yes | Account ID (from `artesca_account.id`). Forces replacement. |
| `bucket_name` | String | Yes | Target bucket name. Forces replacement. |
| `name` | String | No | Workflow name. |
| `enabled` | Boolean | Yes | Whether the workflow is active. |
| `location_name` | String | Yes | Destination storage location for transitioned objects. |
| `apply_to_version` | String | Yes | Which object versions to transition: `current` or `noncurrent`. |
| `trigger_delay_date` | String | No | Transition objects after this date (format: `YYYY-MM-DD`). |
| `trigger_delay_days` | Int | No | Transition objects after this many days. |
| `filter` | Block | No | Object filter criteria. See below. |

### Filter Block

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `object_key_prefix` | String | No | Only apply to objects matching this key prefix. |
| `object_tags` | Block (list) | No | Only apply to objects with these tags. See below. |

### Object Tags Block

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `key` | String | Yes | Tag key. |
| `value` | String | Yes | Tag value. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `workflow_id` | Unique workflow identifier assigned by the server. |
| `instance_id` | The resolved instance ID (useful when auto-discovered). |

## Import

Import is planned for a future release.

## Notes

- `instance_id`, `account_id`, and `bucket_name` force replacement -- the workflow cannot be moved.
- Exactly one of `trigger_delay_date` or `trigger_delay_days` should typically be set.
- The destination `location_name` must reference an existing storage location.
