---
page_title: "artesca_bucket_workflow_replication Resource - artesca"
subcategory: "Bucket Workflows"
description: |-
  Manages a bucket replication rule in ARTESCA via the S3 API.
---

# artesca_bucket_workflow_replication

Manages one rule of a bucket's S3 replication configuration, replicating objects from `bucket_name` to `destination_bucket_name`. Each resource manages a single rule; several resources can target the same bucket, and the provider merges them into the bucket's configuration. Rules created outside Terraform, such as in the ARTESCA UI, are left unchanged. Replication rules appear as replication workflows in the ARTESCA UI, identified by their rule ID.

For instance-level, location-based replication, use `artesca_replication` instead.

## Example

```hcl
resource "artesca_bucket_workflow_replication" "backup" {
  account_name            = artesca_account.app.name
  bucket_name             = artesca_bucket.source.name
  destination_bucket_name = artesca_bucket.backup.name
  enabled                 = true

  filter {
    object_key_prefix = "logs/"
  }
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_name` | String | Yes | Name of the account that owns the resource. Forces replacement. |
| `bucket_name` | String | Yes | Source bucket. Versioning must be enabled. Forces replacement. |
| `destination_bucket_name` | String | Yes | Bucket objects are replicated to. Versioning must be enabled. |
| `enabled` | Boolean | Yes | Whether the rule is active. |
| `rule_id` | String | No | Replication rule ID. Generated if not set. Forces replacement. |
| `filter` | Block | No | Object filter. See below. |

### Filter Block

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `object_key_prefix` | String | No | Only replicate objects whose key starts with this prefix. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `rule_id` | The replication rule ID (shown as the workflow ID in the ARTESCA UI). |

## Import

```bash
tofu import artesca_bucket_workflow_replication.backup <account_name>/<bucket_name>/<rule_id>
```

The import ID starts with the name of the account that owns the resource.

## Notes

- `account_name`, `bucket_name`, and `rule_id` force replacement — a rule cannot be moved.
- Changes to `enabled`, `destination_bucket_name`, and `filter` are applied in place, and changes made outside Terraform are detected on refresh.
- Both buckets must exist with versioning enabled before the rule is created.
