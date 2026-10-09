---
page_title: "artesca_bucket_encryption Resource - artesca"
subcategory: "Storage"
description: |-
  Manages the server-side encryption configuration of an ARTESCA bucket.
---

# artesca_bucket_encryption

Manages the server-side encryption (SSE) configuration of an ARTESCA bucket via the S3 `PutBucketEncryption` / `GetBucketEncryption` / `DeleteBucketEncryption` APIs.

This provider supports only **SSE-S3** (`AES256`), where ARTESCA owns and manages the master keys. ARTESCA also supports `aws:kms` (account-owned keys, optionally a specific `KMSMasterKeyID`); this provider does not configure it.

This is ARTESCA's own server-side encryption, applied by ARTESCA using the cluster's configured key management backend. It is separate from the `server_side_encryption` option of an `artesca_location`, which asks an external cloud storage provider to encrypt the data it stores.

## Example

```hcl
resource "artesca_bucket_encryption" "example" {
  account_name = artesca_account.example.name
  bucket_name  = artesca_bucket.example.name

  sse_algorithm = "AES256"
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_name` | String | Yes | Name of the account that owns the resource. Forces replacement. |
| `bucket_name` | String | Yes | The name of the bucket to configure encryption on. Forces replacement. |
| `sse_algorithm` | String | Yes | The SSE algorithm. Must be `"AES256"`. |

## Import

```bash
tofu import artesca_bucket_encryption.example <account_name>/<bucket_name>
```

The import ID starts with the name of the account that owns the resource.

## Notes

- The encryption configuration replaces any existing configuration on each apply.
- Deleting the resource removes the encryption configuration from the bucket; objects already written remain unchanged.
- If the encryption configuration is removed outside Terraform, the next refresh removes the resource from state.
- S3 Bucket Keys are not available: ARTESCA does not store the `BucketKeyEnabled` setting.
- Where the master keys are kept depends on the cluster's key management configuration. Without an external KMS (KMIP or AWS KMS-compatible), ARTESCA uses its internal backend, which stores master keys alongside the object metadata and is suitable only for development and testing. Configuring the KMS is a cluster administration task outside this provider.
