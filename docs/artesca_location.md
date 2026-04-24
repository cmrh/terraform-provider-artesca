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

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `access_key` | String | No | Access key for the backend. Sensitive. |
| `secret_key` | String | No | Secret key for the backend. Sensitive. |
| `bucket_name` | String | No | Target bucket on the backend. |
| `bucket_match` | Boolean | No | Whether to use bucket matching. |
| `endpoint` | String | No | Custom endpoint URL (for S3-compatible backends). |
| `region` | String | No | AWS region or equivalent. |
| `server_side_encryption` | Boolean | No | Enable server-side encryption. |
| `storage_class` | String | No | Storage class (e.g., `STANDARD`, `GLACIER`). |
| `mpu_bucket_name` | String | No | Separate bucket for multipart uploads. |
| `username` | String | No | Username (for Azure or other backends). |
| `password` | String | No | Password. Sensitive. |
| `tenant_name` | String | No | Azure tenant name. |
| `subscription_id` | String | No | Azure subscription ID. |
| `resource_group` | String | No | Azure resource group. |
| `storage_account_name` | String | No | Azure storage account name. |
| `storage_container_name` | String | No | Azure storage container name. |
| `ns_id` | String | No | Namespace ID (Scality RING). |
| `repo_id` | List(String) | No | Repository IDs (Scality RING). |
| `proxy_path` | String | No | Proxy path (NFS/RING). |
| `bootstrap_list` | List(String) | No | Bootstrap list (Scality RING sproxyd). |
| `chord_cos` | Int | No | Chord COS (sproxyd). |
| `coding_parts` | Int | No | Coding parts for erasure coding. |
| `data_parts` | Int | No | Data parts for erasure coding. |
| `gcp_endpoint` | String | No | GCP endpoint URL. |
| `bucket_prefix` | String | No | Bucket prefix. |

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
