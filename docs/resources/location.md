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
    access_key   = var.aws_access_key
    secret_key   = var.aws_secret_key
    bucket_name  = "my-target-bucket"
    bucket_match = true
    region       = "us-east-1"
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
| `name` | String | Yes | Location name: at least 3 characters, lowercase letters, numbers, and hyphens, starting with a letter. Forces replacement. |
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
| `bucket_match` | Boolean | If `true`, objects are written at the root of the target bucket; if `false`, under a prefix named after the source bucket. Defaults to `false`. See the warning below. |
| `endpoint` | String | Custom endpoint URL (for S3-compatible backends). |
| `sts_endpoint` | String | STS endpoint of the destination site. Required for `location-scality-crr-v1`. |
| `region` | String | AWS region or equivalent. |
| `server_side_encryption` | Boolean | Ask Amazon S3 to encrypt stored objects (SSE-S3). Only valid for `location-aws-s3-v1`. |
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

~> **Warning:** With `bucket_match = true`, using the same location for several buckets can lose data, because objects with the same key overwrite each other. ARTESCA rejects a second location on the same endpoint and target bucket unless both have `bucket_match = false`.

### Required fields by location type

The provider validates these requirements at `plan` time for the location types ARTESCA documents. Other types are passed to ARTESCA unchecked.

| `location_type` | Required `details.*` fields |
|---|---|
| `location-aws-s3-v1` | `access_key`, `secret_key`, `bucket_name` |
| `location-gcp-v1` | `access_key`, `secret_key`, `bucket_name` |
| `location-aws-glacier-v1` | `access_key`, `secret_key`, `bucket_name` |
| `location-azure-v1` | `endpoint`, `bucket_name` |
| `location-azure-archive-v1` | `endpoint`, `bucket_name` |
| `location-wasabi-v1` | `endpoint`, `access_key`, `secret_key`, `bucket_name` |
| `location-scality-ring-s3-v1` | `endpoint`, `access_key`, `secret_key`, `bucket_name` |
| `location-scality-artesca-s3-v1` | `endpoint`, `access_key`, `secret_key`, `bucket_name` |
| `location-scality-sproxyd-v1` | `bootstrap_list`, `chord_cos`, `proxy_path` |
| `location-dmf-v1` | `endpoint`, `username`, `password`, `repo_id`, `ns_id` |
| `location-miria-v1` | `endpoint`, `username`, `password`, `repo_id` |
| `location-scaleway-glacier-v1` | `endpoint`, `access_key`, `secret_key`, `bucket_name` |
| `location-ovh-cold-archive-v1` | `endpoint`, `access_key`, `secret_key`, `bucket_name` |
| `location-versity-tape-archive-v1` | `endpoint`, `access_key`, `secret_key`, `bucket_name` |
| `location-scality-crr-v1` | `endpoint`, `sts_endpoint`, `access_key`, `secret_key` |

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
- ARTESCA supports at most 10 storage locations.
- For Amazon S3 and RING S3 locations, the target bucket must have versioning enabled.
- For RING S3, ARTESCA S3, and CRR locations, the remote server's CA certificate must be in the ARTESCA truststore for a TLS connection.
- A location can't be deleted while it holds buckets or objects.
- CRR locations (`location-scality-crr-v1`) are for replication only: they can't back a bucket or be a transition target.
- `us-east-1` is ARTESCA's default location; don't change it.
