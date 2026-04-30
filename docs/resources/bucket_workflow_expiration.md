---
page_title: "artesca_bucket_workflow_expiration Resource - artesca"
subcategory: "Bucket Workflows"
description: |-
  Manages a bucket object expiration lifecycle workflow that automatically deletes objects based on age, date, or version criteria.
---

# artesca_bucket_workflow_expiration

Manages a bucket object expiration lifecycle workflow in ARTESCA. Expiration workflows automatically delete objects based on age, date, or version criteria.

## Example

```hcl
resource "artesca_bucket_workflow_expiration" "cleanup" {
  account_id  = artesca_account.app.id
  bucket_name = "app-data"
  name        = "expire-old-objects"
  enabled     = true

  current_version_trigger_delay_days = 90

  filter {
    object_key_prefix = "logs/"

    object_tags {
      key   = "environment"
      value = "staging"
    }
  }
}
```

## Example (delete markers and incomplete uploads)

```hcl
resource "artesca_bucket_workflow_expiration" "maintenance" {
  account_id  = artesca_account.app.id
  bucket_name = "app-data"
  name        = "cleanup-maintenance"
  enabled     = true

  expire_delete_markers_trigger                  = true
  incomplete_multipart_upload_trigger_delay_days  = 7
  previous_version_trigger_delay_days             = 30
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
| `current_version_trigger_delay_date` | String | No | Expire current versions after this date (format: `YYYY-MM-DD`). |
| `current_version_trigger_delay_days` | Int | No | Expire current versions after this many days. |
| `expire_delete_markers_trigger` | Boolean | No | Remove expired delete markers. |
| `incomplete_multipart_upload_trigger_delay_days` | Int | No | Abort incomplete multipart uploads after this many days. |
| `previous_version_trigger_delay_days` | Int | No | Expire previous versions after this many days. |
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
- At least one trigger (`current_version_trigger_delay_days`, `expire_delete_markers_trigger`, etc.) should be set.
