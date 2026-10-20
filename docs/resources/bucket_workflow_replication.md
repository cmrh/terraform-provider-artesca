---
page_title: "artesca_bucket_workflow_replication Resource - artesca"
subcategory: "Bucket Workflows"
description: |-
  Manages a bucket replication rule in ARTESCA via the S3 API.
---

# artesca_bucket_workflow_replication

Manages one rule of a bucket's S3 replication configuration, replicating objects from `bucket_name` to a storage location (`destination_location`). Each resource manages a single rule; several resources can target the same bucket, and the provider merges them into the bucket's configuration. Rules created outside Terraform, such as in the ARTESCA UI, are left unchanged. Replication rules appear as replication workflows in the ARTESCA UI, identified by their rule ID.

For instance-level replication streams, use `artesca_replication` instead.

## Example

A bucket on ARTESCA's own storage (`us-east-1`) replicating to a RING location:

```hcl
resource "artesca_bucket_workflow_replication" "offsite" {
  account_name         = artesca_account.app.name
  bucket_name          = artesca_bucket.data.name
  destination_location = artesca_location.ring.name
  enabled              = true
}
```

Objects land in the location's target bucket, under a prefix named after the source bucket unless the location sets `bucket_match = true`.

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_name` | String | Yes | Name of the account that owns the resource. Forces replacement. |
| `bucket_name` | String | Yes | Source bucket. Versioning must be enabled. Forces replacement. |
| `destination_location` | String | Yes | Location objects are replicated to. For RING S3 and cloud locations, objects go to the location's target bucket. |
| `destination_bucket_name` | String | No | For a cross-region replication (CRR) `destination_location`, the bucket on the remote site. Versioning must be enabled. |
| `destination_role` | String | No | For a CRR `destination_location`, the ARN of the IAM role on the remote site that replication assumes. Required for a CRR location and not allowed for other locations. |
| `enabled` | Boolean | Yes | Whether the rule is active. |
| `rule_id` | String | No | Replication rule ID, 1–255 characters. Generated if not set. Forces replacement. |
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
- Changes to `enabled`, `destination_location`, `destination_bucket_name`, `destination_role`, and `filter` are applied in place, and changes made outside Terraform are detected on refresh.
- The source bucket must have versioning enabled before the rule is created.
- ARTESCA rejects a rule naming a location created moments earlier; the provider retries for up to 5 minutes while the location propagates.
- The bucket's replication configuration has one `Role` for all its rules. While any rule targets a CRR location, the provider writes the source role and `destination_role`; when the last CRR rule is removed, it drops `destination_role`. All CRR rules on a bucket must use the same `destination_role`.
- ARTESCA requires every rule on a bucket to name the same destination bucket. When a bucket mixes CRR and other rules, set `destination_bucket_name` on every rule to the CRR destination bucket; RING S3 and cloud locations ignore it and write to their own target bucket.
