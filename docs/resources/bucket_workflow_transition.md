---
page_title: "artesca_bucket_workflow_transition Resource - artesca"
subcategory: "Bucket Workflows"
description: |-
  Manages a bucket lifecycle transition rule in ARTESCA via the S3 API.
---

# artesca_bucket_workflow_transition

Manages one transition rule in a bucket's S3 lifecycle configuration: objects move to another storage location a set number of days after creation. Each resource manages a single rule; several transition and expiration resources can target the same bucket, and the provider merges them into the bucket's lifecycle configuration. Rules created outside Terraform, such as in the ARTESCA UI, are left unchanged. Transition rules appear as transition workflows in the ARTESCA UI.

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

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_name` | String | Yes | Name of the account that owns the resource. Forces replacement. |
| `bucket_name` | String | Yes | Bucket the rule applies to. Forces replacement. |
| `enabled` | Boolean | Yes | Whether the rule is active. |
| `location_name` | String | Yes | Storage location objects transition to (the lifecycle rule's storage class). Must be an existing location, e.g. `artesca_location.<name>.name`. |
| `trigger_delay_days` | Int | Yes | Days after object creation when the object transitions. Must be zero or a positive integer. |
| `rule_id` | String | No | Lifecycle rule ID, 1–255 characters. Generated if not set. Forces replacement. |
| `filter` | Block | No | Object filter. See below. |

### Filter Block

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `object_key_prefix` | String | No | Only transition objects whose key starts with this prefix. Omit to apply to all objects. |

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

- On a versioned (or versioning-suspended) bucket, only the current version transitions; noncurrent versions are unaffected.
- `account_name`, `bucket_name`, and `rule_id` force replacement — a rule cannot be moved.
- Changes to `enabled`, `location_name`, `trigger_delay_days`, and `filter` are applied in place, and changes made outside Terraform are detected on refresh.
- A bucket's lifecycle configuration holds at most 1,000 rules, including rules created outside Terraform.
