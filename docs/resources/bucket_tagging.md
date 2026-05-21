---
page_title: "artesca_bucket_tagging Resource - artesca"
subcategory: "Storage"
description: |-
  Manages the tag set on an ARTESCA bucket.
---

# artesca_bucket_tagging

Manages the tag set on an ARTESCA bucket. Each apply replaces the entire tag set -- there is no key-level merge with externally-applied tags.

## Example

```hcl
resource "artesca_bucket_tagging" "example" {
  account_access_key = artesca_account.example.access_key
  account_secret_key = artesca_account.example.secret_key
  bucket_name        = artesca_bucket.example.name

  tags = {
    environment = "prod"
    team        = "data"
    owner       = "platform"
  }
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | The access key of the account that owns the bucket. Sensitive. Forces replacement. |
| `account_secret_key` | String | Yes | The secret key of the account that owns the bucket. Sensitive. Forces replacement. |
| `bucket_name` | String | Yes | The name of the bucket to tag. Forces replacement. |
| `tags` | Map(String) | Yes | Map of tag key/value pairs. Replacing this map replaces the bucket's entire tag set. |

## Notes

- The S3 PUT bucket-tagging API replaces the full tag set on every call. If another client adds a tag out-of-band, it will be removed on the next Terraform apply.
- Reading is straightforward: drift between configured `tags` and the actual bucket tag set will appear in `terraform plan` and be reconciled.
- Deleting the resource removes all tags from the bucket.
