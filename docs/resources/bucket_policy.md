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
  account_access_key = artesca_account.example.access_key
  account_secret_key = artesca_account.example.secret_key
  bucket_name        = artesca_bucket.example.name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "AllowAccountRead"
        Effect = "Allow"
        # Use the account-root principal form, not `artesca_account.example.arn`.
        # The management API currently returns a malformed ARN (path-style suffix)
        # that the S3 policy validator rejects as MalformedPolicy.
        Principal = { AWS = "arn:aws:iam::${artesca_account.example.id}:root" }
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
| `account_access_key` | String | Yes | The access key of the account that owns the bucket. Sensitive. Forces replacement. |
| `account_secret_key` | String | Yes | The secret key of the account that owns the bucket. Sensitive. Forces replacement. |
| `bucket_name` | String | Yes | The name of the bucket to attach the policy to. Forces replacement. |
| `policy` | String | Yes | The JSON policy document. Whitespace and key-ordering differences are ignored when detecting drift. |

## Notes

- ARTESCA validates the policy on `PUT`. Common rejection: `MalformedPolicy: Policy has invalid resource` when a Resource ARN names a different bucket.
- Anonymous principals (`Principal: "*"`) are accepted.
- The policy field uses semantic JSON comparison for drift detection, so reformatting the policy with `jsonencode` or alternate whitespace will not produce a planned change.
- **Account principals:** Do not use `artesca_account.<x>.arn` for `Principal.AWS`. The management API currently returns a non-standard ARN (`arn:aws:iam::<id>:/<name>/`) that the policy validator rejects with a misleading `MalformedPolicy: This policy contains invalid Json` error. Construct the root form yourself: `"arn:aws:iam::${artesca_account.<x>.id}:root"`.
