---
page_title: "artesca_bucket_workflow_expiration Resource - artesca"
subcategory: "Bucket Workflows"
description: |-
  Manages a bucket lifecycle expiration rule in ARTESCA via the S3 API.
---

# artesca_bucket_workflow_expiration

Manages one expiration rule in a bucket's S3 lifecycle configuration: current versions expire after a number of days or on a date, noncurrent versions expire after they become noncurrent, expired delete markers are removed, and incomplete multipart uploads are aborted. Each resource manages a single rule; several expiration and transition resources can target the same bucket, and the provider merges them into the bucket's lifecycle configuration. Rules created outside Terraform, such as in the ARTESCA UI, are left unchanged. Expiration rules appear as expiration workflows in the ARTESCA UI.

## Example

```hcl
resource "artesca_bucket_workflow_expiration" "cleanup" {
  account_name = artesca_account.app.name
  bucket_name  = artesca_bucket.data.name
  enabled      = true

  current_version_trigger_delay_days = 90

  filter {
    object_key_prefix = "logs/"
  }
}
```

## Example (versioned bucket housekeeping)

```hcl
resource "artesca_bucket_workflow_expiration" "versions" {
  account_name = artesca_account.app.name
  bucket_name  = artesca_bucket.data.name
  enabled      = true

  previous_version_trigger_delay_days            = 30
  incomplete_multipart_upload_trigger_delay_days = 7
}

resource "artesca_bucket_workflow_expiration" "markers" {
  account_name = artesca_account.app.name
  bucket_name  = artesca_bucket.data.name
  enabled      = true

  expire_delete_markers = true
}
```

## Example (date and tag filter)

```hcl
resource "artesca_bucket_workflow_expiration" "project" {
  account_name = artesca_account.app.name
  bucket_name  = artesca_bucket.data.name
  enabled      = true

  current_version_trigger_delay_date = "2027-01-01"

  filter {
    object_key_prefix = "projects/"
    object_tags       = { project = "apollo" }
  }
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_name` | String | Yes | Name of the account that owns the resource. Forces replacement. |
| `bucket_name` | String | Yes | Bucket the rule applies to. Forces replacement. |
| `enabled` | Boolean | Yes | Whether the rule is active. |
| `current_version_trigger_delay_days` | Int | No | Days after object creation when the current version expires. At least 1. |
| `current_version_trigger_delay_date` | String | No | Date (`YYYY-MM-DD`, midnight UTC) on which current versions expire. |
| `expire_delete_markers` | Boolean | No | Remove delete markers that have no noncurrent versions left behind them. |
| `previous_version_trigger_delay_days` | Int | No | Days after a version becomes noncurrent when it expires. At least 1. |
| `incomplete_multipart_upload_trigger_delay_days` | Int | No | Days after a multipart upload starts when an incomplete upload is aborted and its parts deleted. At least 1. |
| `rule_id` | String | No | Lifecycle rule ID, 1–255 characters. Generated if not set. Forces replacement. |
| `filter` | Block | No | Object filter. See below. |

### Filter Block

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `object_key_prefix` | String | No | Only expire objects whose key starts with this prefix. Omit to apply to all objects. |
| `object_tags` | Map of String | No | Only expire objects that have all of these tags. At least one tag. |

### Constraints

ARTESCA rejects these combinations; the provider reports them at plan time:

- The rule must set at least one action: current-version days or date, `expire_delete_markers = true`, `previous_version_trigger_delay_days`, or `incomplete_multipart_upload_trigger_delay_days`.
- `current_version_trigger_delay_days` and `current_version_trigger_delay_date` conflict.
- `expire_delete_markers` conflicts with the current-version days or date.
- `filter.object_tags` can't be combined with `expire_delete_markers` or `incomplete_multipart_upload_trigger_delay_days`.

## Attributes Exported

| Name | Description |
|------|-------------|
| `rule_id` | The lifecycle rule ID. |

## Import

```bash
tofu import artesca_bucket_workflow_expiration.cleanup <account_name>/<bucket_name>/<rule_id>
```

The import ID starts with the name of the account that owns the resource.

## Notes

- On a bucket that has never had versioning enabled, expiration deletes the object. On a versioned (or versioning-suspended) bucket, it adds a delete marker; noncurrent versions remain.
- `account_name`, `bucket_name`, and `rule_id` force replacement — a rule cannot be moved.
- Changes to `enabled`, the expiration actions, and `filter` are applied in place, and changes made outside Terraform are detected on refresh.
- Parts of an imported rule this resource doesn't manage, such as a transition in the same rule, are kept when it is updated.
- A bucket's lifecycle configuration holds at most 1,000 rules, including rules created outside Terraform.
