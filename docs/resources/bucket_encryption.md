---
page_title: "artesca_bucket_encryption Resource - artesca"
subcategory: "Storage"
description: |-
  Manages the server-side encryption configuration of an ARTESCA bucket.
---

# artesca_bucket_encryption

Manages the server-side encryption (SSE) configuration of an ARTESCA bucket via the S3 `PutBucketEncryption` / `GetBucketEncryption` / `DeleteBucketEncryption` APIs.

ARTESCA currently supports **SSE-S3** (`SSEAlgorithm = "AES256"`). SSE-KMS is not yet supported by this resource.

## Example

```hcl
resource "artesca_bucket_encryption" "example" {
  account_access_key = artesca_account.example.access_key
  account_secret_key = artesca_account.example.secret_key
  bucket_name        = artesca_bucket.example.name

  sse_algorithm      = "AES256"
  bucket_key_enabled = false
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | The access key of the account that owns the bucket. Sensitive. Forces replacement. |
| `account_secret_key` | String | Yes | The secret key of the account that owns the bucket. Sensitive. Forces replacement. |
| `bucket_name` | String | Yes | The name of the bucket to configure encryption on. Forces replacement. |
| `sse_algorithm` | String | Yes | The SSE algorithm. Must be `"AES256"`. |
| `bucket_key_enabled` | Bool | Optional | Whether to use an S3 Bucket Key. Defaults to `false` and is also computed from the server. |

## Notes

- The encryption configuration replaces any existing configuration on each apply.
- Deleting the resource removes the encryption configuration from the bucket; objects already written remain unchanged.
- Read returns `nil` when no encryption configuration is set on the bucket (the S3 API surfaces this as `ServerSideEncryptionConfigurationNotFoundError`) — the resource is then removed from Terraform state.
- Whether `bucket_key_enabled = true` is honored depends on the backing storage. Clusters configured purely for transition (no local primary storage) may always report `bucket_key_enabled = false` regardless of the requested value.
