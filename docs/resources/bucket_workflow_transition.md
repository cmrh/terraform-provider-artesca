---
page_title: "artesca_bucket_workflow_transition Resource - artesca"
subcategory: "Bucket Workflows"
description: |-
  Manages a bucket lifecycle transition rule in ARTESCA via the S3 API.
---

# artesca_bucket_workflow_transition

Manages one transition rule in a bucket's S3 lifecycle configuration: current versions move to another storage location after a number of days or on a date, or noncurrent versions move a number of days after they become noncurrent. Each resource manages a single rule; several transition and expiration resources can target the same bucket, and the provider merges them into the bucket's lifecycle configuration. Rules created outside Terraform, such as in the ARTESCA UI, are left unchanged. Transition rules appear as transition workflows in the ARTESCA UI.

## Example

```hcl
resource "artesca_bucket_workflow_transition" "archive" {
  account_name       = artesca_account.app.name
  bucket_name        = artesca_bucket.data.name
  enabled            = true
  location_name      = artesca_location.cold.name
  trigger_delay_days = 30

  filter {
    object_key_prefix = "archive/"
  }
}
```

## Example (noncurrent versions)

```hcl
resource "artesca_bucket_workflow_transition" "old_versions" {
  account_name       = artesca_account.app.name
  bucket_name        = artesca_bucket.data.name
  enabled            = true
  apply_to_version   = "noncurrent"
  location_name      = artesca_location.cold.name
  trigger_delay_days = 7
}
```

## Example (date and tag filter)

```hcl
resource "artesca_bucket_workflow_transition" "project" {
  account_name       = artesca_account.app.name
  bucket_name        = artesca_bucket.data.name
  enabled            = true
  location_name      = artesca_location.cold.name
  trigger_delay_date = "2027-01-01"

  filter {
    object_tags = { project = "apollo" }
  }
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_name` | String | Yes | Name of the account that owns the resource. Forces replacement. |
| `bucket_name` | String | Yes | Bucket the rule applies to. Forces replacement. |
| `enabled` | Boolean | Yes | Whether the rule is active. |
| `location_name` | String | Yes | Storage location objects transition to (the lifecycle rule's storage class). Must be an existing location, e.g. `artesca_location.<name>.name`. |
| `apply_to_version` | String | No | `current` (default) transitions current versions; `noncurrent` transitions versions after they become noncurrent. Forces replacement. |
| `trigger_delay_days` | Int | No | Days after object creation (`current`) or after a version becomes noncurrent (`noncurrent`) when it transitions. Zero or more. |
| `trigger_delay_date` | String | No | Date (`YYYY-MM-DD`, midnight UTC) on which current versions transition. Only with `apply_to_version = "current"`. |
| `rule_id` | String | No | Lifecycle rule ID, 1–255 characters. Generated if not set. Forces replacement. |
| `filter` | Block | No | Object filter. See below. |

### Filter Block

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `object_key_prefix` | String | No | Only transition objects whose key starts with this prefix. Omit to apply to all objects. |
| `object_tags` | Map of String | No | Only transition objects that have all of these tags. At least one tag. |

Exactly one of `trigger_delay_days` and `trigger_delay_date` must be set; `noncurrent` requires `trigger_delay_days`. The provider checks this at plan time.

## Attributes Exported

| Name | Description |
|------|-------------|
| `rule_id` | The lifecycle rule ID. |

## Import

```bash
tofu import artesca_bucket_workflow_transition.archive <account_name>/<bucket_name>/<rule_id>
```

The import ID starts with the name of the account that owns the resource.

## Notes

- With `apply_to_version = "current"`, only current versions transition; use a second resource with `noncurrent` to move noncurrent versions.
- `account_name`, `bucket_name`, `rule_id` and `apply_to_version` force replacement — a rule cannot be moved.
- Changes to `enabled`, `location_name`, the trigger days or date, and `filter` are applied in place, and changes made outside Terraform are detected on refresh.
- On import, `apply_to_version` is taken from the rule: `current` if it has a current-version transition, otherwise `noncurrent`. Parts of an imported rule this resource doesn't manage, such as an expiration or the other kind of transition, are kept when it is updated.
- A bucket's lifecycle configuration holds at most 1,000 rules, including rules created outside Terraform.
