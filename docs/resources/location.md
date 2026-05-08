---
page_title: "artesca_location Resource - artesca"
subcategory: "Infrastructure"
description: |-
  Manages an ARTESCA storage location backed by AWS S3, Azure Blob, GCP Cloud Storage, Scality RING, or other S3-compatible backends.
---

# artesca_location

Manages an ARTESCA storage location. Locations define where data is physically stored -- AWS S3, Azure Blob, GCP Cloud Storage, Scality RING, or other S3-compatible backends.

## Example (AWS S3)

```hcl
resource "artesca_location" "aws_s3" {
  name          = "my-aws-location"
  location_type = "location-aws-s3-v1"

  details {
    access_key             = var.aws_access_key
    secret_key             = var.aws_secret_key
    bucket_name            = "my-target-bucket"
    bucket_match           = true
    region                 = "us-east-1"
    server_side_encryption = false
  }
}
```

## Example (Scality RING S3)

```hcl
resource "artesca_location" "ring_s3" {
  name          = "my-ring-location"
  location_type = "location-scality-ring-s3-v1"

  details {
    access_key  = var.ring_access_key
    secret_key  = var.ring_secret_key
    bucket_name = "ring-bucket"
    endpoint    = "https://ring.internal:8443"
  }
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | String | Yes | Location name. Lowercase alphanumeric and hyphens only. Forces replacement. |
| `location_type` | String | Yes | Backend type (e.g., `location-aws-s3-v1`, `location-azure-v1`, `location-gcp-v1`, `location-scality-ring-s3-v1`). Forces replacement. |
| `is_transient` | Boolean | No | Whether the location is transient. Default: `false`. |
| `legacy_aws_behavior` | Boolean | No | Enable legacy AWS behavior. Default: `false`. |
| `size_limit_gb` | Int | No | Storage size limit in gigabytes. |
| `details` | Block | No | Backend-specific configuration. See below. |

### Details Block

Whether a `details.*` field is required depends on `location_type` -- see [Required fields by location type](#required-fields-by-location-type) below. The fields themselves:

| Name | Type | Description |
|------|------|-------------|
| `access_key` | String | Access key for the backend. Sensitive. |
| `secret_key` | String | Secret key for the backend. Sensitive. |
| `bucket_name` | String | Target bucket on the backend. |
| `bucket_match` | Boolean | Whether to use bucket matching. |
| `endpoint` | String | Custom endpoint URL (for S3-compatible backends). |
| `region` | String | AWS region or equivalent. |
| `server_side_encryption` | Boolean | Enable server-side encryption. |
| `storage_class` | String | Storage class (e.g., `STANDARD`, `GLACIER`). |
| `mpu_bucket_name` | String | Separate bucket for multipart uploads. |
| `username` | String | Username (for Azure or other backends). |
| `password` | String | Password. Sensitive. |
| `tenant_name` | String | Azure tenant name. |
| `subscription_id` | String | Azure subscription ID. |
| `resource_group` | String | Azure resource group. |
| `storage_account_name` | String | Azure storage account name. |
| `storage_container_name` | String | Azure storage container name. |
| `ns_id` | String | Namespace ID (Scality RING). |
| `repo_id` | List(String) | Repository IDs (Scality RING). |
| `proxy_path` | String | Proxy path (NFS/RING). |
| `bootstrap_list` | List(String) | Bootstrap list (Scality RING sproxyd). |
| `chord_cos` | Int | Chord COS (sproxyd). |
| `coding_parts` | Int | Coding parts for erasure coding. |
| `data_parts` | Int | Data parts for erasure coding. |
| `gcp_endpoint` | String | GCP endpoint URL. |
| `bucket_prefix` | String | Bucket prefix. |

### Required fields by location type

The provider validates these requirements at `plan` time. Types not listed below (e.g. `location-mem-v1`, `location-file-v1`) impose no required `details` fields.

| `location_type` | Required `details.*` fields |
|---|---|
| `location-aws-s3-v1` | `access_key`, `secret_key`, `bucket_name` |
| `location-gcp-v1` | `access_key`, `secret_key`, `bucket_name` |
| `location-aws-glacier-v1` | `access_key`, `secret_key`, `bucket_name` |
| `location-azure-v1` | `endpoint`, `bucket_name` |
| `location-azure-archive-v1` | `endpoint`, `bucket_name` |
| `location-wasabi-v1` | `endpoint`, `access_key`, `secret_key`, `bucket_name` |
| `location-do-spaces-v1` | `endpoint`, `access_key`, `secret_key`, `bucket_name` |
| `location-scality-ring-s3-v1` | `endpoint`, `access_key`, `secret_key`, `bucket_name` |
| `location-scality-artesca-s3-v1` | `endpoint`, `access_key`, `secret_key`, `bucket_name` |
| `location-ceph-radosgw-s3-v1` | `endpoint`, `access_key`, `secret_key`, `bucket_name` |
| `location-scality-sproxyd-v1` | `bootstrap_list`, `chord_cos`, `proxy_path` |
| `location-scality-hdclient-v2` | `bootstrap_list` |
| `location-nfs-mount-v1` | `endpoint` |
| `location-dmf-v1` | `endpoint`, `username`, `password`, `repo_id`, `ns_id` |
| `location-miria-v1` | `endpoint`, `username`, `password`, `repo_id` |
| `location-scality-crr-v1` | `endpoint`, `access_key`, `secret_key` |

## Attributes Exported

| Name | Description |
|------|-------------|
| `object_id` | Internal object identifier. |
| `is_builtin` | Whether this is a built-in location. |

## Import

```bash
tofu import artesca_location.aws_s3 my-aws-location
```

After import, sensitive fields (`secret_key`, `password`) will be unknown in state.

## Notes

- `name` and `location_type` force replacement -- locations cannot be renamed or change type.
- The `details` block attributes are backend-specific. Only include fields relevant to your `location_type`.
- Sensitive fields (`secret_key`, `password`) are preserved from state and not re-read from the API.
