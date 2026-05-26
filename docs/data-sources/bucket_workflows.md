---
page_title: "artesca_bucket_workflows Data Source - artesca"
subcategory: "Storage"
description: |-
  Lists the workflows (replication, lifecycle expiration, transition) configured on an ARTESCA bucket.
---

# Data Source: artesca_bucket_workflows

Lists the workflows configured on an ARTESCA bucket. Returns three lists — `replications`, `expirations`, and `transitions` — each summarizing one workflow.

Backed by the management API endpoint `POST /instance/{instanceId}/account/{accountId}/workflow/search` with the request body `{"bucketList": ["<bucket>"]}`.

## Example

```hcl
data "artesca_bucket_workflows" "audit" {
  account_id  = artesca_account.example.id
  bucket_name = "production-data"
}

output "replication_count" {
  value = length(data.artesca_bucket_workflows.audit.replications)
}

output "expiration_workflow_ids" {
  value = [for e in data.artesca_bucket_workflows.audit.expirations : e.workflow_id]
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `instance_id` | String | Optional | Instance ID. Defaults to the provider's `instance_id` if omitted. |
| `account_id` | String | Yes | The account ID that owns the bucket. |
| `bucket_name` | String | Yes | The bucket whose workflows should be listed. |

## Attributes Exported

`replications` — list of:

| Field | Description |
|---|---|
| `workflow_id` | Workflow (stream) ID. |
| `enabled` | Whether the workflow is enabled. |
| `source_bucket_name` | Source bucket name. |
| `source_prefix` | Object key prefix filter. |
| `source_location` | Source location name. |
| `destination_bucket_name` | Destination bucket name. |
| `destination_location` | Destination location name. |
| `destination_preferred_read_location` | Preferred read location. |
| `destination_role` | IAM role for replication. |

> Note: the workflow-search endpoint does not return `name` or `version` for replication workflows (the server returns `null` for those fields). Use `artesca_bucket_workflow_replication` if you need them.

`expirations` — list of:

| Field | Description |
|---|---|
| `workflow_id` | Workflow ID. |
| `name` | Workflow name. |
| `bucket_name` | Bucket name. |
| `type` | Workflow type identifier. |
| `enabled` | Whether the workflow is enabled. |

`transitions` — list of:

| Field | Description |
|---|---|
| `workflow_id` | Workflow ID. |
| `name` | Workflow name. |
| `bucket_name` | Bucket name. |
| `type` | Workflow type identifier. |
| `enabled` | Whether the workflow is enabled. |
| `location_name` | Destination location for the transition. |
| `apply_to_version` | `"current"` or `"noncurrent"`. |
