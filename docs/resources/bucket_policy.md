---
page_title: "artesca_bucket_policy Resource - artesca"
subcategory: "Storage"
description: |-
  Attaches an S3 bucket policy to an ARTESCA bucket.
---

# artesca_bucket_policy

Attaches an S3 bucket policy to an ARTESCA bucket. The policy is the standard AWS S3 bucket policy JSON document. ARTESCA validates the policy server-side -- Resource ARNs that don't match the bucket are rejected with `MalformedPolicy`.

## Example

```hcl
resource "artesca_bucket_policy" "example" {
  account_name = artesca_account.example.name
  bucket_name  = artesca_bucket.example.name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "AllowAccountRead"
        Effect = "Allow"
        Principal = { AWS = artesca_account.example.arn }
        Action    = ["s3:GetObject", "s3:ListBucket"]
        Resource = [
          "arn:aws:s3:::${artesca_bucket.example.name}",
          "arn:aws:s3:::${artesca_bucket.example.name}/*",
        ]
      },
      {
        Sid       = "AllowPublicRead"
        Effect    = "Allow"
        Principal = "*"
        Action    = "s3:GetObject"
        Resource  = "arn:aws:s3:::${artesca_bucket.example.name}/*"
      },
    ]
  })
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_name` | String | Yes | Name of the account that owns the resource. Forces replacement. |
| `bucket_name` | String | Yes | The name of the bucket to attach the policy to. Forces replacement. |
| `policy` | String | Yes | The JSON policy document. Whitespace and key-ordering differences are ignored when detecting drift. |

## Import

```bash
tofu import artesca_bucket_policy.example <account_name>/<bucket_name>
```

The import ID starts with the name of the account that owns the resource.

## Notes

- ARTESCA validates the policy on `PUT`. Common rejection: `MalformedPolicy: Policy has invalid resource` when a Resource ARN names a different bucket.
- Anonymous principals (`Principal: "*"`) are accepted.
- The policy field uses semantic JSON comparison for drift detection, so reformatting the policy with `jsonencode` or alternate whitespace will not produce a planned change.
- **Account principals:** `artesca_account.<x>.arn` is the account root ARN (`arn:aws:iam::<id>:root`) and can be used directly in `Principal.AWS`.
- **No encryption condition keys:** ARTESCA rejects the SSE condition keys `s3:x-amz-server-side-encryption` and `s3:x-amz-server-side-encryption-aws-kms-key-id` with `MalformedPolicy`, so a bucket policy cannot require encrypted uploads. Use `artesca_bucket_encryption` to encrypt new objects by default instead.
