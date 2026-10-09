---
page_title: "artesca_bucket_workflow_expiration Resource - artesca"
subcategory: "Bucket Workflows"
description: |-
  Manages a bucket lifecycle expiration rule in ARTESCA via the S3 API.
---

# artesca_bucket_workflow_expiration

Manages one expiration rule in a bucket's S3 lifecycle configuration: objects expire a set number of days after creation. Each resource manages a single rule; several expiration and transition resources can target the same bucket, and the provider merges them into the bucket's lifecycle configuration. Rules created outside Terraform, such as in the ARTESCA UI, are left unchanged. Expiration rules appear as expiration workflows in the ARTESCA UI.

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

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_name` | String | Yes | Name of the account that owns the resource. Forces replacement. |
| `bucket_name` | String | Yes | Bucket the rule applies to. Forces replacement. |
| `enabled` | Boolean | Yes | Whether the rule is active. |
| `current_version_trigger_delay_days` | Int | Yes | Days after object creation when the current version expires. Must be a positive integer. |
| `rule_id` | String | No | Lifecycle rule ID, 1–255 characters. Generated if not set. Forces replacement. |
| `filter` | Block | No | Object filter. See below. |

### Filter Block

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `object_key_prefix` | String | No | Only expire objects whose key starts with this prefix. Omit to apply to all objects. |

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
- Changes to `enabled`, `current_version_trigger_delay_days`, and `filter` are applied in place, and changes made outside Terraform are detected on refresh.
- A bucket's lifecycle configuration holds at most 1,000 rules, including rules created outside Terraform.
